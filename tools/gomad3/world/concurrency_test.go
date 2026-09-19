package world

import (
	"sync"
	"testing"
)

func TestWorldConcurrentRegistrationAllocatesEveryIDOnce(t *testing.T) {
	w := newTestWorld(t, 8)
	const count = 64
	start := make(chan struct{})
	ids := make(chan RequestID, count)
	errors := make(chan error, count)
	var wait sync.WaitGroup
	wait.Add(count)
	for index := 0; index < count; index++ {
		go func(index int) {
			defer wait.Done()
			<-start
			id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: string(rune('a' + index))}})
			ids <- id
			errors <- err
		}(index)
	}
	close(start)
	wait.Wait()
	close(ids)
	close(errors)
	seen := make(map[RequestID]bool, count)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	for id := range ids {
		if id == 0 || id > count || seen[id] {
			t.Fatalf("invalid concurrent ID %d", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Fatalf("concurrent IDs = %v", seen)
	}
}

func TestWorldCancelQuiesceRaceLinearizes(t *testing.T) {
	for iteration := 0; iteration < 250; iteration++ {
		w := newTestWorld(t, Seed(iteration))
		id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
		if err != nil {
			t.Fatal(err)
		}
		eventID, err := w.Ready(Readiness{RequestID: id, At: InitialTime, Kind: "done"})
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		cancellations := make(chan Cancellation, 1)
		quiescences := make(chan Quiescence, 1)
		errors := make(chan error, 2)
		go func() {
			<-start
			result, cancelErr := w.Cancel(id)
			cancellations <- result
			errors <- cancelErr
		}()
		go func() {
			<-start
			result, quiesceErr := w.Quiesce()
			quiescences <- result
			errors <- quiesceErr
		}()
		close(start)
		cancellation := <-cancellations
		quiescence := <-quiescences
		for index := 0; index < 2; index++ {
			if err := <-errors; err != nil {
				t.Fatal(err)
			}
		}
		switch cancellation.Status {
		case CancelWon:
			if cancellation.EventID != eventID || quiescence.Kind != QuiescenceIdle || len(quiescence.Deliveries) != 0 {
				t.Fatalf("cancel-won race = %#v, %#v", cancellation, quiescence)
			}
		case CancelAlreadyDelivered:
			if quiescence.Kind != QuiescenceDelivered || len(quiescence.Deliveries) != 1 || quiescence.Deliveries[0].EventID != eventID {
				t.Fatalf("delivery-won race = %#v, %#v", cancellation, quiescence)
			}
		default:
			t.Fatalf("race cancellation = %#v", cancellation)
		}
	}
}
