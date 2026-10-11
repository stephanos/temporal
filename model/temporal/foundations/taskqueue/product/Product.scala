// The task queue's Product: its opaque contract, what a feature may rely on of the durable queue
// (fn-126 decision 16). The level's own file holds TaskQueueProduct and its storage-loss variant,
// TaskQueueProductUnderStorageLoss; system/System.scala refines it. The two package clauses read the
// queue's package as well as this one, so its types and signature are in scope.
package temporal
package foundations.taskqueue
package product

import framework.*
import framework.Finite.given

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
// ### The opaque provider: the interface, and the interface under the storage-loss assumption

// The opaque provider.
object TaskQueueProduct
    extends Machine[QueueView, QueueOutcome, QueueFact],
      Phased[QueueView, Outstanding](_.outstanding),
      Queueing:
  val init = QueueView(Outstanding.empty)
  override def end(q: State) = q.outstanding == Outstanding.empty

  object states:
    // A message is outstanding: committed, and not yet acknowledged.
    def holding(s: State) =
      s.outstanding.in(Outstanding.committed, Outstanding.deliveredOnce, Outstanding.deliveredTwice)

  object monitors:
    // A check over the opaque queue rests on the interface alone.
    val queueOpaque = assume("taskQueueProduct.opaque")

  object rules extends Facets(queueing.rules)
