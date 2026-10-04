/* The task queue: the durable queue between history's dispatch and a worker's poll, as a reusable
 * entity of its own, beside temporal/worker. Its opaque contract is what a feature may rely on; the
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
 */
package temporal
package taskqueue

import umpire.*

// The queue was first written in the standalone activity's system contract. Its declarations keep the
// family, the Definition IDs and the type names they had there, so its tables, IDs and answers are
// those that contract was checked with.
given Family = Family("temporal.activity.standalone.system")
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

// ### The entity

/**
 * Named by the task queue's name, as the worker that polls it is. Messages and their deliveries are
 * the queue's state, not entities of their own.
 */
val taskQueueEntity = Entity("taskQueue", key = "taskQueue")

/** The party of the faults the queue's providers suffer. */
val fault = Party()

// ### The interface
//
// What a feature may rely on of the durable queue: an enqueue commits or fails; a committed message
// is delivered up to twice before its acknowledgment; and no committed message is lost. A crash shows
// at this interface only as that second delivery.

val enqueue = internal on taskQueueEntity
val deliver = internal on taskQueueEntity
val acknowledge = internal on taskQueueEntity

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
  // own steps still finds its own `Accepted` alone.
  given Accepted[QueueOutcome] = Accepted(QueueOutcome.internal)

/** The interface's events, and what a provider records of the steps behind them. */
enum QueueFact derives Finite:
  case enqueueCommitted, enqueueFailed, delivered, acknowledged, storageLost
  case addInvoked, taskPersisted, matchReserved, crashed, ackLost

type QueueViewStep = Step[QueueView, QueueOutcome, QueueFact]

val enqueueCommits = choice
val enqueueFails = choice

def enqueueView(q: QueueView) =
  if q.outstanding != Outstanding.empty then disabled
  else
    choose(
      enqueueCommits -> List(
        Step(
          QueueOutcome.committed,
          QueueView(Outstanding.committed),
          List(QueueFact.enqueueCommitted)
        )
      ),
      enqueueFails -> List(Step(QueueOutcome.failed, q, List(QueueFact.enqueueFailed)))
        .because("the durable write fails and no message is outstanding")
    )

def deliverView(q: QueueView) = q.outstanding match
  case Outstanding.committed =>
    List(
      Step(QueueOutcome.delivered, QueueView(Outstanding.deliveredOnce), List(QueueFact.delivered))
    )
  case Outstanding.deliveredOnce =>
    List(
      Step(QueueOutcome.delivered, QueueView(Outstanding.deliveredTwice), List(QueueFact.delivered))
    ).because("a message not yet acknowledged may be delivered again")
  case Outstanding.empty | Outstanding.deliveredTwice => disabled

def acknowledgeView(q: QueueView) = q.outstanding match
  case Outstanding.deliveredOnce | Outstanding.deliveredTwice =>
    List(
      Step(QueueOutcome.acknowledged, QueueView(Outstanding.empty), List(QueueFact.acknowledged))
    )
  case Outstanding.empty | Outstanding.committed => disabled

def storageLossView(q: QueueView) =
  if q.outstanding == Outstanding.empty then disabled
  else List(Step(QueueOutcome.lost, QueueView(Outstanding.empty), List(QueueFact.storageLost)))

/** A check over the opaque queue rests on the interface alone. */
val queueOpaque = assume("dispatchQueue.opaque")

/**
 * Committed storage may be lost. It is a fault of its own, apart from a crash, and only a machine
 * that assumes it has the step.
 */
val storageLossAssumed = assume("storageLoss")
val storageLoss = action(fault)

val emptyQueue = QueueView(Outstanding.empty)

/** The opaque provider. */
val dispatchQueue = machine[QueueView, QueueOutcome, QueueFact] {
  forEntity(taskQueueEntity)
  assumes(queueOpaque)
  starts(emptyQueue)
  ends(q => q.outstanding == Outstanding.empty)
  steps(enqueue ~> enqueueView, deliver ~> deliverView, acknowledge ~> acknowledgeView)
}

/** The interface under the storage-loss assumption: a committed message may also vanish. */
val dispatchQueueUnderStorageLoss =
  dispatchQueue.extend(storageLoss ~> storageLossView).assuming(storageLossAssumed)

// ### The detailed queue: history's dispatch task and matching's custody
//
// The route of one message: history durably schedules the dispatch task; its Execute invokes
// AddActivityTask; matching either persists the task or reserves it for a waiting poller; a poll
// hands it out; and once the consumer has answered, matching completes it. Each step is its own
// transition, so a crash can fall between any two.

val addActivityTask = internal on taskQueueEntity
val persistTask = internal on taskQueueEntity
val syncMatch = internal on taskQueueEntity

// The faults name no entity, as the worker's stop and resume name none.
val crash = action(fault)
val ackLoss = action(fault)

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

val idleQueue = QueueDetail(Custody.nowhere, false, Delivered.never)

/** The interface state a detailed state stands for: a message is outstanding while anyone holds it. */
def viewOf(d: QueueDetail) =
  if d.custody == Custody.nowhere then QueueView(Outstanding.empty)
  else
    d.delivered match
      case Delivered.never => QueueView(Outstanding.committed)
      case Delivered.once  => QueueView(Outstanding.deliveredOnce)
      case Delivered.twice => QueueView(Outstanding.deliveredTwice)

def enqueueDetail(d: QueueDetail) =
  if d.custody != Custody.nowhere then disabled
  else
    choose(
      enqueueCommits -> List(
        Step(
          QueueOutcome.committed,
          QueueDetail(Custody.history, false, Delivered.never),
          List(QueueFact.enqueueCommitted)
        )
      ),
      enqueueFails -> List(Step(QueueOutcome.failed, d, List(QueueFact.enqueueFailed)))
        .because("the durable write fails and no message is outstanding")
    )

