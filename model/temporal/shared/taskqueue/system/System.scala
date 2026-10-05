/* The task queue's System: its provider, the route one message takes through history and matching
 * (fn-126 decision 16). The level's own file holds MatchingQueue, the detailed provider whose
 * refinement says what the opaque contract (product/Product.scala) reads of it, and the providers
 * derived from it, LossyMatchingQueue, ForgetfulQueue and VolatileQueue.
 */
package temporal
package shared.taskqueue
package system

import scala.annotation.unused
import umpire.*
import product.{DispatchQueue, DispatchQueueUnderStorageLoss}

// ### The detailed provider, and the providers derived from it. The violating providers each differ
// from the detailed provider in what one ordinary crash does, and neither assumes storage loss. They
// are negative controls a feature may compose its own design with: the replacement is what must fail.

/** The detailed provider, under ordinary crashes and lost answers to the poller. */
object MatchingQueue extends Machine[QueueDetail, QueueOutcome, QueueFact], FailureModel:
  val entity = taskQueueEntity
  val init = states.idleQueue
  def end(d: State) = states.queueEnds(d)

  object states extends Section:
    val idleQueue = QueueDetail(Custody.nowhere, false, Delivered.never)

    def queueEnds(d: State) = d.custody == Custody.nowhere

    def oneMoreDelivery(d: Delivered) = d match
      case Delivered.never => Delivered.once
      case _               => Delivered.twice

    /** Matching holds a task a poll can take: reserved for a waiting poller, or persisted. */
    def matchable(c: Custody) = c.in(Custody.reserved, Custody.persisted)

    /** A custodian holds the message. */
    def held(s: State) = s.custody != Custody.nowhere

    /** Matching holds the task, no poller holds it, and it was not yet handed out twice. */
    def deliverable(s: State) =
      matchable(s.custody) && !s.polled && s.delivered != Delivered.twice

    /** A poller holds a task matching handed out, whose consumer may now answer. */
    def answerable(s: State) =
      s.polled && s.custody != Custody.nowhere && s.delivered != Delivered.never

  object refinement extends Refinement(DispatchQueue):
    /**
     * The interface state a detailed state stands for: a message is outstanding while anyone holds
     * it.
     */
    def toProduct(d: State) =
      if d.custody == Custody.nowhere then QueueView(Outstanding.empty)
      else
        d.delivered match
          case Delivered.never => QueueView(Outstanding.committed)
          case Delivered.once  => QueueView(Outstanding.deliveredOnce)
          case Delivered.twice => QueueView(Outstanding.deliveredTwice)

    def visible(f: QueueFact) = f match
      case QueueFact.enqueueCommitted | QueueFact.enqueueFailed | QueueFact.delivered |
          QueueFact.acknowledged | QueueFact.storageLost =>
        true
      case QueueFact.addInvoked | QueueFact.taskPersisted | QueueFact.matchReserved |
          QueueFact.crashed | QueueFact.ackLost =>
        false

    def visibleOutcomes(o: QueueOutcome) = o != QueueOutcome.internal

  object effects extends Section:
    def enqueueDetail(s: State) =
      choose(
        enqueueCommits -> List(
          Step(
            QueueOutcome.committed,
            QueueDetail(Custody.history, false, Delivered.never),
            List(QueueFact.enqueueCommitted)
          )
        ),
        enqueueFails -> List(Step(QueueOutcome.failed, s, List(QueueFact.enqueueFailed)))
          .because("the durable write fails and no message is outstanding")
      )

    /** An invocation implies no receiver effect: nothing durable changes until matching persists. */
    def invokeDetail(s: State): List[QueueDetailStep] =
      enter(s.copy(custody = Custody.invoked), QueueFact.addInvoked)

    def persistDetail(s: State): List[QueueDetailStep] =
      enter(s.copy(custody = Custody.persisted), QueueFact.taskPersisted)

    def reserveDetail(s: State): List[QueueDetailStep] =
      enter(s.copy(custody = Custody.reserved), QueueFact.matchReserved)

    def deliverDetail(s: State) =
      List(
        Step(
          QueueOutcome.delivered,
          s.copy(polled = true, delivered = states.oneMoreDelivery(s.delivered)),
          List(QueueFact.delivered)
        )
      )

    /** Matching completes the task, which discharges every custodian's obligation. */
    def acknowledgeDetail(@unused s: State) =
      List(Step(QueueOutcome.acknowledged, states.idleQueue, List(QueueFact.acknowledged)))

    /**
     * The answer to the poller is lost. A persisted task stays queued; a sync match fails back to the
     * invocation, which history retries.
     */
    def ackLossDetail(s: State): List[QueueDetailStep] =
      if s.custody == Custody.reserved then
        enter(s.copy(custody = Custody.invoked, polled = false), QueueFact.ackLost)
      else enter(s.copy(polled = false), QueueFact.ackLost)

    /**
     * An ordinary crash loses what is only in memory, the poll, the invocation and a sync match, and
     * nothing durable: history still holds its dispatch task and retries, and a persisted task is
     * still queued.
     */
    def crashDetail(s: State): List[QueueDetailStep] = s.custody match
      case Custody.invoked | Custody.reserved =>
        enter(s.copy(custody = Custody.history, polled = false), QueueFact.crashed)
      case Custody.nowhere | Custody.history | Custody.persisted =>
        enter(s.copy(polled = false), QueueFact.crashed)

    /** Bound by LossyMatchingQueue alone. */
    def storageLossDetail(@unused s: State) =
      List(Step(QueueOutcome.lost, states.idleQueue, List(QueueFact.storageLost)))

    /**
     * Bound by ForgetfulQueue alone. History drops its dispatch task when it invokes
     * AddActivityTask, before matching persists anything, so a crash there leaves no custodian for a
     * message the interface still calls committed.
     */
    def forgetfulCrash(s: State): List[QueueDetailStep] = s.custody match
      case Custody.invoked | Custody.reserved => enter(states.idleQueue, QueueFact.crashed)
      case Custody.nowhere | Custody.history | Custody.persisted =>
        enter(s.copy(polled = false), QueueFact.crashed)

    /**
     * Bound by VolatileQueue alone. A crash wipes the tasks matching persisted, which history no
     * longer backs.
     */
    def volatileCrash(s: State): List[QueueDetailStep] = s.custody match
      case Custody.persisted                  => enter(states.idleQueue, QueueFact.crashed)
      case Custody.invoked | Custody.reserved =>
        enter(s.copy(custody = Custody.history, polled = false), QueueFact.crashed)
      case Custody.nowhere | Custody.history => enter(s.copy(polled = false), QueueFact.crashed)

  // Each step of the route takes the message from the custodian before it; a poll takes a task
  // matching holds, and the consumer's answer completes it. A crash may fall anywhere.
  object rules extends Rules(_.custody):
    import Custody.*

    in(nowhere)(queue.enqueue ~> effects.enqueueDetail)
    in(history)(queue.addActivityTask ~> effects.invokeDetail)
    in(invoked)(queue.persistTask ~> effects.persistDetail)
    in(invoked)(queue.syncMatch ~> effects.reserveDetail)
    when(states.deliverable)(queue.deliver ~> effects.deliverDetail)
    when(states.answerable)(queue.acknowledge ~> effects.acknowledgeDetail)
    when(s => s.polled)(faults.ackLoss ~> effects.ackLossDetail)
    when(_ => true)(faults.crash ~> effects.crashDetail)

  /** What a provider promises. */
  object properties extends Section:
    /**
     * The laws of the provider `m`: a delivery hands the message out, and a message a custodian holds
     * stays held until it is acknowledged. Each takes its name explicitly, so every provider's
     * instance keeps the name its checks read.
     */
    def queueLaws(m: Machine[State, QueueOutcome, QueueFact]) = QueueLaws(
      m.property("delivers") when queue.deliver holds (after => after.records(QueueFact.delivered)),
      m.property("committedStays")
        .stays(_.custody != Custody.nowhere)
        .unless(_.records(QueueFact.acknowledged))
    )

  /** The crash cuts: one crash at each point of the route, and the delivery that must still follow it. */
  object queries extends Section:
    /**
     * One crash after the invocation, after the sync match, after persistence and after a delivery,
     * declared on the provider `m`. After the acknowledgment nothing is left to deliver. `anyTotal`
     * is the static combination count of its free `any` Query.
     */
    def providerQueries(m: Machine[State, QueueOutcome, QueueFact], anyTotal: Int) =
      val laws = properties.queueLaws(m)
      Vector(
        query(s"${m.name}.crashAfterInvocation") find laws.delivers in
          m.scenario("crashAfterInvocation")
            .actions(
              queue.enqueue,
              queue.addActivityTask,
              faults.crash,
              queue.addActivityTask,
              queue.persistTask,
              queue.deliver
            ) limits seven total 180,
        query(s"${m.name}.crashAfterSyncMatch") find laws.delivers in
          m.scenario("crashAfterSyncMatch")
            .actions(
              queue.enqueue,
              queue.addActivityTask,
              queue.syncMatch,
              faults.crash,
              queue.addActivityTask,
              queue.syncMatch,
              queue.deliver
            ) limits seven total 210,
        query(s"${m.name}.crashAfterPersistence") find laws.delivers in
          m.scenario("crashAfterPersistence")
            .actions(
              queue.enqueue,
              queue.addActivityTask,
              queue.persistTask,
              faults.crash,
              queue.deliver
            ) limits seven total 150,
        query(s"${m.name}.crashAfterDelivery") find laws.delivers in
          m.scenario("crashAfterDelivery")
            .actions(
              queue.enqueue,
              queue.addActivityTask,
              queue.persistTask,
              queue.deliver,
              faults.crash,
              queue.deliver
            ) limits seven total 180,
        query(s"${m.name}.crashAfterAcknowledgment") verify laws.committedStays in
          m.scenario("crashAfterAcknowledgment")
            .actions(
              queue.enqueue,
              queue.addActivityTask,
              queue.persistTask,
              queue.deliver,
              queue.acknowledge,
              faults.crash
            ) limits seven total 180,
        query verify laws.committedStays in m.scenario("any").free limits twelve total anyTotal
      )

    // Eight bound actions for every provider but the lossy one, which binds storage loss as a ninth.
    val matchingQueueQueries = providerQueries(MatchingQueue, anyTotal = 2880)

