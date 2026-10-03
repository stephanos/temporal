/* What a Model says about running its Queries against a system: the roles a run addresses, the
 * values it learns, what it observes and reads as evidence, the controls it needs, and the scripts
 * its controller and its workers follow.
 *
 * These are declarations and nothing else. No code here builds a Case: the lifter emits a
 * declaration into the IR as written, each class by its simple name and each parameter by its own,
 * and Go lowers one Query's witness through it (model/SEMANTICS.md, Realizations). A default
 * is therefore always the empty value, which the IR leaves unset.
 */
package umpire.realize

import io.grpc.MethodDescriptor
import scala.compiletime.error
import scalapb.{GeneratedEnum, GeneratedMessage, GeneratedMessageCompanion, UnrecognizedEnum}
import scala.util.NotGiven
import com.google.protobuf.ByteString
import umpire.{Channel, ClassRef, Machine}

/** How the find Queries of one machine run against a system. */
final case class Realization(
    name: String,
    machine: Machine[?, ?, ?],
    producer: String,
    producerVersion: String,
    roles: Vector[Role],
    correlation: Correlation,
    scripts: Vector[Script],
    learned: Vector[Learned] = Vector.empty,
    observations: Vector[Observed] = Vector.empty,
    evidence: Vector[Evidence | EvidenceRef[?, ?] | TypedEvidence[?]] = Vector.empty,
    controls: Vector[Control] = Vector.empty,
    cleanup: String = ""
)

enum RoleKind:
  case endpoint, worker, taskQueue, participant

/**
 * A symbolic participant commands and activations address. `namespace` and `resource` name the
 * environment bindings a run supplies.
 */
final case class Role(id: String, kind: RoleKind, namespace: String = "", resource: String = "")

enum LearnedKind:
  case text

  /** An effect handle the system issues, which only the command that completes the effect reads. */
  case handle

/** A value a run learns: bound once, by one command, and read by the commands that depend on it. */
final case class Learned(id: String, kind: LearnedKind)

/** A typed value a run records for its checks: one protobuf message, by its full name. */
final case class Observed(id: String, message: String)

object Observed:
  def apply[Message <: GeneratedMessage](id: String)(using
      companion: GeneratedMessageCompanion[Message]
  ): Observed = new Observed(id, companion.scalaDescriptor.fullName)

/** Where one kind of evidence is recorded. */
enum Recorded:
  /** One member of the history event's attributes. */
  case History(attributes: String)

  /** The elements of the repeated field at `path` in the response of a unary method. */
  case Read(method: String, path: String)

  /** The one message at `path` in the response of a unary method. */
  case Single(method: String, path: String)

  case TypedRead[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage,
      Projected <: GeneratedMessage
  ](
      method: MethodDescriptor[Req, Rsp],
      path: Field[Rsp, Seq[Projected]]
  ) extends Recorded
  case TypedSingle[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage,
      Projected <: GeneratedMessage
  ](
      method: MethodDescriptor[Req, Rsp],
      path: Field[Rsp, Projected]
  ) extends Recorded

  case TypedHistory[Root <: GeneratedMessage, Value](
      attributes: Field[Root, Value]
  ) extends Recorded

  /**
   * The Run's own record of the events of one kind that one command of a script records, where
   * `guard` holds of the event's payload, its instruction outcome. `key` names the operation an event
   * is of: the run's own id, or a path of the payload.
   *
   * `attempt` says the events are the record of one attempt of an activity, and a `diagnostic` always
   * names one. A Run records an attempt once it is answered, so the evidence reaches the Run with that
   * answer and not with the carrying command, as the events that record the attempt of that number.
   */
  case RunEvent(
      kind: EventKind,
      script: String,
      command: String,
      key: Operand,
      guard: Option[Operand] = None,
      attempt: Option[AttemptOf] = None
  )

  case TypedRunEvent[Root <: GeneratedMessage](
      kind: EventKind,
      script: String,
      command: String,
      key: ProjectedPath[Root, ?] | RunKey,
      guard: Option[Condition[Root]] = None,
      attempt: Option[AttemptOf] = None
  ) extends Recorded

