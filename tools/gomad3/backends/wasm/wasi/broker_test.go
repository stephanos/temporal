package wasi

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/world"
	"go.temporal.io/server/tools/gomad3sim"
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

func TestBrokerCompressedWorldProfileAdmissionAndTerminalOwnership(t *testing.T) {
	config := BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, WorldProfile: world.CompressedSnapshotSchema, Limits: world.CompressedSnapshotProfile().Limits}
	oversize := config
	oversize.Limits.MaxRequests++
	if broker, err := NewBroker(oversize); !errors.Is(err, world.ErrCapacity) || broker != nil {
		t.Fatalf("oversize configured profile admitted: %v", err)
	}
	unknown := config
	unknown.WorldProfile = "unknown"
	if broker, err := NewBroker(unknown); err == nil || broker != nil {
		t.Fatal("unknown World profile admitted")
	}
	b, err := NewBroker(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := b.EncodeWorldEvidence(); err == nil {
		t.Fatal("live participant World was detached")
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	before := b.DiagnosticCardinality()
	data, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeCompressedSnapshot(data, world.CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
	if b.DiagnosticCardinality() != before {
		t.Fatal("terminal ownership transfer changed diagnostic cardinality")
	}
	if _, err := b.EncodeWorldEvidence(); err == nil {
		t.Fatal("terminal World encoded twice")
	}
	if _, err := b.Restart("a"); err == nil {
		t.Fatal("terminal detached World restarted")
	}
}

func TestBrokerAbsentWorldProfileUsesBinaryOwnerCodec(t *testing.T) {
	b, err := NewBroker(BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, Limits: world.CompressedSnapshotProfile().Limits})
	if err != nil {
		t.Fatal(err)
	}
	if err := b.Session("a").Finished(); err != nil {
		t.Fatal(err)
	}
	data, err := b.EncodeWorldEvidence()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := world.DecodeCompressedSnapshot(data, world.CompressedSnapshotProfile()); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerImplicitProfileAdmitsDefaultSimulationLimits(t *testing.T) {
	simulation := gomad3sim.DefaultLimits()
	limits := world.Limits{
		MaxRequests: simulation.NetworkTransitions * 2, MaxEvents: simulation.NetworkTransitions * 2,
		MaxQueuedEvents: simulation.NetworkDeliveries, MaxTransitions: simulation.NetworkTransitions * 8,
		MaxPayloadBytes: simulation.ObservationBytes, MaxStringBytes: 1024,
	}
	if _, err := NewBroker(BrokerConfig{Nodes: []BrokerNode{{ID: "a", Incarnation: 1}}, Limits: limits}); err != nil {
		t.Fatalf("implicit World profile rejected default simulation limits: %v", err)
	}
}
