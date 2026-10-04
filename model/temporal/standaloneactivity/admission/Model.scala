/* Admission: history's authoritative record of one activity, and how admission at
 * RecordActivityTaskStarted keeps the product's promise that a paused activity is dispatched to no
 * worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
 * chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/specimens/activity.md.
 *
 * Three identities stay apart. The logical activity is the entity. An attempt is what admission
 * commits, counted in the record. A delivery is one dispatch message, which the queue holds and may
 * hand out more than once. The record and the queue are separate machines, so a composition is what
 * checks them together (compositions/). This file declares the record, its monitors, the two
 * admission designs and the two machines a server's Run is checked against.
 *
 * The stale design is a deliberately faulty control, not a claim about a known server defect.
 */
package temporal
package standaloneactivity
package admission

import umpire.*
import SystemFamily.given

// The record was first written in the standalone activity's System.scala. Its declarations keep the
// Definition IDs and type names they had there, so its tables, IDs and answers are those the system
// contract was checked with.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

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

val admissionCommits = choice
val admissionCommitFails = choice

/**
 * The record's vocabulary: the status sets its promises are declared over, which a composition reads
 * through its `activity` member, and the step functions of its designs, named after the actions they
 * answer.
 */
object Admission:
  /** Paused before any attempt was admitted: the pause a delivery must not get past. */
  def paused(s: AdmissionState): Boolean = s.phase == AdmissionPhase.paused
  def running(s: AdmissionState): Boolean = s.phase == AdmissionPhase.started
  def terminal(s: AdmissionState): Boolean =
    s.phase.in(AdmissionPhase.completed, AdmissionPhase.timedOut)
  def twoActive(s: AdmissionState): Boolean = s.active == Active.two
  def phase(s: AdmissionState): AdmissionPhase = s.phase

  /** No unpause is in scope, so a pause is where a path may end, as a completion is. */
  def ends(s: AdmissionState): Boolean =
    !s.phase.in(AdmissionPhase.scheduled, AdmissionPhase.started)

  def oneMore(a: Active): Active = a match
    case Active.none => Active.one
    case _           => Active.two

  def oneLess(a: Active): Active = a match
    case Active.two => Active.one
    case _          => Active.none

  /** The record once an attempt is admitted: started, one more attempt active, an answer owed. */
  def admit(s: AdmissionState): AdmissionState =
    s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed)

  /** The dispatch task's Validate: it is sent only while the activity can start. */
  def dispatch(s: AdmissionState): List[AdmissionStep] =
    if s.phase != AdmissionPhase.scheduled then disabled
    else accept(s, AdmissionFact.dispatchSent)

  /** A pause keeps whatever message is in flight: nothing recalls it. */
  def pause(s: AdmissionState, c: Control): List[AdmissionStep] = c match
    case Control.pause =>
      s.phase match
        case AdmissionPhase.scheduled =>
          accept(s.copy(phase = AdmissionPhase.paused), AdmissionFact.statusPaused)
        case AdmissionPhase.started =>
          accept(s.copy(phase = AdmissionPhase.pausedWhileHeld), AdmissionFact.statusPaused)
        case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => disabled // already paused
        case AdmissionPhase.completed | AdmissionPhase.timedOut     => disabled // over
    case Control.unpause | Control.requestCancel | Control.terminate => disabled // out of scope

  /**
   * An admission, which either commits or fails its durable update. A failed commit records no
   * attempt and owes no answer, so the delivery it came by stays outstanding.
   */
  def admitted(s: AdmissionState): List[AdmissionStep] = choose(
    admissionCommits -> accept(
      admit(s),
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

  /** Admission as the corrected design decides it, with no failure of its durable update. */
  def admitHeld(s: AdmissionState): List[AdmissionStep] =
    if s.phase == AdmissionPhase.scheduled then
      accept(admit(s), AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted)
    else accept(s.copy(answer = Answer.owed), AdmissionFact.admissionRejected)

  def answerDelivery(s: AdmissionState): List[AdmissionStep] =
    if s.answer != Answer.owed then disabled
    else accept(s.copy(answer = Answer.settled), AdmissionFact.deliveryAnswered)

  def attemptResult(s: AdmissionState, r: AttemptResult): List[AdmissionStep] = r match
    case AttemptResult.completed =>
      if s.phase != AdmissionPhase.started then disabled
      else
        accept(
          s.copy(phase = AdmissionPhase.completed, active = oneLess(s.active)),
          AdmissionFact.statusCompleted
        )
    case AttemptResult.failed(_) | AttemptResult.canceled => disabled

  /** Covers the wait for a worker, so it fires only before an attempt is admitted. */
  def scheduleToStart(s: AdmissionState): List[AdmissionStep] =
    if s.phase == AdmissionPhase.scheduled then
      accept(
        s.copy(phase = AdmissionPhase.timedOut, active = Active.none),
        AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart)
      )
    else disabled

  /** Covers the whole activity, so it competes with the other deadline while none has fired. */
  def scheduleToClose(s: AdmissionState): List[AdmissionStep] =
    if terminal(s) then disabled
    else
      accept(
        s.copy(phase = AdmissionPhase.timedOut, active = Active.none),
        AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose)
      )

  def productOf(s: AdmissionState): ProductState = s.phase match
    case AdmissionPhase.scheduled => ProductState(ProductPhase.scheduled)
    case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => ProductState(ProductPhase.paused)
    case AdmissionPhase.started   => ProductState(ProductPhase.started)
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

  /** The active attempts after a step, counted from what it records. */
  def countActive(active: Active, after: AdmissionStep): Active =
    if after.records(AdmissionFact.attemptAdmitted) then oneMore(active)
    else if after.records(AdmissionFact.statusCompleted) then oneLess(active)
    else if after.state.phase == AdmissionPhase.timedOut then Active.none
    else active

  /** Whether the activity is over after a step, and whether it left where it ended. */
  def finality(f: Finality, after: AdmissionStep): Finality = f match
    case Finality.reopened => Finality.reopened
    case _                 =>
      if terminal(after.state) then Finality.closed
      else if f == Finality.closed then Finality.reopened
      else Finality.open

