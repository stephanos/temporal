/* The Nexus caller-side Model: one workflow-scheduled Nexus operation, as the caller sees it. The
 * product machine says what an operation does, the protocol machine says how the server gets there
 * and refines it, and the functional Queries are one per side effect that settles the operation.
 * No cancellation (fn-79) and no concurrency-limit setup parameter.
 *
 * The feature has two levels, each in a folder of its own, because different people read them
 * (model/irgen/testdata/layout/lamp is the template):
 *
 *   - this file: the types; the signature (the entities, the inputs, the parties with their
 *     actions, the derived observation, the timers and the bounds); and last exports, its IR files;
 *   - product/Product.scala: NexusProduct, the product machine, what an operation does;
 *   - system/System.scala: NexusProtocol, the protocol machine that refines it; HandlerWorker, the
 *     handler's worker; and NexusCaller, the protocol with that worker;
 *   - system/ForgedCompletion.scala: ForgedCompletion, the forged control a caller must refuse;
 *   - system/ClosePolicy.scala: the close and reset designs.
 *
 * A machine object reads its header (entity, init, end, evidence), then its sections in order:
 * states, refinement, effects, rules, properties and queries. Realization.scala realizes it.
 */
package temporal
package features.nexuscaller

import umpire.*
import io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
import io.temporal.api.nexus.v1.{HandlerError, StartOperationResponse}
import shared.worker.State as WorkerState
import product.NexusProduct
import system.{ForgedCompletion, HandlerWorker, NexusCaller, NexusProtocol}

// Moved from temporal.nexuscaller; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexuscaller.Model$package$")

/** The family of the caller's machines; the control takes `ControlFamily`. */
object CallerFamily:
  given family: Family = Family("temporal.nexus.caller")

/** The control's family, which `ForgedCompletion` (system/) names where it extends `Machine`. */
object ControlFamily:
  val family: Family = Family("temporal.nexus.control")

// ### Types
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

/**
 * A step's outcome. The product and protocol machines share the two members, and an outcome reads
 * as the refined machine's outcome of the same name.
 */
enum Outcome derives Finite:
  case accepted, notFound

/** The product machine's phases: what an operation does. */
enum ProductPhase derives Finite:
  case scheduled, started, succeeded, failed, canceled, timedOut

final case class ProductState(phase: ProductPhase) derives Finite

enum ProductFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled, nexusOperationTimedOut

type ProductStep = Step[ProductState, Outcome, ProductFact]

/** The protocol machine's phases. It begins before the operation exists, so unscheduled is one. */
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

enum ProtocolFact derives Finite:
  case nexusOperationScheduled, nexusOperationStarted, nexusOperationCompleted,
    nexusOperationFailed,
    nexusOperationCanceled
  case nexusOperationTimedOut(timeoutType: TimeoutType)

  /** The attempt count, read through the observation of that name: no history event records it. */
  case pendingAttempts

type ProtocolStep = Step[ProtocolState, Outcome, ProtocolFact]

final case class NexusCallerState(operation: ProtocolState, worker: WorkerState)

// ### Signature
//
// Parties are names the feature declares by using them. The reserved party system is the server.
// A fault is an ordinary action of a declared party, and a timer is system behavior the machine
// owns, so neither is a separate kind. Each party is an actor object whose members are the actions
// it takes, and the timers are grouped in sections. Actor and section objects are transparent to
// Definition IDs, so every action keeps the ID the file's pin gives it.

// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

val workflow = Entity()
val operation = Entity(key = "scheduledEvent", refer = Map("caller" -> workflow))

// The inputs: the schedule's deadlines, which the deadline timers no longer collide with, the
// handler's reply and its completion's resolution.
val scheduleToClose = input[Timeout]
val scheduleToStart = input[Timeout]
val startToClose = input[Timeout]
val reply = input[Reply]
val resolution = input[Resolution]

/** The caller workflow, which schedules the operation. */
object caller extends Actor:
  val schedule = action(this)
    .input(scheduleToClose)
    .input(scheduleToStart)
    .input(startToClose)
    .creates(operation)
    .schema[ScheduleNexusOperationCommandAttributes]

