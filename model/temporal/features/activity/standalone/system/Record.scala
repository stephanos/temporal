// Admission: history's authoritative record of one activity, and how admission at
// RecordActivityTaskStarted keeps the product's promise that a paused activity is dispatched to no
// worker. Grounded in chasm/lib/activity/tasks.go (the dispatch task) and
// chasm/lib/activity/activity.go (HandleStarted), and reviewed as model/specimens/activity.md.
// Three identities stay apart. The logical activity is the entity. An attempt is what admission
// commits, counted in the record. A delivery is one dispatch message, which the queue holds and may
// hand out more than once. The record and the queue are separate machines, checked together by a
// composition (WithTaskQueue.scala). The stale design is a deliberately faulty control, not a claim about
// a known server defect.
//
// Read top to bottom: the types; the signature (history's internal steps and admission's choices);
// then one object per design -- ActivityRecord, the corrected design, whose status sets, effects
// and monitors every design reads; TrustingActivityRecord, the deliberately faulty design derived from it;
// HeldDispatch, the held race a server is run through; LostStartAnswer, a lost admission
// response. Each reads its header, then its sections in order: states, refinement, effects,
// monitors, rules, properties, implements and queries.
package temporal
package features.activity.standalone
package system

import umpire.*
import umpire.realize.{Cleanup, Conformance, Disposition, MonitorExpectation, PropertyOutcome}
import umpire.realize.{Reason, RunExpectation}
import temporal.capabilities.{given, *}
import temporal.realize.satisfied
import shared.Bounds.{four, three}
import product.ActivityProduct

// ### Types. Both deadlines are armed in every state, which lets them compete with a delivery.

// `pausedWhileHeld` is a pause after admission: the attempt it holds was admitted before it.
enum AdmissionPhase derives Finite:
  case scheduled, paused, pausedWhileHeld, started, completed, timedOut

// Attempts admitted and not closed: an enum, not an `UpTo[2]`, as its cases are frozen keys.
enum Active derives Finite:
  case none, one, two

// Whether admission owes matching the answer that lets it complete the task.
enum Answer derives Finite:
  case settled, owed

final case class AdmissionState(phase: AdmissionPhase, active: Active, answer: Answer)
    derives Finite

// The statuses the product reads, by their product names, and admission's internal facts.
enum AdmissionFact derives Finite:
  case statusStarted, statusPaused, statusCompleted
  case statusTimedOut(timeoutType: TimeoutType)
  case dispatchSent, attemptAdmitted, admissionRejected, admissionCommitFailed, deliveryAnswered

// Whether the activity is over, and whether it left where it ended: what `terminalFinality` counts.
enum Finality derives Finite:
  case open, closed, reopened

// The record with the one response-loss budget a lost admission response consumes.
final case class AdmissionResponseState(record: AdmissionState, lossAvailable: Boolean)
    derives Finite

enum AdmissionResponseFact derives Finite:
  case dispatchSent, attemptAdmitted

// The claims every admission design is held to: the laws its capabilities bring, the record's own
// count of active attempts, and each deadline timing the activity out with the status that says which.
// `notPaused` is the generated `<design>.pausedIsNotDispatched`, which the pinned paths read.
final case class AdmissionClaims(
    notPaused: Property[AdmissionState],
    oneActive: Property[AdmissionState],
    startDeadline: Property[AdmissionState],
    closeDeadline: Property[AdmissionState]
)

// ### Signature

val admissionCommits = choice
val admissionCommitFails = choice

// History's internal steps.
object history:
  // History's dispatch task: its Validate sends the message only while the activity can start.
  val dispatch = internal

  // Admission's answer reaches matching, which may then complete the task.
  val answerMatching = internal

val committedThenLost = choice
val failedThenLost = choice

// ### The corrected design. A design alone takes a delivery whenever one could arrive: what holds of
// it holds over every queue, and what fails of it is confirmed over a queue (WithTaskQueue.scala).

