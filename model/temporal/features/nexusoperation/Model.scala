/* The standalone Nexus operation Model: one operation a caller starts directly through
 * StartNexusOperationExecution, with no workflow around it, grounded in
 * chasm/lib/nexusoperation/{operation.go,operation_statemachine.go}. The caller starts, cancels
 * and terminates it; the endpoint's handler answers it, synchronously or by starting it and
 * completing it later. Its status is read back through DescribeNexusOperationExecution. No retry
 * and no deadline is modeled: an attempt that fails retryably reads as scheduled, as BACKING_OFF
 * does in Describe (RUNNING), and no Case sets a deadline it lives to see.
 */
package temporal
package features.nexusoperation

import umpire.*
import io.temporal.api.workflowservice.v1.*
import OperationFamily.given

// Moved from temporal.nexusoperation; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexusoperation.Model$package$")

object OperationFamily:
  given family: Family = Family("temporal.nexusoperation.standalone")

val caller: Party = Party()
val handler: Party = Party()

/** Named by the id the caller chose: every request and read carries it. */
val operation: Entity = Entity(key = "operationId")

/**
 * The handler's answer to the start: a result, a failure or a cancel at once, or an async start.
 */
enum Reply derives Finite:
  case syncSuccess, syncFailure, syncCanceled, async

/** How a started operation's completion settles it. */
enum Resolution derives Finite:
  case succeeded, failed, canceled

object Inputs:
  val reply = input[Reply]
  val resolution = input[Resolution]

val start = action(caller).creates(operation).schema[StartNexusOperationExecutionRequest]
val requestCancel =
  action(caller).on(operation).schema[RequestCancelNexusOperationExecutionRequest]
val terminate = action(caller).on(operation).schema[TerminateNexusOperationExecutionRequest]
val handlerReply = action(handler).on(operation).input(Inputs.reply)
val complete = action(handler).on(operation).input(Inputs.resolution)

/**
 * A step's outcome. A control of a closed operation is `alreadyCompleted`, the FailedPrecondition
 * ErrOperationAlreadyCompleted (operation.go). A Case's controls carry the run as their request id,
 * so a control it repeats is the server's same request, which it answers as it did the first.
 */
enum Outcome derives Finite:
  case accepted, alreadyCompleted

given Accepted[Outcome] = Accepted(Outcome.accepted)

/** The statuses DescribeNexusOperationExecution reports: scheduled and started read RUNNING. */
enum Phase derives Finite:
  case unstarted, scheduled, started, succeeded, failed, canceled, terminated

/** 7 phases and whether a cancel was requested: 14 states. */
final case class OperationState(phase: Phase, cancelRequested: Boolean) derives Finite

/** What a step records: the status it lands in, and the cancel request Describe reports. */
enum OperationFact derives Finite:
  case statusScheduled, statusStarted, statusCancelRequested
  case statusSucceeded, statusFailed, statusCanceled, statusTerminated

type OperationStep = Step[OperationState, Outcome, OperationFact]

object Operation:
  import Phase.*
  import OperationFact.*

  def phase(s: OperationState): Phase = s.phase
  def terminal(p: Phase): Boolean = p.in(succeeded, failed, canceled, terminated)
  def ends(s: OperationState): Boolean = terminal(s.phase)

  private def closed(s: OperationState): List[OperationStep] =
    List(Step(Outcome.alreadyCompleted, s))
  private def repeated(s: OperationState): List[OperationStep] = List(Step(Outcome.accepted, s))

  def start(s: OperationState): List[OperationStep] =
    if s.phase != unstarted then disabled
    else accept(OperationState(scheduled, false), statusScheduled)

  /**
   * TransitionStarted, or a synchronous completion straight from scheduled; a canceled answer settles
   * it canceled (operation.go invocationResultCancel, onCanceled).
   */
  def handlerReply(s: OperationState, r: Reply): List[OperationStep] =
    if s.phase != scheduled then disabled
    else
      r match
        case Reply.syncSuccess  => accept(s.copy(phase = succeeded), statusSucceeded)
        case Reply.syncFailure  => accept(s.copy(phase = failed), statusFailed)
        case Reply.syncCanceled => accept(s.copy(phase = canceled), statusCanceled)
        case Reply.async        => accept(s.copy(phase = started), statusStarted)

  /** An async operation's completion; a canceled failure settles it canceled. */
  def complete(s: OperationState, r: Resolution): List[OperationStep] =
    if s.phase != started then disabled
    else
      r match
        case Resolution.succeeded => accept(s.copy(phase = succeeded), statusSucceeded)
        case Resolution.failed    => accept(s.copy(phase = failed), statusFailed)
        case Resolution.canceled  => accept(s.copy(phase = canceled), statusCanceled)

  /**
   * RequestCancel records the request and leaves the operation live: it is sent to the handler
   * only once started (operation.go RequestCancel). A repeated request is the same request; one of
   * a closed operation is alreadyCompleted, unless it repeats one the operation took.
   */
  def requestCancel(s: OperationState): List[OperationStep] =
    if s.phase == unstarted then disabled
    else if s.cancelRequested then repeated(s)
    else if terminal(s.phase) then closed(s)
    else accept(s.copy(cancelRequested = true), statusCancelRequested)

  /**
   * Terminate settles a live operation terminated (TransitionTerminated); a repeated terminate of a
   * terminated one is the same request, answered OK; any other control of a closed one is
   * alreadyCompleted (operation.go Terminate).
   */
  def terminate(s: OperationState): List[OperationStep] =
    if s.phase == unstarted then disabled
    else if s.phase == terminated then repeated(s)
    else if terminal(s.phase) then closed(s)
    else accept(s.copy(phase = terminated), statusTerminated)

val nexusOperation = machine[OperationState, Outcome, OperationFact] {
  forEntity(operation)
  starts(OperationState(Phase.unstarted, false))
  ends(Operation.ends)
  steps(
    start ~> Operation.start,
    handlerReply ~> Operation.handlerReply,
    complete ~> Operation.complete,
    requestCancel ~> Operation.requestCancel,
    terminate ~> Operation.terminate
  )
}