object Recorded:
  def history[Root <: GeneratedMessage, Value](
      attributes: Field[Root, Value]
  ): HistoryRef[Root] = new HistoryRef(Recorded.TypedHistory(attributes))

  def read[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage,
      Projected <: GeneratedMessage
  ](
      method: MethodDescriptor[Req, Rsp],
      responseSelection: Field[Rsp, Seq[Projected]]
  ): RecordedRef[Req, Rsp, Projected] =
    new RecordedRef(Recorded.TypedRead(method, responseSelection))

  def single[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage,
      Projected <: GeneratedMessage
  ](
      method: MethodDescriptor[Req, Rsp],
      responseSelection: Field[Rsp, Projected]
  ): RecordedRef[Req, Rsp, Projected] =
    new RecordedRef(Recorded.TypedSingle(method, responseSelection))

  def runEvent[Root <: GeneratedMessage](
      kind: EventKind,
      script: String,
      command: String,
      key: ProjectedPath[Root, ?] | RunKey,
      guard: Option[Condition[Root]] = None,
      attempt: Option[AttemptOf] = None
  ): RunEventRef[Root] =
    new RunEventRef(
      Recorded.TypedRunEvent(kind, script, command, key, guard, attempt)
    )

/**
 * One attempt of the activity one script runs: the `number`-th delivery to the script's worker,
 * counted from one, as the server numbers attempts.
 */
final case class AttemptOf(script: String, number: Long)

/** The kinds of Run Event that carry an instruction outcome. */
enum EventKind:
  /** The command's own completion. */
  case instructionCompleted
  case instructionTimedOut

  /** What a worker reports of an activation the command carries, such as an attempt of an activity. */
  case diagnostic

/** What evidence commits to: what a caller was told, or a durable commit of the receiver. */
enum Commitment:
  case reported, durable

/** The identity a field of evidence names. */
enum FieldRole:
  /** The logical operation, as the kind's operation key names it. */
  case operation

  /** The execution attempt the fact belongs to. */
  case attempt

  /** The delivery the fact belongs to. */
  case delivery

/**
 * One field evidence carries, read at `path` in the recorded data. A field with a role names an
 * identity a check compares; a redacted field is carried without its value, and so names none.
 */
final case class EvidenceField(
    id: String,
    path: String,
    role: Option[FieldRole] = None,
    redacted: Boolean = false
)

/** One step of a path: the `occurrence`-th step, counted from one, of a class. */
final case class Taking(step: ClassRef, occurrence: Long)

/**
 * One kind of evidence: the recorded data that confirms the facts the machine's evidence function
 * names `records`, and the field that keys it to its operation, which a Run Event's evidence leaves
 * empty, since its source's key names the operation. An `exhaustive` kind's source reports
 * every occurrence of those facts for the operations its closing read covers, the read of the command
 * that names the kind in `closes`; only then does a fact nothing reports count as one that did not
 * happen.
 *
 * A kind confirms the one step of a path that records its fact. Where steps of several classes
 * record one fact, or a path takes one class more than once, each such step has evidence of its own:
 * a kind that names the steps it `confirms` is evidence of those and of no other, all of them by one
 * piece of evidence, and several such kinds may record one fact.
 */
final case class Evidence(
    id: String,
    records: String,
    source: String,
    from: Recorded,
    operation: String,
    commitment: Commitment,
    fields: Vector[EvidenceField] = Vector.empty,
    exhaustive: Boolean = false,
    confirms: Vector[Taking] = Vector.empty
)

object Evidence:
  def read[Req, Rsp, Projected](
      id: String,
      records: String,
      source: String,
      from: RecordedRef[Req, Rsp, Projected],
      operation: Field[Projected, ?],
      commitment: Commitment,
      fields: Vector[TypedEvidenceField[Projected, ?]] =
        Vector.empty[TypedEvidenceField[Projected, ?]],
      exhaustive: Boolean = false,
      confirms: Vector[Taking] = Vector.empty
  ): EvidenceRef[Req, Projected] =
    new EvidenceRef(
      Evidence(
        id,
        records,
        source,
        from.recorded,
        "",
        commitment,
        Vector.empty,
        exhaustive,
        confirms
      ),
      operation,
      fields
    )

  def history[Root <: GeneratedMessage](
      id: String,
      records: String,
      source: String,
      from: HistoryRef[Root],
      operation: Field[Root, ?],
      commitment: Commitment,
      fields: Vector[TypedEvidenceField[Root, ?]] = Vector.empty[TypedEvidenceField[Root, ?]],
      exhaustive: Boolean = false,
      confirms: Vector[Taking] = Vector.empty
  ): TypedEvidence[Root] = new TypedEvidence(
    Evidence(
      id,
      records,
      source,
      from.recorded,
      "",
      commitment,
      Vector.empty,
      exhaustive,
      confirms
    ),
    Some(operation),
    fields
  )

  def runEvent[Root <: GeneratedMessage](
      id: String,
      records: String,
      source: String,
      from: RunEventRef[Root],
      commitment: Commitment,
      fields: Vector[TypedEvidenceField[Root, ?]] = Vector.empty[TypedEvidenceField[Root, ?]],
      exhaustive: Boolean = false,
      confirms: Vector[Taking] = Vector.empty
  ): TypedEvidence[Root] = new TypedEvidence(
    Evidence(
      id,
      records,
      source,
      from.recorded,
      "",
      commitment,
      Vector.empty,
      exhaustive,
      confirms
    ),
    None,
    fields
  )

