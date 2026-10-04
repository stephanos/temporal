/* The Nexus caller-side Model: one workflow-scheduled Nexus operation, as the caller sees it. The
 * product machine says what an operation does, the protocol machine says how the server gets there
 * and refines it, and the functional Queries are one per side effect that settles the operation.
 * No cancellation (fn-79) and no concurrency-limit
 * setup parameter.
 *
 * The domains and step functions are in Nexus.scala; this file declares the vocabulary, the two
 * machines, what they promise, and what the Queries ask.
 */
package temporal
package nexuscaller

import umpire.*
import io.temporal.api.command.v1.ScheduleNexusOperationCommandAttributes
import io.temporal.api.nexus.v1.{HandlerError, StartOperationResponse}
import worker.{Phase as WorkerPhase, State as WorkerState}

val Family: umpire.Family = umpire.Family("temporal.nexus.caller")

// Parties are names the feature declares by using them. The reserved party system is the server.
// A fault is an ordinary action of a declared party, and a timer is system behavior the machine
// owns, so neither is a separate kind.
val caller: Party = Party("caller")
val handler: Party = Party("handler")
val network: Party = Party("network")

// ### Entities
//
// An operation is scheduled by a caller workflow, and recorded data names one by its scheduled
// event: every history event of the operation carries that event's id.

val workflow: Entity = Entity("workflow")
val operation: Entity =
  Entity("operation", key = "scheduledEvent", refer = Map("caller" -> workflow))

// ### The input domains
//
given Finite[ProtocolState] =
  // The lifter reads this Int bound; the Go model evaluator uses it to enumerate protocol states.
  given Finite[Int] = Finite.upTo(Protocol.attemptBound)
  Finite.derived

/** What the completion's delivery reports. */
enum Delivery derives Finite:
  case accepted, notFound

// ### Actions

val schedule = action("schedule", caller)
  .input[Timeout]("scheduleToClose")
  .input[Timeout]("scheduleToStart")
  .input[Timeout]("startToClose")
  .creates(operation)
  .schema[ScheduleNexusOperationCommandAttributes]

val handlerReply = action("handlerReply", handler)
  .on(operation)
  .input[Reply]("reply")
  .schema[StartOperationResponse]
  .schema[HandlerError]
  .example(Reply.handlerError(false), "BadRequest")
  .example(Reply.handlerError(true), "Internal")

/**
 * The Nexus HTTP completion carries no protobuf message, so it declares no schema and its classes
 * are names the realization interprets.
 */
val complete =
  action("complete", handler).on(operation).input[Resolution]("resolution").results("Delivery")

val transportFault = action("transportFault", network) on operation

/**
 * The handler's worker stops polling. An action that names no entity is behavior no entity
 * records: the Run records the fault, but nothing recorded names the operation, so the machines
 * keep their state and record nothing at it.
 */
val workerStop = worker.workerStop

// ### The derived observation
//
// A retryable attempt failure writes no history event, so the attempt count is read back through
// DescribeWorkflowExecution. Every other evidence name resolves against the realization's catalog,
// which is why only a derived observation is declared.

val pendingAttempts: Observation = Observation("pendingAttempts", operation, "attempts")

// ### The product machine

val timeout = timer("timeout")

val nexusProduct: Machine[ProductState, Outcome, ProductFact] =
  machine[ProductState, Outcome, ProductFact](Family, "nexusProduct") {
    forEntity(operation)
    starts(ProductState(ProductPhase.scheduled))
    ends(Product.productTerminal)
    evidence {
      case ProductFact.nexusOperationScheduled => "nexusOperationScheduled"
      case ProductFact.nexusOperationStarted   => "nexusOperationStarted"
      case ProductFact.nexusOperationCompleted => "nexusOperationCompleted"
      case ProductFact.nexusOperationFailed    => "nexusOperationFailed"
      case ProductFact.nexusOperationCanceled  => "nexusOperationCanceled"
      case ProductFact.nexusOperationTimedOut  => "nexusOperationTimedOut"
    }
    steps(
      handlerReply ~> Product.handlerReplyStep,
      complete ~> Product.completeStep,
      transportFault ~> Product.transportFaultStep,
      workerStop ~> Product.workerStopStep,
      timeout ~> Product.timeoutStep
    )
  }

// ### The protocol machine

val backoff = timer("backoff")
val scheduleToClose = timer("scheduleToClose")
val scheduleToStart = timer("scheduleToStart")
val startToClose = timer("startToClose")

/** Where every path begins: before the operation exists, with every deadline at its first value. */
val unscheduled: ProtocolState =
  ProtocolState(Phase.unscheduled, 0, Timeout.unset, Timeout.unset, Timeout.unset)

val nexusProtocol: Machine[ProtocolState, Outcome, ProtocolFact] =
  machine[ProtocolState, Outcome, ProtocolFact](Family, "nexusProtocol") {
    forEntity(operation)
    refines(nexusProduct)(Protocol.productOf)
    starts(unscheduled)
    ends(s => Protocol.terminalPhase(s.phase))
    unobservable(backoff)
    evidence {
      case ProtocolFact.nexusOperationScheduled   => "nexusOperationScheduled"
      case ProtocolFact.nexusOperationStarted     => "nexusOperationStarted"
      case ProtocolFact.nexusOperationCompleted   => "nexusOperationCompleted"
      case ProtocolFact.nexusOperationFailed      => "nexusOperationFailed"
      case ProtocolFact.nexusOperationCanceled    => "nexusOperationCanceled"
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
val handlerWorker: Machine[WorkerState, worker.Outcome, worker.Fact] =
  worker.polling.restrict(Family, "handlerWorker")(worker.workerStop, worker.serve)

// ### The operation and the handler's worker
//
// The protocol machine's worker stop is a stutter row: the operation cannot see its handler's
// worker, so the schedule-to-start Scenario orders the stop before the request by convention.
// Composed with the worker of the handler's task queue, the stop is the worker's own phase change
// and every reply is the worker serving, so a reply has a row only while the worker polls. No
// functional Query reads the composition; it is what the cross-entity claim is verified over.

final case class NexusCallerState(operation: ProtocolState, worker: WorkerState)

val nexusCaller: Composition[NexusCallerState] =
  compose[NexusCallerState](Family, "nexusCaller")(
    "operation" -> nexusProtocol,
    "worker" -> handlerWorker
  )
    .sync("workerStop", "operation" -> workerStop, "worker" -> worker.workerStop)
    .sync("handlerReply", "operation" -> handlerReply, "worker" -> worker.serve)
    .ends(s => Protocol.terminalPhase(s.operation.phase))

val pollingWorker: WorkerState = WorkerState(WorkerPhase.polling)
