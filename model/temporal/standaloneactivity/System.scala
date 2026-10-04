/* The standalone activity's system contract: how history's authoritative activity record, the durable
 * dispatch queue and admission at RecordActivityTaskStarted keep the product's promise that a paused
 * activity is dispatched to no worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
 * chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/specimens/activity.md.
 *
 * Three identities stay apart. The logical activity is the entity. An attempt is what admission
 * commits, counted in the record. A delivery is one dispatch message, which the queue holds and may
 * hand out more than once. The record and the queue are separate machines, so a composition is what
 * checks them together, first over the opaque queue and then over the detailed one that replaces it.
 * The queue, its providers and what they promise are the shared task-queue entity's, in
 * temporal/taskqueue; what this file adds is how the activity's dispatch, admission and answer
 * synchronize with it and what the activity promises across both.
 *
 * The stale design is a deliberately faulty control, not a claim about a known server defect.
 */
package temporal
package standaloneactivity

import umpire.*
import taskqueue.*
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

/** The record's status sets, which the promises below are declared over. */
object Admission:
  /** Paused before any attempt was admitted: the pause a delivery must not get past. */
  def paused(s: AdmissionState): Boolean = s.phase == AdmissionPhase.paused
  def running(s: AdmissionState): Boolean = s.phase == AdmissionPhase.started
  def terminal(s: AdmissionState): Boolean = admissionOver(s.phase)
  def twoActive(s: AdmissionState): Boolean = s.active == Active.two
  def phase(s: AdmissionState): AdmissionPhase = s.phase

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

// ### Promises, written once and declared on each design and on each composition with the record
//
// Each takes the states it speaks of as named predicates, so one definition serves the record and
// a composition, which reads the record through its `activity` member.

/** No step admits a paused activity: nothing moves it from paused straight to started. */
def notAdmittedWhilePaused[S](m: Declares[S])(
    paused: S => Boolean,
    running: S => Boolean
): Property[S] =
  m.property("notAdmittedWhilePaused").never(s => running(s.state)).from(paused)

/** No step leaves two admitted attempts active. */
def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
  m.property("atMostOneActive").never(s => twoActive(s.state))

/** Once the activity is over, no step changes its phase. */
def terminalStays[S, P](m: Declares[S])(terminal: S => Boolean, phase: S => P): Property[S] =
  m.property("terminalStays").once(terminal).keeps(phase)

val five = Limits(steps = 5, actions = 5, search = 65536)
val eight = Limits(steps = 8, actions = 8, search = 262144)

/** Every claim and path, declared on one design: a Property or Scenario belongs to one machine. */
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]): Vector[Query] =
  val notPaused = notAdmittedWhilePaused(m)(Admission.paused, Admission.running)
  val oneActive = atMostOneActive(m)(Admission.twoActive)
  val terminal = terminalStays(m)(Admission.terminal, Admission.phase)
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

/** The record's status sets, read through the composition's `activity` member. */
object OverQueue:
  def paused(s: OverQueue): Boolean = Admission.paused(s.activity)
  def running(s: OverQueue): Boolean = Admission.running(s.activity)
  def terminal(s: OverQueue): Boolean = Admission.terminal(s.activity)
  def twoActive(s: OverQueue): Boolean = Admission.twoActive(s.activity)
  def phase(s: OverQueue): AdmissionPhase = s.activity.phase

val currentOverQueue: Composition[OverQueue] =
  compose[OverQueue](_.activity -> currentRecord, _.queue -> dispatchQueue)
    .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .ends(s => admissionEnds(s.activity))

val staleOverQueue: Composition[OverQueue] =
  currentOverQueue.withMember(_.activity -> staleRecord)

