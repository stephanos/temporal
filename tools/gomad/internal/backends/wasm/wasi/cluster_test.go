package wasi

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	runnerbackend "go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

func TestClusterHostTerminationStatus(t *testing.T) {
	for _, name := range []string{"watchdog", "cancelled", "parent deadline"} {
		t.Run(name, func(t *testing.T) {
			helper := filepath.Join(t.TempDir(), "helper")
			if err := os.WriteFile(helper, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0700); err != nil {
				t.Fatal(err)
			}
			input, err := newHelperInput()
			if err != nil {
				t.Fatal(err)
			}
			input.start(nil)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			watchdog := name == "watchdog"
			timeout := 20 * time.Millisecond
			if !watchdog {
				if name == "parent deadline" {
					ctx, cancel = context.WithTimeout(ctx, 20*time.Millisecond)
					defer cancel()
				} else {
					timer := time.AfterFunc(20*time.Millisecond, cancel)
					defer timer.Stop()
				}
				timeout = time.Second
			}
			session := &ClusterSession{parentCtx: ctx, ctx: ctx, cancel: cancel, input: input, states: map[BrokerNode]*clusterState{}, done: make(chan struct{}), began: time.Now()}
			session.run(Request{HelperPath: helper, Timeout: timeout})
			result := session.CompletedResult()
			if result == nil || !result.GroupGone || result.WatchdogTimeout != watchdog || result.Cancelled == watchdog {
				t.Fatalf("host status %+v", result)
			}
			var hostError *ClusterHostError
			if !errors.As(session.err, &hostError) || hostError.WatchdogTimeout != watchdog || hostError.Cancelled == watchdog || !hostError.GroupGone {
				t.Fatalf("startup error lost host status: %v", session.err)
			}
		})
	}
}

func TestCompletedClusterResultRetainsDetachedDiagnosticStderr(t *testing.T) {
	t.Parallel()
	done := make(chan struct{})
	session := &ClusterSession{done: done, result: ClusterResult{helperStderr: []byte("diagnostic profile")}}
	if session.CompletedResult() != nil {
		t.Fatal("published a result before the helper was reaped")
	}
	close(done)
	result := session.CompletedResult()
	if string(result.HelperStderr()) != "diagnostic profile" {
		t.Fatal("completed result lost helper diagnostics")
	}
	result.helperStderr[0] = 'x'
	if string(session.CompletedResult().HelperStderr()) != "diagnostic profile" {
		t.Fatal("completed result shares diagnostic bytes with the session")
	}
}

func TestClusterInternalDeadlineIsWatchdog(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nexec /bin/sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	input.start(nil)
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	session := &ClusterSession{parentCtx: t.Context(), ctx: ctx, cancel: cancel, input: input, states: map[BrokerNode]*clusterState{}, done: make(chan struct{}), began: time.Now()}
	session.run(Request{HelperPath: helper, Timeout: time.Second})
	result := session.CompletedResult()
	var hostError *ClusterHostError
	if result == nil || !result.GroupGone || result.Cancelled || !result.WatchdogTimeout || !errors.As(session.err, &hostError) || hostError.Cancelled || !hostError.WatchdogTimeout || !hostError.GroupGone {
		t.Fatalf("internal deadline lost watchdog classification: result=%+v err=%v", result, session.err)
	}
}

func TestClusterProtocolFailureIsNotCancellation(t *testing.T) {
	helper := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\nprintf 'invalid-frame\\n'\nexec /bin/sleep 30\n"), 0700); err != nil {
		t.Fatal(err)
	}
	input, err := newHelperInput()
	if err != nil {
		t.Fatal(err)
	}
	input.start(nil)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	state := &clusterState{done: make(chan struct{})}
	session := &ClusterSession{parentCtx: t.Context(), ctx: ctx, cancel: cancel, input: input, states: map[BrokerNode]*clusterState{{ID: "a", Incarnation: 1}: state}, done: make(chan struct{}), began: time.Now()}
	session.run(Request{HelperPath: helper, Timeout: time.Second})
	result := session.CompletedResult()
	var hostError *ClusterHostError
	var boundary *BoundaryError
	if result == nil || !result.GroupGone || result.Cancelled || result.WatchdogTimeout || errors.As(session.err, &hostError) || !errors.As(session.err, &boundary) || boundary.Kind != "invalid" {
		t.Fatalf("protocol failure changed classification: result=%+v err=%v", result, session.err)
	}
	select {
	case <-state.done:
		if state.err != session.err || !state.result.Reaped {
			t.Fatalf("incarnation lost protocol error or reaping: %+v %v", state.result, state.err)
		}
	default:
		t.Fatal("incarnation was not released after protocol failure")
	}
}

