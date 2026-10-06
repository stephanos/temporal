// The standalone Nexus operation Model: one operation a client starts directly through
// StartNexusOperationExecution, with no workflow around it, grounded in
// chasm/lib/nexusoperation/{operation.go,operation_statemachine.go}. The client starts, cancels
// and terminates it; the endpoint's handler answers it, synchronously or by starting it and
// completing it later. Its status is read back through DescribeNexusOperationExecution. No retry
// and no deadline is modeled: an attempt that fails retryably reads as scheduled, as BACKING_OFF
// does in Describe (RUNNING), and no Case sets a deadline it lives to see.
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// Read top to bottom: the types; the signature (the client, the handler, the operation and their
// actions); then NexusOperation, the operation's one machine, with its own reading of closed
// rejection and the capabilities it implements; and last exports, its IR file. Realization.scala
// realizes it.
package temporal
package features.nexus.standalone

import scala.annotation.unused
import umpire.*
import umpire.realize.Reason
import temporal.capabilities.{given, *}
import temporal.realize.inconclusive
import io.temporal.api.workflowservice.v1.*
import shared.Bounds.three

// ### Types

// The handler's answer to the start: a result, a failure or a cancel at once, or an async start.
enum Reply derives Finite:
  case syncSuccess, syncFailure, syncCanceled, async

// How a started operation's completion settles it.
enum Resolution derives Finite:
  case succeeded, failed, canceled

// A step's outcome. A control of a closed operation is `alreadyCompleted`, the FailedPrecondition
// ErrOperationAlreadyCompleted (operation.go). A Case's controls carry the run as their request id,
// so a control it repeats is the server's same request, which it answers as it did the first.
enum Outcome derives Finite:
  case accepted, alreadyCompleted

// The statuses DescribeNexusOperationExecution reports: scheduled and started read RUNNING.
enum Phase derives Finite:
  case unstarted, scheduled, started, succeeded, failed, canceled, terminated

// 7 phases and whether a cancel was requested: 14 states.
final case class OperationState(phase: Phase, cancelRequested: Boolean) derives Finite

// What a step records: the status it lands in, and the cancel request Describe reports.
enum OperationFact derives Finite:
  case statusScheduled, statusStarted, statusCancelRequested
  case statusSucceeded, statusFailed, statusCanceled, statusTerminated

// ### Signature

// Named by the id the client chose: every request and read carries it.
val operation: Entity = Entity(key = "operationId")

// The handler's reply to the start and its completion's resolution.
object Inputs:
  val reply = input[Reply]
val resolution = input[Resolution]

// Who acts: each action is declared in the object of who takes it, and named after where it is
// declared, `temporal.features.nexus.standalone.client.start`.

// The client starts the operation, and requests its cancel or terminates it.
object client extends Client:
  val start = action(this).creates(operation).schema[StartNexusOperationExecutionRequest]
  val requestCancel =
    action(this).on(operation).schema[RequestCancelNexusOperationExecutionRequest]
  val terminate = action(this).on(operation).schema[TerminateNexusOperationExecutionRequest]

// The endpoint's handler replies to the start, and completes an operation it started async.
object handler extends Actor:
  val reply = action(this).on(operation).input(Inputs.reply)
  val complete = action(this).on(operation).input(resolution)

given Ok[Outcome] = Ok(Outcome.accepted)

// ### The operation