// The corrected design, whose status sets a composition reads through `activity`.
object ActivityRecord extends Machine[AdmissionState, Outcome, AdmissionFact]:
  import AdmissionFact.*

  val init =
    AdmissionState(phase = AdmissionPhase.scheduled, active = Active.none, answer = Answer.settled)
  def end(s: State) = states.stopped(s)
  val evidence: PartialFunction[AdmissionFact, String] = { case AdmissionFact.statusTimedOut(_) =>
    "statusTimedOut"
  }

  // The record's status sets, which a composition reads through `activity`, and its counts.
  object states:
    // Paused before any attempt was admitted: the pause a delivery must not get past.
    def paused(s: State) = s.phase == AdmissionPhase.paused
    def running(s: State) = s.phase == AdmissionPhase.started
    def terminal(p: AdmissionPhase) = p.in(AdmissionPhase.completed, AdmissionPhase.timedOut)
    def twoActive(s: State) = s.active == Active.two
    def phase(s: State) = s.phase

    // No unpause is in scope, so a pause is where a path may end, as a completion is: `end` reads
    // this named predicate.
    def stopped(s: State) =
      !s.phase.in(AdmissionPhase.scheduled, AdmissionPhase.started)

    def oneMore(a: Active) = if a == Active.none then Active.one else Active.two
    def oneLess(a: Active) = if a == Active.two then Active.one else Active.none

    // The record once an attempt is admitted: started, one more active, its answer owed.
    def admitted(s: State) =
      s.copy(phase = AdmissionPhase.started, active = oneMore(s.active), answer = Answer.owed)

    // The active attempts after a step, counted from what it records.
    def countActive(active: Active, after: Step[State, Outcome, AdmissionFact]) =
      if after.records(attemptAdmitted) then oneMore(active)
      else if after.records(statusCompleted) then oneLess(active)
      else if after.state.phase == AdmissionPhase.timedOut then Active.none
      else active

    // Whether the activity is over after a step, and whether it left where it ended.
    def finality(f: Finality, after: Step[State, Outcome, AdmissionFact]) =
      if f == Finality.reopened then Finality.reopened
      else if terminal(after.state.phase) then Finality.closed
      else if f == Finality.closed then Finality.reopened
      else Finality.open

    // Why the record waives closedIsRejectedUniformly: a delivery that reaches a closed record is
    // admission's to reject, and rejecting it records `admissionRejected` and owes matching the
    // answer, so the record steps where the law has it stay (chasm/lib/activity/activity.go
    // HandleStarted).
    val deliveryAfterClose =
      "admission rejects a delivery to a closed record and owes matching its answer: activity.go HandleStarted"

  // The record refines the product, which sees its statuses alone.
  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: State): product.State = s.phase match
      case AdmissionPhase.scheduled => product.State(product.Phase.scheduled)
      case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld =>
        product.State(product.Phase.paused)
      case AdmissionPhase.started   => product.State(product.Phase.started)
      case AdmissionPhase.completed => product.State(product.Phase.completed)
      case AdmissionPhase.timedOut  => product.State(product.Phase.timedOut)

    // A client reads statuses and nothing of the dispatch, the admission or its answer.
    def visible(f: AdmissionFact) = f match
      case AdmissionFact.statusStarted | AdmissionFact.statusPaused |
          AdmissionFact.statusCompleted | AdmissionFact.statusTimedOut(_) =>
        true
      case AdmissionFact.dispatchSent | AdmissionFact.attemptAdmitted |
          AdmissionFact.admissionRejected | AdmissionFact.admissionCommitFailed |
          AdmissionFact.deliveryAnswered =>
        false

  object effects:
    def sendDispatch(s: State) = enter(s, dispatchSent)

    // A pause keeps whatever message is in flight: nothing recalls it.
    def pause(s: State) = enter(s.copy(phase = AdmissionPhase.paused), statusPaused)

    // A pause after admission: the attempt it holds was admitted before it.
    def pauseHeld(s: State) =
      enter(s.copy(phase = AdmissionPhase.pausedWhileHeld), statusPaused)

    // Its durable update commits or fails; a failed commit owes no answer, so its delivery stays.
    def admit(s: State) = choose(
      admissionCommits -> enter(states.admitted(s), statusStarted, attemptAdmitted),
      admissionCommitFails -> enter(s, admissionCommitFailed)
        .because("the durable update fails: nothing is admitted and the message stays deliverable")
    )

    // A delivery that meets a paused activity or an admitted attempt is answered and admits nothing.
    def rejectDelivery(s: State) = enter(s.copy(answer = Answer.owed), admissionRejected)

    def answerMatching(s: State) =
      enter(s.copy(answer = Answer.settled), deliveryAnswered)

    def complete(s: State) =
      enter(
        s.copy(phase = AdmissionPhase.completed, active = states.oneLess(s.active)),
        statusCompleted
      )

    def timeOut(s: State, t: TimeoutType) =
      enter(s.copy(phase = AdmissionPhase.timedOut, active = Active.none), statusTimedOut(t))

  // Monitors, which count from what a step records whatever the design's state says.
  object monitors:
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

    on(history.dispatch)(in(scheduled) ~> effects.sendDispatch)

    // A pause of a paused or closed record has no rule, and the other controls are out of scope.
    on(client.control(Control.pause)) {
      in(scheduled) ~> effects.pause
      in(started) ~> effects.pauseHeld
    }

    // The corrected design re-reads current eligibility: only a scheduled activity admits.
    on(worker.poll) {
      in(scheduled) ~> effects.admit
      in(paused, pausedWhileHeld, started, completed, timedOut) ~> effects.rejectDelivery
    }
    on(history.answerMatching)(where(_.answer == Answer.owed) ~> effects.answerMatching)
    on(worker.respond(AttemptResult.completed))(in(started) ~> effects.complete)

    // Schedule-to-start covers the wait for a worker, so it fires only before an attempt is
    // admitted; schedule-to-close covers the whole activity, so it competes with the other deadline
    // while none has fired.
    on(deadline.scheduleToStart)(in(scheduled) ~> (effects.timeOut(_, TimeoutType.scheduleToStart)))
    on(deadline.scheduleToClose) {
      in(scheduled, paused, pausedWhileHeld, started) ~> (effects.timeOut(
        _,
        TimeoutType.scheduleToClose
      ))
    }

  // The history record's promises are declared on each admission design and each composition with
  // the record: the laws its capabilities bring, and the record's own count of active attempts. Each
  // takes the states it speaks of as predicates, so one definition serves the record and a
  // composition, which reads the record through its `activity` member.
  object properties:
    // No step leaves two admitted attempts active.
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

  // What an admission design is as the laws of model/temporal/capabilities read it, the record as a
  // machine of its own: it closes, it pauses before an attempt is admitted, and its work is handed out
  // by a worker's poll. Its free Queries run within `five`, as the history record's did. It waives
  // closedIsRejectedUniformly, the one law the record does not keep.
  object implements:
    def admissionCapabilities(m: Machine[State, Outcome, AdmissionFact]) =
      capabilities(m, limits = five)(
        Closable(
          status = states.phase,
          terminal = states.terminal,
          rejected = Outcome.notFound
        ),
        Pausable(
          pause = client.control(Control.pause),
          unpause = client.control(Control.unpause),
          paused = states.paused
        ),
        Pollable(dispatch = worker.poll, running = states.running)
      ).except(closedIsRejectedUniformly, because = states.deliveryAfterClose)

  object queries:
    // The System's own deadlines, which the record's IR file carries: neither is ordered before
    // the other, so each firing is a trace of its own. Both deadlines are set and no attempt started,
    // so either may fire first.
    val bothDeadlinesStartFirst = ActivitySystem.scenario.actions(
      client.start(scheduleToClose := Timeout.expires, scheduleToStart := Timeout.expires),
      deadline.scheduleToStart
    )
    val bothDeadlinesCloseFirst = ActivitySystem.scenario.actions(
      client.start(scheduleToClose := Timeout.expires, scheduleToStart := Timeout.expires),
      deadline.scheduleToClose
    )

    // Every claim and path, declared on the design `m`, since each belongs to one machine.
    def admissionQueries(m: Machine[State, Outcome, AdmissionFact]) =
      val claims = properties.admissionClaims(m)
      val staleDeliveryAfterPause =
        m.scenario.actions(history.dispatch, client.control(Control.pause), worker.poll)
      val admittedBeforePause =
        m.scenario.actions(history.dispatch, worker.poll, client.control(Control.pause))
      val duplicateDelivery =
        m.scenario.actions(history.dispatch, worker.poll, worker.poll)
      val startedAfterCompletion = m.scenario
        .actions(
          history.dispatch,
          worker.poll,
          worker.respond(AttemptResult.completed),
          worker.poll
        )
      val scheduleToStartFirst = m.scenario.actions(history.dispatch, deadline.scheduleToStart)
      val scheduleToCloseFirst = m.scenario.actions(history.dispatch, deadline.scheduleToClose)
      val any = m.scenario.free
      Vector(
        query(s"${m.name}.staleDelivery") verify claims.notPaused in
          staleDeliveryAfterPause limits three,
        query(s"${m.name}.admittedBeforePause") verify claims.notPaused in
          admittedBeforePause limits three,
        query(s"${m.name}.duplicateDelivery") verify claims.oneActive in
          duplicateDelivery limits three,
        // Each asks a Property its path keeps, so a violation is the watching monitor's alone.
        query(s"${m.name}.duplicateDelivery.monitored") verify claims.notPaused in
          duplicateDelivery limits three,
        query(s"${m.name}.startedAfterCompletion.monitored") verify claims.oneActive in
          startedAfterCompletion limits four,
        query verify claims.oneActive in any limits five,
        // Neither deadline is ordered before the other: each firing is a trace of its own.
        query(s"${m.name}.scheduleToStartFirst") find claims.startDeadline in
          scheduleToStartFirst limits three,
        query(s"${m.name}.scheduleToCloseFirst") find claims.closeDeadline in
          scheduleToCloseFirst limits three,
        // The product's own Property, read through the design's declared refinement.
        query(s"${m.name}.product.pausedIsNotDispatched")
          .verify(ActivityProduct.implements.claim(pausedIsNotDispatched))
          .in(staleDeliveryAfterPause) limits three
      )

    val competingTimers = Vector(
      query("competingTimers.scheduleToStartFirst") find
        ActivitySystem.properties.scheduleToStartFires in
        bothDeadlinesStartFirst limits three,
      query("competingTimers.scheduleToCloseFirst") find
        ActivitySystem.properties.scheduleToCloseFires in
        bothDeadlinesCloseFirst limits three
    )

    val activityRecordQueries = admissionQueries(ActivityRecord)

