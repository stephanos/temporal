package temporal
package foundations.taskqueue
package product

import framework.dsl.*
import framework.dsl.Finite.given

// The interface under the storage-loss assumption: a committed message may also vanish.
val _ = () => summon[CanEqual[framework.dsl.outcomes.Outcome, framework.dsl.outcomes.Outcome]]

object TaskQueueProductUnderStorageLoss
    extends Derived(
      TaskQueueProduct
        .extend(on(fault.storageLoss) {
          where(
            TaskQueueProduct.states.holding
          ) ~> TaskQueueProduct.queueing.effects.storageLossView
        })
        .assuming(storageLossAssumed)
    ),
      FailureModel
