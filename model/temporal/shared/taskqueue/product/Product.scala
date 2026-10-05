/* The task queue's Product: its opaque contract, what a feature may rely on of the durable queue
 * (fn-126 decision 16). The level's own file holds DispatchQueue and its storage-loss variant,
 * DispatchQueueUnderStorageLoss; system/System.scala refines it. The two package clauses read the
 * queue's package as well as this one, so its types, signature and family are in scope.
 */
package temporal
package shared.taskqueue
package product

import scala.annotation.unused
import umpire.*

// ### The opaque provider: the interface, and the interface under the storage-loss assumption

/** The opaque provider. */
object DispatchQueue extends Machine[QueueView, QueueOutcome, QueueFact]:
  // First written in the system contract, as the file's declarations were: its assumption keeps the
  // Definition ID it had there.
  given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

  val entity = taskQueueEntity
  val init = QueueView(Outstanding.empty)
  def end(q: State) = q.outstanding == Outstanding.empty

  object states extends Section:
    /** A message is outstanding: committed, and not yet acknowledged. */
    def holding(s: State) =
      s.outstanding.in(Outstanding.committed, Outstanding.deliveredOnce, Outstanding.deliveredTwice)

  object effects extends Section:
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

    /** Bound by DispatchQueueUnderStorageLoss alone. */
    def storageLossView(@unused s: State) =
      List(Step(QueueOutcome.lost, QueueView(Outstanding.empty), List(QueueFact.storageLost)))

  object monitors extends Section:
    /** A check over the opaque queue rests on the interface alone. */
    val queueOpaque = assume("dispatchQueue.opaque")

  // An empty queue takes an enqueue; a committed message is delivered up to twice; a delivered one
  // is acknowledged.
  object rules extends Rules(_.outstanding):
    import Outstanding.*

    in(empty)(queue.enqueue ~> effects.enqueueView)
    in(committed, deliveredOnce)(queue.deliver ~> effects.deliverView)
    in(deliveredOnce, deliveredTwice)(queue.acknowledge ~> effects.acknowledgeView)

/** The interface under the storage-loss assumption: a committed message may also vanish. */
object DispatchQueueUnderStorageLoss
    extends Derived(
      DispatchQueue
        .extend(when(DispatchQueue.states.holding) {
          faults.storageLoss ~> DispatchQueue.effects.storageLossView
        })
        .assuming(storageLossAssumed)
    ),
      FailureModel