/**
 * The detailed provider with the storage-loss fault, which only its assumption allows. It refines
 * the interface that allows the loss.
 */
object LossyMatchingQueue
    extends Derived(
      MatchingQueue
        .extend(when(MatchingQueue.states.held) {
          faults.storageLoss ~> MatchingQueue.effects.storageLossDetail
        })
        .refining(DispatchQueueUnderStorageLoss)(MatchingQueue.refinement.toProduct)
        .assuming(storageLossAssumed)
    ),
      FailureModel:
  object properties extends Section:
    /**
     * Storage loss drops a committed message, and the queue records that it did. Only the lossy
     * provider binds the loss, so this is its own Property, not a law of every provider.
     */
    val storageLossDrops =
      property when faults.storageLoss holds { after =>
        after.state.custody == Custody.nowhere && after.records(QueueFact.storageLost)
      }

  /** The crash cuts of every provider, and the storage loss. */
  object queries extends Section:
    val lossyMatchingQueueQueries =
      MatchingQueue.queries.providerQueries(LossyMatchingQueue, anyTotal = 3240)

    val storageLossQuery =
      query("lossyMatchingQueue.storageLoss") find properties.storageLossDrops in
        scenario("persistedThenLost").actions(
          queue.enqueue,
          queue.addActivityTask,
          queue.persistTask,
          faults.storageLoss
        ) limits seven total 120

/** A crash that loses the message history dropped on invoking AddActivityTask. */
object ForgetfulQueue
    extends Derived(MatchingQueue.rebind(faults.crash ~> MatchingQueue.effects.forgetfulCrash)),
      NegativeControl:
  object queries extends Section:
    val forgetfulQueueQueries =
      MatchingQueue.queries.providerQueries(ForgetfulQueue, anyTotal = 2880)

/** A crash that wipes the tasks matching persisted. */
object VolatileQueue
    extends Derived(MatchingQueue.rebind(faults.crash ~> MatchingQueue.effects.volatileCrash)),
      NegativeControl:
  object queries extends Section:
    val volatileQueueQueries = MatchingQueue.queries.providerQueries(VolatileQueue, anyTotal = 2880)
