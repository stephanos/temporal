/* The standalone Nexus operation Model: one operation a caller starts directly through
 * StartNexusOperationExecution, with no workflow around it, grounded in
 * chasm/lib/nexusoperation/{operation.go,operation_statemachine.go}. The caller starts, cancels
 * and terminates it; the endpoint's handler answers it, synchronously or by starting it and
 * completing it later. Its status is read back through DescribeNexusOperationExecution. No retry
 * and no deadline is modeled: an attempt that fails retryably reads as scheduled, as BACKING_OFF
 * does in Describe (RUNNING), and no Case sets a deadline it lives to see.
 *
 * Read top to bottom: the types; the signature (the caller, the handler, the operation and their
 * actions); then Operation, the operation's one machine, with its own reading of closed rejection
 * and the capabilities it declares; and last Files, its IR file. Realization.scala realizes it.
 */
package temporal
package features.nexusoperation

import umpire.*
import umpire.realize.Reason
import temporal.capabilities.{given, *}
import temporal.realize.inconclusive
import io.temporal.api.workflowservice.v1.*
import shared.Bounds.three
import OperationFamily.given

// Moved from temporal.nexusoperation; the pin keeps its Definition IDs and type names.
given DefinitionScope = DefinitionScope("temporal.nexusoperation.Model$package$")

object OperationFamily:
  given family: Family = Family("temporal.nexusoperation.standalone")

// ### Types

/**
 * The handler's answer to the start: a result, a failure or a cancel at once, or an async start.
 */
enum Reply derives Finite:
  case syncSuccess, syncFailure, syncCanceled, async

/** How a started operation's completion settles it. */
enum Resolution derives Finite:
  case succeeded, failed, canceled

/**
 * A step's outcome. A control of a closed operation is `alreadyCompleted`, the FailedPrecondition
 * ErrOperationAlreadyCompleted (operation.go). A Case's controls carry the run as their request id,
 * so a control it repeats is the server's same request, which it answers as it did the first.
 */
enum Outcome derives Finite:
  case accepted, alreadyCompleted

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

// ### Signature

val caller: Party = Party()
val handler: Party = Party()

/** Named by the id the caller chose: every request and read carries it. */
val operation: Entity = Entity(key = "operationId")

object Inputs:
  val reply = input[Reply]
  val resolution = input[Resolution]

val start = action(caller).creates(operation).schema[StartNexusOperationExecutionRequest]
val requestCancel =
  action(caller).on(operation).schema[RequestCancelNexusOperationExecutionRequest]
val terminate = action(caller).on(operation).schema[TerminateNexusOperationExecutionRequest]
val handlerReply = action(handler).on(operation).input(Inputs.reply)
val complete = action(handler).on(operation).input(Inputs.resolution)

given Ok[Outcome] = Ok(Outcome.accepted)

// ### The operation

