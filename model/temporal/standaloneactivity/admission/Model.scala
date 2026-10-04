/* Admission: history's authoritative record of one activity, and how admission at
 * RecordActivityTaskStarted keeps the product's promise that a paused activity is dispatched to no
 * worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
 * chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/specimens/activity.md.
 * Three identities stay apart. The logical activity is the entity. An attempt is what admission
 * commits, counted in the record. A delivery is one dispatch message, which the queue holds and may
 * hand out more than once. The record and the queue are separate machines, checked together by a
 * composition (compositions/). The stale design is a deliberately faulty control, not a claim about
 * a known server defect.
 */
package temporal
package standaloneactivity
package admission

import umpire.*
import SystemFamily.given

// First written in System.scala: it keeps the Definition IDs and type names it was checked with.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

// ### The record. Both deadlines are armed in every state, which lets them compete with a delivery.

/** `pausedWhileHeld` is a pause after admission: the attempt it holds was admitted before it. */
enum AdmissionPhase derives Finite:
  case scheduled, paused, pausedWhileHeld, started, completed, timedOut

/** Attempts admitted and not closed: an enum, not an `UpTo[2]`, as its cases are frozen keys. */
enum Active derives Finite:
  case none, one, two

/** Whether admission owes matching the answer that lets it complete the task. */
enum Answer derives Finite:
  case settled, owed

final case class AdmissionState(phase: AdmissionPhase, active: Active, answer: Answer)
    derives Finite

/** The statuses the product reads, by their product names, and admission's internal facts. */
enum AdmissionFact derives Finite:
  case statusStarted, statusPaused, statusCompleted
  case statusTimedOut(timeoutType: TimeoutType)
  case dispatchSent, attemptAdmitted, admissionRejected, admissionCommitFailed, deliveryAnswered

type AdmissionStep = Step[AdmissionState, Outcome, AdmissionFact]

val scheduledIdle = AdmissionState(AdmissionPhase.scheduled, Active.none, Answer.settled)
val admissionCommits = choice
val admissionCommitFails = choice

/** History's dispatch task: its Validate sends the message only while the activity can start. */
val dispatch = internal

/** Admission's answer reaches matching, which may then complete the task. */
val answerDelivery = internal

/** The record's status sets, which a composition reads through `activity`, and step functions. */
object Admission:
  import AdmissionFact.*

  /** Paused before any attempt was admitted: the pause a delivery must not get past. */
  def paused(s: AdmissionState) = s.phase == AdmissionPhase.paused
  def running(s: AdmissionState) = s.phase == AdmissionPhase.started
  def terminal(s: AdmissionState) =
    s.phase.in(AdmissionPhase.completed, AdmissionPhase.timedOut)
  def twoActive(s: AdmissionState) = s.active == Active.two
  def phase(s: AdmissionState) = s.phase

  /** No unpause is in scope, so a pause is where a path may end, as a completion is. */
  def ends(s: AdmissionState) =
    !s.phase.in(AdmissionPhase.scheduled, AdmissionPhase.started)

  def oneMore(a: Active) = if a == Active.none then Active.one else Active.two
  def oneLess(a: Active) = if a == Active.two then Active.one else Active.none

  def admit(s: AdmissionState) =
    s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed)

  def dispatch(s: AdmissionState) =
    if s.phase != AdmissionPhase.scheduled then disabled else accept(s, dispatchSent)

  /** A pause keeps whatever message is in flight: nothing recalls it. */
  def control(s: AdmissionState, c: Control) = c match
    case Control.pause =>
      s.phase match
        case AdmissionPhase.scheduled => accept(s.copy(phase = AdmissionPhase.paused), statusPaused)
        case AdmissionPhase.started   =>
          accept(s.copy(phase = AdmissionPhase.pausedWhileHeld), statusPaused)
        case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => disabled // already paused
        case AdmissionPhase.completed | AdmissionPhase.timedOut     => disabled // over
    case Control.unpause | Control.requestCancel | Control.terminate => disabled // out of scope

  /** Its durable update commits or fails; a failed commit owes no answer, so its delivery stays. */
  def admitted(s: AdmissionState) = choose(
    admissionCommits -> accept(admit(s), statusStarted, attemptAdmitted),
    admissionCommitFails -> accept(s, admissionCommitFailed)
      .because("the durable update fails: nothing is admitted and the message stays deliverable")
  )

  /**
   * The corrected design re-reads current eligibility: a delivery that meets a paused activity or
   * an admitted attempt is answered and admits nothing.
   */
  def admitCurrent(s: AdmissionState) =
    if s.phase == AdmissionPhase.scheduled then admitted(s)
    else accept(s.copy(answer = Answer.owed), admissionRejected)

  /** The deliberately faulty design: admission trusts the eligibility the message was sent with. */
  def admitStale(s: AdmissionState) = admitted(s)

  /** Admission as the corrected design decides it, with no failure of its durable update. */
  def admitHeld(s: AdmissionState) =
    if s.phase == AdmissionPhase.scheduled then accept(admit(s), statusStarted, attemptAdmitted)
    else accept(s.copy(answer = Answer.owed), admissionRejected)

  def answerDelivery(s: AdmissionState) =
    if s.answer != Answer.owed then disabled
    else accept(s.copy(answer = Answer.settled), deliveryAnswered)

  def attemptResult(s: AdmissionState, r: AttemptResult) = r match
    case AttemptResult.completed =>
      if s.phase != AdmissionPhase.started then disabled
      else
        accept(
          s.copy(phase = AdmissionPhase.completed, active = oneLess(s.active)),
          statusCompleted
        )
    case AttemptResult.failed(_) | AttemptResult.canceled => disabled

  /** Covers the wait for a worker, so it fires only before an attempt is admitted. */
  def scheduleToStart(s: AdmissionState) =
    if s.phase == AdmissionPhase.scheduled then timeOut(s, TimeoutType.scheduleToStart)
    else disabled

  /** Covers the whole activity, so it competes with the other deadline while none has fired. */
  def scheduleToClose(s: AdmissionState) =
    if terminal(s) then disabled else timeOut(s, TimeoutType.scheduleToClose)

  def timeOut(s: AdmissionState, t: TimeoutType) =
    accept(s.copy(phase = AdmissionPhase.timedOut, active = Active.none), statusTimedOut(t))

  def productOf(s: AdmissionState) = s.phase match
    case AdmissionPhase.scheduled => ProductState(ProductPhase.scheduled)
    case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => ProductState(ProductPhase.paused)
    case AdmissionPhase.started   => ProductState(ProductPhase.started)
    case AdmissionPhase.completed => ProductState(ProductPhase.completed)
    case AdmissionPhase.timedOut  => ProductState(ProductPhase.timedOut)

  /** A caller reads statuses and nothing of the dispatch, the admission or its answer. */
  def productSees(f: AdmissionFact) = f match
    case AdmissionFact.statusStarted | AdmissionFact.statusPaused | AdmissionFact.statusCompleted |
        AdmissionFact.statusTimedOut(_) =>
      true
    case AdmissionFact.dispatchSent | AdmissionFact.attemptAdmitted |
        AdmissionFact.admissionRejected | AdmissionFact.admissionCommitFailed |
        AdmissionFact.deliveryAnswered =>
      false

  /** The active attempts after a step, counted from what it records. */
  def countActive(active: Active, after: AdmissionStep) =
    if after.records(attemptAdmitted) then oneMore(active)
    else if after.records(statusCompleted) then oneLess(active)
    else if after.state.phase == AdmissionPhase.timedOut then Active.none
    else active

  /** Whether the activity is over after a step, and whether it left where it ended. */
  def finality(f: Finality, after: AdmissionStep) =
    if f == Finality.reopened then Finality.reopened
    else if terminal(after.state) then Finality.closed
    else if f == Finality.closed then Finality.reopened
    else Finality.open

