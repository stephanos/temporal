/* The standalone activity's system contract: how history's authoritative activity record, the durable
 * dispatch queue and admission at RecordActivityTaskStarted keep the product's promise that a paused
 * activity is dispatched to no worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
 * chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/scalav2/specimens/activity.md.
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

val SystemFamily: umpire.Family = umpire.Family("temporal.activity.standalone.system")

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
val dispatch = internal("dispatch")

/** Admission's answer reaches matching, which may then complete the task. */
val answerDelivery = internal("answerDelivery")

val scheduledIdle: AdmissionState =
  AdmissionState(AdmissionPhase.scheduled, Active.none, Answer.settled)

def admissionOver(p: AdmissionPhase): Boolean =
  p == AdmissionPhase.completed || p == AdmissionPhase.timedOut

/** The dispatch task's Validate: it is sent only while the activity can start. */
def dispatchStep(s: AdmissionState): List[AdmissionStep] =
  if s.phase != AdmissionPhase.scheduled then Nil
  else List(Step(Outcome.accepted, s, List(AdmissionFact.dispatchSent)))

private def pauses(s: AdmissionState, phase: AdmissionPhase): List[AdmissionStep] =
  List(Step(Outcome.accepted, s.copy(phase = phase), List(AdmissionFact.statusPaused)))

/** A pause keeps whatever message is in flight: nothing recalls it. */
def pauseStep(s: AdmissionState, c: Control): List[AdmissionStep] = c match
  case Control.pause =>
    s.phase match
      case AdmissionPhase.scheduled => pauses(s, AdmissionPhase.paused)
      case AdmissionPhase.started   => pauses(s, AdmissionPhase.pausedWhileHeld)
      case _                        => Nil
  case Control.unpause | Control.requestCancel | Control.terminate => Nil

def oneMore(a: Active): Active = a match
  case Active.none => Active.one
  case _           => Active.two

def oneLess(a: Active): Active = a match
  case Active.two => Active.one
  case _          => Active.none

/**
 * An admission, which either commits or fails its durable update. A failed commit records no
 * attempt and owes no answer, so the delivery it came by stays outstanding.
 */
def admitted(s: AdmissionState): List[AdmissionStep] = List(
  Step(
    Outcome.accepted,
    s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed),
    List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted)
  ),
  Step(
    Outcome.accepted,
    s,
    List(AdmissionFact.admissionCommitFailed),
    "the durable update fails: nothing is admitted and the message stays deliverable"
  )
)

/**
 * The corrected design: admission re-reads current eligibility. A delivery that meets an activity
 * that cannot start, a paused one or one whose attempt is already admitted, is answered and admits
 * nothing.
 */
def admitCurrent(s: AdmissionState): List[AdmissionStep] =
  if s.phase == AdmissionPhase.scheduled then admitted(s)
  else
    List(
      Step(Outcome.accepted, s.copy(answer = Answer.owed), List(AdmissionFact.admissionRejected))
    )

/** The deliberately faulty design: admission trusts the eligibility the message was sent with. */
def admitStale(s: AdmissionState): List[AdmissionStep] = admitted(s)

def answerStep(s: AdmissionState): List[AdmissionStep] =
  if s.answer != Answer.owed then Nil
  else
    List(
      Step(Outcome.accepted, s.copy(answer = Answer.settled), List(AdmissionFact.deliveryAnswered))
    )

def resultStep(s: AdmissionState, r: AttemptResult): List[AdmissionStep] = r match
  case AttemptResult.completed =>
    if s.phase != AdmissionPhase.started then Nil
    else
      List(
        Step(
          Outcome.accepted,
          s.copy(phase = AdmissionPhase.completed, active = oneLess(s.active)),
          List(AdmissionFact.statusCompleted)
        )
      )
  case AttemptResult.failed(_) | AttemptResult.canceled => Nil

private def timesOut(s: AdmissionState, deadline: TimeoutType): List[AdmissionStep] = List(
  Step(
    Outcome.accepted,
    s.copy(phase = AdmissionPhase.timedOut, active = Active.none),
    List(AdmissionFact.statusTimedOut(deadline))
  )
)

