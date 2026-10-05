/* The Nexus caller-side Model: one workflow-scheduled Nexus operation, as the caller sees it. The
 * product machine says what an operation does, the protocol machine says how the server gets there
 * and refines it, and the functional Queries are one per side effect that settles the operation.
 * No cancellation (fn-79) and no concurrency-limit setup parameter.
 *
 * This file declares the vocabulary, the domains and step functions, the two machines, the
 * composition with the handler's worker, and the forged control a caller must refuse.
 */
package temporal
package features.nexuscaller

import scala.annotation.unused
import umpire.*
import io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
import io.temporal.api.nexus.v1.{HandlerError, StartOperationResponse}
import shared.worker.{serve, workerStop, Phase as WorkerPhase, State as WorkerState}
import CallerFamily.given

// Moved from temporal.nexuscaller; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexuscaller.Model$package$")

/** The family of the caller's machines; the control takes its own, in `object Control`. */
object CallerFamily:
  given family: Family = Family("temporal.nexus.caller")

// Parties are names the feature declares by using them. The reserved party system is the server.
// A fault is an ordinary action of a declared party, and a timer is system behavior the machine
// owns, so neither is a separate kind.
val caller = Party()
val handler = Party()
val network = Party()

// ### Entities
//
// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

val workflow = Entity()
val operation = Entity(key = "scheduledEvent", refer = Map("caller" -> workflow))

// ### The input domains
//
// A class is one member of a domain, and a constructor that carries finite fields contributes one
// class per assignment of them: handlerError(retryable) is one constructor and two classes, which
// is the granularity an example is written at and what mirrors a protobuf oneof.

/** Whether the schedule command sets a deadline. */
enum Timeout derives Finite:
  case unset, expires

/** The handler's reply to the server's start request. */
enum Reply derives Finite:
  case syncSuccess, async, operationFailed, operationCanceled
  case handlerError(retryable: Boolean)

/** How an asynchronous completion settles the operation. */
enum Resolution derives Finite:
  case succeeded, failed, canceled

/** The inputs, apart because the deadline timers take their names. */
object Inputs:
  val scheduleToClose = input[Timeout]
  val scheduleToStart = input[Timeout]
  val startToClose = input[Timeout]
  val reply = input[Reply]
  val resolution = input[Resolution]

// ### Actions

val schedule = action(caller)
  .input(Inputs.scheduleToClose)
  .input(Inputs.scheduleToStart)
  .input(Inputs.startToClose)
  .creates(operation)
  .schema[ScheduleNexusOperationCommandAttributes]

val handlerReply = action(handler)
  .on(operation)
  .input(Inputs.reply)
  .schema[StartOperationResponse]
  .schema[HandlerError]
  .example(Reply.handlerError(false), "BadRequest")
  .example(Reply.handlerError(true), "Internal")

/**
 * The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
 * are names the realization interprets. The result text is metadata of the action, not a domain a
 * state holds.
 */
val complete = action(handler).on(operation).input(Inputs.resolution).results("Delivery")

val transportFault = action(network) on operation

// The handler's worker stopping is the worker's own action, `workerStop`: an action that names no
// entity is behavior no entity records. The Run records the fault, but nothing recorded names the
// operation, so the machines keep their state and record nothing at it.

// ### The derived observation
//
// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts = Observation(on = operation, read = "attempts")

/**
 * A step's outcome. The product and protocol machines share the two members, and an outcome reads
 * as the refined machine's outcome of the same name.
 */
enum Outcome derives Finite:
  case accepted, notFound

given Ok[Outcome] = Ok(Outcome.accepted)

// ### The product machine
//
// What an operation does, with no account of how. Every Property written against it is carried to
// the protocol machine by the refinement declared there.

enum ProductPhase derives Finite:
  case scheduled, started, succeeded, failed, canceled, timedOut

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

