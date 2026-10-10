// History's activity dispatch and admission protocol, with its deliberately faulty control.
// The held-delivery and lost-response races live in DispatchRaces.scala; queue compositions
// live in DispatchWithTaskQueue.scala.
package temporal
package features.activity
package standalone
package system

import framework.*
import framework.outcomes.{Outcome, Rejection}
import temporal.capabilities.*
import Bounds.{four, three}
import product.ActivityProduct
import product.ActivityProduct.phased

// `pausedWhileHeld` is a pause after admission: the attempt it holds was admitted before it.
enum AdmissionPhase derives Finite:
  case scheduled extends AdmissionPhase, Waiting
  case paused extends AdmissionPhase, Suspended
  // The admitted attempt remains held while its pause is pending.
  case pausedWhileHeld extends AdmissionPhase, Held
  case started extends AdmissionPhase, Held
  case completed extends AdmissionPhase, Succeeded
  case timedOut extends AdmissionPhase, TimedOut

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

// The claims every admission design is held to: the Properties its capabilities bring, the record's own
// count of active attempts, and each deadline timing the activity out with the status that says which.
// `notPaused` is the generated `<design>.pausedIsNotDispatched`, which the pinned paths read.
final case class AdmissionClaims(
    notPaused: Property[AdmissionState],
    oneActive: Property[AdmissionState],
    startDeadline: Property[AdmissionState],
    closeDeadline: Property[AdmissionState]
)

abstract class AdmissionCapabilities(using
    Declaring[AdmissionState, Outcome, AdmissionFact],
    Phasing[AdmissionState, AdmissionPhase]
) extends Capabilities:
  val closable: Capability = Closable(
    rejected = Outcome.rejected(Rejection.notFound)
  )
  val pausable: Capability = Pausable(
    pause = client.pause,
    unpause = client.unpause
  )
  val pollable: Capability = Pollable(dispatch = worker.poll)
  except(Closable.closedIsRejectedUniformly, because = dispatchWaivers.deliveryAfterClose)

// ### Signature

object dispatchWaivers:
  // A delivery to a closed record is admission's to reject, recording `admissionRejected` and
  // owing matching the answer, where the shared Property has it stay (activity.go HandleStarted).
  val deliveryAfterClose =
    "admission rejects a delivery to a closed record and owes matching its answer: activity.go HandleStarted"

  // The queue keeps its own steps after the record closes; the record's capabilities hold the
  // record itself to closedIsRejectedUniformly.
  val queueStepsOn =
    "the queue member keeps stepping after the record closes; admissionCapabilities holds the record"

val admissionCommits = choice
val admissionCommitFails = choice

// History's internal steps.
object history:
  // History's dispatch task: its Validate sends the message only while the activity can start.
  val dispatch = internal

  // Admission's answer reaches matching, which may then complete the task.
  val answerMatching = internal

// ### The corrected design. A design alone takes a delivery whenever one could arrive: what holds of
// it holds over every queue, and what fails of it is confirmed over a queue (DispatchWithTaskQueue.scala).

