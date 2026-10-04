/* The standalone activity's system contract: how history's authoritative activity record, the durable
 * dispatch queue and admission at RecordActivityTaskStarted keep the product's promise that a paused
 * activity is dispatched to no worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
 * chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/specimens/activity.md.
 *
 * Three identities stay apart. The logical activity is the entity. An attempt is what admission
 * commits, counted in the record. A delivery is one dispatch message, which the queue holds and may
 * hand out more than once. The record and the queue are separate machines, so a composition is what
 * checks them together, first over the opaque queue and then over the detailed one that replaces it.
 *
 * The stale design is a deliberately faulty control, not a claim about a known server defect.
 */
package temporal
package standaloneactivity

import umpire.*
import SystemFamily.given

/**
 * The family of the system contract's declarations. The activity's machines in Model.scala have a
 * family of their own in the same package, so each file imports the one its declarations take.
 */
object SystemFamily:
  given family: Family = Family("temporal.activity.standalone.system")

// ### The record: history's authoritative account of one activity
//
// Only pause is in scope, and no unpause intervenes, so a pause is where a path may end. Both
// deadlines are armed in every state, which is what lets them compete with a delivery.

/**
 * `pausedWhileHeld` is a pause that arrived after admission: the attempt it holds is work admitted
 * before the pause.
 */
enum AdmissionPhase derives Finite:
  case scheduled, paused, pausedWhileHeld, started, completed, timedOut

/**
 * Attempts admission committed and no result closed. An enum rather than an Int, because the lifter
 * gives every Int field of one record the same range.
 */
enum Active derives Finite:
  case none, one, two

/**
 * Whether admission owes matching the answer that lets it complete the task. A commit that fails
 * answers nothing, which is what keeps its message deliverable.
 */
enum Answer derives Finite:
  case settled, owed

final case class AdmissionState(phase: AdmissionPhase, active: Active, answer: Answer)
    derives Finite

/**
 * The statuses the product reads, by their product names, and the internal facts no product fact is
 * named after.
 */
enum AdmissionFact derives Finite:
  case statusStarted, statusPaused, statusCompleted
  case statusTimedOut(timeoutType: TimeoutType)
  case dispatchSent, attemptAdmitted, admissionRejected, admissionCommitFailed, deliveryAnswered

type AdmissionStep = Step[AdmissionState, Outcome, AdmissionFact]

/** History's dispatch task validates the activity and sends the message. */
val dispatch = internal

/** Admission's answer reaches matching, which may then complete the task. */
val answerDelivery = internal

val scheduledIdle: AdmissionState =
  AdmissionState(AdmissionPhase.scheduled, Active.none, Answer.settled)

def admissionOver(p: AdmissionPhase): Boolean =
  p.in(AdmissionPhase.completed, AdmissionPhase.timedOut)

/** The dispatch task's Validate: it is sent only while the activity can start. */
def dispatchStep(s: AdmissionState): List[AdmissionStep] =
  if s.phase != AdmissionPhase.scheduled then disabled
  else accept(s, AdmissionFact.dispatchSent)

/** A pause keeps whatever message is in flight: nothing recalls it. */
def pauseStep(s: AdmissionState, c: Control): List[AdmissionStep] = c match
  case Control.pause =>
    s.phase match
      case AdmissionPhase.scheduled =>
        accept(s.copy(phase = AdmissionPhase.paused), AdmissionFact.statusPaused)
      case AdmissionPhase.started =>
        accept(s.copy(phase = AdmissionPhase.pausedWhileHeld), AdmissionFact.statusPaused)
      case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => disabled // already paused
      case AdmissionPhase.completed | AdmissionPhase.timedOut     => disabled // over
  case Control.unpause | Control.requestCancel | Control.terminate => disabled // out of scope

def oneMore(a: Active): Active = a match
  case Active.none => Active.one
  case _           => Active.two

def oneLess(a: Active): Active = a match
  case Active.two => Active.one
  case _          => Active.none

val admissionCommits = choice
val admissionCommitFails = choice

/**
 * An admission, which either commits or fails its durable update. A failed commit records no
 * attempt and owes no answer, so the delivery it came by stays outstanding.
 */
def admitted(s: AdmissionState): List[AdmissionStep] = choose(
  admissionCommits -> accept(
    s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed),
    AdmissionFact.statusStarted,
    AdmissionFact.attemptAdmitted
  ),
  admissionCommitFails -> accept(s, AdmissionFact.admissionCommitFailed)
    .because("the durable update fails: nothing is admitted and the message stays deliverable")
)

/**
 * The corrected design: admission re-reads current eligibility. A delivery that meets an activity
 * that cannot start, a paused one or one whose attempt is already admitted, is answered and admits
 * nothing.
 */
def admitCurrent(s: AdmissionState): List[AdmissionStep] =
  if s.phase == AdmissionPhase.scheduled then admitted(s)
  else accept(s.copy(answer = Answer.owed), AdmissionFact.admissionRejected)

