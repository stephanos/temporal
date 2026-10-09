// The standalone activity System composed with its task queue's worker.
package temporal
package features.activity
package standalone
package system

import framework.*
import actors.worker.{worker as process, Phase as WorkerPhase, State as WorkerState}
import Timeout.expires

// The System machine and its worker, as the standalone activity composition holds them.
final case class StandaloneActivityState(activity: State, worker: WorkerState)

// ### The worker of the activity's task queue, as the activity sees it: its stop and its serving.

object ActivityWorker extends Derived(actors.worker.Polling.restrict(process.stop, process.serve))

// ### With the worker of its task queue, the stop is the worker's own phase change and every
// attempt start is the worker serving, so an attempt has a row only while the worker polls.

object StandaloneActivity
    extends Composition[StandaloneActivityState](
      _.activity -> ActivitySystem,
      _.worker -> ActivityWorker
    ),
      Phased[StandaloneActivityState, Phase](_.activity.phase):
  object syncs extends Syncs:
    sync(_.activity -> process.stop, _.worker -> process.stop)
    sync(_.activity -> worker.poll, _.worker -> process.serve)

  object properties:
    // The cross-entity claim: no stopped worker starts an attempt.
    val startedByPollingWorker = property
      .whenAction(synced(_.activity -> worker.poll))
      .holds(_.state.worker.phase == WorkerPhase.polling)

  object queries:
    // The cross-entity claim, over the path on which a stopped worker never takes the retry: the
    // first attempt fails retryably and backs off, then the worker stops, so the retry is never
    // dispatched; the attempt start makes the verification exercise the claim. The start is stated:
    // a default would take the worker's from actors/worker/Worker.scala; fn-115's golden compares
    // positions.
    val stoppedBeforeRetry = scenario
      .starts(StandaloneActivityState(ActivitySystem.init, WorkerState(WorkerPhase.polling)))
      .actions(
        own(_.activity, client.start(scheduleToStart := expires)),
        synced(_.activity -> worker.poll),
        own(_.activity, worker.respondFailed(Failure.retryable)),
        own(_.activity, timers.backoff),
        synced(_.activity -> process.stop),
        own(_.activity, deadline.scheduleToStart)
      )
    val stoppedWorkerStartsNothing =
      query verify properties.startedByPollingWorker in stoppedBeforeRetry limits six
