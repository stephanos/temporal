package nexuscaller

import (
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/go/worker"
)

// ### What the machines promise
//
// A same-step claim names the action it is about under When and holds of the step that action
// produces; a transition claim holds of the state before and the step after. A functional Query
// realizes a same-step claim, because the Case's Contract is the claim's clause triggered by the
// action the Case performs; a transition claim is searched and verified, never realized.

// TerminalIsFinal claims once an operation is over, no step changes its phase. Declared on the product
// machine and read on the protocol machine through the map.
var TerminalIsFinal = NexusProduct.Property("terminalIsFinal").
	HoldsAcross(func(before ProductState, after ProductStep) bool {
		return !ProductTerminal(before) || after.State.Phase == before.Phase
	})

// SyncSucceeds claims a synchronous reply settles the operation as succeeded, and the completed event
// records it.
var SyncSucceeds = NexusProtocol.Property("syncSucceeds").
	When(handlerReply.With(SyncSuccess{})).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == Succeeded && records(s, NexusOperationCompleted{})
	})

// AsyncStarts claims an asynchronous reply starts the operation, and the started event records it.
var AsyncStarts = NexusProtocol.Property("asyncStarts").
	When(handlerReply.With(Async{})).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == Started && records(s, NexusOperationStarted{})
	})

// CompletionSucceeds claims a successful completion is recorded by the completed event. Neither the phase
// nor the outcome is fixed: a completion resolves any running phase, and accepted is every earlier
// step's outcome too, so a clause fixing it would be answered before the completion.
var CompletionSucceeds = NexusProtocol.Property("completionSucceeds").
	When(complete.With(ResolvedSucceeded)).
	Holds(func(s ProtocolStep) bool { return records(s, NexusOperationCompleted{}) })

// CompletionFails claims a failed completion is recorded by the failed event.
var CompletionFails = NexusProtocol.Property("completionFails").
	When(complete.With(ResolvedFailed)).
	Holds(func(s ProtocolStep) bool { return records(s, NexusOperationFailed{}) })

// HandlerErrorFails claims a non-retryable handler error settles the operation as failed, and the failed
// event records it.
var HandlerErrorFails = NexusProtocol.Property("handlerErrorFails").
	When(handlerReply.With(HandlerError{Retryable: false})).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == Failed && records(s, NexusOperationFailed{})
	})

// succeededOnRetry is succeeded on the second attempt of an operation with no deadline set. A claim
// fixes one state, so every field is named.
var succeededOnRetry = ProtocolState{Phase: Succeeded, Attempts: 1, ScheduleToClose: Unset,
	ScheduleToStart: Unset, StartToClose: Unset}

// RetrySucceeds claims a synchronous reply to the retried attempt settles the operation as succeeded on
// its second attempt: the count the retryable failure raised is still one, and the completed event
// records the reply.
var RetrySucceeds = NexusProtocol.Property("retrySucceeds").
	When(handlerReply.With(SyncSuccess{})).
	Holds(func(s ProtocolStep) bool {
		return s.State == succeededOnRetry && records(s, NexusOperationCompleted{})
	})

// ScheduleToStartFires claims the schedule-to-start deadline settles an operation no handler started as
// timed out, and the timed-out event records which deadline it was.
var ScheduleToStartFires = NexusProtocol.Property("scheduleToStartFires").
	When(scheduleToStart.With()).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == TimedOut && records(s, NexusOperationTimedOut{TypeScheduleToStart})
	})

