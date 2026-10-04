/* The paths, bounds and Queries of the Nexus caller's machines: a functional Query per side effect
 * that settles the operation, the product claim read on a protocol path, the cross-entity Query and
 * the forged control's Query.
 */
package temporal
package nexuscaller

import umpire.*
import umpire.realize.{Alternative, Conformance, Exploration, Outcome, RunExpectation, Variation}
import worker.workerStop
import Control.inspect
import Timeout.expires

// What a live Run is expected to show.
val satisfied = RunExpectation(Conformance.conformant, Outcome.satisfied)
def inconclusive(reason: String): RunExpectation =
  RunExpectation(Conformance.conformant, Outcome.inconclusive, reason)
val explanationsDisagree = "the executions that explain the evidence disagree"
val neverEvaluated =
  "an execution that explains the evidence never reaches the claim's evaluation point"

// ### The paths the Queries run
//
// Each path below is one upstream functional test's shape, from before the operation exists: the
// schedule command, then the side effects that settle the operation. A schedule that sets no
// deadline is `schedule()`, each input at `unset`.

val syncReplied = nexusProtocol.scenario.actions(schedule(), handlerReply(Reply.syncSuccess))

val asyncThenSucceeded = nexusProtocol.scenario
  .actions(schedule(), handlerReply(Reply.async), complete(Resolution.succeeded))

val asyncThenFailed = nexusProtocol.scenario
  .actions(schedule(), handlerReply(Reply.async), complete(Resolution.failed))

val nonRetryableError =
  nexusProtocol.scenario.actions(schedule(), handlerReply(Reply.handlerError(false)))

/**
 * The retryable error backs the operation off; the backoff timer fires and records nothing; the
 * retried attempt is answered synchronously.
 */
val retriedThenSucceeded = nexusProtocol.scenario.actions(
  schedule(),
  handlerReply(Reply.handlerError(true)),
  backoff,
  handlerReply(Reply.syncSuccess)
)

/**
 * The schedule command sets the schedule-to-start deadline; the handler's worker stops, so nothing
 * answers the start request; the deadline fires. The worker stops after the schedule in the
 * operation's order, where the stop changes nothing; the realization stops it before the workflow
 * starts, where the stop cannot race the dispatch.
 */
val scheduleToStartExpires = nexusProtocol.scenario
  .actions(schedule(Inputs.scheduleToStart := expires), workerStop, scheduleToStart)

/**
 * The schedule command sets the start-to-close deadline; the handler accepts asynchronously and
 * never completes; the deadline fires.
 */
val startToCloseExpires = nexusProtocol.scenario
  .actions(schedule(Inputs.startToClose := expires), handlerReply(Reply.async), startToClose)

/**
 * A retryable reply backs the operation off; the handler's worker then stops, so the retried
 * attempt is never answered and the schedule-to-start deadline fires. The start is stated: a
 * default would take the worker's from worker/Model.scala.
 */
val repliedThenStopped = nexusCaller.scenario
  .starts(NexusCallerState(unscheduled, pollingWorker))
  .actions(
    nexusCaller.own(_.operation, schedule(Inputs.scheduleToStart := expires)),
    nexusCaller.synced(_.operation -> handlerReply(Reply.handlerError(true))),
    nexusCaller.synced(_.operation -> workerStop),
    nexusCaller.own(_.operation, scheduleToStart)
  )

/** The control inspects the operation around a failed completion. */
val inspectedFailure = Control.forgedCompletion.scenario.actions(
  schedule(),
  inspect,
  handlerReply(Reply.async),
  inspect,
  complete(Resolution.failed)
)

// Nine actions are enabled before the operation is scheduled and eleven once it is, so an exact
// sequence of two is found among ninety-nine candidates, one of three among about a thousand and one
// of four among about ten thousand.
val two = Limits(steps = 2, actions = 2, search = 512)
val three = Limits(steps = 3, actions = 3, search = 4096)
val four = Limits(steps = 4, actions = 4, search = 32768)
val control = Limits(steps = 8, actions = 8, search = 262144)

// ### The Queries
//
// The design's seven: sync success, async reply then succeeded callback, async reply then failed
// callback, non-retryable handler error, retryable handler error then sync success after one
// backoff, schedule-to-start timeout with the handler's worker stopped, start-to-close timeout after
// an asynchronous reply. Each finds its same-step claim on its path and is realized as a Case.
// The product claim is verified over every trace of one path, outside `functionalQueries`, because
// a verify Query realizes nothing.

val syncCompletion = (query find syncSucceeds in syncReplied limits two total 384)
  .expect(satisfied)
  .explore(
    Exploration(
      "nexusDeadlines",
      Vector(
        Variation(
          0,
          Vector(
            Alternative("startDeadline", 30, Vector(schedule(Inputs.startToClose := expires))),
            Alternative(
              "scheduleDeadline",
              20,
              Vector(schedule(Inputs.scheduleToStart := expires))
            ),
            Alternative("unbounded", 10, Vector(schedule()))
          )
        )
      ),
      runs = 1,
      edits = 1,
      dropPrefix = true
    )
  )
val asyncCompletion =
  (query find completionSucceeds in asyncThenSucceeded limits three total 576)
    .expect(inconclusive(explanationsDisagree))
val asyncFailure = (query find completionFails in asyncThenFailed limits three total 576)
  .expect(inconclusive(explanationsDisagree))
val handlerError = (query find handlerErrorFails in nonRetryableError limits two total 384)
  .expect(inconclusive(neverEvaluated))
val retry = (query find retrySucceeds in retriedThenSucceeded limits four total 768)
  .expect(inconclusive(explanationsDisagree))
val scheduleToStartTimeout =
  (query find scheduleToStartFires in scheduleToStartExpires limits three total 576)
    .expect(inconclusive(neverEvaluated))
val startToCloseTimeout =
  (query find startToCloseFires in startToCloseExpires limits three total 576)
    .expect(inconclusive(neverEvaluated))

/** A product claim on a protocol path, read through the refinement the protocol machine declares. */
val terminalHolds = query verify terminalIsFinal in asyncThenSucceeded limits three total 576

/** The functional Queries in declaration order. */
val functionalQueries: Vector[Query] = Vector(
  syncCompletion,
  asyncCompletion,
  asyncFailure,
  handlerError,
  retry,
  scheduleToStartTimeout,
  startToCloseTimeout
)

/** The cross-entity claim, verified over that path. */
val stoppedWorkerRepliesNothing =
  query verify repliedByPollingWorker in repliedThenStopped limits four total 1536

/** The forged control, which every modeled execution that explains the evidence refutes. */
val forgedCompletion = (query find forgedSuccess in inspectedFailure limits control total 960)
  .expect(
    RunExpectation(
      Conformance.inconclusive,
      Outcome.violated,
      "every modeled execution that explains the evidence violates it",
      contract = Outcome.violated
    )
  )
  .explore(
    Exploration(
      "nexusControl",
      Vector(
        Variation(
          1,
          Vector(
            Alternative("twice", 20, Vector(inspect, inspect)),
            Alternative("once", 10, Vector(inspect)),
            Alternative("none", 0, Vector.empty)
          )
        )
      ),
      runs = 1,
      edits = 8,
      dropPrefix = true
    )
  )