// ### The deliberately faulty design: admission trusts the eligibility the message was sent with,
// so every delivery is admitted as one that meets a scheduled activity is.

object TrustingActivityRecord
    extends Derived(
      ActivityRecord.rebind(on(worker.poll)(always ~> ActivityRecord.effects.admit))
    ),
      NegativeControl:
  object queries:
    val trustingActivityRecordQueries =
      ActivityRecord.queries.admissionQueries(TrustingActivityRecord)

// ### The held race, run against a server. The stale message is held at the dispatch cut while the
// pause commits, then delivered. The race is declared on the corrected design the server is
// expected to follow, in the scope a Run has: no deadline is set and no fault injected, so none
// fires and the durable update does not fail. The hold makes that scope true of a Run: nothing
// reaches admission before the release, and after the pause the corrected design rejects it. The
// stale design's violation is shown by its own verify Query, never by a Run.

object HeldDispatch extends Machine[AdmissionState, Outcome, AdmissionFact]:
  val init = ActivityRecord.init
  def end(s: State) = ActivityRecord.end(s)
  val evidence: PartialFunction[AdmissionFact, String] = { case AdmissionFact.statusTimedOut(_) =>
    "statusTimedOut"
  }

  // It refines the product as the corrected design does.
  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: State): product.State = ActivityRecord.refinement.toProduct(s)
    def visible(f: AdmissionFact) = ActivityRecord.refinement.visible(f)

  object effects:
    // Admission as the corrected design decides it, with no failure of its durable update.
    def admitCommitted(s: State) =
      enter(
        ActivityRecord.states.admitted(s),
        AdmissionFact.statusStarted,
        AdmissionFact.attemptAdmitted
      )

  // The corrected design's monitors watch the race too.
  object monitors:
    val atMostOneActiveAttempt = ActivityRecord.monitors.atMostOneActiveAttempt
    val terminalFinality = ActivityRecord.monitors.terminalFinality

  object rules extends Rules(_.phase):
    import AdmissionPhase.*

    on(history.dispatch)(in(scheduled) ~> ActivityRecord.effects.sendDispatch)
    on(client.control(Control.pause)) {
      in(scheduled) ~> ActivityRecord.effects.pause
      in(started) ~> ActivityRecord.effects.pauseHeld
    }
    on(worker.poll) {
      in(scheduled) ~> effects.admitCommitted
      in(
        paused,
        pausedWhileHeld,
        started,
        completed,
        timedOut
      ) ~> ActivityRecord.effects.rejectDelivery
    }
    on(history.answerMatching) {
      where(_.answer == Answer.owed) ~> ActivityRecord.effects.answerMatching
    }

  // What the machine a server's Run is checked against promises.
  object properties:
    // Admission met the stale message and rejected it.
    val staleDeliveryRejected =
      property when worker.poll holds (_.records(AdmissionFact.admissionRejected))

  // The held race, as a server's Run is checked.
  object queries:
    val heldStaleDelivery = scenario.actions(
      history.dispatch,
      client.control(Control.pause),
      worker.poll
    )
    val heldStaleDeliveryQuery =
      (query("heldDispatch.staleDelivery") find properties.staleDeliveryRejected in
        heldStaleDelivery limits three).expect(
        RunExpectation(
          Conformance.conformant,
          PropertyOutcome.satisfied,
          contract = PropertyOutcome.satisfied,
          disposition = Disposition.completed,
          cleanup = Cleanup.succeeded,
          monitors = Vector(
            MonitorExpectation(
              ActivityRecord.monitors.atMostOneActiveAttempt,
              PropertyOutcome.inconclusive,
              Some(Reason.neverEvaluated)
            ),
            MonitorExpectation(
              ActivityRecord.monitors.terminalFinality,
              PropertyOutcome.satisfied
            )
          )
        )
      )

