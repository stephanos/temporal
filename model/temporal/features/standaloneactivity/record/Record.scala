/* Admission: history's authoritative record of one activity, and how admission at
 * RecordActivityTaskStarted keeps the product's promise that a paused activity is dispatched to no
 * worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
 * chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/specimens/activity.md.
 * Three identities stay apart. The logical activity is the entity. An attempt is what admission
 * commits, counted in the record. A delivery is one dispatch message, which the queue holds and may
 * hand out more than once. The record and the queue are separate machines, checked together by a
 * composition (withTaskQueue/). The stale design is a deliberately faulty control, not a claim about
 * a known server defect.
 *
 * Read top to bottom: the types; the signature (history's internal steps and admission's choices);
 * then one object per design -- CurrentAdmission, the corrected design, whose status sets, effects
 * and monitors every design reads; StaleAdmission, the deliberately faulty design derived from it;
 * HeldAdmission, the held race a server is run through; AdmissionResponseLoss, a lost admission
 * response. Each reads its header, then its sections in order: states, refinement, effects,
 * monitors, rules, properties, implements and queries.
 */
package temporal
package features.standaloneactivity
package record

import umpire.*
import umpire.realize.{Cleanup, Conformance, Disposition, MonitorExpectation, PropertyOutcome}
import umpire.realize.{Reason, RunExpectation}
import temporal.capabilities.{given, *}
import temporal.realize.satisfied
import shared.Bounds.{four, three}
import SystemFamily.given

// First written in System.scala: it keeps the Definition IDs and type names it was checked with.
given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")

// ### Types. Both deadlines are armed in every state, which lets them compete with a delivery.

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

/** Whether the activity is over, and whether it left where it ended: what `terminalFinality` counts. */
enum Finality derives Finite:
  case open, closed, reopened

/** The record with the one response-loss budget a lost admission response consumes. */
final case class AdmissionResponseState(record: AdmissionState, lossAvailable: Boolean)
    derives Finite

enum AdmissionResponseFact derives Finite:
  case dispatchSent, attemptAdmitted

type ResponseLossStep = Step[AdmissionResponseState, Outcome, AdmissionResponseFact]

/**
 * The claims every admission design is held to: the laws its capabilities bring, the record's own
 * count of active attempts, and each deadline timing the activity out with the status that says which.
 * `notPaused` is the generated `<design>.pausedIsNotDispatched`, which the pinned paths read.
 */
final case class AdmissionClaims(
    notPaused: Property[AdmissionState],
    oneActive: Property[AdmissionState],
    startDeadline: Property[AdmissionState],
    closeDeadline: Property[AdmissionState]
)

// ### Signature

val admissionCommits = choice
val admissionCommitFails = choice

/** History's internal steps. A section is transparent: each keeps the ID the file's pin gives. */
object history extends Section:
  /** History's dispatch task: its Validate sends the message only while the activity can start. */
  val dispatch = internal

  /** Admission's answer reaches matching, which may then complete the task. */
  val answerDelivery = internal

val committedThenLost = choice
val failedThenLost = choice

// ### The corrected design. A design alone takes a delivery whenever one could arrive: what holds of
// it holds over every queue, and what fails of it is confirmed over a queue (withTaskQueue/).