/** How a run's evidence is keyed into operations, and the window a check of it keeps. */
final case class Correlation(
    projection: String,
    run: String,
    operation: String,
    observation: String,
    events: Long,
    buffered: Long,
    keys: Long,
    support: Long,
    work: Long,
    eventSize: Long
)

enum ControlKind:
  /** Holds a channel's deliveries until a command releases them. */
  case HoldDelivery(channel: Channel[?])

  /**
   * Holds what a step of the class dispatches until a command releases it: the step has sent its
   * message, and nothing has delivered it.
   */
  case HoldDispatched(step: ClassRef)

/**
 * An actuator a run needs beyond its commands. `role` is the task-queue role whose deliveries a run
 * holds: a run reaches the channel through the deliveries of that queue.
 */
final case class Control(id: String, kind: ControlKind, role: String = "")

/**
 * A text a Case gets its own copy of: the prefix, then the Case's fixture name when `fixture` is
 * set, then the suffix.
 */
final case class Name(prefix: String, fixture: Boolean = false, suffix: String = "")

/** Who runs a script. */
enum Activation:
  case Controller
  case Workflow(workflowType: Name, worker: String, taskQueue: String)
  case NexusHandler(service: String, operation: String, worker: String, taskQueue: String)

  /**
   * Each attempt of an activity the worker is delivered. `starts` is the classes that delivery is: a
   * step of one of them is the activation itself, and no command performs it. The commands the path
   * places in the script are the attempts in order.
   */
  case Activity(
      activityType: Name,
      worker: String,
      taskQueue: String,
      starts: Vector[ClassRef] = Vector.empty
  )

/** One ordered list of commands and who runs it. */
final case class Script(id: String, activation: Activation, items: Vector[Item])

/**
 * One item of a script: a command every Case carries, a command only the Cases whose path performs
 * one of the `when` classes carry, or the place the path's steps of the classes `performs` binds
 * land, in path order.
 */
final case class Item(
    command: Option[Command] = None,
    when: Vector[ClassRef] = Vector.empty,
    performs: Vector[Performance] = Vector.empty
)

/** The command that performs one class of an action. */
final case class Performance(step: ClassRef, command: Command)

/** The commands of its script a command runs after. */
final case class After(commands: String*)

/**
 * One command of a script. Without `after` it runs after the command before it; `regardless` runs
 * it whatever became of the commands it runs after. `closes` names the exhaustive kinds of evidence
 * its read is the closing read of.
 */
final case class Command(
    id: String,
    instruction: Instruction,
    after: Option[After] = None,
    timeoutMs: Long = 0,
    regardless: Boolean = false,
    closes: Vector[String] = Vector.empty
)

enum FaultKind:
  case workerStop, workerResume, admissionResponseLoss

enum Cardinality:
  case one

  /** One value per element of a repeated path. */
  case each

/** Where a read value goes. */
enum Target:
  case Observe(observation: String)

  /** Binds a learned value. */
  case Bind(learned: String)

  /** Lifts the evidence kinds a history read confirms into an observation. */
  case Lift(observation: String)

final case class Assignment(target: String, value: Operand)

object Assignment:
  def typed[Root, Value](
      target: Field[Root, Value],
      value: TypedOperand[Value]
  ): TypedAssignment[Root, Value] =
    TypedAssignment(target, value)

