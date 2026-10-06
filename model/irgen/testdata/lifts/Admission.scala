// A frozen sketch of activity admission: two designs of one activity's history record, a corrected
// one that re-reads eligibility at admission and a deliberately faulty one that trusts a stale
// dispatch message, with passive monitors in the framework's `monitor` declaration. It refines a
// small product of its own, only the statuses its refinement reads, so a change to the live activity
// Model (temporal/features/activity/standalone, lifted to model/ir/activity-standalone.json) rewrites none of it.
// The lifter's tests lift both designs' Queries and compare the IR with expected/admission.json; the
// Go tests read that IR as a small fixed system.
package fixture.specimens.admission

import umpire.*
import Entities.activity

// ### The product: what a caller reads of the activity, as far as the designs' refinement reads it

object caller extends Actor
object worker extends Actor

// The entity, in an object of its own, which the machine objects read while they initialize.
object Entities:
  val activity = Entity(key = "activityId")

enum Outcome derives Finite:
  case accepted, notFound

given Ok[Outcome] = Ok(Outcome.accepted)

// Only a pause and a completion are in scope. The other controls and answers are classes no step
// takes, kept because a free Scenario's total counts every class.
enum Control derives Finite:
  case pause, unpause, requestCancel, terminate

enum AttemptResult derives Finite:
  case completed
  case failed(retries: Boolean)
  case canceled

// Apart, because the control action takes its input's name.
object Inputs:
  val result = input[AttemptResult]
  val control = input[Control]

val poll = action(worker).on(activity)
val respond = action(worker).on(activity).input(Inputs.result)
val control = action(caller).on(activity).input(Inputs.control)

enum ProductPhase derives Finite:
  case scheduled, started, paused, completed

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case statusStarted, statusPaused, statusCompleted

object Product:
  import ProductPhase.*
  import ProductFact.*

  def poll(s: ProductState) =
    if s.phase == scheduled then enter(ProductState(started), statusStarted) else disabled

  def respond(s: ProductState, r: AttemptResult) =
    if s.phase == started && r == AttemptResult.completed then
      enter(ProductState(completed), statusCompleted)
    else disabled

  // A pause holds before an attempt starts or while one runs; a completed activity is not found.
  def control(s: ProductState, c: Control) =
    if s.phase == completed then List(Step(Outcome.notFound, s))
    else if c == Control.pause && s.phase != paused then enter(ProductState(paused), statusPaused)
    else disabled

// No unpause is in scope, so a path may end paused, as the designs' paths may.
object ActivityProduct extends Machine[ProductState, Outcome, ProductFact]:
  val entity = activity
  val init = ProductState(ProductPhase.scheduled)
  def end(s: State) = s.phase == ProductPhase.completed || s.phase == ProductPhase.paused

  object rules
      extends Bindings(
        poll ~> Product.poll,
        respond ~> Product.respond,
        control ~> Product.control
      )

// The product's promise the designs are read against: no step from paused lands in started.
val pausedIsNotDispatched = ActivityProduct.property
  .never(_.state.phase == ProductPhase.started)
  .from(_.phase == ProductPhase.paused)

// ### System vocabulary: one logical activity, one dispatch message, the attempts admission committed

// The activity as history's authoritative record has it. `pausedWhileHeld` is a pause that
// arrived after admission: the attempt it holds is work admitted before the pause.
enum AdmissionPhase derives Finite:
  case scheduled, paused, pausedWhileHeld, started, completed

// The dispatch channel holds at most one message. It was validated when it was enqueued, and
// nothing re-reads that eligibility while it is in flight.
enum Message derives Finite:
  case empty, queued, redelivery

// Attempts admission committed and no result closed. An enum rather than an Int, because the
// lifter gives every Int field of one record the same range.
enum Active derives Finite:
  case none, one, two

final case class AdmissionState(phase: AdmissionPhase, message: Message, active: Active)
    derives Finite

// The statuses the product reads, by their product names, and the internal facts no product fact
// is named after, which the refinement therefore drops.
enum AdmissionFact derives Finite:
  case statusStarted, statusPaused, statusCompleted
  case dispatchEnqueued, attemptAdmitted, admissionRejected

type AdmissionStep = Step[AdmissionState, Outcome, AdmissionFact]

val committed = choice
val redelivered = choice

// History's dispatch task sends the message. System-internal, so spelled as a timer today.
val dispatch = timer

val scheduledEmpty: AdmissionState =
  AdmissionState(AdmissionPhase.scheduled, Message.empty, Active.none)

// ### Step functions shared by both designs

