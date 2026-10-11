// The Queueing facet of TaskQueueProtocol (fn-161.4): its actions' rules, the effects they use and its
// capabilities.
package temporal
package foundations.taskqueue
package protocol
import scala.annotation.unused
import framework.*
import framework.Finite.given
import QueueOutcome.given

val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]

trait Queueing extends Facet[QueueDetail, QueueOutcome, QueueFact], Phased[QueueDetail, Custody]:
  object queueing:
    object effects:
      def enqueueDetail(s: State) =
        choose(
          enqueueCommits -> List(
            Step(
              QueueOutcome.committed,
              QueueDetail(custody = Custody.history, polled = false, delivered = Delivered.never),
              List(QueueFact.enqueueCommitted)
            )
          ),
          enqueueFails -> List(Step(QueueOutcome.failed, s, List(QueueFact.enqueueFailed)))
            .because("the durable write fails and no message is outstanding")
        )

      // An invocation implies no receiver effect: nothing durable changes until matching persists.
      def invokeDetail(s: State) =
        enter(s.copy(custody = Custody.invoked), QueueFact.addInvoked)

      def persistDetail(s: State) =
        enter(s.copy(custody = Custody.persisted), QueueFact.taskPersisted)

      def reserveDetail(s: State) =
        enter(s.copy(custody = Custody.reserved), QueueFact.matchReserved)

      def deliverDetail(s: State) =
        List(
          Step(
            QueueOutcome.delivered,
            s.copy(polled = true, delivered = TaskQueueProtocol.states.oneMoreDelivery(s.delivered)),
            List(QueueFact.delivered)
          )
        )

      // Matching completes the task, which discharges every custodian's obligation.
      def acknowledgeDetail(@unused s: State) =
        List(Step(QueueOutcome.acknowledged, TaskQueueProtocol.states.idleQueue, List(QueueFact.acknowledged)))

      // The answer to the poller is lost. A persisted task stays queued; a sync match fails back to the
      // invocation, which history retries.
      def ackLossDetail(s: State) =
        if s.custody == Custody.reserved then // scalafix:ok DisableSyntax.==; typed equality
          enter(s.copy(custody = Custody.invoked, polled = false), QueueFact.ackLost)
        else enter(s.copy(polled = false), QueueFact.ackLost)

      // An ordinary crash loses what is only in memory, the poll, the invocation and a sync match, and
      // nothing durable: history still holds its dispatch task and retries, and a persisted task is
      // still queued.
      def crashDetail(s: State) = s.custody match
        case Custody.invoked | Custody.reserved =>
          enter(s.copy(custody = Custody.history, polled = false), QueueFact.crashed)
        case Custody.nowhere | Custody.history | Custody.persisted =>
          enter(s.copy(polled = false), QueueFact.crashed)

      // Bound by LossyMatchingQueue alone.
      def storageLossDetail(@unused s: State) =
        List(Step(QueueOutcome.lost, TaskQueueProtocol.states.idleQueue, List(QueueFact.storageLost)))

      // Bound by ForgetfulQueue alone. History drops its dispatch task when it invokes
      // AddActivityTask, before matching persists anything, so a crash there leaves no custodian for a
      // message the interface still calls committed.
      def forgetfulCrash(s: State) = s.custody match
        case Custody.invoked | Custody.reserved => enter(TaskQueueProtocol.states.idleQueue, QueueFact.crashed)
        case Custody.nowhere | Custody.history | Custody.persisted =>
          enter(s.copy(polled = false), QueueFact.crashed)

      // Bound by VolatileQueue alone. A crash wipes the tasks matching persisted, which history no
      // longer backs.
      def volatileCrash(s: State) = s.custody match
        case Custody.persisted                  => enter(TaskQueueProtocol.states.idleQueue, QueueFact.crashed)
        case Custody.invoked | Custody.reserved =>
          enter(s.copy(custody = Custody.history, polled = false), QueueFact.crashed)
        case Custody.nowhere | Custody.history => enter(s.copy(polled = false), QueueFact.crashed)

    // Each step of the route takes the message from the custodian before it; a poll takes a task
    // matching holds, and the consumer's answer completes it. A crash may fall anywhere.
    object rules extends Rules:
      on(queue.enqueue) {
        when(Custody.nowhere) ~> effects.enqueueDetail
      }
      on(queue.addActivityTask) {
        when(Custody.history) ~> effects.invokeDetail
      }
      on(queue.persistTask) {
        when(Custody.invoked) ~> effects.persistDetail
      }
      on(queue.syncMatch) {
        when(Custody.invoked) ~> effects.reserveDetail
      }
      on(queue.deliver) {
        where(TaskQueueProtocol.states.deliverable) ~> effects.deliverDetail
      }
      on(queue.acknowledge) {
        where(TaskQueueProtocol.states.answerable) ~> effects.acknowledgeDetail
      }
      on(fault.ackLoss) {
        where(_.polled) ~> effects.ackLossDetail
      }
      on(fault.crash) {
        always ~> effects.crashDetail
      }
