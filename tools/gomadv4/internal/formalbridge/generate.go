package formalbridge

import (
	"fmt"

	"go.temporal.io/server/tools/gomadv4/trace"
	"go.temporal.io/server/tools/gomadv4/virtualtime"
)

type Inputs struct {
	ModelIdentity    string
	BoundsIdentity   string
	BaselineIdentity string
}

func Generate(inputs Inputs) (trace.Corpus, error) {
	if inputs.ModelIdentity == "" || inputs.BoundsIdentity == "" || inputs.BaselineIdentity == "" {
		return trace.Corpus{}, fmt.Errorf("formal bridge identities are incomplete")
	}

	empty := snapshot(0, nil, nil)
	timerA := virtualtime.Timer{ID: "timer-a", Deadline: 2, Status: virtualtime.TimerPending}
	timerB := virtualtime.Timer{ID: "timer-b", Deadline: 2, Status: virtualtime.TimerPending}
	withA := snapshot(0, nil, []virtualtime.Timer{timerA})
	withAB := snapshot(0, nil, []virtualtime.Timer{timerA, timerB})
	withWork := snapshot(0, []virtualtime.WorkID{"worker"}, []virtualtime.Timer{timerA, timerB})
	readyA := timerA
	readyA.Status = virtualtime.TimerReady
	readyB := timerB
	readyB.Status = virtualtime.TimerReady
	ready := snapshot(2, nil, []virtualtime.Timer{readyA, readyB})
	firedA := readyA
	firedA.Status = virtualtime.TimerFired
	partlyFired := snapshot(2, nil, []virtualtime.Timer{firedA, readyB})
	cancelledB := readyB
	cancelledB.Status = virtualtime.TimerCancelled
	terminal := snapshot(2, nil, []virtualtime.Timer{firedA, cancelledB})

	immediateEmpty := snapshot(5, nil, nil)
	immediateTimer := virtualtime.Timer{ID: "timer-now", Deadline: 5, Status: virtualtime.TimerReady}
	immediateReady := snapshot(5, nil, []virtualtime.Timer{immediateTimer})
	immediateTimer.Status = virtualtime.TimerFired
	immediateFired := snapshot(5, nil, []virtualtime.Timer{immediateTimer})

	corpus := trace.Corpus{
		Schema: trace.CorpusSchema,
		Generation: trace.GenerationContract{
			Schema: "gomadv4.virtual-time-generation/v1", ModelIdentity: inputs.ModelIdentity, BoundsIdentity: inputs.BoundsIdentity,
		},
		BaselineIdentity: inputs.BaselineIdentity,
		Coverage: []string{
			"action.advance_time", "action.cancel_timer", "action.fire_timer", "action.schedule_timer", "action.set_runnable",
			"rejection.deadline_before_now", "rejection.no_pending_timer", "rejection.ready_timer", "rejection.runnable_work",
			"rejection.runnable_unchanged", "rejection.timer_exists", "rejection.timer_not_ready", "rejection.timer_terminal", "rejection.unknown_timer",
		},
		Traces: []trace.BehaviorTrace{
			{
				Name: "equal-deadline-lifecycle", InitialState: empty,
				Steps: []trace.StepRecord{
					step(0, empty, withA, virtualtime.ScheduleTimer("timer-a", 2), virtualtime.ObservableDelta{Kind: virtualtime.ActionScheduleTimer, TimerID: "timer-a", TimerAfter: virtualtime.TimerPending}),
					step(1, withA, withAB, virtualtime.ScheduleTimer("timer-b", 2), virtualtime.ObservableDelta{Kind: virtualtime.ActionScheduleTimer, TimerID: "timer-b", TimerAfter: virtualtime.TimerPending}),
					step(2, withAB, withWork, virtualtime.SetRunnable("worker", true), virtualtime.ObservableDelta{Kind: virtualtime.ActionSetRunnable, WorkID: "worker", Runnable: true}),
					step(3, withWork, withAB, virtualtime.SetRunnable("worker", false), virtualtime.ObservableDelta{Kind: virtualtime.ActionSetRunnable, WorkID: "worker"}),
					step(4, withAB, ready, virtualtime.AdvanceTime(), virtualtime.ObservableDelta{Kind: virtualtime.ActionAdvanceTime, CurrentTime: 2, ReadyTimers: []virtualtime.TimerID{"timer-a", "timer-b"}}),
					step(5, ready, partlyFired, virtualtime.FireTimer("timer-a"), virtualtime.ObservableDelta{Kind: virtualtime.ActionFireTimer, TimerID: "timer-a", TimerBefore: virtualtime.TimerReady, TimerAfter: virtualtime.TimerFired}),
					step(6, partlyFired, terminal, virtualtime.CancelTimer("timer-b"), virtualtime.ObservableDelta{Kind: virtualtime.ActionCancelTimer, TimerID: "timer-b", TimerBefore: virtualtime.TimerReady, TimerAfter: virtualtime.TimerCancelled}),
				},
			},
			{
				Name: "immediate-timer", InitialState: immediateEmpty,
				Steps: []trace.StepRecord{
					step(0, immediateEmpty, immediateReady, virtualtime.ScheduleTimer("timer-now", 5), virtualtime.ObservableDelta{Kind: virtualtime.ActionScheduleTimer, TimerID: "timer-now", TimerAfter: virtualtime.TimerReady}),
					step(1, immediateReady, immediateFired, virtualtime.FireTimer("timer-now"), virtualtime.ObservableDelta{Kind: virtualtime.ActionFireTimer, TimerID: "timer-now", TimerBefore: virtualtime.TimerReady, TimerAfter: virtualtime.TimerFired}),
				},
			},
		},
		Rejections: []trace.RejectionCase{
			rejection("advance-empty", empty, virtualtime.AdvanceTime(), virtualtime.RejectionNoPendingTimer),
			rejection("advance-ready", immediateReady, virtualtime.AdvanceTime(), virtualtime.RejectionReadyTimer),
			rejection("advance-runnable", withWork, virtualtime.AdvanceTime(), virtualtime.RejectionRunnableWork),
			rejection("cancel-fired", immediateFired, virtualtime.CancelTimer("timer-now"), virtualtime.RejectionTimerTerminal),
			rejection("cancel-unknown", empty, virtualtime.CancelTimer("missing"), virtualtime.RejectionUnknownTimer),
			rejection("duplicate-timer", withA, virtualtime.ScheduleTimer("timer-a", 3), virtualtime.RejectionTimerExists),
			rejection("fire-pending", withA, virtualtime.FireTimer("timer-a"), virtualtime.RejectionTimerNotReady),
			rejection("past-deadline", empty, virtualtime.ScheduleTimer("past", -1), virtualtime.RejectionDeadlineBeforeNow),
			rejection("runnable-unchanged", withWork, virtualtime.SetRunnable("worker", true), virtualtime.RejectionRunnableUnchanged),
		},
	}
	if err := trace.Finalize(&corpus); err != nil {
		return trace.Corpus{}, err
	}
	return corpus, nil
}

func snapshot(now int64, runnable []virtualtime.WorkID, timers []virtualtime.Timer) virtualtime.StateSnapshot {
	return virtualtime.StateSnapshot{Now: now, Runnable: runnable, Timers: timers}
}

func step(ordinal int, pre, post virtualtime.StateSnapshot, action virtualtime.Action, delta virtualtime.ObservableDelta) trace.StepRecord {
	return trace.StepRecord{
		Ordinal: ordinal, Action: action, PreStateIdentity: identity(pre), PostStateIdentity: identity(post), ObservableDelta: delta,
	}
}

func rejection(name string, state virtualtime.StateSnapshot, action virtualtime.Action, code virtualtime.RejectionCode) trace.RejectionCase {
	return trace.RejectionCase{Name: name, InitialState: state, Action: action, PreStateIdentity: identity(state), Code: code}
}

func identity(snapshot virtualtime.StateSnapshot) string {
	state, err := virtualtime.Restore(snapshot)
	if err != nil {
		panic(fmt.Sprintf("invalid formal bridge state: %v", err))
	}
	return state.Identity()
}