object NexusOperation extends Machine[OperationState, Outcome, OperationFact]:
  import Phase.*

  val init = OperationState(phase = unstarted, cancelRequested = false)
  def end(s: State) = states.over(s)

  // The operation's status sets.
  object states:
    def phase(s: State): Phase = s.phase

    def terminal(p: Phase): Boolean = p.in(succeeded, failed, canceled, terminated)

    // A path ends where the operation is over: `end` reads this named predicate.
    def over(s: State): Boolean = terminal(s.phase)

    // Started and not over: the phases a control settles or records a request in.
    def live(p: Phase): Boolean = p.in(scheduled, started)

    // Every phase after the start, live or over: the phases a control is answered in.
    def created(p: Phase): Boolean = live(p) || terminal(p)

  object effects:
    import OperationFact.*

    def start(@unused s: State) =
      enter(OperationState(phase = scheduled, cancelRequested = false), statusScheduled)

    // TransitionStarted, or a synchronous completion straight from scheduled; a canceled answer
    // settles it canceled (operation.go invocationResultCancel, onCanceled).
    def reply(s: State, r: Reply) =
      r match
        case Reply.syncSuccess  => enter(s.copy(phase = succeeded), statusSucceeded)
        case Reply.syncFailure  => enter(s.copy(phase = failed), statusFailed)
        case Reply.syncCanceled => enter(s.copy(phase = canceled), statusCanceled)
        case Reply.async        => enter(s.copy(phase = started), statusStarted)

    // An async operation's completion; a canceled failure settles it canceled.
    def complete(s: State, r: Resolution) =
      r match
        case Resolution.succeeded => enter(s.copy(phase = succeeded), statusSucceeded)
        case Resolution.failed    => enter(s.copy(phase = failed), statusFailed)
        case Resolution.canceled  => enter(s.copy(phase = canceled), statusCanceled)

    // RequestCancel records the request and leaves the operation live: it is sent to the handler
    // only once started (operation.go RequestCancel).
    def requestCancel(s: State) = enter(s.copy(cancelRequested = true), statusCancelRequested)

    // Terminate settles a live operation terminated (TransitionTerminated).
    def terminate(s: State) = enter(s.copy(phase = terminated), statusTerminated)

    // A control of a closed operation is alreadyCompleted (operation.go).
    def closed(s: State) = reject(Outcome.alreadyCompleted, s)

    // A control that repeats a request the operation took is the same request, answered OK.
    def repeated(s: State) = stay(s)

  object rules extends Rules(_.phase):
    on(client.start)(in(unstarted) ~> effects.start)
    on(handler.reply)(in(scheduled) ~> effects.reply)
    on(handler.complete)(in(started) ~> effects.complete)

    // A repeated cancel request is the same request; one of a closed operation is alreadyCompleted,
    // unless it repeats one the operation took (operation.go RequestCancel).
    on(client.requestCancel) {
      in(states.created).where(_.cancelRequested) ~> effects.repeated
      in(states.terminal).where(!_.cancelRequested) ~> effects.closed
      in(states.live).where(!_.cancelRequested) ~> effects.requestCancel
    }

    // A repeated terminate of a terminated operation is the same request, answered OK; any other
    // control of a closed one is alreadyCompleted (operation.go Terminate).
    on(client.terminate) {
      in(terminated) ~> effects.repeated
      in(succeeded, failed, canceled) ~> effects.closed
      in(scheduled, started) ~> effects.terminate
    }

  // What the operation promises of its own: its reading of closed rejection, which its capabilities
  // put in place of the law's.
  object properties:
    // A closed operation keeps its state, and answers a control alreadyCompleted, or OK where it
    // repeats a request the operation took, a recorded cancel or the terminate that closed it: the
    // operation's own reading of closedIsRejectedUniformly.
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

  // What the operation is, as the laws of model/temporal/capabilities read it: it closes, a client
  // terminates it and requests its cancel, and DescribeNexusOperationExecution reports its status.
  // It receives the laws without listing them, each named `nexusOperation.<law>`. It reads the
  // realization, which reads this machine, so it waits in a section, which initializes on its first
  // use.
  //
  // The rejection is the operation's own: alreadyCompleted, a FailedPrecondition, where the activity
  // answers NotFound (operation.go ErrOperationAlreadyCompleted). Each functional law's find starts
  // the operation, which no handler answers, so it stays running, then takes the control. A Run
  // explains an unobserved control of a closed operation too, which records nothing, so a Run of a
  // terminate or cancel find leaves the claim inconclusive: its explanations disagree.
  object implements
      extends Implements(limits = three)(
        Closable(
          status = states.phase,
          terminal = states.terminal,
          rejected = cited(Outcome.alreadyCompleted, "chasm/lib/nexusoperation/operation.go")
        ),
        Terminable(
          terminate = client.terminate,
          settled = OperationFact.statusTerminated,
          reach = Seq(client.start),
          expect = inconclusive(Reason.explanationsDisagree)
        ),
        Cancelable(
          requestCancel = client.requestCancel,
          requested = OperationFact.statusCancelRequested,
          reach = Seq(client.start),
          expect = inconclusive(Reason.explanationsDisagree)
        ),
        Describable(status = OperationRealization.operationStatus)
      ):
    // Why the operation overrides closedIsRejectedUniformly: the server answers a control that
    // repeats a request id the operation took OK, after it closed too (operation.go RequestCancel,
    // Terminate).
    val repeatedRequestsAnswer =
      "a repeated request id is answered OK after close: operation.go RequestCancel and Terminate"

    overriding(
      closedIsRejectedUniformly -> properties.closedRejectsOrRepeats,
      because = repeatedRequestsAnswer
    )

// ### The checked-in IR file of the standalone Nexus operation Model (umpire.irFile).

object exports:
  val nexusStandalone = irFile("nexus-standalone")(
    NexusOperation,
    NexusOperation.implements,
    OperationRealization.standalone
  )
