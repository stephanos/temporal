package temporal
package foundations.taskqueue
package protocol

import framework.*
import framework.Finite.given
import Bounds.seven
import product.TaskQueueProductUnderStorageLoss

// The detailed provider with the storage-loss fault, which only its assumption allows. It refines
// the interface that allows the loss.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object QueueLossyMatching
    extends Derived(
      TaskQueueProtocol
        .extend(on(fault.storageLoss) {
          where(
            TaskQueueProtocol.states.held
          ) ~> TaskQueueProtocol.queueing.effects.storageLossDetail
        })
        .refining(TaskQueueProductUnderStorageLoss)(
          TaskQueueProtocol.refinement.toProduct
        )
        .assuming(storageLossAssumed)
    ),
      FailureModel:

  // The crash cuts of every provider, and the storage loss.
  object queries:
    val lossyMatchingQueueQueries =
      TaskQueueProtocol.queries.providerQueries(QueueLossyMatching)
    val storageLossQuery =
      witness("lossyMatchingQueue.storageLoss")(
        queue.enqueue,
        queue.addActivityTask,
        queue.persistTask,
        fault.storageLoss
      ).records(QueueFact.storageLost)
        .ends(_.custody == Custody.nowhere)
        .limits(seven)