type clusterTestControl struct {
	Type        string `json:"type"`
	ID          uint64 `json:"id"`
	Node        string `json:"node"`
	Incarnation uint64 `json:"incarnation"`
	Command     string `json:"command"`
}

func TestClusterShutdownControlOrder(t *testing.T) {
	t.Parallel()
	want := []BrokerNode{{ID: "a", Incarnation: 1}, {ID: "a", Incarnation: 2}, {ID: "b", Incarnation: 1}}
	for iteration := range 32 {
		t.Run(fmt.Sprint(iteration), func(t *testing.T) {
			var session *ClusterSession
			session, controls := clusterControlSessionRefusing(t, func(control clusterTestControl) string {
				session.mu.Lock()
				defer session.mu.Unlock()
				if control.Command == "stop" {
					close(session.states[BrokerNode{ID: control.Node, Incarnation: control.Incarnation}].done)
				} else if control.Type == "finish" {
					close(session.done)
				}
				return ""
			})
			for index := range want {
				handle := want[(index+iteration)%len(want)]
				session.states[handle] = &clusterState{done: make(chan struct{})}
			}
			completed := make(chan struct{})
			close(completed)
			session.states[BrokerNode{ID: "0-completed", Incarnation: 1}] = &clusterState{done: completed}
			if err := session.Shutdown(t.Context()); err != nil {
				t.Fatal(err)
			}
			var got []BrokerNode
			for range want {
				control := nextClusterControl(t, controls)
				if control.Command != "stop" {
					t.Fatalf("shutdown control = %+v", control)
				}
				got = append(got, BrokerNode{ID: control.Node, Incarnation: control.Incarnation})
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("stop order = %+v, want %+v", got, want)
			}
			if control := nextClusterControl(t, controls); control.Type != "finish" {
				t.Fatalf("shutdown did not finish after live nodes: %+v", control)
			}
		})
	}
}

func clusterControlSession(t *testing.T) (*ClusterSession, <-chan clusterTestControl) {
	t.Helper()
	return clusterControlSessionRefusing(t, nil)
}

// clusterControlSessionRefusing acknowledges each control, refusing those for
// which refuse returns a nonempty message as the helper does.
func clusterControlSessionRefusing(t *testing.T, refuse func(clusterTestControl) string) (*ClusterSession, <-chan clusterTestControl) {
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
			termination, message := "", ""
			if refuse != nil {
				if message = refuse(control); message != "" {
					termination = "invalid"
				}
			}
			ack, err := json.Marshal(map[string]any{"type": "control_result", "id": control.ID, "termination": termination, "message": message})
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

func TestClusterRefusedRestartLeavesIncarnationRetryable(t *testing.T) {
	refused := false
	session, controls := clusterControlSessionRefusing(t, func(control clusterTestControl) string {
		if control.Type == "restart" && !refused {
			refused = true
			return "cluster restart lacks recovery image"
		}
		return ""
	})
	config := testConfig()
	config.Profile = CooperativeProfile
	identity := choice.ExecutionIdentity{TargetSHA256: sha256.Sum256([]byte("guest")), ToolchainBuildKey: strings.Repeat("a", 64), GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: sha256.Sum256([]byte("runtime"))}
	runtime := &RuntimeControl{Choice: &runnerbackend.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: 8192}}
	session.boots = map[string]*clusterBoot{"boot": {request: Request{Config: config}, started: &Started{}}}
	if err := session.Start(t.Context(), "a", 1, "boot", runtime); err != nil {
		t.Fatal(err)
	}
	nextClusterControl(t, controls)
	close(session.states[BrokerNode{ID: "a", Incarnation: 1}].done)
	if err := session.Start(t.Context(), "a", 2, "boot", runtime); err == nil || !strings.Contains(err.Error(), "lacks recovery image") {
		t.Fatalf("refused restart returned %v", err)
	}
	nextClusterControl(t, controls)
	session.mu.Lock()
	_, live := session.states[BrokerNode{ID: "a", Incarnation: 2}]
	current := session.current["a"]
	session.mu.Unlock()
	if live || current != 1 {
		t.Fatalf("refused restart kept Go state: live=%v current=%d", live, current)
	}
	if _, err := session.Wait(t.Context(), "a", 2); err == nil {
		t.Fatal("refused incarnation remained waitable")
	}
	if err := session.Start(t.Context(), "a", 2, "boot", runtime); err != nil {
		t.Fatalf("retry after refusal: %v", err)
	}
	if control := nextClusterControl(t, controls); control.Type != "restart" || control.Incarnation != 2 {
		t.Fatalf("retry control = %+v", control)
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
	if err := state.collect(frame, BrokerConfig{}, false); err == nil {
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
