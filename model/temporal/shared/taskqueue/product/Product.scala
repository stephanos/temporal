// The task queue's Product: its opaque contract, what a feature may rely on of the durable queue
// (fn-126 decision 16). The level's own file holds TaskQueueProduct and its storage-loss variant,
// TaskQueueProductUnderStorageLoss; system/System.scala refines it. The two package clauses read the
// queue's package as well as this one, so its types and signature are in scope.
package temporal
package shared.taskqueue
package product

import scala.annotation.unused
import umpire.*

// ### The opaque provider: the interface, and the interface under the storage-loss assumption

// The opaque provider.
object TaskQueueProduct extends Machine[QueueView, QueueOutcome, QueueFact]:
  val init = QueueView(Outstanding.empty)
  def end(q: State) = q.outstanding == Outstanding.empty

  object states:
    // A message is outstanding: committed, and not yet acknowledged.
    def holding(s: State) =
      s.outstanding.in(Outstanding.committed, Outstanding.deliveredOnce, Outstanding.deliveredTwice)

  object effects:
    def enqueueView(s: State) =
      choose(
        enqueueCommits -> List(
          Step(
            QueueOutcome.committed,
            QueueView(Outstanding.committed),
            List(QueueFact.enqueueCommitted)
          )
        ),
        enqueueFails -> List(Step(QueueOutcome.failed, s, List(QueueFact.enqueueFailed)))
          .because("the durable write fails and no message is outstanding")
      )

    def deliverView(s: State) =
      if s.outstanding == Outstanding.committed then
        List(
          Step(
            QueueOutcome.delivered,
            QueueView(Outstanding.deliveredOnce),
            List(QueueFact.delivered)
          )
        )
      else
        List(
          Step(
            QueueOutcome.delivered,
            QueueView(Outstanding.deliveredTwice),
            List(QueueFact.delivered)
          )
        ).because("a message not yet acknowledged may be delivered again")

    def acknowledgeView(@unused s: State) =
      List(
        Step(
          QueueOutcome.acknowledged,
          QueueView(Outstanding.empty),
          List(QueueFact.acknowledged)
        )
      )

    // Bound by TaskQueueProductUnderStorageLoss alone.
    def storageLossView(@unused s: State) =
      List(Step(QueueOutcome.lost, QueueView(Outstanding.empty), List(QueueFact.storageLost)))

  object monitors:
    // A check over the opaque queue rests on the interface alone.
    val queueOpaque = assume("taskQueueProduct.opaque")

  // An empty queue takes an enqueue; a committed message is delivered up to twice; a delivered one
  // is acknowledged.
  object rules extends Rules(_.outstanding):
    import Outstanding.*

    on(queue.enqueue)(in(empty) ~> effects.enqueueView)
    on(queue.deliver)(in(committed, deliveredOnce) ~> effects.deliverView)
    on(queue.acknowledge)(in(deliveredOnce, deliveredTwice) ~> effects.acknowledgeView)

// The interface under the storage-loss assumption: a committed message may also vanish.
object TaskQueueProductUnderStorageLoss
    extends Derived(
      TaskQueueProduct
        .extend(on(fault.storageLoss) {
          where(TaskQueueProduct.states.holding) ~> TaskQueueProduct.effects.storageLossView
        })
        .assuming(storageLossAssumed)
    ),
      FailureModel