object EvidenceField:
  def typed[Root, Value](
      id: String,
      path: Field[Root, Value],
      role: Option[FieldRole] = None,
      redacted: Boolean = false
  ): TypedEvidenceField[Root, Value] =
    TypedEvidenceField(id, path, role, redacted)

final case class ResponseRead(
    path: String,
    cardinality: Cardinality,
    targets: Vector[Target]
)

object ResponseRead:
  def typed[Root, Value](
      path: Field[Root, Value],
      cardinality: Cardinality,
      targets: Vector[Target]
  ): TypedResponseRead[Root, Value] =
    TypedResponseRead(path, cardinality, targets)

/** What a command does. */
enum Instruction:
  /** A unary call on an endpoint role. */
  case Rpc(
      role: String,
      method: String,
      assign: Vector[Assignment],
      reads: Vector[ResponseRead] = Vector.empty
  )

  case TypedRpc[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
      role: String,
      method: MethodDescriptor[Req, Rsp],
      assign: Vector[TypedAssignment[Req, ?]],
      reads: Vector[TypedResponseRead[Rsp, ?]]
  ) extends Instruction

  /** Polls the read an evidence kind names until an element satisfies `until`. */
  case Poll(
      evidence: String,
      role: String,
      assign: Vector[Assignment],
      until: Operand,
      intervalMs: Long
  )
  case TypedPoll[Req, Projected](
      evidence: EvidenceRef[Req, Projected],
      role: String,
      assign: Vector[TypedAssignment[Req, ?]],
      until: Condition[Projected],
      intervalMs: Long = 0
  ) extends Instruction
  case AwaitLearned(learned: String)

  /** Waits for the operation an earlier command of the script started. */
  case AwaitCommand(command: String)

  /** Completes with a result: a workflow, or the attempt of an activity. */
  case Finish(result: Operand)

  /** Fails the attempt of an activity, with a `temporal.api.failure.v1.Failure`. */
  case AttemptFailure(failure: Proto | TypedProto[?])

  /** Answers the attempt of an activity as canceled. */
  case AttemptCanceled
  case Fault(role: String, kind: FaultKind)

  /** A workflow command, as the message the SDK would emit. */
  case WorkflowCommand(command: Proto | TypedProto[?])

  /** A Nexus handler's answer; an asynchronous one binds the handle `binds` names. */
  case NexusReply(reply: Proto | TypedProto[?], binds: String = "")
  case NexusCompletion(handle: String, result: Proto | TypedProto[?])
  case Hold(control: String)
  case Release(control: String)

object Instruction:
  def rpc[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
      role: String,
      method: MethodDescriptor[Req, Rsp]
  )(
      assign: Vector[TypedAssignment[Req, ?]],
      reads: Vector[TypedResponseRead[Rsp, ?]]
  ): Instruction = TypedRpc(role, method, assign, reads)

  def poll[Req, Projected](evidence: EvidenceRef[Req, Projected], role: String)(
      assign: Vector[TypedAssignment[Req, ?]],
      until: Condition[Projected],
      intervalMs: Long = 0
  ): Instruction = TypedPoll(evidence, role, assign, until, intervalMs)

/** A value a command computes when it runs. */
enum Operand:
  case Literal(value: ProtoValue)
  case Environment(binding: String)

  /** The run's own id. */
  case Run
  case LearnedValue(learned: String)

  /** The value a poll is looking at, or the payload a Run Event's guard and key read. */
  case Projected
  case Path(of: Operand, path: String)
  case Present(of: Operand)
  case Equal(left: Operand, right: Operand)

  /** Holds when every operand does, read left to right up to the first that does not. */
  case All(operands: Operand*)

  /** Holds when the left integer is greater than the right. */
  case Greater(left: Operand, right: Operand)

  /** Holds when its operand, a condition, does not. */
  case Not(of: Operand)

