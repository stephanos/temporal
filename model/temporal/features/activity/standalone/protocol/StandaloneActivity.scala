package temporal
package features.activity
package standalone
package protocol
import framework.*
import framework.Finite.given
import actors.worker.{worker as process, Phase as WorkerPhase, State as WorkerState}
import Timeout.expires
import Bounds.six

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object StandaloneActivity
    extends Composition[StandaloneActivityState](
      _.activity -> ActivityProtocol,
      _.worker -> ActivityWorker
    ),
      Phased[StandaloneActivityState, Phase](_.activity.phase):

  object syncs extends Syncs:
    sync(_.activity -> process.stop, _.worker -> process.stop)
    sync(_.activity -> activityWorker.poll, _.worker -> process.serve)

  object properties:
    // The cross-entity claim: no stopped worker starts an attempt.
    val startedByPollingWorker = property
      .whenAction(synced(_.activity -> activityWorker.poll)) // format: off
      .holds(_.state.worker.phase == WorkerPhase.polling) // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
      // format: on
  object queries:
    // The cross-entity claim, over the path on which a stopped worker never takes the retry: the
    // first attempt fails retryably and backs off, then the worker stops, so the retry is never
    // dispatched; the attempt start makes the verification exercise the claim. The start is stated:
    // a default would take the worker's from actors/worker/Worker.scala; fn-115's golden compares
    // positions.
    val stoppedBeforeRetry = scenario
      .starts(StandaloneActivityState(ActivityProtocol.init, WorkerState(WorkerPhase.polling)))
      .actions(
        own(_.activity, client.start(scheduleToStart := expires)),
        synced(_.activity -> activityWorker.poll),
        own(_.activity, activityWorker.respondFailed(Failure.retryable)),
        own(_.activity, timers.backoff),
        synced(_.activity -> process.stop),
        own(_.activity, deadline.scheduleToStart)
      )
    val stoppedWorkerStartsNothing =
      query verify properties.startedByPollingWorker in stoppedBeforeRetry limits six