def dispatchStep(s: AdmissionState): List[AdmissionStep] =
  if s.phase != AdmissionPhase.scheduled || s.message != Message.empty then Nil
  else
    List(
      Step(Outcome.accepted, s.copy(message = Message.queued), List(AdmissionFact.dispatchEnqueued))
    )

// Only pause is in scope; no unpause intervenes. A pause keeps the message in flight.
def pauseStep(s: AdmissionState, c: Control): List[AdmissionStep] = c match
  case Control.pause =>
    s.phase match
      case AdmissionPhase.scheduled =>
        List(
          Step(
            Outcome.accepted,
            s.copy(phase = AdmissionPhase.paused),
            List(AdmissionFact.statusPaused)
          )
        )
      case AdmissionPhase.started =>
        List(
          Step(
            Outcome.accepted,
            s.copy(phase = AdmissionPhase.pausedWhileHeld),
            List(AdmissionFact.statusPaused)
          )
        )
      case _ => Nil
  case Control.unpause | Control.requestCancel | Control.terminate => Nil

def oneMore(a: Active): Active = a match
  case Active.none => Active.one
  case _           => Active.two

def oneLess(a: Active): Active = a match
  case Active.two => Active.one
  case _          => Active.none

// A committed admission. The channel is at-least-once: the message may be consumed, or retained
// and delivered again before matching acknowledges it.
def admitted(s: AdmissionState): List[AdmissionStep] =
  val next = s.copy(phase = AdmissionPhase.started, active = oneMore(s.active))
  if s.message == Message.redelivery then
    List(
      Step(
        Outcome.accepted,
        next.copy(message = Message.empty),
        List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted)
      )
    )
  else
    choose(
      committed -> List(
        Step(
          Outcome.accepted,
          next.copy(message = Message.empty),
          List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted)
        )
      ),
      redelivered -> List(
        Step(
          Outcome.accepted,
          next.copy(message = Message.redelivery),
          List(AdmissionFact.statusStarted, AdmissionFact.attemptAdmitted),
          "the channel may deliver the message again"
        )
      )
    )

// The corrected design: admission re-reads current eligibility and drops a stale message.
def admitCurrent(s: AdmissionState): List[AdmissionStep] = s.message match
  case Message.empty                       => Nil
  case Message.queued | Message.redelivery =>
    if s.phase == AdmissionPhase.scheduled then admitted(s)
    else
      List(
        Step(
          Outcome.accepted,
          s.copy(message = Message.empty),
          List(AdmissionFact.admissionRejected)
        )
      )

// The deliberately faulty design: admission trusts the eligibility the message was enqueued with.
def admitStale(s: AdmissionState): List[AdmissionStep] = s.message match
  case Message.empty                       => Nil
  case Message.queued | Message.redelivery => admitted(s)

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

// Both pauses read as the product's one paused status; every other phase as its namesake.
def productOfAdmission(s: AdmissionState): ProductState =
  val read = s.phase match
    case AdmissionPhase.scheduled                               => ProductPhase.scheduled
    case AdmissionPhase.paused | AdmissionPhase.pausedWhileHeld => ProductPhase.paused
    case AdmissionPhase.started                                 => ProductPhase.started
    case AdmissionPhase.completed                               => ProductPhase.completed
  ProductState(read)

def admissionEvidence(f: AdmissionFact): String = f match
  case AdmissionFact.statusStarted     => "statusStarted"
  case AdmissionFact.statusPaused      => "statusPaused"
  case AdmissionFact.statusCompleted   => "statusCompleted"
  case AdmissionFact.dispatchEnqueued  => "dispatchEnqueued"
  case AdmissionFact.attemptAdmitted   => "attemptAdmitted"
  case AdmissionFact.admissionRejected => "admissionRejected"

// No unpause is in scope, so a pause is where a path may end, as a completion is.
def admissionEnds(s: AdmissionState): Boolean =
  s.phase != AdmissionPhase.scheduled && s.phase != AdmissionPhase.started

// ### Monitors: at most one admitted active attempt, and terminal finality

// The attempts admission committed and no result closed, counted from the steps' facts.
def countActive(active: Active, before: AdmissionState, after: AdmissionStep): Active =
  if after.facts.contains(AdmissionFact.attemptAdmitted) then oneMore(active)
  else if after.facts.contains(AdmissionFact.statusCompleted) then oneLess(active)
  else active

val atMostOneActiveAttempt: Monitor[AdmissionState, Outcome, AdmissionFact, Active] =
  monitor[AdmissionState, Outcome, AdmissionFact, Active](Active.none)(
    countActive
  )(_ == Active.two).readAfter(after => after.facts.contains(AdmissionFact.attemptAdmitted))

