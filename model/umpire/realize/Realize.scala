/* What a Model says about running its Queries against a system: the roles a run addresses, the
 * values it learns, what it observes and reads as evidence, the controls it needs, and the scripts
 * its controller and the system's own activations follow.
 *
 * These are declarations and nothing else. No code here builds a Case: the lifter emits a
 * declaration into the IR as written, each class by its simple name and each parameter by its own,
 * and Go lowers one Query's witness through it (model/SEMANTICS.md, Realizations). A default
 * is therefore always the empty value, which the IR leaves unset.
 *
 * Nothing here is particular to one system. What a system has of its own (the roles it addresses
 * and their kinds, who runs a script beside the controller, the instructions only it carries out,
 * the records only it keeps, the settings it runs under, how its calls behave between each other and
 * which steps it takes on its own) its realization kit declares, by extending the open traits
 * `Addressee`, `Activation`, `Instruction`, `Recorded`, `Setting`, `Behavior` and `SystemStep`. The
 * lifter reads a kit's classes by name, as it reads these.
 */
package umpire.realize

import io.grpc.MethodDescriptor
import scala.compiletime.error
import scalapb.{GeneratedEnum, GeneratedMessage, GeneratedMessageCompanion, UnrecognizedEnum}
import scala.util.NotGiven
import com.google.protobuf.ByteString
import umpire.{Channel, ClassRef, Machine, Monitor}

/**
 * How the find Queries of one machine run against a system. Declared with named arguments and no
 * `name`, it is named after the `val` that declares it.
 */
final case class Realization(
    name: String = "",
    machine: Machine[?, ?, ?],
    producer: String,
    producerVersion: String,
    roles: Vector[Addressee],
    correlation: Correlation,
    scripts: Vector[Script],
    learned: Vector[Learned] = Vector.empty,
    observations: Vector[Observed] = Vector.empty,
    evidence: Vector[Evidence | EvidenceRef[?, ?] | TypedEvidence[?]] = Vector.empty,
    controls: Vector[Actuator] = Vector.empty,
    cleanup: String = "",
    requiredSettings: Vector[Setting] = Vector.empty,
    behavior: Option[Behavior] = None,
    serverSteps: Vector[SystemStep] = Vector.empty
)

/**
 * A setting the realized system must run under, such as the flag that enables a feature, as the
 * system's kit declares one. A Case lowered through the realization carries it, and preparation
 * refuses an environment that sets it otherwise.
 */
trait Setting

/**
 * How the realized system's calls behave between each other, as the system's kit declares it once
 * for every realization of the system: when a call's effect is visible to a later read, and how long
 * what a read waits for may take. It shapes how long and how often a Case waits, and never what a
 * Model says, what a Property checks or what a Contract asserts.
 */
trait Behavior

/**
 * A step of the machine the realized system takes on its own: no command of a script performs it,
 * and the declaration says what kind of cause it is, so a read that waits for it has a bound.
 */
trait SystemStep

/**
 * A symbolic participant commands, activations and controls address, named by its `id`. The
 * system's kit declares its roles, with the kinds and the environment bindings its system has.
 */
trait Addressee:
  def id: String

enum LearnedKind:
  case text

  /** An effect handle the system issues, which only the command that completes the effect reads. */
  case handle

/** A value a run learns: bound once, by one command, and read by the commands that depend on it. */
final case class Learned(id: String, kind: LearnedKind)

/** A typed value a run records for its checks: one protobuf message, by its full name. */
final class Observed private[realize] (val id: String, val message: String)

object Observed:
  def apply[Message <: GeneratedMessage](id: String)(using
      companion: GeneratedMessageCompanion[Message]
  ): Observed = new Observed(id, companion.scalaDescriptor.fullName)

/**
 * Where one kind of evidence is recorded: in the response of a read, in the Run's own record, or in
 * a record of the system's that its kit declares.
 */
trait Recorded