/** The deliberately faulty design: admission trusts the eligibility the message was sent with. */
def admitStale(s: AdmissionState): List[AdmissionStep] = admitted(s)

def answerStep(s: AdmissionState): List[AdmissionStep] =
  if s.answer != Answer.owed then disabled
  else accept(s.copy(answer = Answer.settled), AdmissionFact.deliveryAnswered)

def resultStep(s: AdmissionState, r: AttemptResult): List[AdmissionStep] = r match
  case AttemptResult.completed =>
    if s.phase != AdmissionPhase.started then disabled
    else
      accept(
        s.copy(phase = AdmissionPhase.completed, active = oneLess(s.active)),
        AdmissionFact.statusCompleted
      )
  case AttemptResult.failed(_) | AttemptResult.canceled => disabled

/** Covers the wait for a worker, so it fires only before an attempt is admitted. */
def admissionScheduleToStart(s: AdmissionState): List[AdmissionStep] =
  if s.phase == AdmissionPhase.scheduled then
    accept(
      s.copy(phase = AdmissionPhase.timedOut, active = Active.none),
      AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart)
    )
  else disabled

/** Covers the whole activity, so it competes with the other deadline while none has fired. */
def admissionScheduleToClose(s: AdmissionState): List[AdmissionStep] =
  if admissionOver(s.phase) then disabled
  else
    accept(
      s.copy(phase = AdmissionPhase.timedOut, active = Active.none),
      AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose)
    )

def productOfAdmission(s: AdmissionState): ProductState = s.phase match
  case AdmissionPhase.scheduled => ProductState(ProductPhase.scheduled)
  case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => ProductState(ProductPhase.paused)
  case AdmissionPhase.started                                 => ProductState(ProductPhase.started)
  case AdmissionPhase.completed => ProductState(ProductPhase.completed)
  case AdmissionPhase.timedOut  => ProductState(ProductPhase.timedOut)

/** A caller reads statuses and nothing of the dispatch, the admission or its answer. */
def productSees(f: AdmissionFact): Boolean = f match
  case AdmissionFact.statusStarted | AdmissionFact.statusPaused | AdmissionFact.statusCompleted |
      AdmissionFact.statusTimedOut(_) =>
    true
  case AdmissionFact.dispatchSent | AdmissionFact.attemptAdmitted |
      AdmissionFact.admissionRejected | AdmissionFact.admissionCommitFailed |
      AdmissionFact.deliveryAnswered =>
    false

/** No unpause is in scope, so a pause is where a path may end, as a completion is. */
def admissionEnds(s: AdmissionState): Boolean =
  !s.phase.in(AdmissionPhase.scheduled, AdmissionPhase.started)

// ### Monitors: at most one admitted active attempt, and terminal finality
//
// They count from what a step records, so they hold a design to the promise whatever its state says.

def countActive(active: Active, after: AdmissionStep): Active =
  if after.records(AdmissionFact.attemptAdmitted) then oneMore(active)
  else if after.records(AdmissionFact.statusCompleted) then oneLess(active)
  else if after.state.phase == AdmissionPhase.timedOut then Active.none
  else active

val atMostOneActiveAttempt =
  monitor[AdmissionState, Outcome, AdmissionFact, Active](Active.none)((active, _, after) =>
    countActive(active, after)
  )(_ == Active.two).readAfter(after => after.records(AdmissionFact.attemptAdmitted))

/** Whether the activity is over, and whether a step after that left where it ended. */
enum Finality derives Finite:
  case open, closed, reopened

def finality(f: Finality, after: AdmissionStep): Finality = f match
  case Finality.reopened => Finality.reopened
  case _                 =>
    if admissionOver(after.state.phase) then Finality.closed
    else if f == Finality.closed then Finality.reopened
    else Finality.open

val terminalFinality =
  monitor[AdmissionState, Outcome, AdmissionFact, Finality](Finality.open)((f, _, after) =>
    finality(f, after)
  )(_ == Finality.reopened)

// ### The two designs, under any delivery
//
// A design alone takes a delivery whenever one could arrive, whatever queue brings it: what holds of
// it holds over every queue, and what fails of it is confirmed over a queue below.

val currentAdmission = machine[AdmissionState, Outcome, AdmissionFact] {
  forEntity(activity)
  monitors(atMostOneActiveAttempt, terminalFinality)
  refines(activityProduct)(productOfAdmission)
  visible(productSees)
  starts(scheduledIdle)
  ends(admissionEnds)
  evidence { case AdmissionFact.statusTimedOut(_) => "statusTimedOut" }
  steps(
    dispatch ~> dispatchStep,
    control ~> pauseStep,
    attemptStart ~> admitCurrent,
    answerDelivery ~> answerStep,
    attemptResult ~> resultStep,
    scheduleToStart ~> admissionScheduleToStart,
    scheduleToClose ~> admissionScheduleToClose
  )
}

