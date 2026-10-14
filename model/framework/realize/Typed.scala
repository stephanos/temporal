package framework.realize

import scala.annotation.publicInBinary

final class Field[Root, Value] @publicInBinary private[realize] (
    val select: Root => Value,
    val path: Option[String],
    val rootMessage: Option[String],
    val selectedMessage: Option[String]
)

object Field:
  inline def apply[Root, Value](inline select: Root => Value): Field[Root, Value] =
    ${ FieldSource.capture('select) }

final class TypedProto[Message] private[realize] (
    val fields: Vector[TypedProtoField[Message, ?]],
    val message: String
)

final class TypedProtoField[Message, Value] private[realize] (
    val field: Field[Message, Value],
    val value: TypedProtoValue[Value]
)

final class TypedProtoValue[Value] private[realize] (
    val value: Any,
    private val elementMessage: Option[String] = None
):
  def message: Option[String] = value match
    case proto: TypedProto[?] => Some(proto.message)
    case _                    => elementMessage

final class TypedProtoEntry private[realize] (
    val key: String,
    val value: TypedProtoValue[com.google.protobuf.ByteString]
)

final class RecordedRef[Req, Rsp, Projected] private[realize] (
    val recorded: Recorded
)

final class EvidenceRef[Req, Projected] private[realize] (
    val evidence: Evidence,
    val operation: Field[Projected, ?],
    val fields: Vector[TypedEvidenceField[Projected, ?]]
)

final class TypedEvidence[Root] private[realize] (
    val evidence: Evidence,
    val operation: Option[Field[Root, ?]],
    val fields: Vector[TypedEvidenceField[Root, ?]]
)

// Recorded data of message type `Root` from a record of the system's that its kit declares, which
// `Evidence.keyed` reads: the kit's own factory writes one.
final class KeyedRef[Root](val recorded: Recorded)

final class RunEventRef[Root] private[realize] (val recorded: Recorded)

final class TypedOperand[Value] private[realize] (val operand: Operand)

final class ProjectedOrigin[Root] @publicInBinary private[realize] (
    val operand: Operand,
    val message: String
)

final class ProjectedPath[Root, Value] private[realize] (
    val origin: ProjectedOrigin[Root],
    val field: Field[Root, Value]
)

final class RunKey private[realize] (val operand: Operand)

final case class TypedAssignment[Root, Value] private[realize] (
    target: Field[Root, Value],
    value: TypedOperand[Value]
)

final case class TypedResponseRead[Root, Value] private[realize] (
    path: Field[Root, Value],
    cardinality: Cardinality,
    targets: Vector[Target]
)

final case class TypedEvidenceField[Root, Value] private[realize] (
    id: String,
    path: Field[Root, Value],
    role: Option[FieldRole] = None,
    redacted: Boolean = false
)

enum ConditionOperator derives CanEqual:
  case present, equal, greater, not, all

final class Condition[Root] private[realize] (
    val operator: ConditionOperator,
    val field: Option[Field[Root, ?]],
    val value: Option[TypedOperand[?]],
    val children: Vector[Condition[Root]]
)

object Condition:
  def present[Root, Value](field: Field[Root, Value]): Condition[Root] =
    new Condition(ConditionOperator.present, Some(field), None, Vector.empty)

  def equal[Root, Value](
      field: Field[Root, Value],
      value: TypedOperand[Value]
  ): Condition[Root] =
    new Condition(ConditionOperator.equal, Some(field), Some(value), Vector.empty)

  def greater[Root, Value <: Int | Long](
      field: Field[Root, Value],
      value: TypedOperand[Value]
  ): Condition[Root] =
    new Condition(ConditionOperator.greater, Some(field), Some(value), Vector.empty)

  def not[Root](of: Condition[Root]): Condition[Root] =
    new Condition(ConditionOperator.not, None, None, Vector(of))

  // Holds when every operand does, read left to right up to the first that does not.
  def all[Root](
      first: Condition[Root],
      rest: Condition[Root]*
  ): Condition[Root] =
    new Condition(ConditionOperator.all, None, None, (first +: rest).toVector)