object Recorded:
  /** The elements of the repeated field selected in the response of a unary method. */
  final case class TypedRead[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage,
      Projected <: GeneratedMessage
  ](
      method: MethodDescriptor[Req, Rsp],
      path: Field[Rsp, Seq[Projected]]
  ) extends Recorded

  /** The one message selected in the response of a unary method. */
  final case class TypedSingle[
      Req <: GeneratedMessage,
      Rsp <: GeneratedMessage,
      Projected <: GeneratedMessage
  ](
      method: MethodDescriptor[Req, Rsp],
      path: Field[Rsp, Projected]
  ) extends Recorded

  /**
   * The Run's own record of the events of one kind that one command of a script records, where
   * `guard` holds of the event's payload, its instruction outcome. `key` names the operation an event
   * is of: the run's own id, or a path of the payload.
   *
   * `attempt` says the events are the record of one attempt a script's activation is delivered, and
   * a `diagnostic` always names one. A Run records an attempt once it is answered, so the evidence
   * reaches the Run with that answer and not with the carrying command, as the events that record
   * the attempt of that number.
   */
  final case class TypedRunEvent[Root <: GeneratedMessage](
      kind: EventKind,
      script: String | Script,
      command: String | Command | Instruction,
      key: ProjectedPath[Root, ?] | RunKey,
      guard: Option[Condition[Root]] = None,
      attempt: Option[AttemptOf] = None
  ) extends Recorded

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
      script: String | Script,
      command: String | Command | Instruction,
      key: ProjectedPath[Root, ?] | RunKey,
      guard: Option[Condition[Root]] = None,
      attempt: Option[AttemptOf] = None
  ): RunEventRef[Root] =
    new RunEventRef(
      Recorded.TypedRunEvent(kind, script, command, key, guard, attempt)
    )

/**
 * One attempt of what one script's activation runs: the `number`-th delivery of it, counted from
 * one, as the system numbers attempts.
 */
final case class AttemptOf(script: String | Script, number: Long)