/** The corrected design, whose status sets a composition reads through `activity`. */
object CurrentAdmission extends Machine[AdmissionState, Outcome, AdmissionFact]:
  // Its monitors, first written in System.scala, keep the Definition IDs they were checked with.
  given DefinitionScope = DefinitionScope("temporal.standaloneactivity.System$package$")
  import AdmissionFact.*

  val entity = activity
  val init = AdmissionState(AdmissionPhase.scheduled, Active.none, Answer.settled)
  def end(s: State) = states.stopped(s)
  val evidence: PartialFunction[AdmissionFact, String] = { case AdmissionFact.statusTimedOut(_) =>
    "statusTimedOut"
  }

  /** The record's status sets, which a composition reads through `activity`, and its counts. */
  object states extends Section:
    /** Paused before any attempt was admitted: the pause a delivery must not get past. */
    def paused(s: State) = s.phase == AdmissionPhase.paused
    def running(s: State) = s.phase == AdmissionPhase.started
    def terminal(p: AdmissionPhase) = p.in(AdmissionPhase.completed, AdmissionPhase.timedOut)
    def twoActive(s: State) = s.active == Active.two
    def phase(s: State) = s.phase

    /**
     * No unpause is in scope, so a pause is where a path may end, as a completion is: `end` reads
     * this named predicate.
     */
    def stopped(s: State) =
      !s.phase.in(AdmissionPhase.scheduled, AdmissionPhase.started)

    def oneMore(a: Active) = if a == Active.none then Active.one else Active.two
    def oneLess(a: Active) = if a == Active.two then Active.one else Active.none

    /** The record once an attempt is admitted: started, one more active, its answer owed. */
    def admitted(s: State) =
      s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed)

    /** The active attempts after a step, counted from what it records. */
    def countActive(active: Active, after: AdmissionStep) =
      if after.records(attemptAdmitted) then oneMore(active)
      else if after.records(statusCompleted) then oneLess(active)
      else if after.state.phase == AdmissionPhase.timedOut then Active.none
      else active

    /** Whether the activity is over after a step, and whether it left where it ended. */
    def finality(f: Finality, after: AdmissionStep) =
      if f == Finality.reopened then Finality.reopened
      else if terminal(after.state.phase) then Finality.closed
      else if f == Finality.closed then Finality.reopened
      else Finality.open

    /**
     * Why the record waives closedIsRejectedUniformly: a delivery that reaches a closed record is
     * admission's to reject, and rejecting it records `admissionRejected` and owes matching the
     * answer, so the record steps where the law has it stay (chasm/lib/activity/activity.go
     * HandleStarted).
     */
    val deliveryAfterClose =
      "admission rejects a delivery to a closed record and owes matching its answer: activity.go HandleStarted"

  /** The record refines the product, which sees its statuses alone. */
  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: State): ProductState = s.phase match
      case AdmissionPhase.scheduled => ProductState(ProductPhase.scheduled)
      case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld =>
        ProductState(ProductPhase.paused)
      case AdmissionPhase.started   => ProductState(ProductPhase.started)
      case AdmissionPhase.completed => ProductState(ProductPhase.completed)
      case AdmissionPhase.timedOut  => ProductState(ProductPhase.timedOut)

    /** A caller reads statuses and nothing of the dispatch, the admission or its answer. */
    def visible(f: AdmissionFact) = f match
      case AdmissionFact.statusStarted | AdmissionFact.statusPaused |
          AdmissionFact.statusCompleted | AdmissionFact.statusTimedOut(_) =>
        true
      case AdmissionFact.dispatchSent | AdmissionFact.attemptAdmitted |
          AdmissionFact.admissionRejected | AdmissionFact.admissionCommitFailed |
          AdmissionFact.deliveryAnswered =>
        false

  object effects extends Section:
    def sendDispatch(s: State) = enter(s, dispatchSent)

    /** A pause keeps whatever message is in flight: nothing recalls it. */
    def pause(s: State) = enter(s.copy(phase = AdmissionPhase.paused), statusPaused)

    /** A pause after admission: the attempt it holds was admitted before it. */
    def pauseHeld(s: State) =
      enter(s.copy(phase = AdmissionPhase.pausedWhileHeld), statusPaused)

    /** Its durable update commits or fails; a failed commit owes no answer, so its delivery stays. */
    def admit(s: State) = choose(
      admissionCommits -> enter(states.admitted(s), statusStarted, attemptAdmitted),
      admissionCommitFails -> enter(s, admissionCommitFailed)
        .because("the durable update fails: nothing is admitted and the message stays deliverable")
    )

    /** A delivery that meets a paused activity or an admitted attempt is answered and admits nothing. */
    def reject(s: State) = enter(s.copy(answer = Answer.owed), admissionRejected)

    def answerDelivery(s: State) =
      enter(s.copy(answer = Answer.settled), deliveryAnswered)

    def complete(s: State) =
      enter(
        s.copy(phase = AdmissionPhase.completed, active = states.oneLess(s.active)),
        statusCompleted
      )

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = AdmissionPhase.timedOut, active = Active.none), statusTimedOut(t))

  // Monitors, which count from what a step records whatever the design's state says.
  object monitors extends Section:
    val atMostOneActiveAttempt =
      monitor[State, Outcome, AdmissionFact, Active](Active.none)((active, _, after) =>
        states.countActive(active, after)
      )(_ == Active.two).readAfter(after => after.records(AdmissionFact.attemptAdmitted))

    val terminalFinality =
      monitor[State, Outcome, AdmissionFact, Finality](Finality.open)((f, _, after) =>
        states.finality(f, after)
      )(_ == Finality.reopened)

  object rules extends Rules(_.phase):
    import AdmissionPhase.*

    in(scheduled)(history.dispatch ~> effects.sendDispatch)

    // A pause of a paused or closed record has no rule, and the other controls are out of scope.
    in(scheduled)(caller.control(Control.pause) ~> effects.pause)
    in(started)(caller.control(Control.pause) ~> effects.pauseHeld)

    // The corrected design re-reads current eligibility: only a scheduled activity admits.
    in(scheduled)(worker.attemptStart ~> effects.admit)
    in(paused, pausedWhileHeld, started, completed, timedOut) {
      worker.attemptStart ~> effects.reject
    }

    when(s => s.answer == Answer.owed)(history.answerDelivery ~> effects.answerDelivery)

    in(started)(worker.attemptResult(AttemptResult.completed) ~> effects.complete)

    // Schedule-to-start covers the wait for a worker, so it fires only before an attempt is
    // admitted; schedule-to-close covers the whole activity, so it competes with the other deadline
    // while none has fired.
    in(scheduled) {
      deadline.scheduleToStart ~> (s => effects.timeOut(s, TimeoutType.scheduleToStart))
    }
    in(scheduled, paused, pausedWhileHeld, started) {
      deadline.scheduleToClose ~> (s => effects.timeOut(s, TimeoutType.scheduleToClose))
    }

  // The system contract's promises are declared on each admission design and each composition with
  // the record: the laws its capabilities bring, and the record's own count of active attempts. Each
  // takes the states it speaks of as predicates, so one definition serves the record and a
  // composition, which reads the record through its `activity` member.
  object properties extends Section:
    /** No step leaves two admitted attempts active. */
    def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
      m.property("atMostOneActive").never(s => twoActive(s.state))

    def admissionClaims(m: Machine[State, Outcome, AdmissionFact]) =
      val declared = implements.admissionCapabilities(m)
      val scheduleToStartTimesOut = m.property when deadline.scheduleToStart holds (after =>
        after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
      )
      val scheduleToCloseTimesOut = m.property when deadline.scheduleToClose holds (after =>
        after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
      )
      AdmissionClaims(
        declared.claim(pausedIsNotDispatched),
        atMostOneActive(m)(states.twoActive),
        scheduleToStartTimesOut,
        scheduleToCloseTimesOut
      )

  /**
   * What an admission design is as the laws of model/temporal/capabilities read it, the record as a
   * machine of its own: it closes, it pauses before an attempt is admitted, and its work is handed out
   * by a worker's poll. Its free Queries run within `five`, as the system contract's did. It waives
   * closedIsRejectedUniformly, the one law the record does not keep.
   */
  object implements extends Section:
    def admissionCapabilities(m: Machine[State, Outcome, AdmissionFact]) =
      capabilities(m, limits = five)(
        Closable(
          status = states.phase,
          terminal = states.terminal,
          rejected = Outcome.notFound
        ),
        Pausable(
          pause = caller.control(Control.pause),
          unpause = caller.control(Control.unpause),
          paused = states.paused
        ),
        Pollable(dispatch = worker.attemptStart, running = states.running)
      ).except(closedIsRejectedUniformly, because = states.deliveryAfterClose)

  object queries extends Section:
    /** Every claim and path, declared on the design `m`, since each belongs to one machine. */
    def admissionQueries(m: Machine[State, Outcome, AdmissionFact]) =
      val claims = properties.admissionClaims(m)
      val staleDeliveryAfterPause =
        m.scenario.actions(history.dispatch, caller.control(Control.pause), worker.attemptStart)
      val admittedBeforePause =
        m.scenario.actions(history.dispatch, worker.attemptStart, caller.control(Control.pause))
      val duplicateDelivery =
        m.scenario.actions(history.dispatch, worker.attemptStart, worker.attemptStart)
      val startedAfterCompletion = m.scenario
        .actions(
          history.dispatch,
          worker.attemptStart,
          worker.attemptResult(AttemptResult.completed),
          worker.attemptStart
        )
      val scheduleToStartFirst = m.scenario.actions(history.dispatch, deadline.scheduleToStart)
      val scheduleToCloseFirst = m.scenario.actions(history.dispatch, deadline.scheduleToClose)
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.staleDelivery") verify claims.notPaused in
          staleDeliveryAfterPause limits three total 108,
        query(s"${m.name}.admittedBeforePause") verify claims.notPaused in
          admittedBeforePause limits three total 108,
        query(s"${m.name}.duplicateDelivery") verify claims.oneActive in
          duplicateDelivery limits three total 108,
        // Each asks a Property its path keeps, so a violation is the watching monitor's alone.
        query(s"${m.name}.duplicateDelivery.monitored") verify claims.notPaused in
          duplicateDelivery limits three total 108,
        query(s"${m.name}.startedAfterCompletion.monitored") verify claims.oneActive in
          startedAfterCompletion limits four total 144,
        query verify claims.oneActive in any limits five total 2340,
        // Neither deadline is ordered before the other: each firing is a trace of its own.
        query(s"${m.name}.scheduleToStartFirst") find claims.startDeadline in
          scheduleToStartFirst limits three total 72,
        query(s"${m.name}.scheduleToCloseFirst") find claims.closeDeadline in
          scheduleToCloseFirst limits three total 72,
        // The product's own Property, read through the design's declared refinement.
        query(s"${m.name}.product.pausedIsNotDispatched")
          .verify(ActivityProduct.implements.all.claim(pausedIsNotDispatched))
          .in(staleDeliveryAfterPause) limits three total 108
      )

    val currentQueries = admissionQueries(CurrentAdmission)

