package temporal
package standaloneactivity

import umpire.*
import umpire.realize.{Conformance, Outcome, RunExpectation}
import worker.{Phase as WorkerPhase, State as WorkerState}

// ### What the machines promise

/**
 * Once an activity is over, no step changes its phase. Declared on the product machine and read on
 * the protocol machine through the map.
 */
val terminalIsFinal: Property[ProductState] =
  activityProduct.property("terminalIsFinal") holdsAcross { (before, after) =>
    !productTerminal(before) || after.state.phase == before.phase
  }

/**
 * A paused activity is dispatched to no worker: nothing moves it straight to started.
 *
 * The predicate fixes no state, outcome or fact: "not started" only says what must not follow
 * paused. It is kept as a function and verified as one, so it claims something all the same:
 * it fails the moment a row from paused to started appears, which is the regression it
 * guards.
 */
val pausedIsNotDispatched: Property[ProductState] =
  activityProduct.property("pausedIsNotDispatched") holdsAcross { (before, after) =>
    before.phase != ProductPhase.paused || after.state.phase != ProductPhase.started
  }

/** A completed answer settles the activity as completed, and the status records it. */
val completes: Property[ProtocolState] =
  activityProtocol.property("completes") when attemptResult(AttemptResult.completed) holds { s =>
    s.state.phase == Phase.completed && s.facts.contains(ProtocolFact.statusCompleted)
  }

/** A non-retryable failure settles the activity as failed. */
val nonRetryableFails: Property[ProtocolState] =
  activityProtocol.property("nonRetryableFails") when attemptResult(
    AttemptResult.failed(false)
  ) holds { s =>
    s.state.phase == Phase.failed && s.facts.contains(ProtocolFact.statusFailed)
  }

/** Completed on the second attempt of an activity with no deadline set. */
val completedOnRetry: ProtocolState =
  ProtocolState(Phase.completed, 2, Timeout.unset, Timeout.unset, Timeout.unset)

/** The retried attempt's completion settles the activity on its second attempt. */
val retryCompletes: Property[ProtocolState] =
  activityProtocol.property("retryCompletes") when attemptResult(AttemptResult.completed) holds {
    s =>
      s.state == completedOnRetry && s.facts.contains(ProtocolFact.statusCompleted)
  }

/** A cancel request is recorded as requested. */
val cancelRequestedWhileStarted: Property[ProtocolState] =
  activityProtocol.property("cancelRequestedWhileStarted") when control(
    Control.requestCancel
  ) holds { s =>
    s.state.phase == Phase.cancelRequested && s.facts.contains(ProtocolFact.statusCancelRequested)
  }

/** The worker's canceled answer settles a cancel-requested activity. */
val canceledByWorker: Property[ProtocolState] =
  activityProtocol.property("canceledByWorker") when attemptResult(AttemptResult.canceled) holds {
    s =>
      s.state.phase == Phase.canceled && s.facts.contains(ProtocolFact.statusCanceled)
  }

/** A terminate settles the activity as terminated. */
val terminated: Property[ProtocolState] =
  activityProtocol.property("terminated") when control(Control.terminate) holds { s =>
    s.state.phase == Phase.terminated && s.facts.contains(ProtocolFact.statusTerminated)
  }

/**
 * The schedule-to-start deadline times the activity out and the status records which deadline it
 * was.
 */
val scheduleToStartFires: Property[ProtocolState] =
  activityProtocol.property("scheduleToStartFires") when scheduleToStart holds { s =>
    s.state.phase == Phase.timedOut && s.facts.contains(
      ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart)
    )
  }

/** The start-to-close deadline times a held attempt out. */
val startToCloseFires: Property[ProtocolState] =
  activityProtocol.property("startToCloseFires") when startToClose holds { s =>
    s.state.phase == Phase.timedOut && s.facts.contains(
      ProtocolFact.statusTimedOut(TimeoutType.startToClose)
    )
  }

// ### The paths the Queries run

import Timeout.{expires, unset}

val completed: Scenario[ProtocolState] =
  activityProtocol
    .scenario("completed")
    .starts(unstarted)
    .actions(start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.completed))

val nonRetryable: Scenario[ProtocolState] =
  activityProtocol
    .scenario("nonRetryable")
    .starts(unstarted)
    .actions(start(unset, unset, unset), attemptStart, attemptResult(AttemptResult.failed(false)))

val retriedThenCompleted: Scenario[ProtocolState] = activityProtocol
  .scenario("retriedThenCompleted")
  .starts(unstarted)
  .actions(
    start(unset, unset, unset),
    attemptStart,
    attemptResult(AttemptResult.failed(true)),
    backoff,
    attemptStart,
    attemptResult(AttemptResult.completed)
  )

