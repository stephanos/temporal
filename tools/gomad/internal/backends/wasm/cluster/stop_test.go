package simulation

import (
	"context"
	"testing"
	"time"

	gomadsim "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

// Shutdown of a cluster whose started incarnations all reached their terminals
// is the final finish, bounded by the helper's execution deadline: a finish
// slower than the stop grace completes. A live incarnation's helper still has
// only the 5 s stop grace, and cleanup ignores the caller's cancellation.
func TestClusterStopBoundsFinishedClusterByExecutionDeadline(t *testing.T) {
	if clusterStopGrace != 5*time.Second {
		t.Fatalf("live helper stop grace = %v, want 5s", clusterStopGrace)
	}
	cluster := func(live bool) (*execution, time.Time) {
		finished := make(chan struct{})
		close(finished)
		nodes := map[gomadsim.NodeID]*node{"active": {handle: gomadsim.NodeHandle{Node: "active", Incarnation: 2}, done: finished}}
		if live {
			nodes["standby"] = &node{handle: gomadsim.NodeHandle{Node: "standby", Incarnation: 1}, done: make(chan struct{})}
		}
		deadline := time.Now().Add(15 * time.Minute)
		return &execution{nodes: nodes, clusterDeadline: deadline}, deadline
	}
	caller, cancelCaller := context.WithCancel(t.Context())
	cancelCaller()

	t.Run("finished", func(t *testing.T) {
		e, deadline := cluster(false)
		stop, cancel := e.stopContext(caller)
		defer cancel()
		if got, ok := stop.Deadline(); !ok || !got.Equal(deadline) {
			t.Fatalf("finished cluster stop deadline %v, want execution deadline %v", got, deadline)
		}
		// The final evidence (helper World/network/inspection streams and the
		// Go decode and recompression of a 40 MB World) outlasts the old 5 s.
		final := make(chan struct{})
		time.AfterFunc(clusterStopGrace+500*time.Millisecond, func() { close(final) })
		select {
		case <-final:
		case <-stop.Done():
			t.Fatalf("slow but successful finish was cut off: %v", stop.Err())
		}
	})

	t.Run("live", func(t *testing.T) {
		e, _ := cluster(true)
		before := time.Now()
		stop, cancel := e.stopContext(caller)
		defer cancel()
		got, ok := stop.Deadline()
		if !ok || got.Before(before.Add(clusterStopGrace)) || got.After(time.Now().Add(clusterStopGrace)) {
			t.Fatalf("live cluster stop deadline %v, want the %v stop grace", got, clusterStopGrace)
		}
		if stop.Err() != nil {
			t.Fatalf("cleanup inherited the caller's cancellation: %v", stop.Err())
		}
	})

	t.Run("never opened", func(t *testing.T) {
		e := &execution{nodes: map[gomadsim.NodeID]*node{}}
		before := time.Now()
		stop, cancel := e.stopContext(caller)
		defer cancel()
		if got, ok := stop.Deadline(); !ok || got.After(time.Now().Add(clusterStopGrace)) || got.Before(before.Add(clusterStopGrace)) {
			t.Fatalf("cluster without an execution deadline stop deadline %v, want the stop grace", got)
		}
	})
}
