/* What the Queries ask of the standalone activity's machines: the paths they run, their bounds, the
 * functional Queries, one per side effect that settles the activity, and the Queries that carry the
 * other promises into the lifted Model.
 */
package temporal
package standaloneactivity

import umpire.*
import umpire.realize.{Conformance, Outcome, RunExpectation}
import worker.{workerStop, Phase as WorkerPhase, State as WorkerState}
import Timeout.{expires, unset}

// ### The paths the Queries run
//
// Each starts where the protocol machine does, before the activity exists. A start that sets no
// deadline is written with its three inputs at their first value, `unset`.

/** The paths, kept apart from the Properties and Queries named after the same outcomes. */
object Paths:
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

  /** Both deadlines set and no attempt started: the schedule-to-start deadline fires first. */
  val bothDeadlinesStartFirst = activityProtocol.scenario.actions(
    start(Inputs.scheduleToClose := expires, Inputs.scheduleToStart := expires),
    scheduleToStart
  )

  /** The same start, and the schedule-to-close deadline fires first. */
  val bothDeadlinesCloseFirst = activityProtocol.scenario.actions(
    start(Inputs.scheduleToClose := expires, Inputs.scheduleToStart := expires),
    scheduleToClose
  )

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

// ### The bounds, for these Queries and the system contract's

val three = Limits(steps = 3, actions = 3, search = 4096)
val four = Limits(steps = 4, actions = 4, search = 32768)
val five = Limits(steps = 5, actions = 5, search = 65536)
val six = Limits(steps = 6, actions = 6, search = 262144)
val eight = Limits(steps = 8, actions = 8, search = 262144)

// ### The functional Queries

/** One Query per side effect that settles the activity, kept apart from the Properties they find. */
object Functional:
  val completion = (query find completes in Paths.completed limits three total 864)
    .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
  val nonRetryableFailure =
    (query find nonRetryableFails in Paths.nonRetryable limits three total 864)
      .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
  val retry =
    (query find retryCompletes in Paths.retriedThenCompleted limits six total 1728)
      .expect(
        RunExpectation(
          Conformance.conformant,
          Outcome.inconclusive,
          "the executions that explain the evidence disagree"
        )
      )
  val cancel =
    query find canceledByWorker in Paths.cancelRequestedThenCanceled limits four total 1152
  val terminate =
    (query find terminated in Paths.terminatedWhileScheduled limits three total 864)
      .expect(
        RunExpectation(
          Conformance.conformant,
          Outcome.inconclusive,
          "the executions that explain the evidence disagree"
        )
      )
  val pauseResume =
    (query find completes in Paths.pausedThenCompleted limits six total 1440)
      .expect(RunExpectation(Conformance.conformant, Outcome.satisfied))
  val scheduleToStartTimeout =
    (query find scheduleToStartFires in Paths.scheduleToStartExpires limits three total 864)
      .expect(
        RunExpectation(
          Conformance.conformant,
          Outcome.inconclusive,
          "an execution that explains the evidence never reaches the claim's evaluation point"
        )
      )
  val startToCloseTimeout =
    query find startToCloseFires in Paths.startToCloseExpires limits three total 864

  /** In declaration order. */
  val all: Vector[Query] = Vector(
    completion,
    nonRetryableFailure,
    retry,
    cancel,
    terminate,
    pauseResume,
    scheduleToStartTimeout,
    startToCloseTimeout
  )

// ### The Queries that carry the other promises

/**
 * Asks the one Property no functional Query asks, over the path that takes a cancel request. It is
 * not one of them: it is what carries the Property into the lifted Model, where it is compared with
 * Go's.
 */
val cancelRequest =
  query find cancelRequestedWhileStarted in Paths.cancelRequestedThenCanceled limits four total 1152

val terminalHolds = query verify terminalIsFinal in Paths.completed limits three total 864
val pauseHolds =
  query verify pausedIsNotDispatched in Paths.pausedThenCompleted limits six total 1440

/** Neither deadline is ordered before the other: each firing is a trace of its own. */
val competingTimers: Vector[Query] = Vector(
  query("competingTimers.scheduleToStartFirst")
    .find(scheduleToStartFires)
    .in(Paths.bothDeadlinesStartFirst) limits three total 576,
  query("competingTimers.scheduleToCloseFirst")
    .find(scheduleToCloseFires)
    .in(Paths.bothDeadlinesCloseFirst) limits three total 576
)

/** The cross-entity claim, verified over the path on which a stopped worker never takes the retry. */
val stoppedWorkerStartsNothing =
  query verify startedByPollingWorker in Paths.stoppedBeforeRetry limits six total 3456
