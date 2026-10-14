// What a Temporal realization says that only Temporal has: the roles of a Temporal deployment and
// their kinds, the worker activations a script runs in, the instructions a worker carries out or a
// run works on it, the history a run reads its evidence from, the dynamic configuration the
// server runs under, and how Temporal's APIs behave between calls.
//
// Each declaration extends an open trait of the framework's realization vocabulary
// (model/framework/realize), which knows no system. As there, the lifter emits a declaration into the
// IR as written, each class by its simple name and each parameter by its own
// (model/irgen/Realizations.scala), and a default is the empty value, which the IR leaves unset.
package temporal.realize

import io.grpc.MethodDescriptor
import io.temporal.api.common.v1.Payloads
import scalapb.GeneratedMessage
import framework.ClassRef
import framework.realize.{
  Activation,
  Addressee,
  Behavior,
  Command,
  Field,
  Instruction,
  KeyedRef,
  Name,
  Recorded,
  Script,
  Setting,
  SystemStep,
  TypedProto
}

// A dynamic-configuration setting the server must run under, such as the flag that enables a
// feature: its key and value as the server's dynamic configuration spells them.
final case class RequiredSetting(key: String, value: String) extends Setting

enum RoleKind:
  case endpoint, worker, taskQueue, participant

// A role of a Temporal deployment: an endpoint such as the frontend's WorkflowService, a worker, or
// a task queue. `namespace` and `resource` name the environment bindings a run supplies.
final case class Role(id: String, kind: RoleKind, namespace: String = "", resource: String = "")
    extends Addressee

// The worker activations a script runs in, besides the controller.
enum WorkerActivation extends Activation:
  case Workflow(workflowType: Name, worker: String | Role, taskQueue: String | Role)
  case NexusHandler(
      service: String,
      operation: String,
      worker: String | Role,
      taskQueue: String | Role
  )

  // Each attempt of an activity the worker is delivered. `starts` is the classes that delivery is: a
  // step of one of them is the activation itself, and no command performs it. The commands the path
  // places in the script are the attempts in order.
  case Activity(
      activityType: Name,
      worker: String | Role,
      taskQueue: String | Role,
      starts: Vector[ClassRef] = Vector.empty
  )

enum FaultKind:
  case workerStop, workerResume, admissionResponseLoss

// What a worker answers the activation it runs in with, and what a run does to a worker. A
// workflow or the attempt of an activity completes with the framework's `Instruction.Finish`.
enum WorkerInstruction extends Instruction:
  // Fails the attempt of an activity, with a `temporal.api.failure.v1.Failure`.
  case AttemptFailure(failure: TypedProto[?])

  // Answers the attempt of an activity as canceled.
  case AttemptCanceled

  // Records details through the worker SDK; Describe supplies the server receipt separately.
  case AttemptHeartbeat(details: TypedProto[Payloads])

  // Offers no answer; the attempt's declared server deadline ends it.
  case AttemptWithheld(
      mode: WithholdingMode = WithholdingMode.context,
      externalSettlement: Option[Instruction.TypedRpc[?, ?]] = None
  )

  case AwaitActivityPublication(publication: ActivityPublication)

  // A deliberate outage of the worker that polls a task-queue role.
  case Fault(role: String | Role, kind: FaultKind)

  // A workflow command, as the message the SDK would emit.
  case WorkflowCommand(command: TypedProto[?])

  // A Nexus handler's answer; an asynchronous one binds the handle `binds` names.
  case NexusReply(reply: TypedProto[?], binds: String | framework.realize.Learned = "")

  // Completes the asynchronous Nexus operation a handle names, with a payload or a failure.
  case NexusCompletion(handle: String | framework.realize.Learned, result: TypedProto[?])

// A workflow's history, as evidence is read from it.
object WorkflowHistory:
  // One member of the history event's attributes.
  final case class TypedHistory[Root <: GeneratedMessage, Value](
      attributes: Field[Root, Value]
  ) extends Recorded

  // The history events whose attributes have the member `attributes`, which `Evidence.keyed` reads:
  // `WorkflowHistory.event(Field(_.attributes.nexusOperationStartedEventAttributes))`.
  def event[Root <: GeneratedMessage, Value](attributes: Field[Root, Value]): KeyedRef[Root] =
    KeyedRef(TypedHistory(attributes))

// ### How Temporal's APIs behave between calls (.plans/API_BEHAVIOR_HINTS.md)
//
// A hint is a fact about an API, declared once in Behavior.scala with the server code it rests on,
// and attached to every Temporal realization by `temporalRealization`. The lifter writes each hint
// with an id derived from what it relates, and with the line it is declared at, which a lowered
// wait names when it runs out.

// How long a condition may take to hold, and how often a wait looks, in milliseconds.
final case class WaitBound(intervalMs: Long, atMostMs: Long)

// When the effect of a write is visible to a read.
enum Visible:
  // In the write's own transaction: a read after the write reads once.
  case atOnce

  // Only after the write returns, within `bound`: a read after the write waits for its condition.
  case eventually(bound: WaitBound)

// A kind of asynchronous cause: something a read waits for that no command of its script does.
enum WithholdingMode:
  case context, sdkPending

enum TimeoutBasis:
  case unspecified, scheduleToClose, startToClose, heartbeat