/** The kinds of Run Event that carry an instruction outcome. */
enum EventKind:
  /** The command's own completion. */
  case instructionCompleted
  case instructionTimedOut

  /** What the system reports of an activation the command carries, such as one attempt of it. */
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
final class EvidenceField private[realize] (
    val id: String,
    val path: String,
    val role: Option[FieldRole] = None,
    val redacted: Boolean = false
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
final class Evidence private[realize] (
    val id: String,
    val records: Fact,
    val source: String,
    val from: Recorded,
    val operation: String,
    val commitment: Commitment,
    val fields: Vector[EvidenceField] = Vector.empty,
    val exhaustive: Boolean = false,
    val confirms: Vector[Taking] = Vector.empty
)

object Evidence:
  def read[Req, Rsp, Projected](
      id: String,
      records: Fact,
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
      new Evidence(
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

  /**
   * Evidence recorded where the system's kit says, `from`, keyed to its operation by the field
   * `operation` of the recorded message.
   */
  def keyed[Root <: GeneratedMessage](
      id: String,
      records: Fact,
      source: String,
      from: KeyedRef[Root],
      operation: Field[Root, ?],
      commitment: Commitment,
      fields: Vector[TypedEvidenceField[Root, ?]] = Vector.empty[TypedEvidenceField[Root, ?]],
      exhaustive: Boolean = false,
      confirms: Vector[Taking] = Vector.empty
  ): TypedEvidence[Root] = new TypedEvidence(
    new Evidence(
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
      records: Fact,
      source: String,
      from: RunEventRef[Root],
      commitment: Commitment,
      fields: Vector[TypedEvidenceField[Root, ?]] = Vector.empty[TypedEvidenceField[Root, ?]],
      exhaustive: Boolean = false,
      confirms: Vector[Taking] = Vector.empty
  ): TypedEvidence[Root] = new TypedEvidence(
    new Evidence(
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
 * An actuator a run needs beyond its commands, which the IR calls a control. `role` is the role
 * whose deliveries a run holds: a run reaches the channel through the deliveries of that role. It is
 * not named `Control`, a word Models take for their own actions' inputs.
 */
final case class Actuator(id: String, kind: ControlKind, role: String | Addressee = "")

/**
 * A text a Case gets its own copy of: the prefix, then the Case's fixture name when `fixture` is
 * set, then the suffix.
 */
final case class Name(prefix: String, fixture: Boolean = false, suffix: String = "")

/** Who runs a script: the controller, or an activation of the system its kit declares. */
trait Activation

object Activation:
  /** The run's own controller, which makes a Case's calls and works its controls. */
  case object Controller extends Activation

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
    closes: Vector[String | EvidenceRef[?, ?] | TypedEvidence[?]] = Vector.empty
)

enum Cardinality:
  case one

  /** One value per element of a repeated path. */
  case each

/** Where a read value goes. */
enum Target:
  case Observe(observation: String)

  /** Binds a learned value. */
  case Bind(learned: String)

  /** Lifts the evidence kinds the read confirms into an observation. */
  case Lift(observation: String)

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

object ResponseRead:
  def typed[Root, Value](
      path: Field[Root, Value],
      cardinality: Cardinality,
      targets: Vector[Target]
  ): TypedResponseRead[Root, Value] =
    TypedResponseRead(path, cardinality, targets)

/** What a command does: one of the instructions below, or one the system's kit declares. */
trait Instruction

object Instruction:
  /** A unary call on an endpoint role. */
  final case class TypedRpc[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
      role: String | Addressee,
      method: MethodDescriptor[Req, Rsp],
      assign: Vector[TypedAssignment[Req, ?]],
      reads: Vector[TypedResponseRead[Rsp, ?]]
  ) extends Instruction

  /** Polls the read an evidence kind names until an element satisfies `until`. */
  final case class TypedPoll[Req, Projected](
      evidence: EvidenceRef[Req, Projected],
      role: String | Addressee,
      assign: Vector[TypedAssignment[Req, ?]],
      until: Condition[Projected],
      intervalMs: Long = 0
  ) extends Instruction
  final case class AwaitLearned(learned: String) extends Instruction

  /** Waits for the operation an earlier command of the script started. */
  final case class AwaitCommand(command: String) extends Instruction

  /** Completes the activation the script runs in with a result. */
  final case class Finish(result: Operand) extends Instruction
  final case class Hold(control: String | Actuator) extends Instruction
  final case class Release(control: String | Actuator) extends Instruction

  def rpc[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
      role: String | Addressee,
      method: MethodDescriptor[Req, Rsp]
  )(
      assign: Vector[TypedAssignment[Req, ?]],
      reads: Vector[TypedResponseRead[Rsp, ?]]
  ): Instruction = TypedRpc(role, method, assign, reads)

  def readUntil[Req, Projected](evidence: EvidenceRef[Req, Projected], role: String | Addressee)(
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
 * A protobuf message written out: its generated type and the fields it sets. A field it does not
 * name stays unset.
 */
object Proto:
  def apply[Message <: GeneratedMessage](
      fields: TypedProtoField[Message, ?]*
  ): TypedProto[Message] = new TypedProto(fields.toVector)

object ProtoField:
  def typed[Message, Value](
      field: Field[Message, Value],
      value: TypedProtoValue[Value]
  ): TypedProtoField[Message, Value] = new TypedProtoField(field, value)

object ProtoEntry:
  def typed(key: String, value: TypedProtoValue[ByteString]): TypedProtoEntry =
    new TypedProtoEntry(key, value)

enum ProtoValue:
  case Text(text: String)
  case Flag(flag: Boolean)
  case Number(number: Long)
  private[realize] case EnumName(name: String)

  /** Bytes, as the UTF-8 text they encode. */
  case Utf8(text: String)

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
  def roleId(value: String | Addressee): TypedProtoValue[String] = new TypedProtoValue(
    RoleId(
      value match
        case r: Addressee => r.id
        case s: String    => s
    )
  )
  def named(value: Name): TypedProtoValue[String] = new TypedProtoValue(Named(value))

/**
 * The assessment a completed live Run must support, independently of the model-search answer: the
 * model assessment's conformance and the Property's outcome, the Contract's Verdict, how the Run
 * ends and how its cleanup ends. Each is declared, none defaulted, so a generated Case's check
 * compares what the Model says and infers nothing. An outcome short of satisfied names its `reason`,
 * the judge's own; a satisfied one names none.
 */
final case class RunExpectation(
    conformance: Conformance,
    property: PropertyOutcome,
    contract: PropertyOutcome,
    disposition: Disposition,
    cleanup: Cleanup,
    reason: Option[Reason] = None,
    monitors: Vector[MonitorExpectation] = Vector.empty
)

enum Conformance:
  case conformant, nonconformant, inconclusive

enum PropertyOutcome:
  case satisfied, violated, inconclusive

/** How a Run ends: it completes, its Contract's monitor stops it, or it does not close complete. */
enum Disposition:
  case completed, stoppedByMonitor, incomplete

/** How a Run's cleanup ends. */
enum Cleanup:
  case succeeded, failed, timedOut

/**
 * Why the model assessment leaves an outcome short of satisfied, by the judge's id: no evidence of
 * the machine, a Run that did not close complete, a hole of the Model in reach, evidence no modeled
 * execution explains, explaining executions that disagree on the claim, one that never reaches the
 * claim's evaluation point, one the claim cannot be read on, or every one violating it. The judge
 * owns each reason's wording (tools/umpire/conformance/conclude.go).
 */
enum Reason:
  case noEvidence, incomplete, hole, unexplained, explanationsDisagree, neverEvaluated, unreadable,
    everyExplanationViolates

/**
 * The verdict a completed live Run must support for one monitor of the Query's machine, named by
 * value, `MonitorExpectation(terminalFinality, PropertyOutcome.satisfied)`, or by its name, with
 * its reason when it is not satisfied.
 */
final case class MonitorExpectation(
    name: String | Monitor[?, ?, ?, ?],
    outcome: PropertyOutcome,
    reason: Option[Reason] = None
)