// ### A lost admission response. The one response-loss budget is consumed whether the update
// committed or failed: the client's missing answer distinguishes neither. The in-process actuator
// supplies the durable decision and realizes the committed arm.

object LostStartAnswer
    extends Machine[AdmissionResponseState, Outcome, AdmissionResponseFact],
      FailureModel:
  val entity = activity
  val init = AdmissionResponseState(record = ActivityRecord.init, lossAvailable = true)
  def end(s: State) = !s.lossAvailable

  object effects:
    def sendDispatch(s: State) = enter(s, AdmissionResponseFact.dispatchSent)

    def loseResponse(s: State) = choose(
      committedThenLost -> enter(
        AdmissionResponseState(
          record = ActivityRecord.states.admitted(s.record),
          lossAvailable = false
        ),
        AdmissionResponseFact.attemptAdmitted
      ),
      failedThenLost -> enter(s.copy(lossAvailable = false))
        .because("the durable update failed before its answer was lost")
    )

  // Both steps take the budget's state: once a loss consumes it, neither fires.
  object rules extends Rules:
    on(history.dispatch)(where(_.lossAvailable) ~> effects.sendDispatch)
    on(shared.taskqueue.fault.ackLoss)(where(_.lossAvailable) ~> effects.loseResponse)

  object properties:
    // A lost response still leaves the attempt admitted when the update committed.
    val committedDespiteLostResponse =
      property when shared.taskqueue.fault.ackLoss holds (after =>
        after.records(AdmissionResponseFact.attemptAdmitted)
      )

  // The lost response, as a server's Run is checked.
  object queries:
    val oneLostResponse = scenario.actions(history.dispatch, shared.taskqueue.fault.ackLoss)
    val lostAdmissionResponseQuery =
      (query("lostStartAnswer.committed") find properties.committedDespiteLostResponse in
        oneLostResponse limits three)
        .expect(satisfied)
