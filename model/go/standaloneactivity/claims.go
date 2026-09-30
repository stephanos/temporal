package standaloneactivity

import (
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

// ### What the machines promise

// TerminalIsFinal claims that once an activity is over, no step changes its phase. Declared on the
// product machine and read on the protocol machine through the map.
var TerminalIsFinal = ActivityProduct.Property("terminalIsFinal").
	HoldsAcross(func(before ProductState, after ProductStep) bool {
		return !ProductTerminal(before) || after.State.Phase == before.Phase
	})

// PausedIsNotDispatched claims that a paused activity is dispatched to no worker: nothing moves it
// straight to started.
//
// Lean rejects this Property: its elaborator lowers a predicate into clauses that each fix a state,
// outcome or fact, and "not started" fixes none, so it reports that the predicate claims nothing.
// Go keeps the predicate as a function and verifies it; it fails the moment a row from paused to
// started appears, which is the regression it guards.
var PausedIsNotDispatched = ActivityProduct.Property("pausedIsNotDispatched").
	HoldsAcross(func(before ProductState, after ProductStep) bool {
		return before.Phase != ProductPaused || after.State.Phase != ProductStarted
	})

// Completes claims that a completed answer settles the activity as completed, and the status
// records it.
var Completes = ActivityProtocol.Property("completes").
	When(attemptResult.With(Completed{})).
	Holds(func(s ProtocolStep) bool { return s.State.Phase == PhaseCompleted && records(s, StatusCompleted{}) })

// NonRetryableFails claims that a non-retryable failure settles the activity as failed.
var NonRetryableFails = ActivityProtocol.Property("nonRetryableFails").
	When(attemptResult.With(Failed{Retryable: false})).
	Holds(func(s ProtocolStep) bool { return s.State.Phase == PhaseFailed && records(s, StatusFailed{}) })

// completedOnRetry is completed on the second attempt of an activity with no deadline set.
var completedOnRetry = ProtocolState{Phase: PhaseCompleted, Attempts: 2, ScheduleToClose: Unset,
	ScheduleToStart: Unset, StartToClose: Unset}

// RetryCompletes claims that the retried attempt's completion settles the activity on its second
// attempt.
var RetryCompletes = ActivityProtocol.Property("retryCompletes").
	When(attemptResult.With(Completed{})).
	Holds(func(s ProtocolStep) bool { return s.State == completedOnRetry && records(s, StatusCompleted{}) })

// CancelRequestedWhileStarted claims that a cancel request is recorded as requested.
var CancelRequestedWhileStarted = ActivityProtocol.Property("cancelRequestedWhileStarted").
	When(control.With(RequestCancel)).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == CancelRequested && records(s, StatusCancelRequested{})
	})

// CanceledByWorker claims that the worker's canceled answer settles a cancel-requested activity.
var CanceledByWorker = ActivityProtocol.Property("canceledByWorker").
	When(attemptResult.With(CanceledByRun{})).
	Holds(func(s ProtocolStep) bool { return s.State.Phase == PhaseCanceled && records(s, StatusCanceled{}) })

// TerminatedClaim claims that a terminate settles the activity as terminated.
var TerminatedClaim = ActivityProtocol.Property("terminated").
	When(control.With(Terminate)).
	Holds(func(s ProtocolStep) bool { return s.State.Phase == Terminated && records(s, StatusTerminated{}) })

// ScheduleToStartFires claims that the schedule-to-start deadline times the activity out and the
// status records which deadline it was.
var ScheduleToStartFires = ActivityProtocol.Property("scheduleToStartFires").
	When(scheduleToStart.With()).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == TimedOut && records(s, StatusTimedOut{TypeScheduleToStart})
	})

// StartToCloseFires claims that the start-to-close deadline times a held attempt out.
var StartToCloseFires = ActivityProtocol.Property("startToCloseFires").
	When(startToClose.With()).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == TimedOut && records(s, StatusTimedOut{TypeStartToClose})
	})

func records(s ProtocolStep, fact ProtocolFact) bool {
	for _, f := range s.Facts {
		if f == fact {
			return true
		}
	}
	return false
}

// ### The paths the Queries run

var unstarted = ProtocolState{Phase: Unstarted, ScheduleToClose: Unset, ScheduleToStart: Unset,
	StartToClose: Unset}

func path(name string, classes ...umpire.Class) *umpire.Scenario[ProtocolState] {
	return ActivityProtocol.Scenario(name).Starts(unstarted).Actions(classes...)
}

var (
	CompletedPath = path("completed",
		start.With(Unset, Unset, Unset), attemptStart.With(), attemptResult.With(Completed{}))
	NonRetryablePath = path("nonRetryable",
		start.With(Unset, Unset, Unset), attemptStart.With(), attemptResult.With(Failed{Retryable: false}))
	RetriedThenCompleted = path("retriedThenCompleted",
		start.With(Unset, Unset, Unset), attemptStart.With(), attemptResult.With(Failed{Retryable: true}),
		backoff.With(), attemptStart.With(), attemptResult.With(Completed{}))
	CancelRequestedThenCanceled = path("cancelRequestedThenCanceled",
		start.With(Unset, Unset, Unset), attemptStart.With(), control.With(RequestCancel),
		attemptResult.With(CanceledByRun{}))
	// TerminatedWhileScheduled: the worker stops before the start, so no attempt is in flight when
	// the caller terminates.
	TerminatedWhileScheduled = path("terminatedWhileScheduled",
		start.With(Unset, Unset, Unset), workerStop.With(), control.With(Terminate))
	PausedThenCompleted = path("pausedThenCompleted",
		start.With(Unset, Unset, Unset), control.With(Pause), control.With(Unpause), attemptStart.With(),
		attemptResult.With(Completed{}))
	ScheduleToStartExpires = path("scheduleToStartExpires",
		start.With(Unset, Expires, Unset), workerStop.With(), scheduleToStart.With())
	StartToCloseExpires = path("startToCloseExpires",
		start.With(Unset, Unset, Expires), attemptStart.With(), startToClose.With())
)

