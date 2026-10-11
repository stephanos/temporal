package temporal
package foundations.taskqueue
package product
import framework.*
import framework.Finite.given

// The interface under the storage-loss assumption: a committed message may also vanish.
val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

object TaskQueueProductUnderStorageLoss
    extends Derived(
      TaskQueueProduct
        .extend(on(fault.storageLoss) {
          where(TaskQueueProduct.states.holding) ~> TaskQueueProduct.queueing.effects.storageLossView
        })
        .assuming(storageLossAssumed)
    ),
      FailureModel
