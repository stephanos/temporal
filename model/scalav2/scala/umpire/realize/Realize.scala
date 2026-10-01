/* What a Model says about running its Queries against a system: the roles a run addresses, the
 * values it learns, what it observes and reads as evidence, the controls it needs, and the scripts
 * its controller and its workers follow.
 *
 * These are declarations and nothing else. No code here builds a Case: the lifter emits a
 * declaration into the IR as written, each class by its simple name and each parameter by its own,
 * and Go lowers one Query's witness through it (model/scalav2/SEMANTICS.md, Realizations). A default
 * is therefore always the empty value, which the IR leaves unset.
 */
package umpire.realize

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
    evidence: Vector[Evidence] = Vector.empty,
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

/** Where one kind of evidence is recorded. */
enum Recorded:
  /** One member of the history event's attributes. */
  case History(attributes: String)

  /** The elements of the repeated field at `path` in the response of a unary method. */
  case Read(method: String, path: String)

/** What evidence commits to: what a caller was told, or a durable commit of the receiver. */
enum Commitment:
  case reported, durable

/**
 * One kind of evidence: the recorded data that confirms the facts the machine's evidence function
 * names `records`, and the field that keys it to its operation.
 */
final case class Evidence(
    id: String,
    records: String,
    source: String,
    from: Recorded,
    operation: String,
    commitment: Commitment
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

/** An actuator a run needs beyond its commands. */
final case class Control(id: String, kind: ControlKind)

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
  case Activity(activityType: Name, worker: String, taskQueue: String)

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
 * it whatever became of the commands it runs after.
 */
final case class Command(
    id: String,
    instruction: Instruction,
    after: Option[After] = None,
    timeoutMs: Long = 0,
    regardless: Boolean = false
)

enum FaultKind:
  case workerStop, workerResume

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

final case class ResponseRead(path: String, cardinality: Cardinality, targets: Vector[Target])

/** What a command does. */
enum Instruction:
  /** A unary call on an endpoint role. */
  case Rpc(
      role: String,
      method: String,
      assign: Vector[Assignment],
      reads: Vector[ResponseRead] = Vector.empty
  )

  /** Polls the read an evidence kind names until an element satisfies `until`. */
  case Poll(
      evidence: String,
      role: String,
      assign: Vector[Assignment],
      until: Operand,
      intervalMs: Long
  )
  case AwaitLearned(learned: String)

  /** Waits for the operation an earlier command of the script started. */
  case AwaitCommand(command: String)
  case Finish(result: Operand)
  case Fault(role: String, kind: FaultKind)

  /** A workflow command, as the message the SDK would emit. */
  case WorkflowCommand(command: Proto)

  /** A Nexus handler's answer; an asynchronous one binds the handle `binds` names. */
  case NexusReply(reply: Proto, binds: String = "")
  case NexusCompletion(handle: String, result: Proto)
  case Hold(control: String)
  case Release(control: String)

/** A value a command computes when it runs. */
enum Operand:
  case Literal(value: ProtoValue)
  case Environment(binding: String)

  /** The run's own id. */
  case Run
  case LearnedValue(learned: String)

  /** The value a poll is looking at. */
  case Projected
  case Path(of: Operand, path: String)
  case Present(of: Operand)
  case Equal(left: Operand, right: Operand)

/**
 * A protobuf message written out: its full name and the fields it sets. A field it does not name
 * stays unset.
 */
final case class Proto(message: String, fields: ProtoField*)

final case class ProtoField(name: String, value: ProtoValue)

final case class ProtoEntry(key: String, value: ProtoValue)

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