var (
	Three = umpire.Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}
	Four  = umpire.Limits{Name: "four", Steps: 4, Actions: 4, Search: 32768}
	Six   = umpire.Limits{Name: "six", Steps: 6, Actions: 6, Search: 262144}
)

// ### The Queries

var viaProduct = ActivityProtocol.Via(ActivityProduct)

var (
	Completion             = CompletedPath.Find("completion", Completes, Three)
	NonRetryableFailure    = NonRetryablePath.Find("nonRetryableFailure", NonRetryableFails, Three)
	Retry                  = RetriedThenCompleted.Find("retry", RetryCompletes, Six)
	Cancel                 = CancelRequestedThenCanceled.Find("cancel", CanceledByWorker, Four)
	TerminateQuery         = TerminatedWhileScheduled.Find("terminate", TerminatedClaim, Three)
	PauseResume            = PausedThenCompleted.Find("pauseResume", Completes, Six)
	ScheduleToStartTimeout = ScheduleToStartExpires.Find("scheduleToStartTimeout", ScheduleToStartFires, Three)
	StartToCloseTimeout    = StartToCloseExpires.Find("startToCloseTimeout", StartToCloseFires, Three)

	TerminalHolds = CompletedPath.VerifyRefined("terminalHolds", TerminalIsFinal, viaProduct, Three)
	PauseHolds    = PausedThenCompleted.VerifyRefined("pauseHolds", PausedIsNotDispatched, viaProduct, Six)
)

// FunctionalQueries is the functional set's Queries in declaration order.
var FunctionalQueries = []*umpire.Query{Completion, NonRetryableFailure, Retry, Cancel, TerminateQuery,
	PauseResume, ScheduleToStartTimeout, StartToCloseTimeout}

// ### The sets
//
// Standalone activities exist only under CHASM, so the functional set does not repeat over the
// implementation switch.

// StandaloneActivityTests is the functional set.
var StandaloneActivityTests = &umpire.Set{Name: "standaloneActivityTests", Purpose: umpire.Functional,
	Bindings: map[umpire.Party]umpire.Binding{Caller: umpire.Driven, worker.Party: umpire.Driven},
	Queries:  FunctionalQueries}

// StandaloneActivityCanary is the canary set: the worker is observed.
var StandaloneActivityCanary = &umpire.Set{Name: "standaloneActivityCanary", Purpose: umpire.Canary,
	Bindings: map[umpire.Party]umpire.Binding{Caller: umpire.Driven, worker.Party: umpire.Observed},
	Queries:  []*umpire.Query{Completion, Cancel}}

// StandaloneActivityExploration is the exploratory set over the protocol machine.
var StandaloneActivityExploration = &umpire.Set{Name: "standaloneActivityExploration",
	Purpose:  umpire.Exploratory,
	Bindings: map[umpire.Party]umpire.Binding{Caller: umpire.Driven, worker.Party: umpire.Driven},
	Machine:  ActivityProtocol,
	Cover:    []umpire.CoverageGoal{umpire.CoverRows, umpire.CoverResults, umpire.CoverClassMembers},
	Budget:   Four}

// ### The cross-entity claim

// StartedByPollingWorker claims that every attempt start leaves the worker polling: no stopped
// worker starts an attempt.
var StartedByPollingWorker = StandaloneActivity.Property("startedByPollingWorker").
	WhenAction("attemptStart").
	Holds(func(s umpire.Step[StandaloneActivityState, string, string]) bool {
		return s.State.Worker.Phase == worker.Polling
	})

// StoppedBeforeRetry is the path where the first attempt is started by the polling worker and fails
// retryably; the backoff returns the activity to scheduled; the worker then stops, so the retry is
// never dispatched and the schedule-to-start deadline fires. The attempt start on the path is what
// makes the verification exercise the claim rather than pass for want of a firing.
var StoppedBeforeRetry = StandaloneActivity.Scenario("stoppedBeforeRetry").
	Starts(StandaloneActivityState{Activity: unstarted, Worker: worker.State{Phase: worker.Polling}}).
	ActionKeys(
		StandaloneActivity.Own("activity", start.With(Unset, Expires, Unset)),
		"attemptStart",
		StandaloneActivity.Own("activity", attemptResult.With(Failed{Retryable: true})),
		StandaloneActivity.Own("activity", backoff.With()),
		"workerStop",
		StandaloneActivity.Own("activity", scheduleToStart.With()))

// StoppedWorkerStartsNothing verifies the cross-entity claim over that path.
var StoppedWorkerStartsNothing = StoppedBeforeRetry.Verify("stoppedWorkerStartsNothing",
	StartedByPollingWorker, Six)
