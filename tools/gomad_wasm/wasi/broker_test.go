package wasi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/volume"
	netmodel "go.temporal.io/server/tools/gomad3/simulation/network"
	"go.temporal.io/server/tools/gomad3/world"
)

func brokerTestConfig() BrokerConfig {
	return BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}, {ID: "b", Incarnation: 1}}, Seed: 7, Limits: world.Limits{MaxRequests: 10000, MaxEvents: 10000, MaxQueuedEvents: 1000, MaxTransitions: 40000, MaxPayloadBytes: 32 << 20, MaxStringBytes: 1024}}
}

func TestBrokerBoundedDiagnosticsAndAtomicWorldCapacity(t *testing.T) {
	config := brokerTestConfig()
	config.Diagnostics = true
	config.Limits.MaxTransitions = 10000
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	request := world.Request{Kind: "diagnostic", Resource: world.ResourceID{Adapter: "broker", Kind: "call", Key: "a"}}
	for i := 0; i < 10000; i++ {
		if _, err := b.world.Register(request); err != nil {
			t.Fatal(err)
		}
	}
	before := b.DiagnosticCardinality()
	if _, err := b.world.Register(request); !errors.Is(err, world.ErrCapacity) {
		t.Fatalf("world transition capacity did not fail typed: %v", err)
	}
	if after := b.DiagnosticCardinality(); after != before {
		t.Fatal("capacity rejection or cardinality read mutated World")
	}
	data := b.Diagnostics()
	if len(data) == 0 || len(data) > 2048 || before.Requests != 10000 || before.Transitions != 10000 {
		t.Fatal("bounded infrastructure diagnostic expanded full canonical World")
	}
	if after := b.DiagnosticCardinality(); after != before {
		t.Fatal("live diagnostic changed global order or time")
	}
}