// ### The deliberately faulty design: admission trusts the eligibility the message was sent with,
// so every delivery is admitted as one that meets a scheduled activity is.

object StaleAdmission
    extends Derived(
      CurrentAdmission.rebind(when(_ => true) {
        worker.attemptStart ~> CurrentAdmission.effects.admit
      })
    ):
  object queries extends Section:
    val staleQueries = CurrentAdmission.queries.admissionQueries(StaleAdmission)

// ### The held race, run against a server. The stale message is held at the dispatch cut while the
// pause commits, then delivered. The race is declared on the corrected design the server is
// expected to follow, in the scope a Run has: no deadline is set and no fault injected, so none
// fires and the durable update does not fail. The hold makes that scope true of a Run: nothing
// reaches admission before the release, and after the pause the corrected design rejects it. The
// stale design's violation is shown by its own verify Query, never by a Run.

object HeldAdmission extends Machine[AdmissionState, Outcome, AdmissionFact]:
  val entity = activity
  val init = CurrentAdmission.init
  def end(s: State) = CurrentAdmission.end(s)
  val evidence: PartialFunction[AdmissionFact, String] = { case AdmissionFact.statusTimedOut(_) =>
    "statusTimedOut"
  }

  /** It refines the product as the corrected design does. */
  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: State): ProductState = CurrentAdmission.refinement.toProduct(s)
    def visible(f: AdmissionFact) = CurrentAdmission.refinement.visible(f)

  object effects extends Section:
    /** Admission as the corrected design decides it, with no failure of its durable update. */
    def admitCommitted(s: State) =
      enter(
        CurrentAdmission.states.admitted(s),
        AdmissionFact.statusStarted,
        AdmissionFact.attemptAdmitted
      )

  // The corrected design's monitors watch the race too.
  object monitors extends Section:
    val atMostOneActiveAttempt = CurrentAdmission.monitors.atMostOneActiveAttempt
    val terminalFinality = CurrentAdmission.monitors.terminalFinality

  object rules extends Rules(_.phase):
    import AdmissionPhase.*

    in(scheduled)(history.dispatch ~> CurrentAdmission.effects.sendDispatch)
    in(scheduled)(caller.control(Control.pause) ~> CurrentAdmission.effects.pause)
    in(started)(caller.control(Control.pause) ~> CurrentAdmission.effects.pauseHeld)
    in(scheduled)(worker.attemptStart ~> effects.admitCommitted)
    in(paused, pausedWhileHeld, started, completed, timedOut) {
      worker.attemptStart ~> CurrentAdmission.effects.reject
    }
    when(s => s.answer == Answer.owed) {
      history.answerDelivery ~> CurrentAdmission.effects.answerDelivery
    }

  // What the machine a server's Run is checked against promises.
  object properties extends Section:
    /** Admission met the stale message and rejected it. */
    val staleDeliveryRejected =
      property when worker.attemptStart holds (_.records(AdmissionFact.admissionRejected))

  // The held race, as a server's Run is checked.
  object queries extends Section:
    val heldStaleDelivery =
      (query("heldAdmission.staleDelivery") find properties.staleDeliveryRejected in
        scenario("heldStaleDelivery").actions(
          history.dispatch,
          caller.control(Control.pause),
          worker.attemptStart
        ) limits three total 108).expect(
        RunExpectation(
          Conformance.conformant,
          PropertyOutcome.satisfied,
          contract = PropertyOutcome.satisfied,
          disposition = Disposition.completed,
          cleanup = Cleanup.succeeded,
          monitors = Vector(
            MonitorExpectation(
              CurrentAdmission.monitors.atMostOneActiveAttempt,
              PropertyOutcome.inconclusive,
              Some(Reason.neverEvaluated)
            ),
            MonitorExpectation(
              CurrentAdmission.monitors.terminalFinality,
              PropertyOutcome.satisfied
            )
          )
        )
      )