val staleAdmission = currentAdmission.rebind(attemptStart ~> admitStale)

// ### Promises, written once and declared on each design

def admitsWhilePaused(before: AdmissionPhase, after: AdmissionPhase): Boolean =
  before == AdmissionPhase.paused && after == AdmissionPhase.started

def leavesTheEnd(before: AdmissionPhase, after: AdmissionPhase): Boolean =
  admissionOver(before) && after != before

def notAdmittedWhilePaused(before: AdmissionState, after: AdmissionStep): Boolean =
  !admitsWhilePaused(before.phase, after.state.phase)

def atMostOneActive(after: AdmissionStep): Boolean = after.state.active != Active.two

def terminalStays(before: AdmissionState, after: AdmissionStep): Boolean =
  !leavesTheEnd(before.phase, after.state.phase)

val five = Limits(steps = 5, actions = 5, search = 65536)
val seven = Limits(steps = 7, actions = 7, search = 262144)
val eight = Limits(steps = 8, actions = 8, search = 262144)

/** Past the depth of the detailed queue's table and of a design composed with it, which is ten. */
val twelve = Limits(steps = 12, actions = 12, search = 262144)

/** Every claim and path, declared on one design: a Property or Scenario belongs to one machine. */
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]): Vector[Query] =
  val notPaused = m.property("notAdmittedWhilePaused") holdsAcross notAdmittedWhilePaused
  val oneActive = m.property("atMostOneActive") holds atMostOneActive
  val terminal = m.property("terminalStays") holdsAcross terminalStays
  val startDeadline = m.property("scheduleToStartTimesOut") when scheduleToStart holds (after =>
    after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
  )
  val closeDeadline = m.property("scheduleToCloseTimesOut") when scheduleToClose holds (after =>
    after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
  )
  val stale =
    m.scenario("staleDeliveryAfterPause").actions(dispatch, control(Control.pause), attemptStart)
  val prePause =
    m.scenario("admittedBeforePause").actions(dispatch, attemptStart, control(Control.pause))
  val duplicate = m.scenario("duplicateDelivery").actions(dispatch, attemptStart, attemptStart)
  val reopened = m
    .scenario("startedAfterCompletion")
    .actions(dispatch, attemptStart, attemptResult(AttemptResult.completed), attemptStart)
  val startFirst = m.scenario("scheduleToStartFirst").actions(dispatch, scheduleToStart)
  val closeFirst = m.scenario("scheduleToCloseFirst").actions(dispatch, scheduleToClose)
  val any = m.scenario("any").free
  Vector(
    query(s"${m.name}.staleDelivery") verify notPaused in stale limits three total 108,
    query(s"${m.name}.admittedBeforePause") verify notPaused in prePause limits three total 108,
    query(s"${m.name}.duplicateDelivery") verify oneActive in duplicate limits three total 108,
    // Each asks a Property its path keeps, so a violation is the watching monitor's alone.
    query(s"${m.name}.duplicateDelivery.monitored") verify notPaused in
      duplicate limits three total 108,
    query(s"${m.name}.startedAfterCompletion.monitored") verify oneActive in
      reopened limits four total 144,
    query(s"${m.name}.any.notAdmittedWhilePaused") verify notPaused in any limits five total 2340,
    query(s"${m.name}.any.atMostOneActive") verify oneActive in any limits five total 2340,
    query(s"${m.name}.any.terminalStays") verify terminal in any limits five total 2340,
    // Neither deadline is ordered before the other: each firing is a trace of its own.
    query(s"${m.name}.scheduleToStartFirst") find startDeadline in startFirst limits three total 72,
    query(s"${m.name}.scheduleToCloseFirst") find closeDeadline in closeFirst limits three total 72,
    // The product's own Property, read through the design's declared refinement.
    query(s"${m.name}.product.pausedIsNotDispatched")
      .verify(pausedIsNotDispatched)
      .in(stale) limits three total 108
  )

val currentQueries: Vector[Query] = admissionQueries(currentAdmission)
val staleQueries: Vector[Query] = admissionQueries(staleAdmission)

// ### The competing deadlines of the protocol machine
//
// With both deadlines set and no attempt started, each may fire first, and which did is what the
// status records.

/** The schedule-to-close deadline times the activity out and the status records which it was. */
val scheduleToCloseFires =
  activityProtocol.property when scheduleToClose holds { s =>
    s.state.phase == Phase.timedOut &&
    s.records(ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose))
  }

val bothDeadlinesStartFirst =
  activityProtocol.scenario.actions(
    start(Inputs.scheduleToClose := Timeout.expires, Inputs.scheduleToStart := Timeout.expires),
    scheduleToStart
  )

val bothDeadlinesCloseFirst =
  activityProtocol.scenario.actions(
    start(Inputs.scheduleToClose := Timeout.expires, Inputs.scheduleToStart := Timeout.expires),
    scheduleToClose
  )