// ### Monitors, which count from what a step records whatever the design's state says.

val atMostOneActiveAttempt =
  monitor[AdmissionState, Outcome, AdmissionFact, Active](Active.none)((active, _, after) =>
    Admission.countActive(active, after)
  )(_ == Active.two).readAfter(after => after.records(AdmissionFact.attemptAdmitted))

enum Finality derives Finite:
  case open, closed, reopened

val terminalFinality =
  monitor[AdmissionState, Outcome, AdmissionFact, Finality](Finality.open)((f, _, after) =>
    Admission.finality(f, after)
  )(_ == Finality.reopened)

// ### The two designs. A design alone takes a delivery whenever one could arrive: what holds of it
// holds over every queue, and what fails of it is confirmed over a queue (compositions/).

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
    control ~> Admission.control,
    attemptStart ~> Admission.admitCurrent,
    answerDelivery ~> Admission.answerDelivery,
    attemptResult ~> Admission.attemptResult,
    scheduleToStart ~> Admission.scheduleToStart,
    scheduleToClose ~> Admission.scheduleToClose
  )
}

val staleAdmission = currentAdmission.rebind(attemptStart ~> Admission.admitStale)

// ### The held race, run against a server. The stale message is held at the dispatch cut while the
// pause commits, then delivered. The race is declared on the corrected design the server is
// expected to follow, in the scope a Run has: no deadline is set and no fault injected, so none
// fires and the durable update does not fail. The hold makes that scope true of a Run: nothing
// reaches admission before the release, and after the pause the corrected design rejects it. The
// stale design's violation is shown by its own verify Query, never by a Run.

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
    control ~> Admission.control,
    attemptStart ~> Admission.admitHeld,
    answerDelivery ~> Admission.answerDelivery
  )
}

// ### A lost admission response. The one response-loss budget is consumed whether the update
// committed or failed: the caller's missing answer distinguishes neither. The in-process actuator
// supplies the durable decision and realizes the committed arm.

final case class AdmissionResponseState(record: AdmissionState, lossAvailable: Boolean)
    derives Finite

enum AdmissionResponseFact derives Finite:
  case dispatchSent, attemptAdmitted

type ResponseLossStep = Step[AdmissionResponseState, Outcome, AdmissionResponseFact]

val responseLossInitial = AdmissionResponseState(scheduledIdle, true)
val committedThenLost = choice
val failedThenLost = choice

object ResponseLoss:
  def dispatch(s: AdmissionResponseState) =
    if !s.lossAvailable then disabled else accept(s, AdmissionResponseFact.dispatchSent)

  def ackLoss(s: AdmissionResponseState) =
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