// ### Monitors: at most one admitted active attempt, and terminal finality
//
// They count from what a step records, so they hold a design to the promise whatever its state says.

val atMostOneActiveAttempt =
  monitor[AdmissionState, Outcome, AdmissionFact, Active](Active.none)((active, _, after) =>
    Admission.countActive(active, after)
  )(_ == Active.two).readAfter(after => after.records(AdmissionFact.attemptAdmitted))

/** Whether the activity is over, and whether a step after that left where it ended. */
enum Finality derives Finite:
  case open, closed, reopened

val terminalFinality =
  monitor[AdmissionState, Outcome, AdmissionFact, Finality](Finality.open)((f, _, after) =>
    Admission.finality(f, after)
  )(_ == Finality.reopened)

// ### The two designs, under any delivery
//
// A design alone takes a delivery whenever one could arrive, whatever queue brings it: what holds of
// it holds over every queue, and what fails of it is confirmed over a queue (compositions/).

val currentAdmission = machine[AdmissionState, Outcome, AdmissionFact] {
  forEntity(activity)
  monitors(atMostOneActiveAttempt, terminalFinality)
  refines(activityProduct)(Admission.productOf)
  visible(Admission.productSees)
  starts(scheduledIdle)
  ends(Admission.ends)
  evidence { case AdmissionFact.statusTimedOut(_) => "statusTimedOut" }
  steps(
    dispatch ~> Admission.dispatch,
    control ~> Admission.pause,
    attemptStart ~> Admission.admitCurrent,
    answerDelivery ~> Admission.answerDelivery,
    attemptResult ~> Admission.attemptResult,
    scheduleToStart ~> Admission.scheduleToStart,
    scheduleToClose ~> Admission.scheduleToClose
  )
}

val staleAdmission = currentAdmission.rebind(attemptStart ~> Admission.admitStale)

// ### The held race, run against a server
//
// The stale message is held at the dispatch cut while the pause commits, then delivered to
// admission. The server is expected to follow the corrected design, so the race is declared on that
// design, in the scope a Run of it has: the start sets no schedule deadline, so none fires, and no
// fault is injected, so admission's durable update does not fail. The hold is what makes that scope
// true of a Run: no delivery reaches admission before the release, and after the pause the corrected
// design rejects whatever arrives. The stale design's violation is shown by its own verify Query,
// never by a Run.

val heldAdmission = machine[AdmissionState, Outcome, AdmissionFact] {
  forEntity(activity)
  monitors(atMostOneActiveAttempt, terminalFinality)
  refines(activityProduct)(Admission.productOf)
  visible(Admission.productSees)
  starts(scheduledIdle)
  ends(Admission.ends)
  evidence { case AdmissionFact.statusTimedOut(_) => "statusTimedOut" }
  steps(
    dispatch ~> Admission.dispatch,
    control ~> Admission.pause,
    attemptStart ~> Admission.admitHeld,
    answerDelivery ~> Admission.answerDelivery
  )
}

// ### A lost admission response
//
// A bounded projection of admission and its lost answer. The one response-loss budget is consumed
// whether the update committed or failed: the caller's missing answer alone distinguishes neither.
// The in-process actuator supplies the durable decision separately and realizes the committed arm.

final case class AdmissionResponseState(record: AdmissionState, lossAvailable: Boolean)
    derives Finite

enum AdmissionResponseFact derives Finite:
  case dispatchSent, attemptAdmitted

type ResponseLossStep = Step[AdmissionResponseState, Outcome, AdmissionResponseFact]

val responseLossInitial: AdmissionResponseState = AdmissionResponseState(scheduledIdle, true)

val committedThenLost = choice
val failedThenLost = choice

/** The response-loss machine's step functions, named after the actions they answer. */
object ResponseLoss:
  def dispatch(s: AdmissionResponseState): List[ResponseLossStep] =
    if !s.lossAvailable then disabled
    else accept(s, AdmissionResponseFact.dispatchSent)

  def ackLoss(s: AdmissionResponseState): List[ResponseLossStep] =
    if !s.lossAvailable then disabled
    else
      choose(
        committedThenLost -> accept(
          AdmissionResponseState(Admission.admit(s.record), false),
          AdmissionResponseFact.attemptAdmitted
        ),
        failedThenLost -> accept(s.copy(lossAvailable = false))
          .because("the durable update failed before its answer was lost")
      )

val admissionResponseLoss = machine[AdmissionResponseState, Outcome, AdmissionResponseFact] {
  forEntity(activity)
  starts(responseLossInitial)
  ends(s => !s.lossAvailable)
  steps(dispatch ~> ResponseLoss.dispatch, taskqueue.ackLoss ~> ResponseLoss.ackLoss)
}