val cancelRequestedThenCanceled: Scenario[ProtocolState] = activityProtocol
  .scenario("cancelRequestedThenCanceled")
  .starts(unstarted)
  .actions(
    start(unset, unset, unset),
    attemptStart,
    control(Control.requestCancel),
    attemptResult(AttemptResult.canceled)
  )

/** The worker stops before the start, so no attempt is in flight when the caller terminates. */
val terminatedWhileScheduled: Scenario[ProtocolState] =
  activityProtocol
    .scenario("terminatedWhileScheduled")
    .starts(unstarted)
    .actions(start(unset, unset, unset), workerStop, control(Control.terminate))

val pausedThenCompleted: Scenario[ProtocolState] = activityProtocol
  .scenario("pausedThenCompleted")
  .starts(unstarted)
  .actions(
    start(unset, unset, unset),
    control(Control.pause),
    control(Control.unpause),
    attemptStart,
    attemptResult(AttemptResult.completed)
  )

val scheduleToStartExpires: Scenario[ProtocolState] =
  activityProtocol
    .scenario("scheduleToStartExpires")
    .starts(unstarted)
    .actions(start(unset, expires, unset), workerStop, scheduleToStart)

val startToCloseExpires: Scenario[ProtocolState] =
  activityProtocol
    .scenario("startToCloseExpires")
    .starts(unstarted)
    .actions(start(unset, unset, expires), attemptStart, startToClose)

val three: Limits = Limits("three", steps = 3, actions = 3, search = 4096)
val four: Limits = Limits("four", steps = 4, actions = 4, search = 32768)
val six: Limits = Limits("six", steps = 6, actions = 6, search = 262144)

// ### The Queries

val completion: Query = (query("completion") find completes in completed limits three total 864)
  .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
val nonRetryableFailure: Query =
  (query("nonRetryableFailure") find nonRetryableFails in nonRetryable limits three total 864)
    .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
val retry: Query =
  (query("retry") find retryCompletes in retriedThenCompleted limits six total 1728)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "the executions that explain the evidence disagree"
      )
    )
val cancel: Query =
  query("cancel") find canceledByWorker in cancelRequestedThenCanceled limits four total 1152
val terminate: Query =
  (query("terminate") find terminated in terminatedWhileScheduled limits three total 864)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "the executions that explain the evidence disagree"
      )
    )
val pauseResume: Query =
  (query("pauseResume") find completes in pausedThenCompleted limits six total 1440)
    .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
val scheduleToStartTimeout: Query =
  (query("scheduleToStartTimeout") find scheduleToStartFires in scheduleToStartExpires limits three)
    .total(864)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "an execution that explains the evidence never reaches the claim's evaluation point"
      )
    )
val startToCloseTimeout: Query =
  query("startToCloseTimeout") find startToCloseFires in startToCloseExpires limits three total 864

/**
 * Asks the one Property no functional Query asks, over the path that takes a cancel request. It is
 * not one of them: it is what carries the Property into the lifted Model, where it is compared with
 * Go's.
 */
val cancelRequest: Query =
  query("cancelRequest") find cancelRequestedWhileStarted in
    cancelRequestedThenCanceled limits four total 1152

val terminalHolds: Query =
  query("terminalHolds") verify terminalIsFinal in completed limits three total 864
val pauseHolds: Query =
  query("pauseHolds") verify pausedIsNotDispatched in pausedThenCompleted limits six total 1440

/** The functional Queries in declaration order. */
val functionalQueries: Vector[Query] = Vector(
  completion,
  nonRetryableFailure,
  retry,
  cancel,
  terminate,
  pauseResume,
  scheduleToStartTimeout,
  startToCloseTimeout
)

// ### The cross-entity claim

/** Every attempt start leaves the worker polling: no stopped worker starts an attempt. */
val startedByPollingWorker: Property[StandaloneActivityState] =
  standaloneActivity.property("startedByPollingWorker").whenAction("attemptStart") holds
    (_.state.worker.phase == WorkerPhase.polling)

/**
 * The first attempt is started by the polling worker and fails retryably; the backoff returns the
 * activity to scheduled; the worker then stops, so the retry is never dispatched and the
 * schedule-to-start deadline fires. The attempt start on the path is what makes the verification
 * exercise the claim rather than pass for want of a firing.
 */
val stoppedBeforeRetry: Scenario[StandaloneActivityState] = standaloneActivity
  .scenario("stoppedBeforeRetry")
  .starts(StandaloneActivityState(unstarted, WorkerState(WorkerPhase.polling)))
  .actionKeys(
    "activity_start-unset-expires-unset",
    "attemptStart",
    "activity_attemptResult-failed-true",
    "activity_backoff",
    "workerStop",
    "activity_scheduleToStart"
  )

/** The cross-entity claim, verified over that path. */
val stoppedWorkerStartsNothing: Query =
  query("stoppedWorkerStartsNothing") verify startedByPollingWorker in
    stoppedBeforeRetry limits six total 3456