val competingTimers: Vector[Query] = Vector(
  query("competingTimers.scheduleToStartFirst")
    .find(scheduleToStartFires)
    .in(bothDeadlinesStartFirst) limits three total 576,
  query("competingTimers.scheduleToCloseFirst")
    .find(scheduleToCloseFires)
    .in(bothDeadlinesCloseFirst) limits three total 576
)

// ### The dispatch queue's interface
//
// What the activity may rely on of the durable queue between history and a worker's poll: an enqueue
// commits or fails; a committed message is delivered up to twice before its acknowledgment; and no
// committed message is lost. A crash shows at this interface only as that second delivery.

val fault: Party = Party("fault")

val enqueue = internal
val deliver = internal
val acknowledge = internal

/** The one message the queue may hold, by how far its delivery got. */
enum Outstanding derives Finite:
  case empty, committed, deliveredOnce, deliveredTwice

final case class QueueView(outstanding: Outstanding) derives Finite

/** `internal` answers every step of a provider that its interface does not show. */
enum QueueOutcome derives Finite:
  case committed, failed, delivered, acknowledged, lost, internal

object QueueOutcome:
  // A provider's step behind the interface is the one that answers `internal`. Kept in the
  // companion, where only a step known to answer a QueueOutcome finds it, so a `choose` of the
  // activity's steps still finds the package's `Accepted[Outcome]` alone.
  given Accepted[QueueOutcome] = Accepted(QueueOutcome.internal)

/** The interface's events, and what a provider records of the steps behind them. */
enum QueueFact derives Finite:
  case enqueueCommitted, enqueueFailed, delivered, acknowledged, storageLost
  case addInvoked, taskPersisted, matchReserved, crashed, ackLost

type QueueViewStep = Step[QueueView, QueueOutcome, QueueFact]

val enqueueCommits = choice
val enqueueFails = choice

def enqueueView(q: QueueView): List[QueueViewStep] =
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

def deliverView(q: QueueView): List[QueueViewStep] = q.outstanding match
  case Outstanding.committed =>
    List(
      Step(QueueOutcome.delivered, QueueView(Outstanding.deliveredOnce), List(QueueFact.delivered))
    )
  case Outstanding.deliveredOnce =>
    List(
      Step(QueueOutcome.delivered, QueueView(Outstanding.deliveredTwice), List(QueueFact.delivered))
    ).because("a message not yet acknowledged may be delivered again")
  case Outstanding.empty | Outstanding.deliveredTwice => disabled

def acknowledgeView(q: QueueView): List[QueueViewStep] = q.outstanding match
  case Outstanding.deliveredOnce | Outstanding.deliveredTwice =>
    List(
      Step(QueueOutcome.acknowledged, QueueView(Outstanding.empty), List(QueueFact.acknowledged))
    )
  case Outstanding.empty | Outstanding.committed => disabled

def storageLossView(q: QueueView): List[QueueViewStep] =
  if q.outstanding == Outstanding.empty then disabled
  else List(Step(QueueOutcome.lost, QueueView(Outstanding.empty), List(QueueFact.storageLost)))

/** A check over the opaque queue rests on the interface alone. */
val queueOpaque: Assumption = assume("dispatchQueue.opaque")

/**
 * Committed storage may be lost. It is a fault of its own, apart from a crash, and only a machine
 * that assumes it has the step.
 */
val storageLossAssumed: Assumption = assume("storageLoss")
val storageLoss = action(fault)

val emptyQueue: QueueView = QueueView(Outstanding.empty)

/** The opaque provider. */
val dispatchQueue = machine[QueueView, QueueOutcome, QueueFact] {
  assumes(queueOpaque)
  starts(emptyQueue)
  ends(q => q.outstanding == Outstanding.empty)
  steps(enqueue ~> enqueueView, deliver ~> deliverView, acknowledge ~> acknowledgeView)
}

/** The interface under the storage-loss assumption: a committed message may also vanish. */
val dispatchQueueUnderStorageLoss = machine[QueueView, QueueOutcome, QueueFact] {
  assumes(queueOpaque, storageLossAssumed)
  starts(emptyQueue)
  ends(q => q.outstanding == Outstanding.empty)
  steps(
    enqueue ~> enqueueView,
    deliver ~> deliverView,
    acknowledge ~> acknowledgeView,
    storageLoss ~> storageLossView
  )
}

// ### The detailed queue: history's dispatch task and matching's custody
//
// The route of one message: history durably schedules the dispatch task; its Execute invokes
// AddActivityTask; matching either persists the task or reserves it for a waiting poller; a poll
// hands it out; and once admission has answered, matching completes it. Each step is its own
// transition, so a crash can fall between any two.

val addActivityTask = internal
val persistTask = internal
val syncMatch = internal
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

