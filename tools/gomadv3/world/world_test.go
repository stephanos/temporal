package world

import (
	"errors"
	"fmt"
	"testing"
)

func TestWorldLifecycleCopiesDataAndDeliversWholeInstant(t *testing.T) {
	w := newTestWorld(t, 7)
	payload := []byte("request")
	first, err := w.Register(Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}, Priority: 2, Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = 'X'
	second, err := w.Register(Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "b"}, Priority: 1})
	if err != nil {
		t.Fatal(err)
	}
	if first != 1 || second != 2 {
		t.Fatalf("request IDs = %d, %d", first, second)
	}
	if _, err := w.Ready(Readiness{RequestID: first, At: InitialTime + 10, Kind: "result", Payload: []byte("one")}); err != nil {
		t.Fatal(err)
	}
	if _, err := w.Ready(Readiness{RequestID: second, At: InitialTime + 10, Kind: "result", Payload: []byte("two")}); err != nil {
		t.Fatal(err)
	}
	quiescence, err := w.Quiesce()
	if err != nil {
		t.Fatal(err)
	}
	if quiescence.Kind != QuiescenceDelivered || quiescence.Before != InitialTime || quiescence.After != InitialTime+10 || len(quiescence.Deliveries) != 2 {
		t.Fatalf("quiescence = %#v", quiescence)
	}
	if quiescence.Deliveries[0].RequestID != second || quiescence.Deliveries[1].RequestID != first {
		t.Fatalf("delivery order = %#v", quiescence.Deliveries)
	}
	quiescence.Deliveries[1].Payload[0] = 'X'
	snapshot := w.Snapshot()
	if got := string(snapshot.Requests[0].Request.Payload); got != "request" {
		t.Fatalf("stored request payload = %q", got)
	}
	if got := string(snapshot.Events[0].Readiness.Payload); got != "one" {
		t.Fatalf("stored event payload = %q", got)
	}
}

func TestWorldOrdinaryOrderingIsLexicographicAndSeedIndependent(t *testing.T) {
	orders := map[string]struct{}{}
	for _, seed := range []Seed{0, 1, 99, ^Seed(0)} {
		w := newTestWorld(t, seed)
		requests := []Request{
			{Kind: "z", Resource: ResourceID{Adapter: "z", Kind: "r", Key: "k"}, Priority: 1},
			{Kind: "a", Resource: ResourceID{Adapter: "a", Kind: "r", Key: "k"}, Priority: 1},
			{Kind: "a", Resource: ResourceID{Adapter: "a", Kind: "r", Key: "k"}, Priority: 0},
		}
		for _, request := range requests {
			id, err := w.Register(request)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "done"}); err != nil {
				t.Fatal(err)
			}
		}
		batch, err := w.Quiesce()
		if err != nil {
			t.Fatal(err)
		}
		order := fmt.Sprintf("%d%d%d", batch.Deliveries[0].RequestID, batch.Deliveries[1].RequestID, batch.Deliveries[2].RequestID)
		orders[order] = struct{}{}
		if order != "321" {
			t.Fatalf("seed %d order = %s", seed, order)
		}
	}
	if len(orders) != 1 {
		t.Fatalf("ordinary orders varied: %v", orders)
	}
}

func TestWorldEquivalentOrderingRepeatsAndCanVaryBySeed(t *testing.T) {
	orders := map[string]struct{}{}
	for seed := Seed(0); seed < 64; seed++ {
		first := equivalentOrder(t, seed)
		second := equivalentOrder(t, seed)
		if first != second {
			t.Fatalf("seed %d orders = %q and %q", seed, first, second)
		}
		orders[first] = struct{}{}
	}
	if len(orders) < 2 {
		t.Fatalf("equivalent order did not vary: %v", orders)
	}
}

func TestWorldCancellationIdleAndDeadlock(t *testing.T) {
	w := newTestWorld(t, 1)
	id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	deadlock, err := w.Quiesce()
	if err != nil {
		t.Fatal(err)
	}
	if deadlock.Kind != QuiescenceDeadlock || len(deadlock.Blocked) != 1 || deadlock.Blocked[0] != id {
		t.Fatalf("deadlock = %#v", deadlock)
	}
	cancellation, err := w.Cancel(id)
	if err != nil {
		t.Fatal(err)
	}
	if cancellation.Status != CancelWon || cancellation.EventID != 0 {
		t.Fatalf("cancellation = %#v", cancellation)
	}
	again, err := w.Cancel(id)
	if err != nil {
		t.Fatal(err)
	}
	if again.Status != CancelAlreadyCanceled {
		t.Fatalf("second cancellation = %#v", again)
	}
	idle, err := w.Quiesce()
	if err != nil {
		t.Fatal(err)
	}
	if idle.Kind != QuiescenceIdle {
		t.Fatalf("idle = %#v", idle)
	}

	deliveredID, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "b"}})
	if err != nil {
		t.Fatal(err)
	}
	eventID, err := w.Ready(Readiness{RequestID: deliveredID, At: InitialTime, Kind: "done"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Quiesce(); err != nil {
		t.Fatal(err)
	}
	after, err := w.Cancel(deliveredID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Status != CancelAlreadyDelivered || after.EventID != eventID {
		t.Fatalf("post-delivery cancellation = %#v", after)
	}
}

func TestWorldInvalidAndCapacityErrorsDoNotMutateState(t *testing.T) {
	limits := testLimits()
	limits.MaxRequests = 1
	w, err := New(Config{Seed: 1, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	before := w.Snapshot()
	if _, err := w.Register(Request{Kind: "", Resource: ResourceID{Adapter: "Bad", Kind: "cell", Key: "a"}}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("invalid request error = %v", err)
	}
	afterInvalid := w.Snapshot()
	if before.StateDigest != afterInvalid.StateDigest {
		t.Fatal("invalid request mutated World")
	}
	id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "b"}}); !errors.Is(err, ErrCapacity) {
		t.Fatalf("capacity error = %v", err)
	}
	if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime - 1, Kind: "done"}); !errors.Is(err, ErrTimeRegression) {
		t.Fatalf("time regression error = %v", err)
	}
	eventID, err := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "done"})
	if err != nil {
		t.Fatal(err)
	}
	if eventID != 1 {
		t.Fatalf("event ID after failed readiness = %d, want 1", eventID)
	}
}

func equivalentOrder(t *testing.T, seed Seed) string {
	t.Helper()
	w := newTestWorld(t, seed)
	for index := 0; index < 4; index++ {
		id, err := w.Register(Request{Kind: "receive", Resource: ResourceID{Adapter: "network", Kind: "queue", Key: "q"}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "message", EquivalenceClass: "exchangeable"}); err != nil {
			t.Fatal(err)
		}
	}
	batch, err := w.Quiesce()
	if err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%d%d%d%d", batch.Deliveries[0].RequestID, batch.Deliveries[1].RequestID, batch.Deliveries[2].RequestID, batch.Deliveries[3].RequestID)
}

func newTestWorld(t *testing.T, seed Seed) *World {
	t.Helper()
	w, err := New(Config{Seed: seed, Limits: testLimits()})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func testLimits() Limits {
	return Limits{MaxRequests: 100, MaxEvents: 100, MaxQueuedEvents: 100, MaxTransitions: 500, MaxPayloadBytes: 1 << 20, MaxStringBytes: 1024}
}