// The corrected design, whose status sets a composition reads through `activity`.
object ActivityRecord
    extends Machine[AdmissionState, Outcome, AdmissionFact],
      Phased[AdmissionState, AdmissionPhase](_.phase):
  import AdmissionFact.*
  val init =
    AdmissionState(phase = AdmissionPhase.scheduled, active = Active.none, answer = Answer.settled)
  override def end(s: State) = states.stopped(s)
  val evidence: PartialFunction[AdmissionFact, String] = { case AdmissionFact.statusTimedOut(_) =>
    "statusTimedOut"
  }
  // The record's status sets, which a composition reads through `activity`, and its counts.
  object states:
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
      else if after.state.phase.in[Closed] then Finality.closed
      else if f == Finality.closed then Finality.reopened
      else Finality.open

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

  object rules extends Rules:
    import AdmissionPhase.*

    on(history.dispatch) {
      when(scheduled) ~> effects.sendDispatch
    }

    // A pause of a paused or closed record has no rule, and the other controls are out of scope.
    on(client.pause) {
      when(scheduled) ~> effects.pause
      when(started) ~> effects.pauseHeld
    }

    // The corrected design re-reads current eligibility: only a scheduled activity admits.
    on(worker.poll) {
      when(scheduled) ~> effects.admit
      when(paused, pausedWhileHeld, started, completed, timedOut) ~> effects.rejectDelivery
    }
    on(history.answerMatching) {
      where(_.answer == Answer.owed) ~> effects.answerMatching
    }
    on(worker.respondCompleted) {
      when(started) ~> effects.complete
    }

    // Schedule-to-start covers the wait for a worker, so it fires only before an attempt is
    // admitted; schedule-to-close covers the whole activity, so it competes with the other deadline
    // while none has fired.
    on(deadline.scheduleToStart) {
      when(scheduled) ~> (effects.timeOut(_, TimeoutType.scheduleToStart))
    }
    on(deadline.scheduleToClose) {
      when(scheduled, paused, pausedWhileHeld, started) ~> (effects.timeOut(
        _,
        TimeoutType.scheduleToClose
      ))
    }

  // The history record's promises are declared on each admission design and each composition with
  // the record: the Properties its capabilities bring, and the record's own count of active attempts. Each
  // takes the states it speaks of as predicates, so one definition serves the record and a
  // composition, which reads the record through its `activity` member.
  object properties:
    // No step leaves two admitted attempts active.
    def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
      m.property("atMostOneActive").never(s => twoActive(s.state))

    def admissionClaims(
        m: Machine[State, Outcome, AdmissionFact],
        declared: Capabilities[State, Outcome, AdmissionFact]
    ) =
      val scheduleToStartTimesOut = m.property when deadline.scheduleToStart holds (after =>
        after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToStart))
      )
      val scheduleToCloseTimesOut = m.property when deadline.scheduleToClose holds (after =>
        after.records(AdmissionFact.statusTimedOut(TimeoutType.scheduleToClose))
      )
      AdmissionClaims(
        declared.claim(Pausable.pausedIsNotDispatched[State, AdmissionPhase]),
        atMostOneActive(m)(states.twoActive),
        scheduleToStartTimesOut,
        scheduleToCloseTimesOut
      )

  object capabilities extends AdmissionCapabilities

  object queries:
    capabilities.bound(five)

    // Every claim and path, declared on the design `m`, since each belongs to one machine.
    def admissionQueries(
        m: Machine[State, Outcome, AdmissionFact],
        declared: Capabilities[State, Outcome, AdmissionFact]
    ) =
      val claims = properties.admissionClaims(m, declared)
      val staleDeliveryAfterPause =
        m.scenario.actions(history.dispatch, client.pause, worker.poll)
      val admittedBeforePause =
        m.scenario.actions(history.dispatch, worker.poll, client.pause)
      val duplicateDelivery =
        m.scenario.actions(history.dispatch, worker.poll, worker.poll)
      val startedAfterCompletion = m.scenario
        .actions(
          history.dispatch,
          worker.poll,
          worker.respondCompleted,
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
          .verify(
            ActivityProduct.capabilities
              .claim(Pausable.pausedIsNotDispatched[product.State, product.Phase])
          )
          .in(staleDeliveryAfterPause) limits three
      )

    val activityRecordQueries = admissionQueries(ActivityRecord, capabilities)

// ### The deliberately faulty design: admission trusts the eligibility the message was sent with,
// so every delivery is admitted as one that meets a scheduled activity is.

object TrustingActivityRecord
    extends Derived(
      ActivityRecord.rebind(on(worker.poll) {
        always ~> ActivityRecord.effects.admit
      })
    ),
      NegativeControl:
  object capabilities extends AdmissionCapabilities
  object queries:
    capabilities.bound(five)
    val trustingActivityRecordQueries =
      ActivityRecord.queries.admissionQueries(TrustingActivityRecord, capabilities)
