package temporal
package features.activity
package standalone
package protocol

import framework.*
import framework.Finite.given
import Bounds.three
import actors.worker.{worker as process, Phase as WorkerPhase}

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object ServedDispatch
    extends Composition[StandaloneActivityState](
      _.activity -> DispatchingActivity,
      _.worker -> ServingWorker
    ),
      Phased[StandaloneActivityState, Phase](_.activity.phase):

  object states:
    // The source reads both members: an attempt the worker may take now, while the worker polls.
    def dispatchable(s: State): Boolean =
      s.activity.phase == Phase.scheduled && s.activity.dispatch == Dispatch.now && s.worker.phase == WorkerPhase.polling

    def started(s: State): Boolean = s.activity.phase == Phase.started

  object syncs extends Syncs:
    sync(_.activity -> activityWorker.poll, _.worker -> process.serve)

  object properties:
    // The cross-entity safety promise: only a polling worker starts an attempt.
    def startedByPollingWorker(c: Composition[State]) = c
      .property("startedByPollingWorker")
      .whenAction(c.synced(_.activity -> activityWorker.poll))
      .holds(_.state.worker.phase == WorkerPhase.polling)

    object liveness:
      // The bounded progress promise, over both members.
      val startsWithinOneStep = ServedDispatch.leadsTo("startsWithinOneStep")(
        states.dispatchable,
        states.started,
        within = 1,
        servedDispatchRuns
      )

  object queries:
    val anyDispatch = scenario.free
    val onlyPollingWorkersStart =
      query verify properties.startedByPollingWorker(ServedDispatch) in anyDispatch limits three

// ### The missing prerequisite: a worker that may stop, and never resumes. The same claim, under the
// same kind of fairness, is refuted by a deadlock: a stopped worker leaves the sync disabled.