// ### A lost admission response. The one response-loss budget is consumed whether the update
// committed or failed: the caller's missing answer distinguishes neither. The in-process actuator
// supplies the durable decision and realizes the committed arm.

object AdmissionResponseLoss
    extends Machine[AdmissionResponseState, Outcome, AdmissionResponseFact], FailureModel:
  val entity = activity
  val init = AdmissionResponseState(CurrentAdmission.init, true)
  def end(s: State) = !s.lossAvailable

  object effects extends Section:
    def sendDispatch(s: State) = enter(s, AdmissionResponseFact.dispatchSent)

    def loseResponse(s: State) = choose(
      committedThenLost -> enter(
        AdmissionResponseState(CurrentAdmission.states.admitted(s.record), false),
        AdmissionResponseFact.attemptAdmitted
      ),
      failedThenLost -> enter(s.copy(lossAvailable = false))
        .because("the durable update failed before its answer was lost")
    )

  // Both steps take the budget's state: once a loss consumes it, neither fires.
  object rules extends Rules:
    when(_.lossAvailable) {
      history.dispatch ~> effects.sendDispatch
      shared.taskqueue.faults.ackLoss ~> effects.loseResponse
    }

  object properties extends Section:
    /** A lost response still leaves the attempt admitted when the update committed. */
    val committedDespiteLostResponse =
      property when shared.taskqueue.faults.ackLoss holds (after =>
        after.records(AdmissionResponseFact.attemptAdmitted)
      )

  // The lost response, as a server's Run is checked.
  object queries extends Section:
    val lostAdmissionResponseQuery =
      (query("admissionResponseLoss.committed") find properties.committedDespiteLostResponse in
        scenario("oneLostResponse").actions(history.dispatch, shared.taskqueue.faults.ackLoss)
        limits three total 144)
        .expect(satisfied)
