package virtualtime

import (
	"errors"
	"testing"
)

func TestStepAdvancesAllTimersAtEarliestDeadline(t *testing.T) {
	state := NewState(10)
	state = mustStep(t, state, ScheduleTimer("later", 15)).PostState
	state = mustStep(t, state, ScheduleTimer("first-b", 12)).PostState
	state = mustStep(t, state, ScheduleTimer("first-a", 12)).PostState

	transition, err := Step(state, AdvanceTime())
	if err != nil {
		t.Fatal(err)
	}
	if transition.PreStateIdentity != state.Identity() {
		t.Fatalf("pre-state identity = %q, want %q", transition.PreStateIdentity, state.Identity())
	}
	if transition.PostState.Now() != 12 {
		t.Fatalf("now = %d, want 12", transition.PostState.Now())
	}
	if got, want := transition.Delta.ReadyTimers, []TimerID{"first-a", "first-b"}; !equalTimerIDs(got, want) {
		t.Fatalf("ready timers = %v, want %v", got, want)
	}
	if transition.PostState.Timer("first-a").Status != TimerReady || transition.PostState.Timer("first-b").Status != TimerReady {
		t.Fatalf("earliest timers were not marked ready: %#v", transition.PostState.Snapshot().Timers)
	}
	if transition.PostState.Timer("later").Status != TimerPending {
		t.Fatalf("later timer status = %q, want %q", transition.PostState.Timer("later").Status, TimerPending)
	}
	if transition.PostStateIdentity != transition.PostState.Identity() {
		t.Fatalf("post-state identity = %q, want %q", transition.PostStateIdentity, transition.PostState.Identity())
	}
}

func TestStepRequiresQuiescenceBeforeAdvance(t *testing.T) {
	state := NewState(10)
	state = mustStep(t, state, ScheduleTimer("timer", 12)).PostState
	state = mustStep(t, state, SetRunnable("worker", true)).PostState

	assertRejectedWithoutMutation(t, state, AdvanceTime(), RejectionRunnableWork)
}

func TestStepTimerLifecycle(t *testing.T) {
	state := NewState(10)
	ready := mustStep(t, state, ScheduleTimer("ready", 10))
	if ready.PostState.Timer("ready").Status != TimerReady {
		t.Fatalf("timer status = %q, want %q", ready.PostState.Timer("ready").Status, TimerReady)
	}
	fired := mustStep(t, ready.PostState, FireTimer("ready"))
	if fired.PostState.Timer("ready").Status != TimerFired {
		t.Fatalf("timer status = %q, want %q", fired.PostState.Timer("ready").Status, TimerFired)
	}

	pending := mustStep(t, fired.PostState, ScheduleTimer("cancel", 20))
	cancelled := mustStep(t, pending.PostState, CancelTimer("cancel"))
	if cancelled.PostState.Timer("cancel").Status != TimerCancelled {
		t.Fatalf("timer status = %q, want %q", cancelled.PostState.Timer("cancel").Status, TimerCancelled)
	}
	assertRejectedWithoutMutation(t, cancelled.PostState, FireTimer("cancel"), RejectionTimerNotReady)
}

func TestStepRejectsInvalidActionsWithoutMutation(t *testing.T) {
	state := NewState(10)
	state = mustStep(t, state, ScheduleTimer("timer", 12)).PostState

	tests := []struct {
		name   string
		action Action
		code   RejectionCode
	}{
		{name: "duplicate timer", action: ScheduleTimer("timer", 13), code: RejectionTimerExists},
		{name: "past deadline", action: ScheduleTimer("past", 9), code: RejectionDeadlineBeforeNow},
		{name: "unknown cancellation", action: CancelTimer("missing"), code: RejectionUnknownTimer},
		{name: "redundant runnable remove", action: SetRunnable("missing", false), code: RejectionRunnableUnchanged},
		{name: "fire pending", action: FireTimer("timer"), code: RejectionTimerNotReady},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertRejectedWithoutMutation(t, state, test.action, test.code)
		})
	}
}

func TestStepDetachesInputAndOutputState(t *testing.T) {
	state := NewState(0)
	transition := mustStep(t, state, ScheduleTimer("timer", 1))
	snapshot := transition.PostState.Snapshot()
	snapshot.Timers[0].Status = TimerCancelled

	if transition.PostState.Timer("timer").Status != TimerPending {
		t.Fatal("mutating a snapshot changed transition state")
	}
	if _, found := state.TimerOK("timer"); found {
		t.Fatal("Step mutated its input state")
	}
}

func mustStep(t *testing.T, state State, action Action) Transition {
	t.Helper()
	transition, err := Step(state, action)
	if err != nil {
		t.Fatal(err)
	}
	return transition
}

func assertRejectedWithoutMutation(t *testing.T, state State, action Action, want RejectionCode) {
	t.Helper()
	before := state.Identity()
	transition, err := Step(state, action)
	if err == nil {
		t.Fatal("Step() succeeded")
	}
	var rejection *Rejection
	if !errors.As(err, &rejection) {
		t.Fatalf("Step() error = %T %v, want *Rejection", err, err)
	}
	if rejection.Code != want {
		t.Fatalf("rejection code = %q, want %q", rejection.Code, want)
	}
	if state.Identity() != before {
		t.Fatalf("input state identity changed from %q to %q", before, state.Identity())
	}
	if transition.PostStateIdentity != "" {
		t.Fatalf("rejected transition returned post-state identity %q", transition.PostStateIdentity)
	}
}

func equalTimerIDs(left, right []TimerID) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
