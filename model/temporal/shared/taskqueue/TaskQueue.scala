/* The task queue: the durable queue between history's dispatch and a worker's poll, as a reusable
 * entity of its own, beside shared/worker. Its opaque contract is what a feature may rely on; the
 * matching provider is the route one message takes through history and matching, which refines it;
 * and the forgetful and volatile providers are deliberately violating controls. Grounded in
 * chasm/lib/activity/tasks.go (the dispatch task) and service/matching (AddActivityTask, sync match,
 * the persisted task queue and its completion).
 *
 * A feature composes a provider with its own machine and synchronizes its actions with enqueue,
 * deliver and acknowledge; the standalone activity's system contract does. The queue names nothing
 * of a feature.
 *
 * The abstraction is bounded: one message at a time, delivered at most twice before its
 * acknowledgment. It is not a general multi-message queue, and a check of it says nothing of two
 * messages in flight at once.
 *
 * Read top to bottom: the types; the signature (the entity, the interface's actions, the faults,
 * the storage-loss assumption and the bounds); then one object per machine, each before the
 * machines derived from it -- DispatchQueue, the opaque contract, and DispatchQueueUnderStorageLoss,
 * its storage-loss variant; MatchingQueue, the detailed provider that refines it, and the providers
 * derived from it, LossyMatchingQueue, ForgetfulQueue and VolatileQueue. A machine object reads its
 * header (entity, init, end), then its sections in order: states, refinement, effects, monitors,
 * rules, properties and queries.
 */
package temporal
package shared.taskqueue

import scala.annotation.unused
import umpire.*

// The queue was first written in the standalone activity's system contract. Its declarations keep the
// family, the Definition IDs and the type names they had there, so its tables, IDs and answers are
// those that contract was checked with.
given Family = Family("temporal.activity.standalone.system")
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

// ### Types

/** The one message the queue may hold, by how far its delivery got. */
enum Outstanding derives Finite:
  case empty, committed, deliveredOnce, deliveredTwice

final case class QueueView(outstanding: Outstanding) derives Finite

/** `internal` answers every step of a provider that its interface does not show. */
enum QueueOutcome derives Finite:
  case committed, failed, delivered, acknowledged, lost, internal

object QueueOutcome:
  // A provider's step behind the interface is the one that answers `internal`. Kept in the
  // companion, where only a step known to answer a QueueOutcome finds it, so a `choose` of a feature's
  // own steps still finds its own `Ok` alone.
  given Ok[QueueOutcome] = Ok(QueueOutcome.internal)

/** The interface's events, and what a provider records of the steps behind them. */
enum QueueFact derives Finite:
  case enqueueCommitted, enqueueFailed, delivered, acknowledged, storageLost
  case addInvoked, taskPersisted, matchReserved, crashed, ackLost

type QueueViewStep = Step[QueueView, QueueOutcome, QueueFact]

/**
 * Who holds the message. `history` is the dispatch task alone; `invoked` an AddActivityTask in
 * flight and `reserved` a sync match, both only in memory; `persisted` a task in matching's durable
 * queue, which is what lets history's own task finish.
 */
enum Custody derives Finite:
  case nowhere, history, invoked, reserved, persisted

enum Delivered derives Finite:
  case never, once, twice

/** `polled` is a poller holding the task while the consumer decides. */
final case class QueueDetail(custody: Custody, polled: Boolean, delivered: Delivered) derives Finite

type QueueDetailStep = Step[QueueDetail, QueueOutcome, QueueFact]

/** The laws every provider of the detailed queue is held to, declared once by `queueLaws`. */
final case class QueueLaws(delivers: Property[QueueDetail], committedStays: Property[QueueDetail])

// ### Signature

/**
 * Named by the task queue's name, as the worker that polls it is. Messages and their deliveries are
 * the queue's state, not entities of their own.
 */
val taskQueueEntity = Entity("taskQueue", key = "taskQueue")

/** The party of the faults the queue's providers suffer. */
val fault = Party()

val enqueueCommits = choice
val enqueueFails = choice

/**
 * The queue's own steps. A section is transparent, so each keeps the Definition ID the file's pin
 * gives it.
 *
 * The interface: what a feature may rely on of the durable queue. An enqueue commits or fails; a
 * committed message is delivered up to twice before its acknowledgment; and no committed message
 * is lost. A crash shows at this interface only as that second delivery.
 *
 * The detailed queue: history's dispatch task and matching's custody. The route of one message:
 * history durably schedules the dispatch task; its Execute invokes AddActivityTask; matching either
 * persists the task or reserves it for a waiting poller; a poll hands it out; and once the consumer
 * has answered, matching completes it. Each step is its own transition, so a crash can fall between
 * any two.
 */
object queue extends Section:
  val enqueue = internal on taskQueueEntity
  val deliver = internal on taskQueueEntity
  val acknowledge = internal on taskQueueEntity

  val addActivityTask = internal on taskQueueEntity
  val persistTask = internal on taskQueueEntity
  val syncMatch = internal on taskQueueEntity

/** The faults the providers suffer. Like the worker's stop and resume, they name no entity. */
object faults extends Section:
  /**
   * Committed storage may be lost. It is a fault of its own, apart from a crash, and only a machine
   * that assumes it has the step.
   */
  val storageLoss = action(fault)

  val crash = action(fault)
  val ackLoss = action(fault)

/**
 * Committed storage may be lost. No machine makes the assumption of its own: a derivation that binds
 * the storage-loss step adds it. The file's pin is the one the system contract had, so it keeps its
 * Definition ID.
 */
val storageLossAssumed = assume("storageLoss")

val seven = Limits(steps = 7, actions = 7, search = 262144)

/** Past the depth of the detailed queue's table and of a design composed with it, which is ten. */
val twelve = Limits(steps = 12, actions = 12, search = 262144)

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