object Product:
  import ProductPhase.*
  import ProductFact.*

  /** The four phases the product machine ends on. */
  def productTerminal(s: ProductState) = s.phase.in(succeeded, failed, canceled, timedOut)

  /**
   * The handler's reply to the server's start request. An operation that has not started yet is
   * the only one a reply can move.
   */
  def handlerReplyStep(s: ProductState, reply: Reply) =
    if s.phase != scheduled then disabled
    else
      reply match
        case Reply.syncSuccess       => enter(ProductState(succeeded), nexusOperationCompleted)
        case Reply.async             => enter(ProductState(started), nexusOperationStarted)
        case Reply.operationFailed   => enter(ProductState(failed), nexusOperationFailed)
        case Reply.operationCanceled => enter(ProductState(canceled), nexusOperationCanceled)
        // A retryable handler error leaves the operation where it is: the product machine does not
        // know about backing off, which is the whole of what the protocol machine adds.
        case Reply.handlerError(retryable) =>
          if retryable then disabled else enter(ProductState(failed), nexusOperationFailed)

  /**
   * An asynchronous completion. A completion that arrives after the operation is over is not
   * found, and changes nothing.
   */
  def completeStep(s: ProductState, resolution: Resolution) =
    if productTerminal(s) then List(Step(Outcome.notFound, s))
    else
      resolution match
        case Resolution.succeeded => enter(ProductState(succeeded), nexusOperationCompleted)
        case Resolution.failed    => enter(ProductState(failed), nexusOperationFailed)
        case Resolution.canceled  => enter(ProductState(canceled), nexusOperationCanceled)

  /**
   * A transport fault is an ordinary action of the network. The product machine cannot see one:
   * whether a delivery was retried is the protocol's account of how, not what.
   */
  def transportFaultStep(@unused s: ProductState): List[ProductStep] = disabled

  /**
   * The handler's worker stopping is a fault the Run records and the operation does not feel. The
   * product machine cannot see it, like the transport fault: a step that kept the state and
   * recorded nothing would be indistinguishable from a stutter, and the refinement would read every
   * stutter as this step.
   */
  def workerStopStep(@unused s: ProductState): List[ProductStep] = disabled

  /**
   * One of the operation's deadlines firing. Which deadline is the protocol's account of how, so the
   * product machine has one timer, and it fires while the operation runs.
   */
  def timeoutStep(s: ProductState) =
    if s.phase.in(scheduled, started) then enter(ProductState(timedOut), nexusOperationTimedOut)
    else disabled

val timeout = timer

/** Every fact the product machine records is confirmed by the history event of its name. */
val nexusProduct = machine[ProductState, Outcome, ProductFact] {
  forEntity(operation)
  starts(ProductState(ProductPhase.scheduled))
  ends(Product.productTerminal)
  steps(
    handlerReply ~> Product.handlerReplyStep,
    complete ~> Product.completeStep,
    transportFault ~> Product.transportFaultStep,
    workerStop ~> Product.workerStopStep,
    timeout ~> Product.timeoutStep
  )
}

// ### The protocol machine
//
// How the server gets there: the retry the product machine cannot see, the three timers the
// schedule command sets, and the attempt count a retryable failure raises. Written against the same
// actions, so a Property proved on the product machine is carried here by the refinement.
//
// The machine begins before the operation exists: a state structure has no "no instance yet"
// member, so unscheduled is that member, and it is what makes the three deadline fields reachable
// at anything but their first value -- the schedule command is what sets them.
//
// Not here, for reasons recorded rather than silent: the cancel field and its rows (fn-79), and the
// concurrency-limit rejection, which names no operation and is not modeled until a Query needs it.

enum Phase derives Finite:
  case unscheduled, scheduled, backingOff, started, succeeded, failed, canceled, timedOut

/**
 * Which timer fired. The history event records it, so a Contract that did not check it would pass a
 * run that timed out on the wrong deadline.
 */
enum TimeoutType derives Finite:
  case scheduleToClose, scheduleToStart, startToClose

/** The attempt count is `0..attemptBound`. */
final case class ProtocolState(
    phase: Phase,
    attempts: Int,
    scheduleToClose: Timeout,
    scheduleToStart: Timeout,
    startToClose: Timeout
)

given Finite[ProtocolState] =
  // The lifter reads this Int bound; the Go model evaluator uses it to enumerate protocol states.
  given Finite[Int] = Finite.upTo(Protocol.attemptBound)
  Finite.derived

enum ProtocolFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)

  /** The attempt count, read through the observation of that name: no history event records it. */
  case pendingAttempts

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

