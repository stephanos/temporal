package simulation

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func stopTestExecution(live bool, deadline time.Time) *execution {
	finished := make(chan struct{})
	close(finished)
	nodes := map[gomadsim.NodeID]*node{"active": {handle: gomadsim.NodeHandle{Node: "active", Incarnation: 2}, done: finished}}
	if live {
		nodes["standby"] = &node{handle: gomadsim.NodeHandle{Node: "standby", Incarnation: 1}, done: make(chan struct{})}
	}
	return &execution{nodes: nodes, cluster: &wasi.ClusterSession{}, clusterDeadline: deadline}
}

// stubClusterShutdown replaces the helper session shutdown and records the
// deadline Shutdown gave it.
func stubClusterShutdown(t *testing.T, shutdown func(*execution) error, e *execution) *time.Time {
	t.Helper()
	var observed time.Time
	original := shutdownClusterSession
	t.Cleanup(func() { shutdownClusterSession = original })
	shutdownClusterSession = func(_ *wasi.ClusterSession, ctx context.Context) error {
		if ctx.Err() != nil {
			t.Errorf("cleanup inherited the caller's cancellation: %v", ctx.Err())
		}
		observed, _ = ctx.Deadline()
		return shutdown(e)
	}
	return &observed
}

func withinGrace(got, before, after time.Time) bool {
	return !got.Before(before.Add(clusterStopGrace)) && !got.After(after.Add(clusterStopGrace))
}

// Shutdown of a cluster whose started incarnations all reached their terminals
// is the final finish: it gets the helper's execution deadline, never less
// than the stop grace. A live incarnation's helper, a cluster near its deadline
// and a failed session shutdown keep the 5 s stop grace. Cleanup ignores the
// caller's cancellation.
func TestClusterShutdownBoundsFinishedClusterByExecutionDeadline(t *testing.T) {
	if clusterStopGrace != 5*time.Second {
		t.Fatalf("live helper stop grace = %v, want 5s", clusterStopGrace)
	}
	caller, cancelCaller := context.WithCancel(t.Context())
	cancelCaller()

	t.Run("finished", func(t *testing.T) {
		deadline := time.Now().Add(15 * time.Minute)
		e := stopTestExecution(false, deadline)
		observed := stubClusterShutdown(t, func(*execution) error { return nil }, e)
		if err := e.Shutdown(caller); err != nil {
			t.Fatal(err)
		}
		if !observed.Equal(deadline) {
			t.Fatalf("finished cluster shutdown deadline %v, want execution deadline %v", *observed, deadline)
		}
	})

	for _, test := range []struct {
		name     string
		live     bool
		deadline time.Duration
	}{
		{name: "live", live: true, deadline: 15 * time.Minute},
		{name: "near deadline", deadline: time.Second},
		{name: "past deadline", deadline: -time.Minute},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := stopTestExecution(test.live, time.Now().Add(test.deadline))
			observed := stubClusterShutdown(t, func(e *execution) error {
				for _, n := range e.nodes {
					select {
					case <-n.done:
					default:
						close(n.done)
					}
				}
				return nil
			}, e)
			before := time.Now()
			if err := e.Shutdown(caller); err != nil {
				t.Fatal(err)
			}
			if !withinGrace(*observed, before, time.Now()) {
				t.Fatalf("shutdown deadline %v, want the %v stop grace", *observed, clusterStopGrace)
			}
		})
	}

	t.Run("never opened", func(t *testing.T) {
		e := stopTestExecution(false, time.Time{})
		observed := stubClusterShutdown(t, func(*execution) error { return nil }, e)
		before := time.Now()
		if err := e.Shutdown(caller); err != nil {
			t.Fatal(err)
		}
		if !withinGrace(*observed, before, time.Now()) {
			t.Fatalf("shutdown deadline %v, want the %v stop grace", *observed, clusterStopGrace)
		}
	})

	t.Run("failed shutdown waits only the stop grace", func(t *testing.T) {
		grace := clusterStopGrace
		t.Cleanup(func() { clusterStopGrace = grace })
		e := stopTestExecution(false, time.Now().Add(15*time.Minute))
		live := errors.New("cluster finish has a live incarnation")
		stubClusterShutdown(t, func(e *execution) error {
			// A Start raced the bound: its helper incarnation never ends here.
			clusterStopGrace = 50 * time.Millisecond
			e.mu.Lock()
			e.nodes["standby"] = &node{handle: gomadsim.NodeHandle{Node: "standby", Incarnation: 1}, done: make(chan struct{})}
			e.mu.Unlock()
			return live
		}, e)
		returned := make(chan error, 1)
		go func() { returned <- e.Shutdown(caller) }()
		select {
		case err := <-returned:
			if !errors.Is(err, live) || !errors.Is(err, context.DeadlineExceeded) {
				t.Fatalf("failed shutdown returned %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("failed shutdown waited for the finished cluster's execution deadline")
		}
	})
}

// The recorded execution deadline is the longest instance or boot timeout,
// as wasi.OpenCluster bounds the session, taken before the session opens.
func TestOpenClusterRecordsHelperExecutionDeadline(t *testing.T) {
	request := wasi.ClusterRequest{
		Instances: []wasi.ClusterInstance{{ID: "active", Request: wasi.Request{Timeout: 3 * time.Minute}}, {ID: "standby", Request: wasi.Request{Timeout: 7 * time.Minute}}},
		Boots:     map[string]wasi.Request{"active-reopen": {Timeout: 11 * time.Minute}, "standby-reopen": {Timeout: time.Minute}},
	}
	if got := clusterExecutionTimeout(request); got != 11*time.Minute {
		t.Fatalf("cluster execution timeout %v, want 11m", got)
	}
	request.Boots = nil
	if got := clusterExecutionTimeout(request); got != 7*time.Minute {
		t.Fatalf("cluster execution timeout without boots %v, want 7m", got)
	}
	request.Boots = map[string]wasi.Request{"active-reopen": {Timeout: 11 * time.Minute}}
	original := openClusterSession
	t.Cleanup(func() { openClusterSession = original })
	var opened time.Time
	session := &wasi.ClusterSession{}
	openClusterSession = func(_ context.Context, got wasi.ClusterRequest) (*wasi.ClusterSession, error) {
		if len(got.Instances) != 2 || len(got.Boots) != 1 {
			t.Errorf("OpenCluster received %+v", got)
		}
		opened = time.Now()
		return session, nil
	}
	e := &execution{ctx: t.Context(), clusterRequest: &request}
	before := time.Now()
	if err := e.openCluster(); err != nil {
		t.Fatal(err)
	}
	if e.cluster != session || e.clusterRequest != nil {
		t.Fatal("openCluster did not keep the opened session")
	}
	if e.clusterDeadline.Before(before.Add(11*time.Minute)) || e.clusterDeadline.After(opened.Add(11*time.Minute)) {
		t.Fatalf("recorded deadline %v, want 11m from before the session opened (%v..%v)", e.clusterDeadline, before, opened)
	}
}
