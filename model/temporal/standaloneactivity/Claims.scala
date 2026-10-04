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
val terminalIsFinal = activityProduct.property.once(terminal).keeps(_.phase)

/**
 * A paused activity is dispatched to no worker: nothing moves it straight to started.
 *
 * The predicate fixes no state, outcome or fact: "not started" only says what must not follow
 * paused. It is kept as a function and verified as one, so it claims something all the same:
 * it fails the moment a row from paused to started appears, which is the regression it
 * guards.
 */
val pausedIsNotDispatched = activityProduct.property.never(s => running(s.state)).from(paused)

/** A completed answer settles the activity as completed, and the status records it. */
val completes = activityProtocol.property when attemptResult(AttemptResult.completed) holds { s =>
  s.state.phase == Phase.completed && s.records(ProtocolFact.statusCompleted)
}

/** A non-retryable failure settles the activity as failed. */
val nonRetryableFails =
  activityProtocol.property when attemptResult(AttemptResult.failed(false)) holds { s =>
    s.state.phase == Phase.failed && s.records(ProtocolFact.statusFailed)
  }

/** Completed on the second attempt of an activity with no deadline set. */
val completedOnRetry: ProtocolState =
  ProtocolState(Phase.completed, UpTo(attemptBound), Timeout.unset, Timeout.unset, Timeout.unset)

/**
 * The retried attempt's completion settles the activity on its second attempt. The attempt count
 * saturates at `attemptBound`, so the claim is bounded by it: a completion on any later attempt reads
 * as this one.
 */
val retryCompletes =
  activityProtocol.property when attemptResult(AttemptResult.completed) holds { s =>
    s.state == completedOnRetry && s.records(ProtocolFact.statusCompleted)
  }

/** A cancel request is recorded as requested. */
val cancelRequestedWhileStarted =
  activityProtocol.property when control(Control.requestCancel) holds { s =>
    s.state.phase == Phase.cancelRequested && s.records(ProtocolFact.statusCancelRequested)
  }

/** The worker's canceled answer settles a cancel-requested activity. */
val canceledByWorker =
  activityProtocol.property when attemptResult(AttemptResult.canceled) holds { s =>
    s.state.phase == Phase.canceled && s.records(ProtocolFact.statusCanceled)
  }

/** A terminate settles the activity as terminated. */
val terminated = activityProtocol.property when control(Control.terminate) holds { s =>
  s.state.phase == Phase.terminated && s.records(ProtocolFact.statusTerminated)
}

/**
 * The schedule-to-start deadline times the activity out and the status records which deadline it
 * was.
 */
val scheduleToStartFires = activityProtocol.property when scheduleToStart holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToStart))
}

/** The start-to-close deadline times a held attempt out. */
val startToCloseFires = activityProtocol.property when startToClose holds { s =>
  s.state.phase == Phase.timedOut &&
  s.records(ProtocolFact.statusTimedOut(TimeoutType.startToClose))
}

// ### The paths the Queries run
//
// Each starts where the protocol machine does, before the activity exists. A start that sets no
// deadline is written with its three inputs at their first value, `unset`.

import Timeout.{expires, unset}

val completed = activityProtocol.scenario.actions(
  start(unset, unset, unset),
  attemptStart,
  attemptResult(AttemptResult.completed)
)

val nonRetryable = activityProtocol.scenario.actions(
  start(unset, unset, unset),
  attemptStart,
  attemptResult(AttemptResult.failed(false))
)

val retriedThenCompleted = activityProtocol.scenario.actions(
  start(unset, unset, unset),
  attemptStart,
  attemptResult(AttemptResult.failed(true)),
  backoff,
  attemptStart,
  attemptResult(AttemptResult.completed)
)

val cancelRequestedThenCanceled = activityProtocol.scenario.actions(
  start(unset, unset, unset),
  attemptStart,
  control(Control.requestCancel),
  attemptResult(AttemptResult.canceled)
)