/** `polled` is a poller holding the task while admission decides. */
final case class QueueDetail(custody: Custody, polled: Boolean, delivered: Delivered) derives Finite

type QueueDetailStep = Step[QueueDetail, QueueOutcome, QueueFact]

val idleQueue: QueueDetail = QueueDetail(Custody.nowhere, false, Delivered.never)

/** The interface state a detailed state stands for: a message is outstanding while anyone holds it. */
def viewOf(d: QueueDetail): QueueView =
  if d.custody == Custody.nowhere then QueueView(Outstanding.empty)
  else
    d.delivered match
      case Delivered.never => QueueView(Outstanding.committed)
      case Delivered.once  => QueueView(Outstanding.deliveredOnce)
      case Delivered.twice => QueueView(Outstanding.deliveredTwice)

def enqueueDetail(d: QueueDetail): List[QueueDetailStep] =
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

def oneMoreDelivery(d: Delivered): Delivered = d match
  case Delivered.never => Delivered.once
  case _               => Delivered.twice

/** Matching holds a task a poll can take: reserved for a waiting poller, or persisted. */
def matchable(c: Custody): Boolean = c.in(Custody.reserved, Custody.persisted)

def deliverDetail(d: QueueDetail): List[QueueDetailStep] =
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
def acknowledgeDetail(d: QueueDetail): List[QueueDetailStep] =
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

def storageLossDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody == Custody.nowhere then disabled
  else List(Step(QueueOutcome.lost, idleQueue, List(QueueFact.storageLost)))

def interfaceSees(f: QueueFact): Boolean = f match
  case QueueFact.enqueueCommitted | QueueFact.enqueueFailed | QueueFact.delivered |
      QueueFact.acknowledged | QueueFact.storageLost =>
    true
  case QueueFact.addInvoked | QueueFact.taskPersisted | QueueFact.matchReserved |
      QueueFact.crashed | QueueFact.ackLost =>
    false

def interfaceAnswers(o: QueueOutcome): Boolean = o != QueueOutcome.internal

def queueEnds(d: QueueDetail): Boolean = d.custody == Custody.nowhere

/** The detailed provider. */
val matchingQueue = machine[QueueDetail, QueueOutcome, QueueFact] {
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

/** The detailed provider with the storage-loss fault, which only its assumption allows. */
val lossyMatchingQueue = machine[QueueDetail, QueueOutcome, QueueFact] {
  assumes(storageLossAssumed)
  refines(dispatchQueueUnderStorageLoss)(viewOf)
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
    crash ~> crashDetail,
    storageLoss ~> storageLossDetail
  )
}

// ### The violating providers
//
// Each differs from the detailed provider in what one ordinary crash does, and neither assumes
// storage loss.

/**
 * History drops its dispatch task when it invokes AddActivityTask, before matching persists
 * anything, so a crash there leaves no custodian for a message the interface still calls committed.
 */
def forgetfulCrash(d: QueueDetail): List[QueueDetailStep] = d.custody match
  case Custody.invoked | Custody.reserved                    => accept(idleQueue, QueueFact.crashed)
  case Custody.nowhere | Custody.history | Custody.persisted =>
    accept(d.copy(polled = false), QueueFact.crashed)

val forgetfulQueue = machine[QueueDetail, QueueOutcome, QueueFact] {
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
    crash ~> forgetfulCrash
  )
}

/** A crash wipes the tasks matching persisted, which history no longer backs. */
def volatileCrash(d: QueueDetail): List[QueueDetailStep] = d.custody match
  case Custody.persisted                  => accept(idleQueue, QueueFact.crashed)
  case Custody.invoked | Custody.reserved =>
    accept(d.copy(custody = Custody.history, polled = false), QueueFact.crashed)
  case Custody.nowhere | Custody.history => accept(d.copy(polled = false), QueueFact.crashed)

val volatileQueue = machine[QueueDetail, QueueOutcome, QueueFact] {
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
    crash ~> volatileCrash
  )
}

// ### What a provider promises, and the crash cuts

/** A message a custodian holds stays held until it is acknowledged. */
def committedStays(before: QueueDetail, after: QueueDetailStep): Boolean =
  before.custody != Custody.nowhere implies
    (after.state.custody != Custody.nowhere || after.records(QueueFact.acknowledged))

/**
 * One crash at each point of the route, and the delivery that must still follow it: after the
 * invocation, after the sync match, after persistence and after a delivery. After the
 * acknowledgment nothing is left to deliver. `anyTotal` is the static combination count of its free
 * `any` Queries.
 */
