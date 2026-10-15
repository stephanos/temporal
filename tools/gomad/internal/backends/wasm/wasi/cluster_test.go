package wasi

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

type clusterTestControl struct {
	Type        string `json:"type"`
	ID          uint64 `json:"id"`
	Node        string `json:"node"`
	Incarnation uint64 `json:"incarnation"`
	Command     string `json:"command"`
}

func clusterControlSession(t *testing.T) (*ClusterSession, <-chan clusterTestControl) {
	t.Helper()
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	input.start(nil)
	ctx, cancel := context.WithCancel(t.Context())
	session := &ClusterSession{states: map[BrokerNode]*clusterState{}, current: map[string]uint64{}, pending: map[uint64]chan error{}, input: input, ctx: ctx, cancel: cancel, done: make(chan struct{})}
	controls := make(chan clusterTestControl, 16)
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		stream := &clusterStream{session: session}
		scanner := bufio.NewScanner(input.reader)
		for scanner.Scan() {
			var control clusterTestControl
			if err := json.Unmarshal(scanner.Bytes(), &control); err != nil {
				t.Error(err)
				return
			}
			if control.ID == 0 {
				continue
			}
			ack, err := json.Marshal(map[string]any{"type": "control_result", "id": control.ID, "termination": "", "message": ""})
			if err == nil {
				err = stream.frame(ack)
			}
			if err != nil {
				t.Error(err)
				return
			}
			controls <- control
		}
	}()
	t.Cleanup(func() {
		cancel()
		session.mu.Lock()
		for _, state := range session.states {
			select {
			case <-state.done:
			default:
				close(state.done)
			}
		}
		session.mu.Unlock()
		input.closeInput()
		<-readDone
		if err := input.finish(); err != nil {
			t.Error(err)
		}
	})
	return session, controls
}

func nextClusterControl(t *testing.T, controls <-chan clusterTestControl) clusterTestControl {
	t.Helper()
	select {
	case control := <-controls:
		return control
	case <-time.After(5 * time.Second):
		t.Fatal("cluster control did not arrive")
		return clusterTestControl{}
	}
}

func TestClusterStartAndObserveLeaveControllerPaused(t *testing.T) {
	session, controls := clusterControlSession(t)
	config := testConfig()
	config.Profile = CooperativeProfile
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	runtime := &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	session.boots = map[string]*clusterBoot{"boot": {request: Request{Config: config}, started: &Started{}}}
	if err := session.Start(t.Context(), "a", 1, "boot", runtime); err != nil {
		t.Fatal(err)
	}
	if control := nextClusterControl(t, controls); control.Type != "start" {
		t.Fatalf("Start drove a guest: %+v", control)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := session.Observe(ctx, "a", 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("passive observer returned %v", err)
	}
	if count := session.FrameCounts().Controls; count != 1 {
		t.Fatalf("passive observation sent %d controls", count)
	}
	close(session.states[BrokerNode{ID: "a", Incarnation: 1}].done)
	if err := session.Start(t.Context(), "a", 2, "boot", runtime); err != nil {
		t.Fatal(err)
	}
	if control := nextClusterControl(t, controls); control.Type != "restart" {
		t.Fatalf("Restart drove a guest: %+v", control)
	}
	if count := session.FrameCounts().Controls; count != 2 {
		t.Fatalf("restart emitted extra controls: %d", count)
	}
}

func TestClusterWaitSerializesTurnsWithoutBlockingLiveControls(t *testing.T) {
	session, controls := clusterControlSession(t)
	for _, node := range []string{"a", "b", "cached"} {
		session.states[BrokerNode{ID: node, Incarnation: 1}] = &clusterState{done: make(chan struct{})}
	}
	complete := func(node string) {
		session.mu.Lock()
		defer session.mu.Unlock()
		state := session.states[BrokerNode{ID: node, Incarnation: 1}]
		state.result = Result{Termination: "stop", Reaped: true}
		close(state.done)
	}
	complete("cached")
	observed := make(chan error, 1)
	go func() { _, err := session.Observe(t.Context(), "a", 1); observed <- err }()
	ctx, cancel := context.WithCancel(t.Context())
	driver := make(chan error, 1)
	go func() { _, err := session.Wait(ctx, "a", 1); driver <- err }()
	if control := nextClusterControl(t, controls); control.Type != "run" || control.Node != "a" || control.Incarnation != 1 {
		t.Fatalf("wrong first turn: %+v", control)
	}
	cancel()
	if err := <-driver; !errors.Is(err, context.Canceled) {
		t.Fatalf("driver cancellation: %v", err)
	}
	queued, cancelQueued := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancelQueued()
	if _, err := session.Wait(queued, "b", 1); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("canceled queued turn: %v", err)
	}
	if result, err := session.Wait(ctx, "cached", 1); err != nil || result.Termination != "stop" {
		t.Fatalf("cached terminal blocked behind an active turn: %+v %v", result, err)
	}
	if err := session.Revoke(t.Context(), "stop", "a", 1); err != nil {
		t.Fatal(err)
	}
	if control := nextClusterControl(t, controls); control.Type != "control" || control.Command != "stop" {
		t.Fatalf("live revoke did not bypass driver: %+v", control)
	}
	if err := session.Link(t.Context(), "heal", "a", "b", 0, true); err != nil {
		t.Fatal(err)
	}
	if control := nextClusterControl(t, controls); control.Command != "link" {
		t.Fatalf("live link did not bypass driver: %+v", control)
	}
	second := make(chan error, 1)
	go func() { _, err := session.Wait(t.Context(), "b", 1); second <- err }()
	complete("a")
	if err := <-observed; err != nil {
		t.Fatal(err)
	}
	if control := nextClusterControl(t, controls); control.Type != "run" || control.Node != "b" {
		t.Fatalf("wrong next turn: %+v", control)
	}
	complete("b")
	if err := <-second; err != nil {
		t.Fatal(err)
	}
	if _, err := session.Wait(t.Context(), "a", 1); err != nil {
		t.Fatal(err)
	}
	if count := session.FrameCounts().Controls; count != 4 {
		t.Fatalf("cached/canceled waits emitted extra controls: %d", count)
	}
}

