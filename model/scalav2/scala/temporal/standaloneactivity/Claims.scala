package temporal
package standaloneactivity

import umpire.*
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
 * Lean rejects this Property: its elaborator lowers a predicate into clauses that each fix a state,
 * outcome or fact, and "not started" fixes none, so it reports that the predicate claims nothing.
 * Scala keeps the predicate as a function and verifies it; it fails the moment a row from paused to
 * started appears, which is the regression it guards.
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

/** A product Property read on the protocol machine goes through the declared refinement. */
given Reads[ProtocolState, ProductState] = Reads.through(activityProtocol, activityProduct)

val completion: Query = query("completion") find completes in completed limits three
val nonRetryableFailure: Query =
  query("nonRetryableFailure") find nonRetryableFails in nonRetryable limits three
val retry: Query = query("retry") find retryCompletes in retriedThenCompleted limits six
val cancel: Query = query("cancel") find canceledByWorker in cancelRequestedThenCanceled limits four
val terminate: Query = query("terminate") find terminated in terminatedWhileScheduled limits three
val pauseResume: Query = query("pauseResume") find completes in pausedThenCompleted limits six
val scheduleToStartTimeout: Query =
  query("scheduleToStartTimeout") find scheduleToStartFires in scheduleToStartExpires limits three
val startToCloseTimeout: Query =
  query("startToCloseTimeout") find startToCloseFires in startToCloseExpires limits three

/**
 * Asks the one Property no Query of the sets asks, over the path that takes a cancel request. It is
 * in no set: it is what carries the Property into the lifted Model, where it is compared with Go's.
 */
val cancelRequest: Query =
  query("cancelRequest") find cancelRequestedWhileStarted in cancelRequestedThenCanceled limits four

val terminalHolds: Query = query("terminalHolds") verify terminalIsFinal in completed limits three
val pauseHolds: Query =
  query("pauseHolds") verify pausedIsNotDispatched in pausedThenCompleted limits six

/** The functional set's Queries in declaration order. */
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

// ### The sets
//
// Standalone activities exist only under CHASM, so the functional set does not repeat over the
// implementation switch.

private val drivenAll: Map[Party, Binding] =
  Map(caller -> Binding.driven, worker.party -> Binding.driven)

/** The functional set. */
val standaloneActivityTests: UmpireSet =
  UmpireSet("standaloneActivityTests", Purpose.functional, drivenAll, queries = functionalQueries)

/** The canary set: the worker is observed. */
val standaloneActivityCanary: UmpireSet = UmpireSet(
  "standaloneActivityCanary",
  Purpose.canary,
  drivenAll.updated(worker.party, Binding.observed),
  queries = Vector(completion, cancel)
)

/** The exploratory set over the protocol machine. */
val standaloneActivityExploration: UmpireSet = UmpireSet(
  "standaloneActivityExploration",
  Purpose.exploratory,
  drivenAll,
  machine = Some(activityProtocol),
  cover = Vector(CoverageGoal.rows, CoverageGoal.results, CoverageGoal.classMembers),
  budget = Some(four)
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
  query("stoppedWorkerStartsNothing") verify startedByPollingWorker in stoppedBeforeRetry limits six
