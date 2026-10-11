// The Queueing facet of TaskQueueProduct (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package foundations.taskqueue
package product
import scala.annotation.unused
import framework.*
import framework.Finite.given

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Queueing extends Facet[QueueView, QueueOutcome, QueueFact], Phased[QueueView, Outstanding]:
  object queueing:
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

      def deliverView(s: State) = // format: off
        if s.outstanding == Outstanding.committed then // scalafix:ok DisableSyntax.==; strictEquality checks native comparisons
          List( // format: on
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

    // An empty queue takes an enqueue; a committed message is delivered up to twice; a delivered one
    // is acknowledged.
    object rules extends Rules:
      on(queue.enqueue) {
        when(Outstanding.empty) ~> effects.enqueueView
      }
      on(queue.deliver) {
        when(Outstanding.committed, Outstanding.deliveredOnce) ~> effects.deliverView
      }
      on(queue.acknowledge) {
        when(Outstanding.deliveredOnce, Outstanding.deliveredTwice) ~> effects.acknowledgeView
      }