def overQueueQueries(c: Composition[OverQueue]): Vector[Query] =
  val notPaused = notAdmittedWhilePaused(c)(OverQueue.paused, OverQueue.running)
  val oneActive = atMostOneActive(c)(OverQueue.twoActive)
  val terminal = terminalStays(c)(OverQueue.terminal, OverQueue.phase)
  // A failed commit admits nothing and leaves its message with the queue.
  val failedCommit = c.property("failedCommitKeepsTheMessage") holds (after =>
    after.records(_.activity, AdmissionFact.admissionCommitFailed) implies
      (after.state.queue.outstanding != Outstanding.empty &&
        !after.records(_.activity, AdmissionFact.attemptAdmitted))
  )
  val stale = c
    .scenario("staleDeliveryAfterPause")
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.activity, control(Control.pause)),
      c.synced(_.activity -> attemptStart)
    )
  val prePause = c
    .scenario("admittedBeforePause")
    .actions(
      c.synced(_.activity -> dispatch),
      c.synced(_.activity -> attemptStart),
      c.own(_.activity, control(Control.pause))
    )
  val duplicate = c
    .scenario("duplicateDelivery")
    .actions(
      c.synced(_.activity -> dispatch),
      c.synced(_.activity -> attemptStart),
      c.synced(_.activity -> attemptStart)
    )
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
// Each later design replaces one member of the first, and the interface it replaces is the one its
// new provider refines.

final case class OverMatching(activity: AdmissionState, queue: QueueDetail)

/** The record's status sets, read through the composition's `activity` member. */
object OverMatching:
  def paused(s: OverMatching): Boolean = Admission.paused(s.activity)
  def running(s: OverMatching): Boolean = Admission.running(s.activity)
  def terminal(s: OverMatching): Boolean = Admission.terminal(s.activity)
  def twoActive(s: OverMatching): Boolean = Admission.twoActive(s.activity)
  def phase(s: OverMatching): AdmissionPhase = s.activity.phase

val currentOverMatching: Composition[OverMatching] =
  compose[OverMatching](_.activity -> currentRecord, _.queue -> matchingQueue)
    .sync("dispatch", _.activity -> dispatch, _.queue -> enqueue)
    .sync("admit", _.activity -> attemptStart, _.queue -> deliver)
    .sync("settle", _.activity -> answerDelivery, _.queue -> acknowledge)
    .replaces(_.queue, dispatchQueue)
    .ends(s => admissionEnds(s.activity))

val staleOverMatching: Composition[OverMatching] =
  currentOverMatching.withMember(_.activity -> staleRecord)

/** The corrected design over each violating provider: the replacement is what must fail. */
val currentOverForgetful: Composition[OverMatching] =
  currentOverMatching.withMember(_.queue -> forgetfulQueue)

val currentOverVolatile: Composition[OverMatching] =
  currentOverMatching.withMember(_.queue -> volatileQueue)

/** The corrected design where storage loss is assumed, over the interface that allows it. */
val currentOverLossyMatching: Composition[OverMatching] =
  currentOverMatching.withMember(_.queue -> lossyMatchingQueue)

/** `anyTotal` is the static combination count of its free `any` Queries. */
def overMatchingQueries(c: Composition[OverMatching], anyTotal: Int): Vector[Query] =
  val notPaused = notAdmittedWhilePaused(c)(OverMatching.paused, OverMatching.running)
  val oneActive = atMostOneActive(c)(OverMatching.twoActive)
  val terminal = terminalStays(c)(OverMatching.terminal, OverMatching.phase)
  val stale = c
    .scenario("staleDeliveryAfterPause")
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, persistTask),
      c.own(_.activity, control(Control.pause)),
      c.synced(_.activity -> attemptStart)
    )
  val prePause = c
    .scenario("admittedBeforePause")
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, persistTask),
      c.synced(_.activity -> attemptStart),
      c.own(_.activity, control(Control.pause))
    )
  // The answer to the poller is lost after the commit, so the persisted task is handed out again.
  val lostAck = c
    .scenario("deliveredAgainAfterLostAck")
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, persistTask),
      c.synced(_.activity -> attemptStart),
      c.own(_.queue, ackLoss),
      c.synced(_.activity -> attemptStart)
    )
  // A crash after the admission commit and before the acknowledgment: history retries the sync match.
  val crashAfterCommit = c
    .scenario("crashAfterAdmissionCommit")
    .actions(
      c.synced(_.activity -> dispatch),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, syncMatch),
      c.synced(_.activity -> attemptStart),
      c.own(_.queue, crash),
      c.own(_.queue, addActivityTask),
      c.own(_.queue, syncMatch),
      c.synced(_.activity -> attemptStart)
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