object Protocol:
  import Phase.*
  import ProtocolFact.*

  /**
   * Bounds the attempt count. Nothing wires the Limits into a machine's state, so the bound is
   * written here; the Go model evaluator checks each require precondition against it.
   */
  val attemptBound = 2

  def validAttempts(a: Int) = 0 <= a && a <= attemptBound

  /** A retry past the bound stays at it, rather than wrapping as `Fin` arithmetic would. */
  def saturatingSucc(a: Int) =
    require(validAttempts(a))
    if a < attemptBound then a + 1 else a

  /** The four phases the design ends on. A completion that arrives after one of them is not found. */
  def terminalPhase(p: Phase) = p.in(succeeded, failed, canceled, timedOut)

  /** Scheduled and not yet over: the phases a completion resolves and a timer can fire in. */
  def running(p: Phase) = p.in(scheduled, backingOff, started)

  /**
   * The caller's schedule command. It names the operation's three deadlines, and every one of them
   * is a state field because whether a timer fires is a question about the operation and not about
   * the command that started it.
   */
  def scheduleStep(
      s: ProtocolState,
      scheduleToClose: Timeout,
      scheduleToStart: Timeout,
      startToClose: Timeout
  ) =
    if s.phase != Phase.unscheduled then disabled
    else
      enter(
        ProtocolState(scheduled, 0, scheduleToClose, scheduleToStart, startToClose),
        nexusOperationScheduled
      )

  /**
   * The handler's reply to the server's start request. What the product machine cannot see is the
   * last arm: a retryable failure backs the operation off and raises its attempt count, and the
   * count is read back through the pendingAttempts observation because no history event records it.
   */
  def handlerReplyStep(s: ProtocolState, reply: Reply) =
    require(validAttempts(s.attempts))
    if s.phase != scheduled then disabled
    else
      reply match
        case Reply.syncSuccess       => enter(s.copy(phase = succeeded), nexusOperationCompleted)
        case Reply.async             => enter(s.copy(phase = started), nexusOperationStarted)
        case Reply.operationFailed   => enter(s.copy(phase = failed), nexusOperationFailed)
        case Reply.operationCanceled => enter(s.copy(phase = canceled), nexusOperationCanceled)
        case Reply.handlerError(retryable) =>
          if !retryable then enter(s.copy(phase = failed), nexusOperationFailed)
          else
            enter(
              s.copy(phase = backingOff, attempts = saturatingSucc(s.attempts)),
              ProtocolFact.pendingAttempts
            )

  /** A transport fault is the same failure arriving as a dropped delivery rather than as a reply. */
  def transportFaultStep(s: ProtocolState) =
    require(validAttempts(s.attempts))
    if s.phase != scheduled then disabled
    else
      enter(
        s.copy(phase = backingOff, attempts = saturatingSucc(s.attempts)),
        ProtocolFact.pendingAttempts
      )

  /**
   * The handler's worker stopping is a fault the Run records and the operation does not feel, so the
   * step keeps the state and records nothing. On a path it is confirmed by the evidence of the step
   * after it, and the Case says so in a Known Gap.
   */
  def workerStopStep(s: ProtocolState): List[ProtocolStep] = stay(s)

  /**
   * An asynchronous completion. Before a start, the server records a Started event first, which is
   * why the evidence is two facts and not one -- and why the product machine, which has no
   * backingOff phase to have skipped, could write the completion alone.
   */
  def completeStep(s: ProtocolState, resolution: Resolution) =
    if terminalPhase(s.phase) then List(Step(Outcome.notFound, s))
    else if s.phase == Phase.unscheduled then disabled
    else
      val startedFirst =
        if s.phase != started then List(nexusOperationStarted) else Nil
      resolution match
        case Resolution.succeeded =>
          enter(s.copy(phase = succeeded), (startedFirst ++ List(nexusOperationCompleted))*)
        case Resolution.failed =>
          enter(s.copy(phase = failed), (startedFirst ++ List(nexusOperationFailed))*)
        case Resolution.canceled =>
          enter(s.copy(phase = canceled), (startedFirst ++ List(nexusOperationCanceled))*)

  /**
   * The backoff timer. It is what makes backingOff a phase the operation leaves rather than a state
   * it is stuck in, and it records nothing: a retry writes no history event.
   */
  def backoffStep(s: ProtocolState): List[ProtocolStep] =
    if s.phase != backingOff then disabled else enter(s.copy(phase = scheduled))

  /**
   * The schedule-to-close deadline covers the whole operation, so it fires in every running phase
   * -- and only when the schedule command set it.
   */
  def scheduleToCloseStep(s: ProtocolState) =
    if running(s.phase) && s.scheduleToClose == Timeout.expires then
      enter(s.copy(phase = timedOut), nexusOperationTimedOut(TimeoutType.scheduleToClose))
    else disabled

  /**
   * The schedule-to-start deadline covers the wait for the handler to accept, so it stops at the
   * start.
   */
  def scheduleToStartStep(s: ProtocolState) =
    if s.phase.in(scheduled, backingOff) && s.scheduleToStart == Timeout.expires then
      enter(s.copy(phase = timedOut), nexusOperationTimedOut(TimeoutType.scheduleToStart))
    else disabled

  /** The start-to-close deadline covers the handler's own work, so it begins at the start. */
  def startToCloseStep(s: ProtocolState) =
    if s.phase == started && s.startToClose == Timeout.expires then
      enter(s.copy(phase = timedOut), nexusOperationTimedOut(TimeoutType.startToClose))
    else disabled

  /**
   * How a protocol state reads as a product state. A phase of the same name is that phase; backing
   * off is still scheduled, because the product machine cannot see a retry; and an operation not
   * yet scheduled reads as scheduled, because the product machine begins there. Every other field
   * is hidden, which is what a map that does not read it says.
   */
  def productOf(s: ProtocolState) = s.phase match
    case Phase.unscheduled | Phase.scheduled | Phase.backingOff =>
      ProductState(ProductPhase.scheduled)
    case Phase.started   => ProductState(ProductPhase.started)
    case Phase.succeeded => ProductState(ProductPhase.succeeded)
    case Phase.failed    => ProductState(ProductPhase.failed)
    case Phase.canceled  => ProductState(ProductPhase.canceled)
    case Phase.timedOut  => ProductState(ProductPhase.timedOut)

