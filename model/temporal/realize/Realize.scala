/* What a Temporal realization says that only Temporal has: the roles of a Temporal deployment and
 * their kinds, the worker activations a script runs in, the instructions a worker carries out or a
 * run works on it, the history a run reads its evidence from, and the dynamic configuration the
 * server runs under.
 *
 * Each declaration extends an open trait of the framework's realization vocabulary
 * (model/umpire/realize), which knows no system. As there, the lifter emits a declaration into the
 * IR as written, each class by its simple name and each parameter by its own
 * (model/lifter/Realizations.scala), and a default is the empty value, which the IR leaves unset.
 */
package temporal.realize

import scalapb.GeneratedMessage
import umpire.ClassRef
import umpire.realize.{
  Activation,
  Addressee,
  Field,
  Instruction,
  KeyedRef,
  Name,
  Recorded,
  Setting,
  TypedProto
}

/**
 * A dynamic-configuration setting the server must run under, such as the flag that enables a
 * feature: its key and value as the server's dynamic configuration spells them.
 */
final case class RequiredSetting(key: String, value: String) extends Setting

enum RoleKind:
  case endpoint, worker, taskQueue, participant

/**
 * A role of a Temporal deployment: an endpoint such as the frontend's WorkflowService, a worker, or
 * a task queue. `namespace` and `resource` name the environment bindings a run supplies.
 */
final case class Role(id: String, kind: RoleKind, namespace: String = "", resource: String = "")
    extends Addressee

/** The worker activations a script runs in, besides the controller. */
enum WorkerActivation extends Activation:
  case Workflow(workflowType: Name, worker: String | Role, taskQueue: String | Role)
  case NexusHandler(
      service: String,
      operation: String,
      worker: String | Role,
      taskQueue: String | Role
  )

  /**
   * Each attempt of an activity the worker is delivered. `starts` is the classes that delivery is: a
   * step of one of them is the activation itself, and no command performs it. The commands the path
   * places in the script are the attempts in order.
   */
  case Activity(
      activityType: Name,
      worker: String | Role,
      taskQueue: String | Role,
      starts: Vector[ClassRef] = Vector.empty
  )

enum FaultKind:
  case workerStop, workerResume, admissionResponseLoss

/**
 * What a worker answers the activation it runs in with, and what a run does to a worker. A
 * workflow or the attempt of an activity completes with the framework's `Instruction.Finish`.
 */
enum WorkerInstruction extends Instruction:
  /** Fails the attempt of an activity, with a `temporal.api.failure.v1.Failure`. */
  case AttemptFailure(failure: TypedProto[?])

  /** Answers the attempt of an activity as canceled. */
  case AttemptCanceled

  /** A deliberate outage of the worker that polls a task-queue role. */
  case Fault(role: String | Role, kind: FaultKind)

  /** A workflow command, as the message the SDK would emit. */
  case WorkflowCommand(command: TypedProto[?])

  /** A Nexus handler's answer; an asynchronous one binds the handle `binds` names. */
  case NexusReply(reply: TypedProto[?], binds: String = "")

  /** Completes the asynchronous Nexus operation a handle names, with a payload or a failure. */
  case NexusCompletion(handle: String, result: TypedProto[?])

/** A workflow's history, as evidence is read from it. */
object WorkflowHistory:
  /** One member of the history event's attributes. */
  final case class TypedHistory[Root <: GeneratedMessage, Value](
      attributes: Field[Root, Value]
  ) extends Recorded

  /**
   * The history events whose attributes have the member `attributes`, which `Evidence.keyed` reads:
   * `WorkflowHistory.event(Field(_.attributes.nexusOperationStartedEventAttributes))`.
   */
  def event[Root <: GeneratedMessage, Value](attributes: Field[Root, Value]): KeyedRef[Root] =
    KeyedRef(TypedHistory(attributes))