// Whether the activity completed, and whether a step after that left completed.
enum Finality derives Finite:
  case open, completed, reopened

def finality(f: Finality, before: AdmissionState, after: AdmissionStep): Finality = f match
  case Finality.reopened => Finality.reopened
  case _                 =>
    if after.state.phase == AdmissionPhase.completed then Finality.completed
    else if f == Finality.completed then Finality.reopened
    else Finality.open

val terminalFinality: Monitor[AdmissionState, Outcome, AdmissionFact, Finality] =
  monitor[AdmissionState, Outcome, AdmissionFact, Finality](Finality.open)(
    finality
  )(_ == Finality.reopened)

// ### The two designs

object ActivityRecord extends Machine[AdmissionState, Outcome, AdmissionFact]:
  val entity = activity
  def init = scheduledEmpty
  def end(s: State) = admissionEnds(s)
  val evidence: AdmissionFact => String = admissionEvidence

  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: AdmissionState) = productOfAdmission(s)

  object monitors:
    val oneActive = atMostOneActiveAttempt
    val finality = terminalFinality

  object rules
      extends Bindings(
        dispatch ~> dispatchStep,
        control ~> pauseStep,
        poll ~> admitCurrent,
        respond ~> resultStep
      )

object TrustingActivityRecord extends Machine[AdmissionState, Outcome, AdmissionFact]:
  val entity = activity
  def init = scheduledEmpty
  def end(s: State) = admissionEnds(s)
  val evidence: AdmissionFact => String = admissionEvidence

  object refinement extends Refinement(ActivityProduct):
    def toProduct(s: AdmissionState) = productOfAdmission(s)

  object monitors:
    val oneActive = atMostOneActiveAttempt
    val finality = terminalFinality

  object rules
      extends Bindings(
        dispatch ~> dispatchStep,
        control ~> pauseStep,
        poll ~> admitStale,
        respond ~> resultStep
      )

// ### Promises, written once and declared on each design

def notAdmittedWhilePaused(before: AdmissionState, after: AdmissionStep): Boolean =
  before.phase != AdmissionPhase.paused || after.state.phase != AdmissionPhase.started

def atMostOneActive(after: AdmissionStep): Boolean = after.state.active != Active.two

def terminalStays(before: AdmissionState, after: AdmissionStep): Boolean =
  before.phase != AdmissionPhase.completed || after.state.phase == AdmissionPhase.completed

val three: Limits = Limits(steps = 3, actions = 3, search = 4096)
val five: Limits = Limits(steps = 5, actions = 5, search = 65536)

// Every claim and path, declared on one design: a Property or Scenario belongs to one machine.
def admissionQueries(m: Machine[AdmissionState, Outcome, AdmissionFact]): Vector[Query] =
  val notPaused = m.property("notAdmittedWhilePaused") holdsAcross notAdmittedWhilePaused
  val oneActive = m.property("atMostOneActive") holds atMostOneActive
  val terminal = m.property("terminalStays") holdsAcross terminalStays
  val stale = m
    .scenario("staleDeliveryAfterPause")
    .starts(scheduledEmpty)
    .actions(dispatch, control(Control.pause), poll)
  val prePause = m
    .scenario("admittedBeforePause")
    .starts(scheduledEmpty)
    .actions(dispatch, poll, control(Control.pause))
  val duplicate = m
    .scenario("duplicateDelivery")
    .starts(scheduledEmpty)
    .actions(dispatch, poll, poll)
  val any = m.scenario.starts(scheduledEmpty).free
  Vector(
    query(s"${m.name}.staleDelivery") verify notPaused in stale limits three total 135,
    query(s"${m.name}.admittedBeforePause") verify notPaused in prePause limits three total 135,
    query(s"${m.name}.duplicateDelivery") verify oneActive in duplicate limits three total 135,
    query(s"${m.name}.any.notAdmittedWhilePaused") verify notPaused in any limits five total 2250,
    query(s"${m.name}.any.atMostOneActive") verify oneActive in any limits five total 2250,
    query(s"${m.name}.any.terminalStays") verify terminal in any limits five total 2250,
    // The product's own Property, read through the design's declared refinement.
    query(s"${m.name}.product.pausedIsNotDispatched")
      .verify(pausedIsNotDispatched)
      .in(stale) limits three total 135
  )

val currentQueries: Vector[Query] = admissionQueries(ActivityRecord)
val staleQueries: Vector[Query] = admissionQueries(TrustingActivityRecord)