object Operation:
  import Phase.*

  def phase(s: OperationState): Phase = s.phase
  def terminal(p: Phase): Boolean = p.in(succeeded, failed, canceled, terminated)
  def end(s: OperationState): Boolean = terminal(s.phase)

  object effects:
    import OperationFact.*

    private def closed(s: OperationState): List[OperationStep] =
      List(Step(Outcome.alreadyCompleted, s))
    private def repeated(s: OperationState): List[OperationStep] = List(Step(Outcome.accepted, s))

    def start(s: OperationState): List[OperationStep] =
      if s.phase != unstarted then disabled
      else enter(OperationState(scheduled, false), statusScheduled)

    /**
     * TransitionStarted, or a synchronous completion straight from scheduled; a canceled answer
     * settles it canceled (operation.go invocationResultCancel, onCanceled).
     */
    def handlerReply(s: OperationState, r: Reply): List[OperationStep] =
      if s.phase != scheduled then disabled
      else
        r match
          case Reply.syncSuccess  => enter(s.copy(phase = succeeded), statusSucceeded)
          case Reply.syncFailure  => enter(s.copy(phase = failed), statusFailed)
          case Reply.syncCanceled => enter(s.copy(phase = canceled), statusCanceled)
          case Reply.async        => enter(s.copy(phase = started), statusStarted)

    /** An async operation's completion; a canceled failure settles it canceled. */
    def complete(s: OperationState, r: Resolution): List[OperationStep] =
      if s.phase != started then disabled
      else
        r match
          case Resolution.succeeded => enter(s.copy(phase = succeeded), statusSucceeded)
          case Resolution.failed    => enter(s.copy(phase = failed), statusFailed)
          case Resolution.canceled  => enter(s.copy(phase = canceled), statusCanceled)

    /**
     * RequestCancel records the request and leaves the operation live: it is sent to the handler
     * only once started (operation.go RequestCancel). A repeated request is the same request; one of
     * a closed operation is alreadyCompleted, unless it repeats one the operation took.
     */
    def requestCancel(s: OperationState): List[OperationStep] =
      if s.phase == unstarted then disabled
      else if s.cancelRequested then repeated(s)
      else if terminal(s.phase) then closed(s)
      else enter(s.copy(cancelRequested = true), statusCancelRequested)

    /**
     * Terminate settles a live operation terminated (TransitionTerminated); a repeated terminate of
     * a terminated one is the same request, answered OK; any other control of a closed one is
     * alreadyCompleted (operation.go Terminate).
     */
    def terminate(s: OperationState): List[OperationStep] =
      if s.phase == unstarted then disabled
      else if s.phase == terminated then repeated(s)
      else if terminal(s.phase) then closed(s)
      else enter(s.copy(phase = terminated), statusTerminated)

  val nexusOperation = machine[OperationState, Outcome, OperationFact] {
    forEntity(operation)
    starts(OperationState(Phase.unstarted, false))
    ends(end)
    steps(
      start ~> effects.start,
      handlerReply ~> effects.handlerReply,
      complete ~> effects.complete,
      requestCancel ~> effects.requestCancel,
      terminate ~> effects.terminate
    )
  }

  /**
   * What the operation promises of its own: its reading of closed rejection, which its capabilities
   * put in place of the law's.
   */
  object properties:
    /**
     * A closed operation keeps its state, and answers a control alreadyCompleted, or OK where it
     * repeats a request the operation took, a recorded cancel or the terminate that closed it: the
     * operation's own reading of closedIsRejectedUniformly.
     */
    def closedRejectsOrRepeats(m: Machine[OperationState, Outcome, OperationFact])(
        status: OperationState => Phase,
        terminal: Phase => Boolean,
        rejected: Outcome
    ): Property[OperationState] =
      m.property holdsAcross ((before, after) =>
        !terminal(status(before)) ||
          (after.state == before && (after.outcome == rejected || after.outcome == Outcome.accepted &&
            (before.cancelRequested || status(before) == Phase.terminated)))
      )

  /**
   * What the operation is, as the laws of model/temporal/capabilities read it: it closes, a caller
   * terminates it and requests its cancel, and DescribeNexusOperationExecution reports its status.
   * It receives the laws without listing them, each named `nexusOperation.<law>`. It reads the
   * realization, which reads this machine, so it waits in an object of its own until it is used.
   */
  object laws:
    /**
     * Why the operation overrides closedIsRejectedUniformly: the server answers a control that
     * repeats a request id the operation took OK, after it closed too (operation.go RequestCancel,
     * Terminate).
     */
    val repeatedRequestsAnswer =
      "a repeated request id is answered OK after close: operation.go RequestCancel and Terminate"

    /**
     * The rejection is the operation's own: alreadyCompleted, a FailedPrecondition, where the
     * activity answers NotFound (operation.go ErrOperationAlreadyCompleted). Each functional law's
     * find starts the operation, which no handler answers, so it stays running, then takes the
     * control. A Run explains an unobserved control of a closed operation too, which records nothing,
     * so a Run of a terminate or cancel find leaves the claim inconclusive: its explanations disagree.
     */
    val operationCapabilities = capabilities(nexusOperation, limits = three)(
      Closable(
        status = Operation.phase,
        terminal = Operation.terminal,
        rejected = cited(Outcome.alreadyCompleted, "chasm/lib/nexusoperation/operation.go")
      ),
      Terminable(
        terminate = terminate,
        settled = OperationFact.statusTerminated,
        reach = Seq(start),
        expect = inconclusive(Reason.explanationsDisagree)
      ),
      Cancelable(
        requestCancel = requestCancel,
        requested = OperationFact.statusCancelRequested,
        reach = Seq(start),
        expect = inconclusive(Reason.explanationsDisagree)
      ),
      Describable(status = OperationRealization.operationStatus)
    ).overriding(
      closedIsRejectedUniformly -> properties.closedRejectsOrRepeats,
      because = repeatedRequestsAnswer
    )

// ### The checked-in IR file of the standalone Nexus operation Model (umpire.irFile).

object Files:
  val nexusOperationFile = irFile("nexus-operation")(
    Operation.nexusOperation,
    Operation.laws.operationCapabilities,
    OperationRealization.standalone
  )
