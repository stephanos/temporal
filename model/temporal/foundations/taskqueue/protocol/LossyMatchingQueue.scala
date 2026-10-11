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

object LossyMatchingQueue
    extends Derived(
      TaskQueueProtocol
        .extend(on(fault.storageLoss) {
          where(
            TaskQueueProtocol.states.held
          ) ~> TaskQueueProtocol.queueing.effects.storageLossDetail
        })
        .refining(TaskQueueProductUnderStorageLoss)(TaskQueueProtocol.refinement.toProduct)
        .assuming(storageLossAssumed)
    ),
      FailureModel:

  object properties:
    // Storage loss drops a committed message, and the queue records that it did. Only the lossy
    // provider binds the loss, so this is its own Property, not a claim of every provider.
    val storageLossDrops =
      property when fault.storageLoss holds { after =>
        after.state.custody == Custody.nowhere && after.records(QueueFact.storageLost)
      }

  // The crash cuts of every provider, and the storage loss.
  object queries:
    val persistedThenLost = scenario.actions(
      queue.enqueue,
      queue.addActivityTask,
      queue.persistTask,
      fault.storageLoss
    )
    val lossyMatchingQueueQueries =
      TaskQueueProtocol.queries.providerQueries(LossyMatchingQueue)
    val storageLossQuery =
      query("lossyMatchingQueue.storageLoss") find properties.storageLossDrops in
        persistedThenLost limits seven