// StartToCloseFires claims the start-to-close deadline settles a started operation no handler completed
// as timed out.
var StartToCloseFires = NexusProtocol.Property("startToCloseFires").
	When(startToClose.With()).
	Holds(func(s ProtocolStep) bool {
		return s.State.Phase == TimedOut && records(s, NexusOperationTimedOut{TypeStartToClose})
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
//
// A protocol Scenario names its classed actions with their inputs and its start. Each path below is
// one upstream functional test's shape: the schedule command with no deadline set, then the side
// effects that settle the operation.

var unscheduled = ProtocolState{Phase: Unscheduled, ScheduleToClose: Unset, ScheduleToStart: Unset,
	StartToClose: Unset}

var (
	SyncReplied = NexusProtocol.Scenario("syncReplied").Starts(unscheduled).
			Actions(schedule.With(Unset, Unset, Unset), handlerReply.With(SyncSuccess{}))

	AsyncThenSucceeded = NexusProtocol.Scenario("asyncThenSucceeded").Starts(unscheduled).
				Actions(schedule.With(Unset, Unset, Unset), handlerReply.With(Async{}),
			complete.With(ResolvedSucceeded))

	AsyncThenFailed = NexusProtocol.Scenario("asyncThenFailed").Starts(unscheduled).
			Actions(schedule.With(Unset, Unset, Unset), handlerReply.With(Async{}),
			complete.With(ResolvedFailed))

	NonRetryableError = NexusProtocol.Scenario("nonRetryableError").Starts(unscheduled).
				Actions(schedule.With(Unset, Unset, Unset), handlerReply.With(HandlerError{Retryable: false}))

	// RetriedThenSucceeded: the retryable error backs the operation off; the backoff timer fires and
	// records nothing; the retried attempt is answered synchronously.
	RetriedThenSucceeded = NexusProtocol.Scenario("retriedThenSucceeded").Starts(unscheduled).
				Actions(schedule.With(Unset, Unset, Unset), handlerReply.With(HandlerError{Retryable: true}),
			backoff.With(), handlerReply.With(SyncSuccess{}))

	// ScheduleToStartExpires: the schedule command sets the schedule-to-start deadline; the handler's
	// worker stops, so nothing answers the start request; the deadline fires. The worker stops after
	// the schedule in the operation's order, where the stop changes nothing; the realization stops it
	// before the workflow starts, where the stop cannot race the dispatch.
	ScheduleToStartExpires = NexusProtocol.Scenario("scheduleToStartExpires").Starts(unscheduled).
				Actions(schedule.With(Unset, Expires, Unset), workerStop.With(), scheduleToStart.With())

	// StartToCloseExpires: the schedule command sets the start-to-close deadline; the handler accepts
	// asynchronously and never completes; the deadline fires.
	StartToCloseExpires = NexusProtocol.Scenario("startToCloseExpires").Starts(unscheduled).
				Actions(schedule.With(Unset, Unset, Expires), handlerReply.With(Async{}), startToClose.With())
)

// Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
// sequence of two is found among ninety-nine candidates, one of three among about a thousand and
// one of four among about ten thousand.
var (
	Two   = umpire.Limits{Name: "two", Steps: 2, Actions: 2, Search: 512}
	Three = umpire.Limits{Name: "three", Steps: 3, Actions: 3, Search: 4096}
	Four  = umpire.Limits{Name: "four", Steps: 4, Actions: 4, Search: 32768}
)

// ### The Queries
//
// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one
// backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after
// an asynchronous reply. Each finds its same-step claim on its path and is realized by the set
// below. The product claim is verified over every trace of one path, outside the set, because a
// verify Query realizes nothing.

var (
	SyncCompletion         = SyncReplied.Find("syncCompletion", SyncSucceeds, Two)
	AsyncCompletion        = AsyncThenSucceeded.Find("asyncCompletion", CompletionSucceeds, Three)
	AsyncFailure           = AsyncThenFailed.Find("asyncFailure", CompletionFails, Three)
	HandlerErrorQuery      = NonRetryableError.Find("handlerError", HandlerErrorFails, Two)
	Retry                  = RetriedThenSucceeded.Find("retry", RetrySucceeds, Four)
	ScheduleToStartTimeout = ScheduleToStartExpires.Find("scheduleToStartTimeout", ScheduleToStartFires, Three)
	StartToCloseTimeout    = StartToCloseExpires.Find("startToCloseTimeout", StartToCloseFires, Three)

	TerminalHolds = AsyncThenSucceeded.VerifyRefined("terminalHolds", TerminalIsFinal,
		NexusProtocol.Via(NexusProduct), Three)
)

// FunctionalQueries is the functional set's Queries in declaration order.
var FunctionalQueries = []*umpire.Query{SyncCompletion, AsyncCompletion, AsyncFailure,
	HandlerErrorQuery, Retry, ScheduleToStartTimeout, StartToCloseTimeout}

// NexusCallerTests is the functional set.
//
// Every party but system is bound: the Case drives the caller, the handler and the worker, and
// observes the network. The set repeats over the implementation switch, so each Query's Case runs
// once under HSM and once under CHASM.
var NexusCallerTests = &umpire.Set{Name: "nexusCallerTests", Purpose: umpire.Functional,
	Bindings: map[umpire.Party]umpire.Binding{Caller: umpire.Driven, Handler: umpire.Driven,
		Network: umpire.Observed, worker.Party: umpire.Driven},
	Repeat:  "implementation",
	Queries: FunctionalQueries}

// NexusCallerCanary is the canary set.
//
// A canary runs a Query against a deployment that performs the handler's part itself: the handler
// is observed, so the verifier reads which reply occurred and checks the machine allows it. What
// admits a canary is that a deployment can close every gap its Case carries, and every step of the
// sync and async completion paths records evidence; a path with a silent step -- the backoff, the
// worker stop -- is a capability gap no deployment closes, so a canary naming it is rejected.
var NexusCallerCanary = &umpire.Set{Name: "nexusCallerCanary", Purpose: umpire.Canary,
	Bindings: map[umpire.Party]umpire.Binding{Caller: umpire.Driven, Handler: umpire.Observed,
		Network: umpire.Observed, worker.Party: umpire.Driven},
	Queries: []*umpire.Query{SyncCompletion, AsyncCompletion}}

// NexusCallerExploration is the exploratory set.
//
// An exploration covers the protocol machine rather than listing Queries. Its targets are the rows
// an exploration within the budget's steps of a start can take, the results those rows reach and
// the members of the classes their actions claim, each in the machine's catalog order and cut at the
// budget's search count, so the enumeration is the same on every reading.
var NexusCallerExploration = &umpire.Set{Name: "nexusCallerExploration", Purpose: umpire.Exploratory,
	Bindings: map[umpire.Party]umpire.Binding{Caller: umpire.Driven, Handler: umpire.Driven,
		Network: umpire.Observed, worker.Party: umpire.Driven},
	Machine: NexusProtocol,
	Cover:   []umpire.CoverageGoal{umpire.CoverRows, umpire.CoverResults, umpire.CoverClassMembers},
	Budget:  Four}

// ### The cross-entity claim

// RepliedByPollingWorker claims every reply, of any class, leaves the handler's worker polling: no
// handler replies while its worker is stopped.
var RepliedByPollingWorker = NexusCaller.Property("repliedByPollingWorker").
	WhenAction("handlerReply").
	Holds(func(s umpire.Step[NexusCallerState, string, string]) bool {
		return s.State.Worker.Phase == worker.Polling
	})

// RepliedThenStopped is the path where a retryable reply backs the operation off; the handler's worker then stops,
// so the retried attempt is never answered and the schedule-to-start deadline fires.
var RepliedThenStopped = NexusCaller.Scenario("repliedThenStopped").
	Starts(NexusCallerState{Operation: unscheduled, Worker: worker.State{Phase: worker.Polling}}).
	ActionKeys(
		NexusCaller.Own("operation", schedule.With(Unset, Expires, Unset)),
		NexusCaller.Synced("handlerReply", handlerReply.With(HandlerError{Retryable: true})),
		"workerStop",
		NexusCaller.Own("operation", scheduleToStart.With()))

// StoppedWorkerRepliesNothing verifies the cross-entity claim over that path.
var StoppedWorkerRepliesNothing = RepliedThenStopped.Verify("stoppedWorkerRepliesNothing",
	RepliedByPollingWorker, Four)