func TestBrokerPreparationCancellationAndFailureRelease(t *testing.T) {
	b, err := NewBroker(brokerTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	a, peer := b.Session("a"), b.Session("b")
	if err := peer.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := a.Prepare(ctx); err != context.Canceled {
		t.Fatalf("canceled preparation: %v", err)
	}
	if b.preparing != peer || b.active != a {
		t.Fatal("preparation changed visible continuation ordering")
	}
	peer.Prepared()
	peer.Prepared()
	if err := a.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := a.Finished(); err != nil {
		t.Fatal(err)
	}
	if b.preparing != nil {
		t.Fatal("failed helper retained compilation slot")
	}
	if err := peer.Prepare(t.Context()); err != nil {
		t.Fatal(err)
	}
	peer.Prepared()
	if err := peer.Activate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := a.Prepare(t.Context()); err == nil {
		t.Fatal("stale incarnation obtained preparation slot")
	}
}

func TestBrokerRevokedInFlightCompletionRetainsTypedBoundary(t *testing.T) {
	b, err := NewBroker(brokerTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	s := b.Session("a")
	s.request = 1
	if err := b.Stop("a"); err != nil {
		t.Fatal(err)
	}
	var boundary *BoundaryError
	if err := s.Delivered(Call{ID: 1, Op: "fd_write"}); !errors.As(err, &boundary) || boundary.Kind != "stop" {
		t.Fatalf("revoked in-flight reply lost typed interruption: %v", err)
	}
	for _, op := range []string{"runtime_idle", "runtime_net_stream"} {
		if err := s.Delivered(Call{ID: 1, Op: op}); !errors.As(err, &boundary) || boundary.Kind != "stop" {
			t.Fatalf("revoked %s completion lost typed interruption: %v", op, err)
		}
	}
	if err := s.beginCall(&Environment{}, Call{ID: 2, Op: "fd_write"}); !errors.As(err, &boundary) {
		t.Fatalf("revoked request lost boundary: %v", err)
	}
	if err := s.finishCall(&Environment{}, Call{ID: 1, Op: "runtime_net_stream"}); !errors.As(err, &boundary) {
		t.Fatalf("revoked readiness lost boundary: %v", err)
	}
	if _, err := s.prepareIdle(1000); !errors.As(err, &boundary) || s.state != "revoked" {
		t.Fatalf("revoked participant became idle: state=%s err=%v", s.state, err)
	}
	if err := s.Finished(); err != nil {
		t.Fatal(err)
	}
	if err := s.Delivered(Call{ID: 1, Op: "runtime_idle"}); !errors.As(err, &boundary) || boundary.Kind != "invalid" {
		t.Fatalf("finished incarnation retained live boundary: %v", err)
	}
}

func TestBrokerRevocationReleasesSurvivorProgressAfterReap(t *testing.T) {
	for _, kind := range []string{"crash", "stop"} {
		t.Run(kind, func(t *testing.T) {
			b, err := NewBroker(brokerTestConfig())
			if err != nil {
				t.Fatal(err)
			}
			old := b.Session("a")
			if kind == "crash" {
				err = b.Crash("a")
			} else {
				err = b.Stop("a")
			}
			if err != nil {
				t.Fatal(err)
			}
			if b.Current() != 1 || b.active != old {
				t.Fatal("revocation released continuation before helper reap")
			}
			if err := old.Finished(); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			now, err := b.Session("b").idle(ctx, 1000)
			if err != nil || now != 1000 {
				t.Fatalf("surviving timer blocked without restart: now=%d err=%v", now, err)
			}
		})
	}
}

func TestBrokerCrashRecoveryFreshGuest(t *testing.T) {
	request := cooperativeRequest(t, "../simulation/testdata/service")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.Storage = &StorageModel{Volumes: []volume.VolumeConfig{{ID: "a-store", Path: "/workspace", CapacityBytes: 8 << 20}}, Limits: volume.VolumeLimits{PendingOperations: 1024, Transitions: 10000}}
	run := func() ([]Result, world.Snapshot) {
		b, err := NewBroker(brokerTestConfig())
		if err != nil {
			t.Fatal(err)
		}
		if err = b.Session("a").CrashOnOutput([]byte("committed durable-a")); err != nil {
			t.Fatal(err)
		}
		peer := request
		peer.Config = request.Config
		peer.Config.Storage = nil
		peer.Config.Args = []string{"service", "park"}
		peer.Broker = b.Session("b")
		peerDone := make(chan Result, 1)
		peerErr := make(chan error, 1)
		go func() { result, err := Run(t.Context(), peer); peerDone <- result; peerErr <- err }()
		first := request
		first.Config = request.Config
		first.Config.Args = []string{"service", "durable"}
		first.Broker = b.Session("a")
		crashed, err := Run(t.Context(), first)
		if err != nil || crashed.Termination != "crash" || crashed.StorageImage == nil || !crashed.Reaped {
			t.Fatalf("crash: %s %s err=%v stdout=%s stderr=%s", crashed.Termination, crashed.Message, err, crashed.Stdout, crashed.Stderr)
		}
		session, err := b.Restart("a")
		if err != nil {
			t.Fatal(err)
		}
		restart := request
		restart.Config = request.Config
		restart.Config.Args = []string{"service", "recover"}
		restart.Broker = session
		restart.StorageImage = crashed.StorageImage
		recovered, err := Run(t.Context(), restart)
		if err != nil || recovered.Termination != "exit" || !recovered.Reaped || !bytes.Contains(recovered.Stdout, []byte("fresh heap 1\nrecovered durable-a")) {
			t.Fatalf("recover: %s %s %v stdout=%s stderr=%s", recovered.Termination, recovered.Message, err, recovered.Stdout, recovered.Stderr)
		}
		other := <-peerDone
		if err = <-peerErr; err != nil || other.Termination != "exit" || !other.Reaped {
			t.Fatalf("peer: %s %s %v", other.Termination, other.Message, err)
		}
		return []Result{crashed, recovered, other}, b.Snapshot()
	}
	left, leftOrder := run()
	right, rightOrder := run()
	for i := range left {
		if !bytes.Equal(left[i].Stdout, right[i].Stdout) || !bytes.Equal(left[i].Transcript, right[i].Transcript) || !bytes.Equal(left[i].StorageTranscript, right[i].StorageTranscript) {
			t.Fatalf("crash/recover node incarnation %d differs", i)
		}
	}
	if !reflect.DeepEqual(leftOrder, rightOrder) {
		t.Fatal("crash recovery global event order differs")
	}
}

func TestBrokerStaleLifecycleAndContinuationGate(t *testing.T) {
	b, err := NewBroker(brokerTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	a, old := b.Session("a"), b.Session("b")
	if err = a.Activate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = b.Crash("b"); err != nil {
		t.Fatal(err)
	}
	if err = old.Activate(t.Context()); err == nil {
		t.Fatal("stale incarnation authorized")
	}
	if err = old.Finished(); err != nil {
		t.Fatal(err)
	}
	next, err := b.Restart("b")
	if err != nil || next.Node.Incarnation != 2 {
		t.Fatalf("restart %v %v", next, err)
	}
	if err = old.Delivered(Call{ID: 1}); err == nil {
		t.Fatal("stale completion accepted")
	}
	if err = a.Finished(); err != nil {
		t.Fatal(err)
	}
	if err = next.Activate(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = next.Finished(); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerIndependentGuestStreamsReplay(t *testing.T) {
	request := cooperativeRequest(t, "../simulation/testdata/service")
	request.Config.Clock.ReadStepNanos = 0
	request.Config.NetworkV2 = &NetworkModelV2{Profile: NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 128, Connections: 128, Deliveries: 1024, Bytes: 8 << 20, Transitions: 10000}}
	request.Config.Limits.Calls = 100000
	request.Config.Limits.TranscriptBytes = 32 << 20
	request.Runtime.Choice.Limit = 8 << 20
	run := func(tapes []*choice.ReplayPlan) ([]Result, world.Snapshot) {
		b, err := NewBroker(func() BrokerConfig {
			c := brokerTestConfig()
			c.Network = request.Config.NetworkV2
			c.Faults = []BrokerNetworkFault{{Name: "partition", Action: "partition", From: "a", To: "b", AtNanos: 1, Symmetric: true}, {Name: "heal", Action: "heal", From: "a", To: "b", AtNanos: 1_000_000_001, Symmetric: true}}
			return c
		}())
		if err != nil {
			t.Fatal(err)
		}
		results := make([]Result, 2)
		errs := make([]error, 2)
		var wg sync.WaitGroup
		for index, id := range []string{"a", "b"} {
			r := request
			r.Config = request.Config
			r.Runtime = &RuntimeControl{Seed: request.Runtime.Seed, Diagnostics: true, Choice: new(*request.Runtime.Choice)}
			r.Broker = b.Session(id)
			if index == 0 {
				r.Config.Args = []string{"service", "server"}
			} else {
				r.Config.Args = []string{"service", "client", b.Address("a")}
			}
			if tapes != nil {
				r.Runtime.Choice.Mode = choice.ModeReplay
				r.Runtime.Choice.Tape = tapes[index]
			}
			wg.Add(1)
			go func() { defer wg.Done(); results[index], errs[index] = Run(context.Background(), r) }()
		}
		wg.Wait()
		for i, result := range results {
			if errs[i] != nil || result.Termination != "exit" || result.ExitCode == nil || *result.ExitCode != 0 || !result.Reaped {
				t.Fatalf("node %d: %s %s err=%v stdout=%s stderr=%s", i, result.Termination, result.Message, errs[i], result.Stdout, result.Stderr)
			}
			if !bytes.Contains(result.Stdout, []byte("fresh heap 1")) || !bytes.Contains(result.Stdout, []byte("ordered bytes ordered-byte")) {
				t.Fatalf("node %d witnesses: %s", i, result.Stdout)
			}
		}
		return results, b.Snapshot()
	}
	observed, order := run(nil)
	tapes := make([]*choice.ReplayPlan, 2)
	for i, result := range observed {
		plan, err := choice.ProjectReplayPlan(result.ChoiceTrace, request.Runtime.Choice.ExecutionIdentity)
		if err != nil {
			t.Fatal(err)
		}
		tapes[i] = &plan
	}
	replayed, replayOrder := run(tapes)
	for i := range observed {
		if !bytes.Equal(observed[i].Stdout, replayed[i].Stdout) || !bytes.Equal(observed[i].Transcript, replayed[i].Transcript) || !bytes.Equal(observed[i].NetworkTranscript, replayed[i].NetworkTranscript) {
			t.Fatalf("node %d fresh replay differs", i)
		}
	}
	if !reflect.DeepEqual(order, replayOrder) {
		left, _ := json.Marshal(order)
		right, _ := json.Marshal(replayOrder)
		t.Fatalf("global broker event order replay differs: %s / %s", left, right)
	}
}
