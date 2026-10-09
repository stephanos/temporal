// The Temporal kit's sugar: definitions whose meaning a core realization declaration already
// expresses, kept for readability. Each names the core form it stands for, and the IR
// generator (model/irgen/Syntax.scala) lowers it to the IR that core form lifts to. No core file
// of the kit uses them.
package temporal.realize

import scala.annotation.targetName
import com.google.protobuf.ByteString
import scalapb.GeneratedMessage
import framework.realize.{
  Addressee,
  Cardinality,
  Field,
  Name,
  Observed,
  Proto,
  RequestScope,
  Target,
  TypedOperand,
  TypedProto,
  TypedProtoValue
}

// A field of the request of the scope it is written in, the named slot `:=` gives an operand.
// Core form: the typed field `Field[Req, V](_.name)` that `Assignment.typed` takes.
final class RequestField[Req, V] private[realize] (val target: Field[Req, V]):
  // The field receives the operand.
  def :=(value: TypedOperand[V]): Unit = ()

  // A message written out, assigned to a message field of the request: each field the message
  // sets is assigned at its own path below this one.
  @targetName("assignMessage")
  def :=(value: TypedProto[V]): Unit = ()

// The scope `proto[M] { … }` opens for a protobuf message of type `M`: each line sets one field of
// that message. Core form: a line is `ProtoField.typed(Field[M, V](_.name), value)`.
final class ProtoScope[M] private[realize] ()

// A field of the message of the literal scope it is written in. `:=` gives it a value, and a scope
// after it, `field(_.info) { … }`, writes the message it holds. Core form: the typed field
// `Field[M, V](_.name)` that `ProtoField.typed` takes.
final class LiteralField[M, V] private[realize] (val target: Field[M, V]):
  // The field holds `value`, of a type `ProtoValueOf` admits for the field's type.
  def :=[X](value: X)(using ProtoValueOf[V, X]): Unit = ()

  // The field holds the message its own scope writes.
  def apply(build: ProtoScope[V] ?=> Unit): Unit = build(using ProtoScope())

// The values a protobuf literal's field of type `V` takes, written as `X`: the field's own type,
// such as a string, a flag, a number or an enum value; a message written out for a message field;
// a role or a per-Case name for a string field; text for a bytes field, encoded as UTF-8; and a
// map of texts for a map of bytes. Core form: the `ProtoValue` factory of the value, such as
// `ProtoValue.text(v)`.
final class ProtoValueOf[V, -X] private[realize] ()

// The values `ProtoValueOf` admits, one given each. Core form: `ProtoValue.text(v)` and its siblings.
object ProtoValueOf:
  // A value of the field's own type. Core form: `ProtoValue.text(v)`, `ProtoValue.flag(v)`,
  // `ProtoValue.number(v)` or `ProtoValue.enumValue(v)`.
  given scalarValue: [V] => ProtoValueOf[V, V] = ProtoValueOf()

  // A number field of 64 bits given an `Int`. Core form: `ProtoValue.number(v)`.
  given widenedNumber: ProtoValueOf[Long, Int] = ProtoValueOf()

  // A message written out. Core form: `ProtoValue.message(Proto[M](…))`.
  given messageValue: [M] => ProtoValueOf[M, TypedProto[M]] = ProtoValueOf()

  // A protobuf value written in its core form. Core form: the value itself, `ProtoValue.text(v)`.
  given coreValue: [V] => ProtoValueOf[V, TypedProtoValue[V]] = ProtoValueOf()

  // A role, by its id. Core form: `ProtoValue.roleId(role)`.
  given roleValue: ProtoValueOf[String, Addressee] = ProtoValueOf()

  // A name each Case gets its own copy of. Core form: `ProtoValue.named(name)`.
  given nameValue: ProtoValueOf[String, Name] = ProtoValueOf()

  // Text, as the UTF-8 bytes it encodes. Core form: `ProtoValue.utf8(text)`.
  given textBytes: ProtoValueOf[ByteString, String] = ProtoValueOf()

  // A map of texts, each as the UTF-8 bytes it encodes. Core form:
  // `ProtoValue.mapping(ProtoEntry.typed(key, ProtoValue.utf8(text)), …)`.
  given textMap: ProtoValueOf[Map[String, ByteString], Map[String, String]] = ProtoValueOf()

// The scope a field selector is read in: a request scope or a protobuf literal scope, and the slot
// a field of that scope is. Core form: `Field[Root, V](_.name)`.
trait FieldScope[Root]:
  type SlotOf[V]
  def slotOf[V](target: Field[Root, V]): SlotOf[V]

// The two scopes a field is read in. Core form: `Field[Root, V](_.name)`.
object FieldScope:
  // A field of a call's request. Core form: `Field[Req, V](_.name)`, as `Assignment.typed` takes it.
  given requests: [Req] => RequestScope[Req] => FieldScope[Req]:
    type SlotOf[V] = RequestField[Req, V]
    def slotOf[V](target: Field[Req, V]) = RequestField(target)

  // A field of a protobuf literal. Core form: `Field[M, V](_.name)`, as `ProtoField.typed` takes it.
  given protos: [M] => ProtoScope[M] => FieldScope[M]:
    type SlotOf[V] = LiteralField[M, V]
    def slotOf[V](target: Field[M, V]) = LiteralField(target)

// `field(_.namespace) := operand`, written inside `rpc(…) { … }` or `readUntil(…) { … }`: the
// field of the call's request the selector names receives the operand. The request type is the
// scope's, so the selector is typed against it and no line repeats it; `:=` is the named slot's one
// operator, as for an action's input. Inside `proto[M] { … }` the field is one of the message's,
// and it takes the values `ProtoValueOf` admits. Core form:
// `Assignment.typed(Field[Req, V](_.namespace), operand)`, and in a literal
// `ProtoField.typed(Field[M, V](_.name), value)`.
def field[Root, V](using scope: FieldScope[Root])(select: Root => V): scope.SlotOf[V] =
  scope.slotOf(Field(select))

// `proto[Failure] { field(_.message) := "failed" }`: a protobuf message written out, the fields
// its scope sets in order, each value's type inferred from its field, a nested message written in
// the scope after its field, `field(_.getInfo) { … }`. Core form:
// `Proto[Failure](ProtoField.typed(Field[Failure, String](_.message), ProtoValue.text("failed")))`.
def proto[M <: GeneratedMessage](build: ProtoScope[M] ?=> Unit): TypedProto[M] =
  build(using ProtoScope())
  Proto[M]()

// A read of the call's response, written in its scope: `read(path, cardinality)` and then
// `.into(targets…)`, where an observation stands for observing into it. Core form:
// `ResponseRead.typed(path, cardinality, targets)`.
final class ResponseReadLine[Rsp, V] private[realize] (
    val path: Field[Rsp, V],
    val cardinality: Cardinality
):
  // The targets the read values go to, in order.
  def into(targets: (Observed | Target)*): Unit = ()

// `read(historyEvents, Cardinality.each).into(historyEvent, Target.Lift(correlated.id))`, written
// inside `rpc(…) { … }`: the call reads `path` of its response into the targets, an observation
// written by value for `Target.Observe(observation.id)`. Core form:
// `ResponseRead.typed(path, Cardinality.each, Vector(Target.Observe(…), Target.Lift(…)))`, in the
// reads of `Instruction.rpc(role, method)(assign, reads)`.
def read[Rsp, V](path: Field[Rsp, V], cardinality: Cardinality)(using
    RequestScope[?]
): ResponseReadLine[Rsp, V] = ResponseReadLine(path, cardinality)