val backoff = timer
val scheduleToClose = timer
val scheduleToStart = timer
val startToClose = timer

/** Where every path begins: before the operation exists, with every deadline at its first value. */
val unscheduled =
  ProtocolState(Phase.unscheduled, 0, Timeout.unset, Timeout.unset, Timeout.unset)

/**
 * A timeout is confirmed by the one timed-out event, whichever deadline fired, and the attempt
 * count by its observation.
 */
val nexusProtocol = machine[ProtocolState, Outcome, ProtocolFact] {
  forEntity(operation)
  refines(nexusProduct)(Protocol.productOf)
  starts(unscheduled)
  ends(s => Protocol.terminalPhase(s.phase))
  unobservable(backoff)
  evidence {
    case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
    case ProtocolFact.pendingAttempts           => pendingAttempts.name
  }
  steps(
    schedule ~> Protocol.scheduleStep,
    handlerReply ~> Protocol.handlerReplyStep,
    complete ~> Protocol.completeStep,
    transportFault ~> Protocol.transportFaultStep,
    workerStop ~> Protocol.workerStopStep,
    backoff ~> Protocol.backoffStep,
    scheduleToClose ~> Protocol.scheduleToCloseStep,
    scheduleToStart ~> Protocol.scheduleToStartStep,
    startToClose ~> Protocol.startToCloseStep
  )
}

/**
 * The caller's view of the handler's worker: it stops and it serves. It never resumes, because an
 * action no sync line names would stay executable on its own and admit a stop, a resume and then a
 * reply; the operation's timers settle every state a stop leaves.
 */
val handlerWorker = shared.worker.polling.restrict(workerStop, serve)

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No
// functional Query reads the composition; it is what the cross-entity claim is verified over.

final case class NexusCallerState(operation: ProtocolState, worker: WorkerState)

val nexusCaller =
  compose[NexusCallerState](_.operation -> nexusProtocol, _.worker -> handlerWorker)
    .sync(_.operation -> workerStop, _.worker -> workerStop)
    .sync(_.operation -> handlerReply, _.worker -> serve)
    .ends(s => Protocol.terminalPhase(s.operation.phase))

val pollingWorker = WorkerState(WorkerPhase.polling)

// ### The control
//
// A caller design that predicts success for a failed completion, which the forged-completion Query
// must refuse. It keeps its own family, and its Definition IDs hang off this object.

object Control:
  // Moved from temporal.nexuscaller; the pin keeps its Definition IDs.
  given DefinitionScope = DefinitionScope("temporal.nexuscaller.Control$")

  given family: Family = Family("temporal.nexus.control")

  val inspect = action(caller).on(operation)

  def inspectStep(s: ProtocolState): List[ProtocolStep] = stay(s)

  // A failed callback completes the operation two ways: as the success the control forges, and as
  // the failure the runtime still sends.
  val forged = choice
  val sent = choice

  // The control deliberately predicts success for a failed callback. The runtime still sends failure.
  def forgedComplete(s: ProtocolState, resolution: Resolution) =
    if resolution == Resolution.failed then
      choose(
        forged -> Protocol.completeStep(s, Resolution.succeeded),
        sent -> Protocol.completeStep(s, resolution)
      )
    else Protocol.completeStep(s, resolution)

  // The protocol machine without its refinement, its completion forged and an inspection added.
  // It is declared, not derived from nexusProtocol: a derivation lifts its source machines, and
  // nexus-control.json holds this machine alone.
  val forgedCompletion = machine[ProtocolState, Outcome, ProtocolFact] {
    forEntity(operation)
    starts(unscheduled)
    ends(s => Protocol.terminalPhase(s.phase))
    unobservable(backoff)
    evidence {
      case ProtocolFact.nexusOperationTimedOut(_) => "nexusOperationTimedOut"
      case ProtocolFact.pendingAttempts           => pendingAttempts.name
    }
    steps(
      schedule ~> Protocol.scheduleStep,
      handlerReply ~> Protocol.handlerReplyStep,
      complete ~> forgedComplete,
      inspect ~> inspectStep,
      transportFault ~> Protocol.transportFaultStep,
      workerStop ~> Protocol.workerStopStep,
      backoff ~> Protocol.backoffStep,
      scheduleToClose ~> Protocol.scheduleToCloseStep,
      scheduleToStart ~> Protocol.scheduleToStartStep,
      startToClose ~> Protocol.startToCloseStep
    )
  }