/** The worker stops before the start, so no attempt is in flight when the caller terminates. */
val terminatedWhileScheduled = activityProtocol.scenario.actions(
  start(unset, unset, unset),
  workerStop,
  control(Control.terminate)
)

val pausedThenCompleted = activityProtocol.scenario.actions(
  start(unset, unset, unset),
  control(Control.pause),
  control(Control.unpause),
  attemptStart,
  attemptResult(AttemptResult.completed)
)

val scheduleToStartExpires = activityProtocol.scenario.actions(
  start(Inputs.scheduleToStart := expires),
  workerStop,
  scheduleToStart
)

val startToCloseExpires = activityProtocol.scenario.actions(
  start(Inputs.startToClose := expires),
  attemptStart,
  startToClose
)

val three = Limits(steps = 3, actions = 3, search = 4096)
val four = Limits(steps = 4, actions = 4, search = 32768)
val six = Limits(steps = 6, actions = 6, search = 262144)

// ### The Queries

val completion = (query find completes in completed limits three total 864)
  .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
val nonRetryableFailure =
  (query find nonRetryableFails in nonRetryable limits three total 864)
    .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
val retry =
  (query find retryCompletes in retriedThenCompleted limits six total 1728)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "the executions that explain the evidence disagree"
      )
    )
val cancel = query find canceledByWorker in cancelRequestedThenCanceled limits four total 1152
val terminate =
  (query find terminated in terminatedWhileScheduled limits three total 864)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "the executions that explain the evidence disagree"
      )
    )
val pauseResume =
  (query find completes in pausedThenCompleted limits six total 1440)
    .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
val scheduleToStartTimeout =
  (query find scheduleToStartFires in scheduleToStartExpires limits three total 864)
    .expect(
      RunExpectation(
        Conformance.conformant,
        Outcome.inconclusive,
        "an execution that explains the evidence never reaches the claim's evaluation point"
      )
    )
val startToCloseTimeout =
  query find startToCloseFires in startToCloseExpires limits three total 864

/**
 * Asks the one Property no functional Query asks, over the path that takes a cancel request. It is
 * not one of them: it is what carries the Property into the lifted Model, where it is compared with
 * Go's.
 */
val cancelRequest =
  query find cancelRequestedWhileStarted in cancelRequestedThenCanceled limits four total 1152

val terminalHolds = query verify terminalIsFinal in completed limits three total 864
val pauseHolds = query verify pausedIsNotDispatched in pausedThenCompleted limits six total 1440

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
val startedByPollingWorker = standaloneActivity.property
  .whenAction(standaloneActivity.synced(_.activity -> attemptStart))
  .holds(_.state.worker.phase == WorkerPhase.polling)

/**
 * The first attempt is started by the polling worker and fails retryably; the backoff returns the
 * activity to scheduled; the worker then stops, so the retry is never dispatched and the
 * schedule-to-start deadline fires. The attempt start on the path is what makes the verification
 * exercise the claim rather than pass for want of a firing.
 *
 * It starts where both members do, and says so: a start left to default would take the worker's
 * from Worker.scala, and fn-115's migration golden compares the file of each position.
 */
val stoppedBeforeRetry = standaloneActivity.scenario
  .starts(StandaloneActivityState(unstarted, WorkerState(WorkerPhase.polling)))
  .actions(
    standaloneActivity.own(_.activity, start(Inputs.scheduleToStart := expires)),
    standaloneActivity.synced(_.activity -> attemptStart),
    standaloneActivity.own(_.activity, attemptResult(AttemptResult.failed(true))),
    standaloneActivity.own(_.activity, backoff),
    standaloneActivity.synced(_.activity -> workerStop),
    standaloneActivity.own(_.activity, scheduleToStart)
  )

/** The cross-entity claim, verified over that path. */
val stoppedWorkerStartsNothing =
  query verify startedByPollingWorker in stoppedBeforeRetry limits six total 3456