enum CauseKind:
  // An activity script's answer: RespondActivityTaskCompleted, Failed or Canceled.
  case activityAnswer

  // An SDK heartbeat, whose invocation is distinct from its persisted server receipt.
  case activityHeartbeat

  // A workflow script's commands: RespondWorkflowTaskCompleted.
  case workflowTask

  // A Nexus handler script's reply: RespondNexusTaskCompleted or Failed.
  case handlerReply

  // The server's dispatch of a task to a worker.
  case delivery

  // A server timer at a deadline the realization set.
  case timer

// When the effect of `write`, a method the API binds to HTTP POST or a cause no call is, is visible
// to `read`, a method the API binds to HTTP GET: `write.visibleTo(read, when)`.
final class Visibility private[realize] (
    val write: MethodDescriptor[?, ?] | CauseKind,
    val read: MethodDescriptor[?, ?],
    val when: Visible
)

// How long one kind of cause may take: `kind.boundedBy(bound)`.
final class CauseBound private[realize] (val kind: CauseKind, val bound: WaitBound)

// How the server numbers the attempts of one activity: the Nth delivery is the attempt numbered
// `first + N - 1`, and with `oneRun` every attempt belongs to the activity's one run.
final case class AttemptNumbering(first: Long, oneRun: Boolean)

// The limits of one instruction: its deadline in milliseconds, and the attempts it may take.
final case class InstructionLimit(timeoutMs: Long, attempts: Long)

// The behavior every Temporal realization carries: Behavior.scala declares it once. Besides the
// hints, how attempts are numbered, the limits of an instruction no hint bounds and that writes
// none, and whether a run's record order is the causal order of one operation's evidence across
// sources. A lifter fixture leaves them out, and the IR leaves them unset.
final case class ApiBehavior(
    visibility: Vector[Visibility],
    causes: Vector[CauseBound],
    attemptNumbering: Option[AttemptNumbering] = None,
    instructionDefaults: Option[InstructionLimit] = None,
    runOrderIsCausal: Boolean = false
) extends Behavior

// A step class no command performs, and the kind of cause it is: an activity's `poll` is a
// delivery, a timeout class a timer. A timer carries the deadline, in milliseconds, its request set
// from the same kit value; the wait for it is that deadline plus the timer's bound.
final case class ServerStep(
    step: ClassRef,
    kind: CauseKind,
    deadlineMs: Long = 0,
    timeoutBasis: TimeoutBasis = TimeoutBasis.unspecified
) extends SystemStep

final case class ActivityPublication(id: String)

final case class ActivityExternalSettlement[
    Req <: GeneratedMessage,
    Rsp <: GeneratedMessage
](
    carrier: Command | Instruction,
    activity: Script,
    attempt: Long,
    pending: ActivityPublication,
    held: Command | Instruction,
    answer: Instruction.TypedRpc[Req, Rsp],
    settlement: Command | Instruction,
    cleanup: Command,
    requestCancel: Option[Instruction.TypedRpc[
      io.temporal.api.workflowservice.v1.RequestCancelActivityExecutionRequest,
      io.temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse
    ]] = None
) extends SystemStep

object ActivityExternalSettlement:
  final case class Scheduled(
      carrier: Command | Instruction,
      answer: Instruction.TypedRpc[
        io.temporal.api.workflowservice.v1.RespondActivityTaskCompletedByIdRequest,
        io.temporal.api.workflowservice.v1.RespondActivityTaskCompletedByIdResponse
      ],
      settlement: Command | Instruction,
      cleanup: Command
  ) extends SystemStep

// A held attempt's pending publication, then the controller's reset of it, which the selected
// per-attempt `timer` applies. The server's next delivery is a first attempt again, the attempt group
// `freshAttempt` runs; the activity script's numbering restarts there, never by relabeling one.
final case class ActivityResetSettlement(
    carrier: Command | Instruction,
    activity: Script,
    attempt: Long,
    pending: ActivityPublication,
    held: Command | Instruction,
    resetRequest: Instruction.TypedRpc[
      io.temporal.api.workflowservice.v1.ResetActivityExecutionRequest,
      io.temporal.api.workflowservice.v1.ResetActivityExecutionResponse
    ],
    timer: ClassRef,
    freshAttempt: Long,
    settlement: Command | Instruction,
    cleanup: Command
) extends SystemStep

extension [Req <: GeneratedMessage, Rsp <: GeneratedMessage](write: MethodDescriptor[Req, Rsp])
  // That the effect of the call `write` is visible to `read` `when`.
  def visibleTo[RReq <: GeneratedMessage, RRsp <: GeneratedMessage](
      read: MethodDescriptor[RReq, RRsp],
      when: Visible
  ): Visibility = Visibility(write, read, when)

extension (cause: CauseKind)
  // That the effect of a cause of this kind is visible to `read` `when`.
  def visibleTo[RReq <: GeneratedMessage, RRsp <: GeneratedMessage](
      read: MethodDescriptor[RReq, RRsp],
      when: Visible
  ): Visibility = Visibility(cause, read, when)

  // That a cause of this kind takes at most `bound` from the step before it.
  def boundedBy(bound: WaitBound): CauseBound = CauseBound(cause, bound)