/** Covers the wait for a worker, so it fires only before an attempt is admitted. */
def admissionScheduleToStart(s: AdmissionState): List[AdmissionStep] =
  if s.phase == AdmissionPhase.scheduled then timesOut(s, TimeoutType.scheduleToStart) else Nil

/** Covers the whole activity, so it competes with the other deadline while none has fired. */
def admissionScheduleToClose(s: AdmissionState): List[AdmissionStep] =
  if admissionOver(s.phase) then Nil else timesOut(s, TimeoutType.scheduleToClose)

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

def admissionEvidence(f: AdmissionFact): String = f match
  case AdmissionFact.statusStarted         => "statusStarted"
  case AdmissionFact.statusPaused          => "statusPaused"
  case AdmissionFact.statusCompleted       => "statusCompleted"
  case AdmissionFact.statusTimedOut(_)     => "statusTimedOut"
  case AdmissionFact.dispatchSent          => "dispatchSent"
  case AdmissionFact.attemptAdmitted       => "attemptAdmitted"
  case AdmissionFact.admissionRejected     => "admissionRejected"
  case AdmissionFact.admissionCommitFailed => "admissionCommitFailed"
  case AdmissionFact.deliveryAnswered      => "deliveryAnswered"

/** No unpause is in scope, so a pause is where a path may end, as a completion is. */
def admissionEnds(s: AdmissionState): Boolean =
  s.phase != AdmissionPhase.scheduled && s.phase != AdmissionPhase.started

// ### Monitors: at most one admitted active attempt, and terminal finality
//
// They count from what a step records, so they hold a design to the promise whatever its state says.

def countActive(active: Active, after: AdmissionStep): Active =
  if after.facts.contains(AdmissionFact.attemptAdmitted) then oneMore(active)
  else if after.facts.contains(AdmissionFact.statusCompleted) then oneLess(active)
  else if after.state.phase == AdmissionPhase.timedOut then Active.none
  else active

val atMostOneActiveAttempt: Monitor[AdmissionState, Outcome, AdmissionFact, Active] =
  monitor[AdmissionState, Outcome, AdmissionFact, Active]("atMostOneActiveAttempt", Active.none)(
    (active, _, after) => countActive(active, after)
  )(_ == Active.two).readAfter(after => after.facts.contains(AdmissionFact.attemptAdmitted))

/** Whether the activity is over, and whether a step after that left where it ended. */
enum Finality derives Finite:
  case open, closed, reopened

def finality(f: Finality, after: AdmissionStep): Finality = f match
  case Finality.reopened => Finality.reopened
  case _                 =>
    if admissionOver(after.state.phase) then Finality.closed
    else if f == Finality.closed then Finality.reopened
    else Finality.open

val terminalFinality: Monitor[AdmissionState, Outcome, AdmissionFact, Finality] =
  monitor[AdmissionState, Outcome, AdmissionFact, Finality]("terminalFinality", Finality.open)(
    (f, _, after) => finality(f, after)
  )(_ == Finality.reopened)

// ### The two designs, under any delivery
//
// A design alone takes a delivery whenever one could arrive, whatever queue brings it: what holds of
// it holds over every queue, and what fails of it is confirmed over a queue below.