/** An invocation implies no receiver effect: nothing durable changes until matching persists. */
def invokeDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.history then disabled
  else accept(d.copy(custody = Custody.invoked), QueueFact.addInvoked)

def persistDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.invoked then disabled
  else accept(d.copy(custody = Custody.persisted), QueueFact.taskPersisted)

def reserveDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.invoked then disabled
  else accept(d.copy(custody = Custody.reserved), QueueFact.matchReserved)

def oneMoreDelivery(d: Delivered) = d match
  case Delivered.never => Delivered.once
  case _               => Delivered.twice

/** Matching holds a task a poll can take: reserved for a waiting poller, or persisted. */
def matchable(c: Custody) = c.in(Custody.reserved, Custody.persisted)

def deliverDetail(d: QueueDetail) =
  if matchable(d.custody) && !d.polled && d.delivered != Delivered.twice then
    List(
      Step(
        QueueOutcome.delivered,
        d.copy(polled = true, delivered = oneMoreDelivery(d.delivered)),
        List(QueueFact.delivered)
      )
    )
  else disabled

/** Matching completes the task, which discharges every custodian's obligation. */
def acknowledgeDetail(d: QueueDetail) =
  if d.polled && d.custody != Custody.nowhere && d.delivered != Delivered.never then
    List(Step(QueueOutcome.acknowledged, idleQueue, List(QueueFact.acknowledged)))
  else disabled

/**
 * The answer to the poller is lost. A persisted task stays queued; a sync match fails back to the
 * invocation, which history retries.
 */
def ackLossDetail(d: QueueDetail): List[QueueDetailStep] =
  if !d.polled then disabled
  else if d.custody == Custody.reserved then
    accept(d.copy(custody = Custody.invoked, polled = false), QueueFact.ackLost)
  else accept(d.copy(polled = false), QueueFact.ackLost)

/**
 * An ordinary crash loses what is only in memory, the poll, the invocation and a sync match, and
 * nothing durable: history still holds its dispatch task and retries, and a persisted task is still
 * queued.
 */
def crashDetail(d: QueueDetail): List[QueueDetailStep] = d.custody match
  case Custody.invoked | Custody.reserved =>
    accept(d.copy(custody = Custody.history, polled = false), QueueFact.crashed)
  case Custody.nowhere | Custody.history | Custody.persisted =>
    accept(d.copy(polled = false), QueueFact.crashed)

def storageLossDetail(d: QueueDetail) =
  if d.custody == Custody.nowhere then disabled
  else List(Step(QueueOutcome.lost, idleQueue, List(QueueFact.storageLost)))

def interfaceSees(f: QueueFact) = f match
  case QueueFact.enqueueCommitted | QueueFact.enqueueFailed | QueueFact.delivered |
      QueueFact.acknowledged | QueueFact.storageLost =>
    true
  case QueueFact.addInvoked | QueueFact.taskPersisted | QueueFact.matchReserved |
      QueueFact.crashed | QueueFact.ackLost =>
    false

def interfaceAnswers(o: QueueOutcome) = o != QueueOutcome.internal

def queueEnds(d: QueueDetail) = d.custody == Custody.nowhere

/** The detailed provider. */
val matchingQueue = machine[QueueDetail, QueueOutcome, QueueFact] {
  forEntity(taskQueueEntity)
  refines(dispatchQueue)(viewOf)
  visible(interfaceSees)
  visibleOutcomes(interfaceAnswers)
  starts(idleQueue)
  ends(queueEnds)
  steps(
    enqueue ~> enqueueDetail,
    addActivityTask ~> invokeDetail,
    persistTask ~> persistDetail,
    syncMatch ~> reserveDetail,
    deliver ~> deliverDetail,
    acknowledge ~> acknowledgeDetail,
    ackLoss ~> ackLossDetail,
    crash ~> crashDetail
  )
}

/**
 * The detailed provider with the storage-loss fault, which only its assumption allows. It refines
 * the interface that allows the loss.
 */
val lossyMatchingQueue = matchingQueue
  .extend(storageLoss ~> storageLossDetail)
  .refining(dispatchQueueUnderStorageLoss)(viewOf)
  .assuming(storageLossAssumed)

// ### The violating providers
//
// Each differs from the detailed provider in what one ordinary crash does, and neither assumes
// storage loss. They are negative controls a feature may compose its own design with: the replacement
// is what must fail.

/**
 * History drops its dispatch task when it invokes AddActivityTask, before matching persists
 * anything, so a crash there leaves no custodian for a message the interface still calls committed.
 */
def forgetfulCrash(d: QueueDetail): List[QueueDetailStep] = d.custody match
  case Custody.invoked | Custody.reserved                    => accept(idleQueue, QueueFact.crashed)
  case Custody.nowhere | Custody.history | Custody.persisted =>
    accept(d.copy(polled = false), QueueFact.crashed)

val forgetfulQueue = matchingQueue.rebind(crash ~> forgetfulCrash)

/** A crash wipes the tasks matching persisted, which history no longer backs. */
def volatileCrash(d: QueueDetail): List[QueueDetailStep] = d.custody match
  case Custody.persisted                  => accept(idleQueue, QueueFact.crashed)
  case Custody.invoked | Custody.reserved =>
    accept(d.copy(custody = Custody.history, polled = false), QueueFact.crashed)
  case Custody.nowhere | Custody.history => accept(d.copy(polled = false), QueueFact.crashed)

val volatileQueue = matchingQueue.rebind(crash ~> volatileCrash)