func TestClusterFinishPreservesSessionWithLiveIncarnations(t *testing.T) {
	session := &ClusterSession{states: map[BrokerNode]*clusterState{{ID: "a", Incarnation: 1}: {done: make(chan struct{})}}}
	if _, err := session.Finish(context.Background()); err == nil {
		t.Fatal("finish accepted a live incarnation")
	}
	if session.finishing {
		t.Fatal("refused finish prevented subsequent lifecycle control")
	}
}

func TestClusterPreservesWholeRunInfrastructureFailure(t *testing.T) {
	stream := &clusterStream{session: &ClusterSession{}}
	err := stream.frame([]byte(`{"type":"cluster_result","termination":"infrastructure","message":"helper panic","instances":[],"virtual_elapsed_nanos":0}`))
	var boundary *BoundaryError
	if !errors.As(err, &boundary) || boundary.Kind != "infrastructure" {
		t.Fatalf("helper failure lost infrastructure classification: %v", err)
	}
}

func TestClusterRejectsGuestCallbacks(t *testing.T) {
	for _, frame := range []string{
		`{"instance":"a","type":"call","id":1,"op":"runtime_idle","input":{}}`,
		`{"instance":"a","type":"sync","id":1}`,
	} {
		session := &ClusterSession{states: map[BrokerNode]*clusterState{}}
		stream := &clusterStream{session: session}
		if err := stream.frame([]byte(frame)); err == nil {
			t.Fatal("cluster accepted a guest callback to Go")
		}
		if session.FrameCounts().GuestCallbacks != 1 {
			t.Fatal("guest callback refusal was absent from the transport counter")
		}
	}
}

func TestClusterRejectsResultWithoutModelEvidence(t *testing.T) {
	config := testConfig()
	e, err := NewEnvironment(config)
	if err != nil {
		t.Fatal(err)
	}
	state := &clusterState{sink: &frameSink{request: Request{Config: config, Fuel: 100, MemoryBytes: 65536}, environment: e, started: &Started{}, authorizedAt: time.Now()}}
	frame, err := newValidatedFrame([]byte(`{"type":"result","termination":"capacity","message":"memory limit","fuel_remaining":100,"peak_memory_bytes":0,"output_bytes":0}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := state.collect(frame, BrokerConfig{}); err == nil {
		t.Fatal("accepted an incarnation result with truncated model evidence")
	}
}

func TestClusterRejectsUnattributableFrames(t *testing.T) {
	for _, frame := range []string{
		`{"instance":"unknown","type":"result"}`,
		`{"instance":"a","incarnation":0,"type":"evidence"}`,
		`{"instance":"a","instance":"b","type":"result"}`,
		`{"type":"evidence","stream":"world","data_base64":"!"}`,
	} {
		stream := &clusterStream{session: &ClusterSession{states: map[BrokerNode]*clusterState{}}}
		if err := stream.frame([]byte(frame)); err == nil {
			t.Fatalf("accepted corrupt cluster frame %s", frame)
		}
	}
}
