// The script helpers: what a realization's scripts say, written as steps rather than records.
//
// A step is a command every Case carries (`everyCase`), a command only the Cases whose path takes
// one of some classes carry (`onPath`), or the place where a path's steps of the classes it binds
// land, in path order (`perform`). A command is a `val`, named after it in kebab case
// (`val pauseOrder` is the command `pause-order`), and other declarations refer to it by value: an
// instruction stands for the command with no options, and `command(instruction, …)` is one with
// options. A call opens a scope with its request type fixed, and the request's fields are assigned
// inside it.
//
// Each helper is core: it writes the IR record its scaladoc names (Script, Item, Performance,
// Command, Rpc, Poll), which the IR generator writes by name (model/irgen/Realizations.scala). No
// body here runs.
package framework.realize

import io.grpc.MethodDescriptor
import scala.annotation.unused
import scalapb.GeneratedMessage
import framework.{Action, ClassRef}

// A fact as a machine records it: a case of its fact enum (`system.Fact.statusPaused`), or for a
// case with fields the case itself (`system.Fact.statusTimedOut`), which names every value of it.
// The lifter writes the case's name, which is the evidence name the fact is confirmed by unless the
// machine's evidence function says otherwise. A string is the name written out.
type Fact = AnyRef

// The script `id`, run by `activation`, of the steps `items` in order. IR: Script.
def script(id: String, activation: Activation)(items: Item*): Script =
  Script(id, activation, items.toVector)

// Where the steps of a path that take one of the classes land, each performed by its command, in
// path order: `perform(control(Control.pause) -> pauseOrder)`. At least one class. IR: an Item
// of `performs`, each a Performance.
def perform(bindings: (ClassRef, Command | Instruction)*): Item =
  Item(performs = bindings.toVector.map((step, command) => Performance(step, commanded(command))))

// A command only the Cases whose path takes one of the classes carry. At least one class. An
// action with inputs, `onPath(caller.schedule)`, stands for each of its classes a `deadlines`
// declaration of the realization performs. IR: an Item of `command` and `when`.
def onPath(classes: (ClassRef | Action[?])*)(command: Command | Instruction): Item =
  Item(
    command = Some(commanded(command)),
    // An action stands for its classes; the lifter reads them off the source, not this value.
    when = classes.toVector.map(_.asInstanceOf[ClassRef]) // scalafix:ok DisableSyntax.asInstanceOf
  )

// A command every Case carries. IR: an Item of `command`.
def everyCase(command: Command | Instruction): Item = Item(command = Some(commanded(command)))

// A command with options, named after the `val` that declares it: the commands it runs `after`, a
// timeout bound in `timeoutMs`, whether it runs `regardless` of what became of them, and the
// exhaustive kinds of evidence its read `closes`. IR: Command.
def command(
    instruction: Instruction,
    after: Option[After] = None,
    timeoutMs: Long = 0,
    regardless: Boolean = false,
    closes: Vector[String | EvidenceRef[?, ?] | TypedEvidence[?]] = Vector.empty
): Command = Command("", instruction, after, timeoutMs, regardless, closes)

// A command that takes the name of another, `target`, as a race does whose evidence reads one
// command name in every realization that runs it: `aliasOf(releaseDispatch)(fault(…))`. It is the
// one way two commands share a name; the lifter refuses two commands of one realization with one
// name otherwise. A call `withFields` extends keeps its call's name without an alias. IR: Command,
// named as `target` is.
def aliasOf(@unused target: Command | Instruction)(instruction: Instruction): Command =
  Command("", instruction)

private def commanded(c: Command | Instruction): Command = c match
  case c: Command     => c
  case i: Instruction => Command("", i)

// The scope a call opens for its request of type `Req`: inside `rpc(…) { … }` or
// `readUntil(…) { … }` each line assigns one field of that request. Core form of a line:
// `Assignment.typed(Field[Req, V](_.name), operand)`.
final class RequestScope[Req] private[realize] ()

// A unary call of `method` on `role`, its request's fields assigned in the scope it opens:
// `rpc(orderService, METHOD_PAUSE_ORDER) { … }`. IR: Rpc.
def rpc[Req <: GeneratedMessage, Rsp <: GeneratedMessage](
    role: String | Addressee,
    method: MethodDescriptor[Req, Rsp]
)(assign: RequestScope[Req] ?=> Unit): Instruction.TypedRpc[Req, Rsp] =
  assign(using RequestScope())
  new Instruction.TypedRpc(role, method, Vector.empty, Vector.empty)

// Reads what `evidence` names on `role` every `intervalMs` until an element satisfies `until`, its
// request's fields assigned in the scope it opens. IR: Poll.
def readUntil[Req, Projected](
    evidence: EvidenceRef[Req, Projected],
    role: String | Addressee,
    until: Condition[Projected],
    intervalMs: Long
)(assign: RequestScope[Req] ?=> Unit): Instruction =
  assign(using RequestScope())
  Instruction.TypedPoll(evidence, role, Vector.empty, until, intervalMs)

extension [Req <: GeneratedMessage, Rsp <: GeneratedMessage](call: Instruction.TypedRpc[Req, Rsp])
  // The same call with more fields assigned after its own: one command written once, with what a
  // class of an action adds. It keeps the call's command name, so evidence that names the call,
  // such as `answered(fact, call)`, matches every `withFields` variant of it. IR: the Rpc with the
  // assignments appended.
  def withFields(assign: RequestScope[Req] ?=> Unit): Instruction.TypedRpc[Req, Rsp] =
    assign(using RequestScope())
    call

  // A call of its own that extends this one: its request is this call's, with the fields its scope
  // assigns after them, and it reads what its scope reads. Unlike `withFields` it is a command of
  // its own, named after the `val` that declares it. IR: the Rpc with the assignments and reads
  // appended.
  def extended(assign: RequestScope[Req] ?=> Unit): Instruction.TypedRpc[Req, Rsp] =
    assign(using RequestScope())
    call

// What a fact reads as on a system: one value of type `V` for each fact the table lists, such as
// the status a describe call reports while the fact holds. The table is declared once beside the
// realization and consulted by value, `table(fact)`; the lifter looks the fact up when it lifts.
final class StatusTable[V] private[realize] (val entries: Vector[(Fact, V)]):
  def apply(fact: Fact): V = entries.collectFirst { case (f, v) if f == fact => v }.get

// A table of what each fact reads as, `statusTable(fact -> value, …)`, each fact listed once.
def statusTable[V](entries: (Fact, V)*): StatusTable[V] = StatusTable(entries.toVector)
