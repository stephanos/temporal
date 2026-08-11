package world

import (
	"errors"
	"testing"
)

func TestWorldReplayValidatesEveryTransitionAndFinalDigest(t *testing.T) {
	initialWorld := newTestWorld(t, 41)
	initial := initialWorld.Snapshot()
	recorded := newTestWorld(t, 41)
	request := Request{Kind: "receive", Resource: ResourceID{Adapter: "network", Kind: "queue", Key: "inbox"}, Payload: []byte("request")}
	id, err := recorded.Register(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := recorded.Ready(Readiness{RequestID: id, At: InitialTime + 3, Kind: "message", Payload: []byte("result")}); err != nil {
		t.Fatal(err)
	}
	if _, err := recorded.Quiesce(); err != nil {
		t.Fatal(err)
	}
	if _, err := recorded.Quiesce(); err != nil {
		t.Fatal(err)
	}
	final := recorded.Snapshot()
	plan := ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: initial.StateDigest, Transitions: copyTransitions(final.Transitions), FinalDigest: final.StateDigest}

	replayed, err := Restore(initial, &plan)
	if err != nil {
		t.Fatal(err)
	}
	replayedID, err := replayed.Register(request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := replayed.Ready(Readiness{RequestID: replayedID, At: InitialTime + 3, Kind: "message", Payload: []byte("result")}); err != nil {
		t.Fatal(err)
	}
	if _, err := replayed.Quiesce(); err != nil {
		t.Fatal(err)
	}
	if _, err := replayed.Quiesce(); err != nil {
		t.Fatal(err)
	}
	if progress := replayed.ReplayProgress(); progress.Cursor != progress.Expected || progress.Expected != 4 {
		t.Fatalf("replay progress = %#v", progress)
	}
	if got := replayed.Snapshot().StateDigest; got != plan.FinalDigest {
		t.Fatalf("final digest = %s, want %s", got, plan.FinalDigest)
	}
}

func TestWorldReplayDivergenceDoesNotMutateStateOrCursor(t *testing.T) {
	initialWorld := newTestWorld(t, 2)
	initial := initialWorld.Snapshot()
	recorded := newTestWorld(t, 2)
	request := Request{Kind: "read", Resource: ResourceID{Adapter: "memory", Kind: "cell", Key: "expected"}}
	if _, err := recorded.Register(request); err != nil {
		t.Fatal(err)
	}
	final := recorded.Snapshot()
	plan := ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: initial.StateDigest, Transitions: copyTransitions(final.Transitions), FinalDigest: final.StateDigest}
	replayed, err := Restore(initial, &plan)
	if err != nil {
		t.Fatal(err)
	}
	before := replayed.Snapshot()
	request.Resource.Key = "actual"
	_, err = replayed.Register(request)
	var divergence *ReplayDivergenceError
	if !errors.As(err, &divergence) || divergence.Field != "register.request.resource.key" {
		t.Fatalf("Register() error = %#v", err)
	}
	if after := replayed.Snapshot(); after.StateDigest != before.StateDigest || after.Replay.Cursor != 0 {
		t.Fatalf("divergence mutated replay: before=%#v after=%#v", before.Replay, after.Replay)
	}
}

func TestWorldReplayRejectsInvalidPlanBeforeUse(t *testing.T) {
	initial := newTestWorld(t, 5).Snapshot()
	plan := ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: initial.StateDigest, FinalDigest: initial.StateDigest, Transitions: []Transition{{Kind: "unknown"}}}
	if restored, err := Restore(initial, &plan); !errors.Is(err, ErrInvalidSnapshot) || restored != nil {
		t.Fatalf("Restore() = %#v, %v", restored, err)
	}
}

func TestWorldReplayRejectsUnexpectedOperationAfterPlan(t *testing.T) {
	initial := newTestWorld(t, 6).Snapshot()
	plan := ReplayPlan{SchemaVersion: SchemaVersion, InitialDigest: initial.StateDigest, FinalDigest: initial.StateDigest}
	w, err := Restore(initial, &plan)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.Quiesce()
	var divergence *ReplayDivergenceError
	if !errors.As(err, &divergence) || divergence.ExpectedKind != "end" || divergence.ActualKind != "quiesce" {
		t.Fatalf("Quiesce() error = %#v", err)
	}
}