def providerQueries(
    m: Machine[QueueDetail, QueueOutcome, QueueFact],
    anyTotal: Int
): Vector[Query] =
  val stays = m.property("committedStays") holdsAcross committedStays
  val delivers =
    m.property("delivers") when deliver holds (after => after.records(QueueFact.delivered))
  val afterInvocation = m
    .scenario("crashAfterInvocation")
    .actions(enqueue, addActivityTask, crash, addActivityTask, persistTask, deliver)
  val afterSyncMatch = m
    .scenario("crashAfterSyncMatch")
    .actions(enqueue, addActivityTask, syncMatch, crash, addActivityTask, syncMatch, deliver)
  val afterPersistence = m
    .scenario("crashAfterPersistence")
    .actions(enqueue, addActivityTask, persistTask, crash, deliver)
  val afterDelivery = m
    .scenario("crashAfterDelivery")
    .actions(enqueue, addActivityTask, persistTask, deliver, crash, deliver)
  val afterAcknowledgment = m
    .scenario("crashAfterAcknowledgment")
    .actions(enqueue, addActivityTask, persistTask, deliver, acknowledge, crash)
  val any = m.scenario("any").free
  Vector(
    query(s"${m.name}.crashAfterInvocation") find delivers in
      afterInvocation limits seven total 180,
    query(s"${m.name}.crashAfterSyncMatch") find delivers in afterSyncMatch limits seven total 210,
    query(s"${m.name}.crashAfterPersistence") find delivers in
      afterPersistence limits seven total 150,
    query(s"${m.name}.crashAfterDelivery") find delivers in afterDelivery limits seven total 180,
    query(s"${m.name}.crashAfterAcknowledgment") verify stays in
      afterAcknowledgment limits seven total 180,
    query(s"${m.name}.any.committedStays") verify stays in any limits twelve total anyTotal
  )

val matchingQueueQueries: Vector[Query] = providerQueries(matchingQueue, anyTotal = 2880)
val forgetfulQueueQueries: Vector[Query] = providerQueries(forgetfulQueue, anyTotal = 2880)
val volatileQueueQueries: Vector[Query] = providerQueries(volatileQueue, anyTotal = 2880)
val lossyMatchingQueueQueries: Vector[Query] = providerQueries(lossyMatchingQueue, anyTotal = 3240)

/** Storage loss drops a committed message, and the queue records that it did. */
val storageLossDrops =
  lossyMatchingQueue.property when storageLoss holds { after =>
    after.state.custody == Custody.nowhere && after.records(QueueFact.storageLost)
  }

val persistedThenLost =
  lossyMatchingQueue.scenario.actions(enqueue, addActivityTask, persistTask, storageLoss)

val storageLossQuery =
  query("lossyMatchingQueue.storageLoss") find storageLossDrops in
    persistedThenLost limits seven total 120

// ### The record as a member
//
// A composition's Queries are not answered while a member names monitors, so the members are the
// two designs again without them. They declare no refinement: the designs above are what refine the
// product.

val currentRecord = currentAdmission.unmonitored
val staleRecord = staleAdmission.unmonitored

// ### A design over the opaque queue
//
// The dispatch is the queue's enqueue, a delivery is admission's one input, and admission's answer
// is the acknowledgment. Split that way, a commit, its answer and the acknowledgment are three
// points a fault can fall between.

final case class OverQueue(activity: AdmissionState, queue: QueueView)

val idleOverQueue: OverQueue = OverQueue(scheduledIdle, emptyQueue)

