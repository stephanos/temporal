package world

import (
	"errors"
	"math"
	"testing"
)

func TestWorldIdentityAndTransitionExhaustionDoNotWrap(t *testing.T) {
	request := Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}}

	t.Run("request", func(t *testing.T) {
		w := newTestWorld(t, 1)
		w.nextRequestID = RequestID(math.MaxUint64)
		before := w.Snapshot()
		if id, err := w.Register(request); !errors.Is(err, ErrCapacity) || id != 0 {
			t.Fatalf("Register() error = %v", err)
		}
		if after := w.Snapshot(); after.StateDigest != before.StateDigest {
			t.Fatal("request ID exhaustion mutated World")
		}
	})

	t.Run("event", func(t *testing.T) {
		w := newTestWorld(t, 1)
		firstRequestID, err := w.Register(request)
		if err != nil {
			t.Fatal(err)
		}
		w.nextEventID = EventID(math.MaxUint64)
		before := w.Snapshot()
		if id, err := w.Ready(Readiness{RequestID: firstRequestID, At: InitialTime, Kind: "done"}); !errors.Is(err, ErrCapacity) || id != 0 {
			t.Fatalf("Ready() error = %v", err)
		}
		if after := w.Snapshot(); after.StateDigest != before.StateDigest {
			t.Fatal("event ID exhaustion mutated World")
		}
	})

	t.Run("transition", func(t *testing.T) {
		w := newTestWorld(t, 1)
		w.nextTransition = Sequence(math.MaxUint64)
		before := w.Snapshot()
		if _, err := w.Register(request); !errors.Is(err, ErrCapacity) {
			t.Fatalf("Register() error = %v", err)
		}
		if after := w.Snapshot(); after.StateDigest != before.StateDigest {
			t.Fatal("transition exhaustion mutated World")
		}
	})
}

func TestWorldQueueCapacityRollbackPreservesEventSequence(t *testing.T) {
	limits := testLimits()
	limits.MaxQueuedEvents = 1
	w, err := New(Config{Seed: 1, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	first, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	second, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "b"}})
	if err != nil {
		t.Fatal(err)
	}
	if eventID, err := w.Ready(Readiness{RequestID: first, At: InitialTime, Kind: "done"}); err != nil || eventID != 1 {
		t.Fatalf("first Ready() = %d, %v", eventID, err)
	}
	before := w.Snapshot()
	if _, err := w.Ready(Readiness{RequestID: second, At: InitialTime, Kind: "done"}); !errors.Is(err, ErrCapacity) {
		t.Fatalf("queue capacity error = %v", err)
	}
	if after := w.Snapshot(); after.StateDigest != before.StateDigest {
		t.Fatal("queue capacity rejection mutated World")
	}
	if _, err := w.Cancel(first); err != nil {
		t.Fatal(err)
	}
	if eventID, err := w.Ready(Readiness{RequestID: second, At: InitialTime, Kind: "done"}); err != nil || eventID != 2 {
		t.Fatalf("second Ready() after capacity rollback = %d, %v", eventID, err)
	}
}

func TestWorldTransitionCapacityIsTerminalWithoutMutation(t *testing.T) {
	limits := testLimits()
	limits.MaxTransitions = 1
	w, err := New(Config{Seed: 1, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}})
	if err != nil {
		t.Fatal(err)
	}
	before := w.Snapshot()
	if _, err := w.Cancel(id); !errors.Is(err, ErrCapacity) {
		t.Fatalf("transition capacity error = %v", err)
	}
	if after := w.Snapshot(); after.StateDigest != before.StateDigest {
		t.Fatal("transition capacity rejection mutated World")
	}
}

func TestWorldPayloadCapacityIncludesTranscriptCopies(t *testing.T) {
	limits := testLimits()
	limits.MaxPayloadBytes = 5
	w, err := New(Config{Seed: 1, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}, Payload: []byte("abc")}); !errors.Is(err, ErrCapacity) {
		t.Fatalf("payload capacity error = %v", err)
	}
	id, err := w.Register(Request{Kind: "wait", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "a"}, Payload: []byte("ab")})
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 || w.Snapshot().PayloadBytes != 4 {
		t.Fatalf("payload accounting = id %d snapshot %#v", id, w.Snapshot())
	}
}

func TestWorldRejectsEveryZeroLimit(t *testing.T) {
	for name, mutate := range map[string]func(*Limits){
		"requests":    func(limits *Limits) { limits.MaxRequests = 0 },
		"events":      func(limits *Limits) { limits.MaxEvents = 0 },
		"queued":      func(limits *Limits) { limits.MaxQueuedEvents = 0 },
		"transitions": func(limits *Limits) { limits.MaxTransitions = 0 },
		"payload":     func(limits *Limits) { limits.MaxPayloadBytes = 0 },
		"string":      func(limits *Limits) { limits.MaxStringBytes = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			limits := testLimits()
			mutate(&limits)
			if world, err := New(Config{Limits: limits}); !errors.Is(err, ErrInvalidConfig) || world != nil {
				t.Fatalf("New() = %#v, %v", world, err)
			}
		})
	}
}

func TestWorldStringLimitCoversResourceComponents(t *testing.T) {
	limits := testLimits()
	limits.MaxStringBytes = 3
	w, err := New(Config{Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Register(Request{Kind: "get", Resource: ResourceID{Adapter: "memory", Kind: "key", Key: "a"}}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("resource string limit error = %v", err)
	}
}