val currentAdmission: Machine[AdmissionState, Outcome, AdmissionFact] =
  machine[AdmissionState, Outcome, AdmissionFact](SystemFamily, "currentAdmission") {
    forEntity(activity)
    monitors(atMostOneActiveAttempt, terminalFinality)
    refines(activityProduct)(productOfAdmission)
    visible(productSees)
    starts(scheduledIdle)
    ends(admissionEnds)
    evidence(admissionEvidence)
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

val staleAdmission: Machine[AdmissionState, Outcome, AdmissionFact] =
  machine[AdmissionState, Outcome, AdmissionFact](SystemFamily, "staleAdmission") {
    forEntity(activity)
    monitors(atMostOneActiveAttempt, terminalFinality)
    refines(activityProduct)(productOfAdmission)
    visible(productSees)
    starts(scheduledIdle)
    ends(admissionEnds)
    evidence(admissionEvidence)
    steps(
      dispatch ~> dispatchStep,
      control ~> pauseStep,
      attemptStart ~> admitStale,
      answerDelivery ~> answerStep,
      attemptResult ~> resultStep,
      scheduleToStart ~> admissionScheduleToStart,
      scheduleToClose ~> admissionScheduleToClose
    )
  }

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

val five: Limits = Limits("five", steps = 5, actions = 5, search = 65536)
val seven: Limits = Limits("seven", steps = 7, actions = 7, search = 262144)
val eight: Limits = Limits("eight", steps = 8, actions = 8, search = 262144)

/** Past the depth of the detailed queue's table and of a design composed with it, which is ten. */
val twelve: Limits = Limits("twelve", steps = 12, actions = 12, search = 262144)

/** Every claim and path, declared on one design: a Property or Scenario belongs to one machine. */
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]): Vector[Query] =
  val notPaused = m.property("notAdmittedWhilePaused") holdsAcross notAdmittedWhilePaused
  val oneActive = m.property("atMostOneActive") holds atMostOneActive
  val terminal = m.property("terminalStays") holdsAcross terminalStays
  val startDeadline = m.property("scheduleToStartTimesOut") when scheduleToStart holds (after =>
    after.facts.contains(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
  )
  val closeDeadline = m.property("scheduleToCloseTimesOut") when scheduleToClose holds (after =>
    after.facts.contains(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
  )
  val stale = m
    .scenario("staleDeliveryAfterPause")
    .starts(scheduledIdle)
    .actions(dispatch, control(Control.pause), attemptStart)
  val prePause = m
    .scenario("admittedBeforePause")
    .starts(scheduledIdle)
    .actions(dispatch, attemptStart, control(Control.pause))
  val duplicate = m
    .scenario("duplicateDelivery")
    .starts(scheduledIdle)
    .actions(dispatch, attemptStart, attemptStart)
  val reopened = m
    .scenario("startedAfterCompletion")
    .starts(scheduledIdle)
    .actions(dispatch, attemptStart, attemptResult(AttemptResult.completed), attemptStart)
  val startFirst =
    m.scenario("scheduleToStartFirst").starts(scheduledIdle).actions(dispatch, scheduleToStart)
  val closeFirst =
    m.scenario("scheduleToCloseFirst").starts(scheduledIdle).actions(dispatch, scheduleToClose)
  val any = m.scenario("any").starts(scheduledIdle).free
  Vector(
    query(s"${m.name}.staleDelivery") verify notPaused in stale limits three,
    query(s"${m.name}.admittedBeforePause") verify notPaused in prePause limits three,
    query(s"${m.name}.duplicateDelivery") verify oneActive in duplicate limits three,
    // Each asks a Property its path keeps, so a violation is the watching monitor's alone.
    query(s"${m.name}.duplicateDelivery.monitored") verify notPaused in duplicate limits three,
    query(s"${m.name}.startedAfterCompletion.monitored") verify oneActive in reopened limits four,
    query(s"${m.name}.any.notAdmittedWhilePaused") verify notPaused in any limits five,
    query(s"${m.name}.any.atMostOneActive") verify oneActive in any limits five,
    query(s"${m.name}.any.terminalStays") verify terminal in any limits five,
    // Neither deadline is ordered before the other: each firing is a trace of its own.
    query(s"${m.name}.scheduleToStartFirst") find startDeadline in startFirst limits three,
    query(s"${m.name}.scheduleToCloseFirst") find closeDeadline in closeFirst limits three,
    // The product's own Property, read through the design's declared refinement.
    query(s"${m.name}.product.pausedIsNotDispatched")
      .verify(pausedIsNotDispatched)
      .in(stale)(using Reads.through(m, activityProduct)) limits three
  )

val currentQueries: Vector[Query] = admissionQueries(currentAdmission)
val staleQueries: Vector[Query] = admissionQueries(staleAdmission)

// ### The competing deadlines of the protocol machine
//
// With both deadlines set and no attempt started, each may fire first, and which did is what the
// status records.

/** The schedule-to-close deadline times the activity out and the status records which it was. */
val scheduleToCloseFires: Property[ProtocolState] =
  activityProtocol.property("scheduleToCloseFires") when scheduleToClose holds { s =>
    s.state.phase == Phase.timedOut && s.facts.contains(
      ProtocolFact.statusTimedOut(TimeoutType.scheduleToClose)
    )
  }

val bothDeadlinesStartFirst: Scenario[ProtocolState] =
  activityProtocol
    .scenario("bothDeadlinesStartFirst")
    .starts(unstarted)
    .actions(start(Timeout.expires, Timeout.expires, Timeout.unset), scheduleToStart)

val bothDeadlinesCloseFirst: Scenario[ProtocolState] =
  activityProtocol
    .scenario("bothDeadlinesCloseFirst")
    .starts(unstarted)
    .actions(start(Timeout.expires, Timeout.expires, Timeout.unset), scheduleToClose)

val competingTimers: Vector[Query] = Vector(
  query("competingTimers.scheduleToStartFirst")
    .find(scheduleToStartFires)
    .in(bothDeadlinesStartFirst) limits three,
  query("competingTimers.scheduleToCloseFirst")
    .find(scheduleToCloseFires)
    .in(bothDeadlinesCloseFirst) limits three
)

// ### The dispatch queue's interface
//
// What the activity may rely on of the durable queue between history and a worker's poll: an enqueue
// commits or fails; a committed message is delivered up to twice before its acknowledgment; and no
// committed message is lost. A crash shows at this interface only as that second delivery.

val fault: Party = Party("fault")

val enqueue = internal("enqueue")
val deliver = internal("deliver")
val acknowledge = internal("acknowledge")

/** The one message the queue may hold, by how far its delivery got. */
enum Outstanding derives Finite:
  case empty, committed, deliveredOnce, deliveredTwice

final case class QueueView(outstanding: Outstanding) derives Finite

/** `internal` answers every step of a provider that its interface does not show. */
enum QueueOutcome derives Finite:
  case committed, failed, delivered, acknowledged, lost, internal

/** The interface's events, and what a provider records of the steps behind them. */
enum QueueFact derives Finite:
  case enqueueCommitted, enqueueFailed, delivered, acknowledged, storageLost
  case addInvoked, taskPersisted, matchReserved, crashed, ackLost

type QueueViewStep = Step[QueueView, QueueOutcome, QueueFact]

private def views(o: QueueOutcome, to: Outstanding, recorded: QueueFact): List[QueueViewStep] =
  List(Step(o, QueueView(to), List(recorded)))

def enqueueView(q: QueueView): List[QueueViewStep] =
  if q.outstanding != Outstanding.empty then Nil
  else
    List(
      Step(
        QueueOutcome.committed,
        QueueView(Outstanding.committed),
        List(QueueFact.enqueueCommitted)
      ),
      Step(
        QueueOutcome.failed,
        q,
        List(QueueFact.enqueueFailed),
        "the durable write fails and no message is outstanding"
      )
    )

def deliverView(q: QueueView): List[QueueViewStep] = q.outstanding match
  case Outstanding.committed =>
    views(QueueOutcome.delivered, Outstanding.deliveredOnce, QueueFact.delivered)
  case Outstanding.deliveredOnce =>
    List(
      Step(
        QueueOutcome.delivered,
        QueueView(Outstanding.deliveredTwice),
        List(QueueFact.delivered),
        "a message not yet acknowledged may be delivered again"
      )
    )
  case Outstanding.empty | Outstanding.deliveredTwice => Nil

def acknowledgeView(q: QueueView): List[QueueViewStep] = q.outstanding match
  case Outstanding.deliveredOnce | Outstanding.deliveredTwice =>
    views(QueueOutcome.acknowledged, Outstanding.empty, QueueFact.acknowledged)
  case Outstanding.empty | Outstanding.committed => Nil

def storageLossView(q: QueueView): List[QueueViewStep] =
  if q.outstanding == Outstanding.empty then Nil
  else views(QueueOutcome.lost, Outstanding.empty, QueueFact.storageLost)

def queueEvidence(f: QueueFact): String = f match
  case QueueFact.enqueueCommitted => "enqueueCommitted"
  case QueueFact.enqueueFailed    => "enqueueFailed"
  case QueueFact.delivered        => "delivered"
  case QueueFact.acknowledged     => "acknowledged"
  case QueueFact.storageLost      => "storageLost"
  case QueueFact.addInvoked       => "addInvoked"
  case QueueFact.taskPersisted    => "taskPersisted"
  case QueueFact.matchReserved    => "matchReserved"
  case QueueFact.crashed          => "crashed"
  case QueueFact.ackLost          => "ackLost"

/** A check over the opaque queue rests on the interface alone. */
val queueOpaque: Assumption = assume("dispatchQueue.opaque")

/**
 * Committed storage may be lost. It is a fault of its own, apart from a crash, and only a machine
 * that assumes it has the step.
 */
val storageLossAssumed: Assumption = assume("storageLoss")
val storageLoss = action("storageLoss", fault)

val emptyQueue: QueueView = QueueView(Outstanding.empty)

/** The opaque provider. */
val dispatchQueue: Machine[QueueView, QueueOutcome, QueueFact] =
  machine[QueueView, QueueOutcome, QueueFact](SystemFamily, "dispatchQueue") {
    assumes(queueOpaque)
    starts(emptyQueue)
    ends(q => q.outstanding == Outstanding.empty)
    evidence(queueEvidence)
    steps(enqueue ~> enqueueView, deliver ~> deliverView, acknowledge ~> acknowledgeView)
  }

/** The interface under the storage-loss assumption: a committed message may also vanish. */
val dispatchQueueUnderStorageLoss: Machine[QueueView, QueueOutcome, QueueFact] =
  machine[QueueView, QueueOutcome, QueueFact](SystemFamily, "dispatchQueueUnderStorageLoss") {
    assumes(queueOpaque, storageLossAssumed)
    starts(emptyQueue)
    ends(q => q.outstanding == Outstanding.empty)
    evidence(queueEvidence)
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

val addActivityTask = internal("addActivityTask")
val persistTask = internal("persistTask")
val syncMatch = internal("syncMatch")
val crash = action("crash", fault)
val ackLoss = action("ackLoss", fault)

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

/** A step of a provider that its interface does not show. */
private def behind(to: QueueDetail, recorded: QueueFact): List[QueueDetailStep] =
  List(Step(QueueOutcome.internal, to, List(recorded)))

def enqueueDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.nowhere then Nil
  else
    List(
      Step(
        QueueOutcome.committed,
        QueueDetail(Custody.history, false, Delivered.never),
        List(QueueFact.enqueueCommitted)
      ),
      Step(
        QueueOutcome.failed,
        d,
        List(QueueFact.enqueueFailed),
        "the durable write fails and no message is outstanding"
      )
    )

/** An invocation implies no receiver effect: nothing durable changes until matching persists. */
def invokeDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.history then Nil
  else behind(d.copy(custody = Custody.invoked), QueueFact.addInvoked)

def persistDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.invoked then Nil
  else behind(d.copy(custody = Custody.persisted), QueueFact.taskPersisted)

def reserveDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody != Custody.invoked then Nil
  else behind(d.copy(custody = Custody.reserved), QueueFact.matchReserved)

def oneMoreDelivery(d: Delivered): Delivered = d match
  case Delivered.never => Delivered.once
  case _               => Delivered.twice

def deliverDetail(d: QueueDetail): List[QueueDetailStep] =
  if (d.custody == Custody.reserved || d.custody == Custody.persisted) && !d.polled &&
    d.delivered != Delivered.twice
  then
    List(
      Step(
        QueueOutcome.delivered,
        d.copy(polled = true, delivered = oneMoreDelivery(d.delivered)),
        List(QueueFact.delivered)
      )
    )
  else Nil

/** Matching completes the task, which discharges every custodian's obligation. */
def acknowledgeDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.polled && d.custody != Custody.nowhere && d.delivered != Delivered.never then
    List(Step(QueueOutcome.acknowledged, idleQueue, List(QueueFact.acknowledged)))
  else Nil

/**
 * The answer to the poller is lost. A persisted task stays queued; a sync match fails back to the
 * invocation, which history retries.
 */
def ackLossDetail(d: QueueDetail): List[QueueDetailStep] =
  if !d.polled then Nil
  else if d.custody == Custody.reserved then
    behind(d.copy(custody = Custody.invoked, polled = false), QueueFact.ackLost)
  else behind(d.copy(polled = false), QueueFact.ackLost)

/**
 * An ordinary crash loses what is only in memory, the poll, the invocation and a sync match, and
 * nothing durable: history still holds its dispatch task and retries, and a persisted task is still
 * queued.
 */
def crashDetail(d: QueueDetail): List[QueueDetailStep] = d.custody match
  case Custody.invoked | Custody.reserved =>
    behind(d.copy(custody = Custody.history, polled = false), QueueFact.crashed)
  case Custody.nowhere | Custody.history | Custody.persisted =>
    behind(d.copy(polled = false), QueueFact.crashed)

def storageLossDetail(d: QueueDetail): List[QueueDetailStep] =
  if d.custody == Custody.nowhere then Nil
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
val matchingQueue: Machine[QueueDetail, QueueOutcome, QueueFact] =
  machine[QueueDetail, QueueOutcome, QueueFact](SystemFamily, "matchingQueue") {
    refines(dispatchQueue)(viewOf)
    visible(interfaceSees)
    visibleOutcomes(interfaceAnswers)
    starts(idleQueue)
    ends(queueEnds)
    evidence(queueEvidence)
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
val lossyMatchingQueue: Machine[QueueDetail, QueueOutcome, QueueFact] =
  machine[QueueDetail, QueueOutcome, QueueFact](SystemFamily, "lossyMatchingQueue") {
    assumes(storageLossAssumed)
    refines(dispatchQueueUnderStorageLoss)(viewOf)
    visible(interfaceSees)
    visibleOutcomes(interfaceAnswers)
    starts(idleQueue)
    ends(queueEnds)
    evidence(queueEvidence)
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
  case Custody.invoked | Custody.reserved                    => behind(idleQueue, QueueFact.crashed)
  case Custody.nowhere | Custody.history | Custody.persisted =>
    behind(d.copy(polled = false), QueueFact.crashed)

val forgetfulQueue: Machine[QueueDetail, QueueOutcome, QueueFact] =
  machine[QueueDetail, QueueOutcome, QueueFact](SystemFamily, "forgetfulQueue") {
    refines(dispatchQueue)(viewOf)
    visible(interfaceSees)
    visibleOutcomes(interfaceAnswers)
    starts(idleQueue)
    ends(queueEnds)
    evidence(queueEvidence)
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
  case Custody.persisted                  => behind(idleQueue, QueueFact.crashed)
  case Custody.invoked | Custody.reserved =>
    behind(d.copy(custody = Custody.history, polled = false), QueueFact.crashed)
  case Custody.nowhere | Custody.history => behind(d.copy(polled = false), QueueFact.crashed)

val volatileQueue: Machine[QueueDetail, QueueOutcome, QueueFact] =
  machine[QueueDetail, QueueOutcome, QueueFact](SystemFamily, "volatileQueue") {
    refines(dispatchQueue)(viewOf)
    visible(interfaceSees)
    visibleOutcomes(interfaceAnswers)
    starts(idleQueue)
    ends(queueEnds)
    evidence(queueEvidence)
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
  before.custody == Custody.nowhere || after.state.custody != Custody.nowhere ||
    after.facts.contains(QueueFact.acknowledged)

/**
 * One crash at each point of the route, and the delivery that must still follow it: after the
 * invocation, after the sync match, after persistence and after a delivery. After the
 * acknowledgment nothing is left to deliver.
 */
def providerQueries(m: Machine[QueueDetail, QueueOutcome, QueueFact]): Vector[Query] =
  val stays = m.property("committedStays") holdsAcross committedStays
  val delivers =
    m.property("delivers") when deliver holds (after => after.facts.contains(QueueFact.delivered))
  val afterInvocation = m
    .scenario("crashAfterInvocation")
    .starts(idleQueue)
    .actions(enqueue, addActivityTask, crash, addActivityTask, persistTask, deliver)
  val afterSyncMatch = m
    .scenario("crashAfterSyncMatch")
    .starts(idleQueue)
    .actions(enqueue, addActivityTask, syncMatch, crash, addActivityTask, syncMatch, deliver)
  val afterPersistence = m
    .scenario("crashAfterPersistence")
    .starts(idleQueue)
    .actions(enqueue, addActivityTask, persistTask, crash, deliver)
  val afterDelivery = m
    .scenario("crashAfterDelivery")
    .starts(idleQueue)
    .actions(enqueue, addActivityTask, persistTask, deliver, crash, deliver)
  val afterAcknowledgment = m
    .scenario("crashAfterAcknowledgment")
    .starts(idleQueue)
    .actions(enqueue, addActivityTask, persistTask, deliver, acknowledge, crash)
  val any = m.scenario("any").starts(idleQueue).free
  Vector(
    query(s"${m.name}.crashAfterInvocation") find delivers in afterInvocation limits seven,
    query(s"${m.name}.crashAfterSyncMatch") find delivers in afterSyncMatch limits seven,
    query(s"${m.name}.crashAfterPersistence") find delivers in afterPersistence limits seven,
    query(s"${m.name}.crashAfterDelivery") find delivers in afterDelivery limits seven,
    query(s"${m.name}.crashAfterAcknowledgment") verify stays in afterAcknowledgment limits seven,
    query(s"${m.name}.any.committedStays") verify stays in any limits twelve
  )

val matchingQueueQueries: Vector[Query] = providerQueries(matchingQueue)
val forgetfulQueueQueries: Vector[Query] = providerQueries(forgetfulQueue)
val volatileQueueQueries: Vector[Query] = providerQueries(volatileQueue)
val lossyMatchingQueueQueries: Vector[Query] = providerQueries(lossyMatchingQueue)

/** Storage loss drops a committed message, and the queue records that it did. */
val storageLossDrops: Property[QueueDetail] =
  lossyMatchingQueue.property("storageLossDrops") when storageLoss holds { after =>
    after.state.custody == Custody.nowhere && after.facts.contains(QueueFact.storageLost)
  }

val persistedThenLost: Scenario[QueueDetail] =
  lossyMatchingQueue
    .scenario("persistedThenLost")
    .starts(idleQueue)
    .actions(enqueue, addActivityTask, persistTask, storageLoss)

val storageLossQuery: Query =
  query("lossyMatchingQueue.storageLoss") find storageLossDrops in persistedThenLost limits seven

// ### The record as a member
//
// A composition's Queries are not answered while a member names monitors, so the members are the
// two designs again without them. They declare no refinement: the designs above are what refine the
// product.

val currentRecord: Machine[AdmissionState, Outcome, AdmissionFact] =
  machine[AdmissionState, Outcome, AdmissionFact](SystemFamily, "currentRecord") {
    forEntity(activity)
    starts(scheduledIdle)
    ends(admissionEnds)
    evidence(admissionEvidence)
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

val staleRecord: Machine[AdmissionState, Outcome, AdmissionFact] =
  machine[AdmissionState, Outcome, AdmissionFact](SystemFamily, "staleRecord") {
    forEntity(activity)
    starts(scheduledIdle)
    ends(admissionEnds)
    evidence(admissionEvidence)
    steps(
      dispatch ~> dispatchStep,
      control ~> pauseStep,
      attemptStart ~> admitStale,
      answerDelivery ~> answerStep,
      attemptResult ~> resultStep,
      scheduleToStart ~> admissionScheduleToStart,
      scheduleToClose ~> admissionScheduleToClose
    )
  }

// ### A design over the opaque queue
//
// The dispatch is the queue's enqueue, a delivery is admission's one input, and admission's answer
// is the acknowledgment. Split that way, a commit, its answer and the acknowledgment are three
// points a fault can fall between.

final case class OverQueue(activity: AdmissionState, queue: QueueView)

val idleOverQueue: OverQueue = OverQueue(scheduledIdle, emptyQueue)

val currentOverQueue: Composition[OverQueue] =
  compose[OverQueue](SystemFamily, "currentOverQueue")(
    "activity" -> currentRecord,
    "queue" -> dispatchQueue
  )
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .ends(s => admissionEnds(s.activity))

val staleOverQueue: Composition[OverQueue] =
  compose[OverQueue](SystemFamily, "staleOverQueue")(
    "activity" -> staleRecord,
    "queue" -> dispatchQueue
  )
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
    .starts(idleOverQueue)
    .actionKeys("dispatch", "activity_control-pause", "admit")
  val prePause = c
    .scenario("admittedBeforePause")
    .starts(idleOverQueue)
    .actionKeys("dispatch", "admit", "activity_control-pause")
  val duplicate =
    c.scenario("duplicateDelivery").starts(idleOverQueue).actionKeys("dispatch", "admit", "admit")
  val any = c.scenario("any").starts(idleOverQueue).free
  Vector(
    query(s"${c.name}.staleDelivery") verify notPaused in stale limits three,
    query(s"${c.name}.admittedBeforePause") verify notPaused in prePause limits three,
    query(s"${c.name}.duplicateDelivery") verify oneActive in duplicate limits three,
    query(s"${c.name}.failedCommit") verify failedCommit in duplicate limits three,
    query(s"${c.name}.any.notAdmittedWhilePaused") verify notPaused in any limits five,
    query(s"${c.name}.any.atMostOneActive") verify oneActive in any limits five,
    query(s"${c.name}.any.terminalStays") verify terminal in any limits five
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
  compose[OverMatching](SystemFamily, "currentOverMatching")(
    "activity" -> currentRecord,
    "queue" -> matchingQueue
  )
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

val staleOverMatching: Composition[OverMatching] =
  compose[OverMatching](SystemFamily, "staleOverMatching")(
    "activity" -> staleRecord,
    "queue" -> matchingQueue
  )
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

/** The corrected design over each violating provider: the replacement is what must fail. */
val currentOverForgetful: Composition[OverMatching] =
  compose[OverMatching](SystemFamily, "currentOverForgetful")(
    "activity" -> currentRecord,
    "queue" -> forgetfulQueue
  )
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

val currentOverVolatile: Composition[OverMatching] =
  compose[OverMatching](SystemFamily, "currentOverVolatile")(
    "activity" -> currentRecord,
    "queue" -> volatileQueue
  )
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueue)
    .ends(s => admissionEnds(s.activity))

/** The corrected design where storage loss is assumed, over the interface that allows it. */
val currentOverLossyMatching: Composition[OverMatching] =
  compose[OverMatching](SystemFamily, "currentOverLossyMatching")(
    "activity" -> currentRecord,
    "queue" -> lossyMatchingQueue
  )
    .sync("dispatch", "activity" -> dispatch, "queue" -> enqueue)
    .sync("admit", "activity" -> attemptStart, "queue" -> deliver)
    .sync("settle", "activity" -> answerDelivery, "queue" -> acknowledge)
    .replaces("queue", dispatchQueueUnderStorageLoss)
    .ends(s => admissionEnds(s.activity))

def overMatchingQueries(c: Composition[OverMatching]): Vector[Query] =
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
    .starts(idleOverMatching)
    .actionKeys(
      "dispatch",
      "queue_addActivityTask",
      "queue_persistTask",
      "activity_control-pause",
      "admit"
    )
  val prePause = c
    .scenario("admittedBeforePause")
    .starts(idleOverMatching)
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
    .starts(idleOverMatching)
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
    .starts(idleOverMatching)
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
  val any = c.scenario("any").starts(idleOverMatching).free
  Vector(
    query(s"${c.name}.staleDelivery") verify notPaused in stale limits five,
    query(s"${c.name}.admittedBeforePause") verify notPaused in prePause limits five,
    query(s"${c.name}.deliveredAgainAfterLostAck") verify oneActive in lostAck limits seven,
    query(s"${c.name}.crashAfterAdmissionCommit") verify oneActive in crashAfterCommit limits eight,
    query(s"${c.name}.any.notAdmittedWhilePaused") verify notPaused in any limits twelve,
    query(s"${c.name}.any.atMostOneActive") verify oneActive in any limits twelve,
    query(s"${c.name}.any.terminalStays") verify terminal in any limits twelve
  )

val currentOverMatchingQueries: Vector[Query] = overMatchingQueries(currentOverMatching)
val staleOverMatchingQueries: Vector[Query] = overMatchingQueries(staleOverMatching)
val currentOverLossyMatchingQueries: Vector[Query] = overMatchingQueries(currentOverLossyMatching)
