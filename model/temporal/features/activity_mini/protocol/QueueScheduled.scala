package temporal
package features.activity
package mini
package protocol

import framework.*
import framework.Finite.given
import foundations.taskqueue.{QueueView, Outstanding, queue as foundationQueue}
import foundations.taskqueue.product.TaskQueueProduct

val _ = () => summon[CanEqual[Outstanding, Outstanding]]

object QueueScheduled extends Derived(TaskQueueProduct
    .restrict(foundationQueue.deliver, foundationQueue.acknowledge)
    .extend(on(taskQueue.schedule) {
      where((s: QueueView) => s.outstanding == Outstanding.empty) ~> ActivityProtocol.dispatching.effects.scheduleQueue
    }))