val currentOverQueue: Composition[OverQueue] =
  compose[OverQueue]("activity" -> currentRecord, "queue" -> dispatchQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .ends(s => admissionEnds(s.activity))

val staleOverQueue: Composition[OverQueue] =
  compose[OverQueue]("activity" -> staleRecord, "queue" -> dispatchQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .ends(s => admissionEnds(s.activity))

def overQueueQueries(c: Composition[OverQueue]): Vector[Query] =
  val notPaused = c.property("notAdmittedWhilePaused") holdsAcross ((before, after) =>
    !admitsWhilePaused(before.activity.phase, after.state.activity.phase)
  )
  val oneActive =
    c.property("atMostOneActive") holds (after => after.state.activity.active != Active.two)
  val terminal = c.property("terminalStays") holdsAcross ((before, after) =>
    !leavesTheEnd(before.activity.phase, after.state.activity.phase)
  )
  // A failed commit admits nothing and leaves its message with the queue.
  val failedCommit = c.property("failedCommitKeepsTheMessage") holds (after =>
    !after.facts.contains("activity_admissionCommitFailed") ||
      (after.state.queue.outstanding != Outstanding.empty &&
        !after.facts.contains("activity_attemptAdmitted"))
  )
  val stale = c
    .scenario("staleDeliveryAfterPause")
    .actionKeys("dispatch", "activity_control-pause", "admit")
  val prePause = c
    .scenario("admittedBeforePause")
    .actionKeys("dispatch", "admit", "activity_control-pause")
  val duplicate = c.scenario("duplicateDelivery").actionKeys("dispatch", "admit", "admit")
  val any = c.scenario("any").free
  Vector(
    query(s"${c.name}.staleDelivery") verify notPaused in stale limits three total 432,
    query(s"${c.name}.admittedBeforePause") verify notPaused in prePause limits three total 432,
    query(s"${c.name}.duplicateDelivery") verify oneActive in duplicate limits three total 432,
    query(s"${c.name}.failedCommit") verify failedCommit in duplicate limits three total 432,
    query(s"${c.name}.any.notAdmittedWhilePaused") verify notPaused in any limits five total 9360,
    query(s"${c.name}.any.atMostOneActive") verify oneActive in any limits five total 9360,
    query(s"${c.name}.any.terminalStays") verify terminal in any limits five total 9360
  )

val currentOverQueueQueries: Vector[Query] = overQueueQueries(currentOverQueue)
val staleOverQueueQueries: Vector[Query] = overQueueQueries(staleOverQueue)

// ### A design over the detailed queue, which replaces the opaque one
//
// The replacement is scoped to the composition: it holds only where the detailed provider refines
// the interface it stands in for, and its checks then rely on that provider, not on the interface.

final case class OverMatching(activity: AdmissionState, queue: QueueDetail)

val idleOverMatching: OverMatching = OverMatching(scheduledIdle, idleQueue)

val currentOverMatching: Composition[OverMatching] =
  compose[OverMatching]("activity" -> currentRecord, "queue" -> matchingQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

val staleOverMatching: Composition[OverMatching] =
  compose[OverMatching]("activity" -> staleRecord, "queue" -> matchingQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

/** The corrected design over each violating provider: the replacement is what must fail. */
val currentOverForgetful: Composition[OverMatching] =
  compose[OverMatching]("activity" -> currentRecord, "queue" -> forgetfulQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

val currentOverVolatile: Composition[OverMatching] =
  compose[OverMatching]("activity" -> currentRecord, "queue" -> volatileQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

/** The corrected design where storage loss is assumed, over the interface that allows it. */
val currentOverLossyMatching: Composition[OverMatching] =
  compose[OverMatching]("activity" -> currentRecord, "queue" -> lossyMatchingQueue)
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueueUnderStorageLoss)
    .ends(s => admissionEnds(s.activity))

/** `anyTotal` is the static combination count of its free `any` Queries. */
def overMatchingQueries(c: Composition[OverMatching], anyTotal: Int): Vector[Query] =
  val notPaused = c.property("notAdmittedWhilePaused") holdsAcross ((before, after) =>
    !admitsWhilePaused(before.activity.phase, after.state.activity.phase)
  )
  val oneActive =
    c.property("atMostOneActive") holds (after => after.state.activity.active != Active.two)
  val terminal = c.property("terminalStays") holdsAcross ((before, after) =>
    !leavesTheEnd(before.activity.phase, after.state.activity.phase)
  )
  val stale = c
    .scenario("staleDeliveryAfterPause")
    .actionKeys(
      "dispatch",
      "queue_addActivityTask",
      "queue_persistTask",
      "activity_control-pause",
      "admit"
    )
  val prePause = c
    .scenario("admittedBeforePause")
    .actionKeys(
      "dispatch",
      "queue_addActivityTask",
      "queue_persistTask",
      "admit",
      "activity_control-pause"
    )
  // The answer to the poller is lost after the commit, so the persisted task is handed out again.
  val lostAck = c
    .scenario("deliveredAgainAfterLostAck")
    .actionKeys(
      "dispatch",
      "queue_addActivityTask",
      "queue_persistTask",
      "admit",
      "queue_ackLoss",
      "admit"
    )
  // A crash after the admission commit and before the acknowledgment: history retries the sync match.
  val crashAfterCommit = c
    .scenario("crashAfterAdmissionCommit")
    .actionKeys(
      "dispatch",
      "queue_addActivityTask",
      "queue_syncMatch",
      "admit",
      "queue_crash",
      "queue_addActivityTask",
      "queue_syncMatch",
      "admit"
    )
  val any = c.scenario("any").free
  Vector(
    query(s"${c.name}.staleDelivery") verify notPaused in stale limits five total 5400,
    query(s"${c.name}.admittedBeforePause") verify notPaused in prePause limits five total 5400,
    query(s"${c.name}.deliveredAgainAfterLostAck") verify oneActive in
      lostAck limits seven total 6480,
    query(s"${c.name}.crashAfterAdmissionCommit") verify oneActive in
      crashAfterCommit limits eight total 8640,
    query(s"${c.name}.any.notAdmittedWhilePaused") verify notPaused in
      any limits twelve total anyTotal,
    query(s"${c.name}.any.atMostOneActive") verify oneActive in any limits twelve total anyTotal,
    query(s"${c.name}.any.terminalStays") verify terminal in any limits twelve total anyTotal
  )

val currentOverMatchingQueries: Vector[Query] =
  overMatchingQueries(currentOverMatching, anyTotal = 233280)
val staleOverMatchingQueries: Vector[Query] =
  overMatchingQueries(staleOverMatching, anyTotal = 233280)
val currentOverLossyMatchingQueries: Vector[Query] =
  overMatchingQueries(currentOverLossyMatching, anyTotal = 246240)

// ### The held race, run against a server
//
// The stale message is held at the dispatch cut while the pause commits, then delivered to
// admission. The server is expected to follow the corrected design, so the race is declared on that
// design, in the scope a Run of it has: the start sets no schedule deadline, so none fires, and no
// fault is injected, so admission's durable update does not fail. The hold is what makes that scope
// true of a Run: no delivery reaches admission before the release, and after the pause the corrected
// design rejects whatever arrives. The stale design's violation is shown by its own verify Query,
// never by a Run.

/** Admission as the corrected design decides it, with no failure of its durable update. */
def admitHeld(s: AdmissionState): List[AdmissionStep] =
  if s.phase == AdmissionPhase.scheduled then
    accept(
      s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed),
      AdmissionFact.statusStarted,
      AdmissionFact.attemptAdmitted
    )
  else accept(s.copy(answer = Answer.owed), AdmissionFact.admissionRejected)

val heldAdmission = machine[AdmissionState, Outcome, AdmissionFact] {
  forEntity(activity)
  monitors(atMostOneActiveAttempt, terminalFinality)
  refines(activityProduct)(productOfAdmission)
  visible(productSees)
  starts(scheduledIdle)
  ends(admissionEnds)
  evidence { case AdmissionFact.statusTimedOut(_) => "statusTimedOut" }
  steps(
    dispatch ~> dispatchStep,
    control ~> pauseStep,
    attemptStart ~> admitHeld,
    answerDelivery ~> answerStep
  )
}

/** Admission met the stale message and rejected it. */
val staleDeliveryRejected =
  heldAdmission.property when attemptStart holds (after =>
    after.records(AdmissionFact.admissionRejected)
  )

val heldStaleDelivery =
  (query("heldAdmission.staleDelivery") find staleDeliveryRejected in heldAdmission
    .scenario("heldStaleDelivery")
    .actions(dispatch, control(Control.pause), attemptStart) limits three total 108).expect(
    umpire.realize.RunExpectation(
      umpire.realize.Conformance.conformant,
      umpire.realize.Outcome.satisfied,
      monitors = Vector(
        umpire.realize
          .MonitorExpectation(
            "atMostOneActiveAttempt",
            umpire.realize.Outcome.inconclusive,
            "an execution that explains the evidence never reaches the claim's evaluation point"
          ),
        umpire.realize.MonitorExpectation("terminalFinality", umpire.realize.Outcome.satisfied)
      )
    )
  )

// A bounded projection of admission and its lost answer. The one response-loss budget is consumed
// whether the update committed or failed: the caller's missing answer alone distinguishes neither.
// The in-process actuator supplies the durable decision separately and realizes the committed arm.
final case class AdmissionResponseState(record: AdmissionState, lossAvailable: Boolean)
    derives Finite

enum AdmissionResponseFact derives Finite:
  case dispatchSent, attemptAdmitted

val responseLossInitial: AdmissionResponseState = AdmissionResponseState(scheduledIdle, true)

def responseLossDispatch(
    s: AdmissionResponseState
): List[Step[AdmissionResponseState, Outcome, AdmissionResponseFact]] =
  if !s.lossAvailable then disabled
  else accept(s, AdmissionResponseFact.dispatchSent)

val committedThenLost = choice
val failedThenLost = choice

def loseAdmissionAnswer(
    s: AdmissionResponseState
): List[Step[AdmissionResponseState, Outcome, AdmissionResponseFact]] =
  if !s.lossAvailable then disabled
  else
    choose(
      committedThenLost -> accept(
        AdmissionResponseState(
          s.record.copy(
            phase = AdmissionPhase.started,
            active = oneMore(s.record.active),
            answer = Answer.owed
          ),
          false
        ),
        AdmissionResponseFact.attemptAdmitted
      ),
      failedThenLost -> accept(s.copy(lossAvailable = false))
        .because("the durable update failed before its answer was lost")
    )

val admissionResponseLoss = machine[AdmissionResponseState, Outcome, AdmissionResponseFact] {
  forEntity(activity)
  starts(responseLossInitial)
  ends(s => !s.lossAvailable)
  steps(dispatch ~> responseLossDispatch, ackLoss ~> loseAdmissionAnswer)
}

val committedDespiteLostResponse =
  admissionResponseLoss.property when ackLoss holds (after =>
    after.records(AdmissionResponseFact.attemptAdmitted)
  )

val lostAdmissionResponseQuery =
  (query("admissionResponseLoss.committed") find committedDespiteLostResponse in
    admissionResponseLoss
      .scenario("oneLostResponse")
      .actions(dispatch, ackLoss) limits three total 144).expect(
    umpire.realize
      .RunExpectation(umpire.realize.Conformance.conformant, umpire.realize.Outcome.satisfied)
  )