/** The endpoint's handler, which replies to the start and completes the operation. */
object handler extends Actor:
  val handlerReply = action(this)
    .on(operation)
    .input(reply)
    .schema[StartOperationResponse]
    .schema[HandlerError]
    .example(Reply.handlerError(false), "BadRequest")
    .example(Reply.handlerError(true), "Internal")

  /**
   * The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
   * are names the realization interprets. The result text is metadata of the action, not a domain a
   * state holds.
   */
  val complete = action(this).on(operation).input(resolution).results("Delivery")

/** The network between the caller and the handler, which can fail a transport. */
object network extends Actor:
  val transportFault = action(this) on operation

// The handler's worker stopping is the worker's own action, `worker.workerStop`: an action that
// names no entity is behavior no entity records. The Run records the fault, but nothing recorded
// names the operation, so the machines keep their state and record nothing at it.

// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts = Observation(on = operation, read = "attempts")

given Ok[Outcome] = Ok(Outcome.accepted)

/** One of the operation's deadlines firing, as the product machine sees it, and the backoff. */
object timers extends Section:
  val timeout = timer
  val backoff = timer

/** The protocol's three deadlines, each armed by the schedule's input of its name. */
object deadline extends Section:
  val scheduleToClose = timer
  val scheduleToStart = timer
  val startToClose = timer

/**
 * Bounds the attempt count. Nothing wires the Limits into a machine's state, so the bound is
 * written here, beside the state type's `Finite`, which reads it before any machine exists; the Go
 * model evaluator checks each require precondition against it.
 */
val attemptBound = 2

given Finite[ProtocolState] =
  // The lifter reads this Int bound; the Go model evaluator uses it to enumerate protocol states.
  given Finite[Int] = Finite.upTo(attemptBound)
  Finite.derived

// The bounds of the Queries, beside three and four (shared.Bounds). Nine actions are enabled before
// the operation is scheduled and eleven once it is, so an exact sequence of two is found among
// ninety-nine candidates, one of three among about a thousand and one of four among about ten
// thousand.
val two = Limits(steps = 2, actions = 2, search = 512)
val control = Limits(steps = 8, actions = 8, search = 262144)

// ### The checked-in IR files of the Nexus caller Model and its close and reset designs (umpire.irFile).

object exports:
  // The functional Queries and the realization that runs them are roots beside the machines: Go
  // lowers each Query's witness through the realization into a Testpilot Case (tools/umpire/lower).
  // The product claim read on a protocol path and the cross-entity Query, which carries the
  // composition with the handler's worker and its claim, are roots too.
  val nexusCaller = irFile("nexus-caller")(
    NexusProduct,
    NexusProtocol,
    HandlerWorker,
    NexusCaller,
    shared.worker.Polling,
    NexusProtocol.queries.functionalQueries,
    NexusProtocol.queries.terminalHolds,
    NexusCaller.queries.stoppedWorkerRepliesNothing,
    NexusRealization.asyncNexus
  )

  // The forged completion a caller must refuse, and the realization that offers it.
  val nexusControl =
    irFile("nexus-control")(
      ForgedCompletion.queries.forgedCompletion,
      NexusRealization.forgedCompletion
    )

  // The close and reset designs. Each design's Queries are a root, and so is each progress claim.
  val nexusClose = irFile("nexus-close")(
    system.RejectAfterClose.queries.all,
    system.AckByOriginal.queries.all,
    system.RetainAndRoute.queries.all,
    system.ForgetsCancelOnReset.queries.all,
    system.TruncatesOnReset.queries.all,
    system.RetainAndRouteBoundedRetry.queries.all,
    system.RejectAfterCloseWithDeadline.queries.all,
    system.AckByOriginalWithDeadline.queries.all,
    system.RetainAndRouteWithDeadline.queries.all,
    system.RejectAfterClose.properties.rejectAfterCloseProgress,
    system.AckByOriginal.properties.ackByOriginalProgress,
    system.RetainAndRoute.properties.retainAndRouteProgress,
    system.RetainAndRouteBoundedRetry.properties.retainAndRouteBoundedRetryProgress,
    system.RejectAfterCloseWithDeadline.properties.rejectAfterCloseWithDeadlineProgress,
    system.AckByOriginalWithDeadline.properties.ackByOriginalWithDeadlineProgress,
    system.RetainAndRouteWithDeadline.properties.retainAndRouteWithDeadlineProgress,
    system.RetainAndRoute.properties.retainedReachesOwner,
    system.RetainAndRoute.properties.retainedWaitsWithoutRecovery,
    system.RetainAndRouteBoundedRetry.properties.retainedReachesOwnerBoundedRetry
  )
