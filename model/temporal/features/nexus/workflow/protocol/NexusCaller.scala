package temporal
package features.nexus
package workflow
package protocol
import framework.*
import framework.Finite.given
import temporal.Bounds.four
import temporal.actors.worker.{worker, Phase as WorkerPhase, State as WorkerState}
import Timeout.expires

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object NexusCaller
    extends Composition[NexusCallerState](
      _.operation -> NexusProtocol,
      _.worker -> HandlerWorker
    ),
      Phased[NexusCallerState, Phase](_.operation.phase):

  object syncs extends Syncs:
    sync(_.operation -> worker.stop, _.worker -> worker.stop)
    sync(_.operation -> handler.reply, _.worker -> worker.serve)
  // format: off
  object properties:
    // The cross-entity claim: every reply, of any class, leaves the handler's worker polling, so no
    // handler replies while its worker is stopped.
    val repliedByPollingWorker = property
      .whenAction(synced(_.operation -> handler.reply))
      .holds(_.state.worker.phase == WorkerPhase.polling) // scalafix:ok DisableSyntax.==; typed equality
  // format: on
  object queries:
    // The cross-entity claim, verified over the path on which a retryable reply backs the operation
    // off; the handler's worker then stops, so the retried attempt is never answered and the
    // schedule-to-start deadline fires. The start is stated: a default would take the worker's from
    // actors/worker/Worker.scala.
    val repliedThenStopped = scenario
      .starts(NexusCallerState(NexusProtocol.init, WorkerState(WorkerPhase.polling)))
      .actions(
        own(_.operation, caller.schedule(scheduleToStart := expires)),
        synced(_.operation -> handler.reply(Reply.handlerError(true))),
        synced(_.operation -> worker.stop),
        own(_.operation, deadline.scheduleToStart)
      )
    val stoppedWorkerRepliesNothing =
      query verify properties.repliedByPollingWorker in repliedThenStopped limits four