object Operand:
  def run(): TypedOperand[String] = new TypedOperand(Run)
  def runKey(): RunKey = new RunKey(Run)
  def environment[Value](binding: String)(using
      Value =:= String
  ): TypedOperand[Value] =
    new TypedOperand(Environment(binding))
  def learnedValue[Value](learned: String)(using
      Value =:= String
  ): TypedOperand[Value] =
    new TypedOperand(LearnedValue(learned))
  def text(value: String): TypedOperand[String] = new TypedOperand(Literal(ProtoValue.Text(value)))
  def flag(value: Boolean): TypedOperand[Boolean] =
    new TypedOperand(Literal(ProtoValue.Flag(value)))
  def number(value: Long): TypedOperand[Long] =
    new TypedOperand(Literal(ProtoValue.Number(value)))
  def integer(value: Int): TypedOperand[Int] =
    new TypedOperand(Literal(ProtoValue.Number(value.toLong)))
  def enumValue[Value <: GeneratedEnum, Actual <: Value](value: Actual)(using
      NotGiven[Actual <:< UnrecognizedEnum]
  ): TypedOperand[Value] =
    new TypedOperand(Literal(ProtoValue.EnumName(value.name)))
  def named(value: Name): TypedOperand[String] =
    new TypedOperand(Literal(ProtoValue.Named(value)))
  def path[Root, Value](
      of: ProjectedOrigin[Root],
      field: Field[Root, Value]
  ): ProjectedPath[Root, Value] = new ProjectedPath(of, field)

  extension (inline projected: Operand)
    transparent inline def as[Root <: GeneratedMessage]: ProjectedOrigin[Root] =
      inline projected match
        case Operand.Projected => new ProjectedOrigin(Operand.Projected)
        case _                 => error("only Operand.Projected has a dynamic message root")

/**
 * A protobuf message written out: its full name and the fields it sets. A field it does not name
 * stays unset.
 */
final case class Proto(message: String, fields: ProtoField*)

object Proto:
  def apply[Message <: GeneratedMessage](
      fields: TypedProtoField[Message, ?]*
  ): TypedProto[Message] = new TypedProto(fields.toVector)

final case class ProtoField(name: String, value: ProtoValue)

object ProtoField:
  def typed[Message, Value](
      field: Field[Message, Value],
      value: TypedProtoValue[Value]
  ): TypedProtoField[Message, Value] = new TypedProtoField(field, value)

final case class ProtoEntry(key: String, value: ProtoValue)

object ProtoEntry:
  def typed(key: String, value: TypedProtoValue[ByteString]): TypedProtoEntry =
    new TypedProtoEntry(key, value)

enum ProtoValue:
  case Text(text: String)
  case Flag(flag: Boolean)
  case Number(number: Long)
  case EnumName(name: String)

  /** Bytes, as the UTF-8 text they encode. */
  case Utf8(text: String)
  case Message(proto: Proto)
  case Mapping(entries: ProtoEntry*)

  /** A role, where a field names the role a run resolves. */
  case RoleId(role: String)
  case Named(name: Name)

object ProtoValue:
  def text(value: String): TypedProtoValue[String] = new TypedProtoValue(Text(value))
  def flag(value: Boolean): TypedProtoValue[Boolean] = new TypedProtoValue(Flag(value))
  def number(value: Long): TypedProtoValue[Long] = new TypedProtoValue(Number(value))
  def integer(value: Int): TypedProtoValue[Int] = new TypedProtoValue(Number(value.toLong))
  def enumValue[Value <: GeneratedEnum, Actual <: Value](value: Actual)(using
      NotGiven[Actual <:< UnrecognizedEnum]
  ): TypedProtoValue[Value] = new TypedProtoValue(EnumName(value.name))
  def utf8(value: String): TypedProtoValue[ByteString] = new TypedProtoValue(Utf8(value))
  def message[Message <: GeneratedMessage](value: TypedProto[Message]): TypedProtoValue[Message] =
    new TypedProtoValue(value)
  def mapping(entries: TypedProtoEntry*): TypedProtoValue[Map[String, ByteString]] =
    new TypedProtoValue(entries.toVector)
  def roleId(value: String): TypedProtoValue[String] = new TypedProtoValue(RoleId(value))
  def named(value: Name): TypedProtoValue[String] = new TypedProtoValue(Named(value))

/** The assessment a completed live Run must support, independently of the model-search answer. */
final case class RunExpectation(
    conformance: Conformance,
    property: Outcome,
    reason: String = "",
    monitors: Vector[MonitorExpectation] = Vector.empty,
    contract: Outcome = Outcome.satisfied
)

enum Conformance:
  case conformant, nonconformant, inconclusive

enum Outcome:
  case satisfied, violated, inconclusive

final case class MonitorExpectation(name: String, outcome: Outcome, reason: String = "")
