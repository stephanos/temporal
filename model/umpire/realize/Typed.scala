package umpire.realize

import scala.annotation.publicInBinary
final case class Field[Root, Value](select: Root => Value)

final class TypedProto[Message] private[realize] (
    val fields: Vector[TypedProtoField[Message, ?]]
)

final class TypedProtoField[Message, Value] private[realize] (
    val field: Field[Message, Value],
    val value: TypedProtoValue[Value]
)

final class TypedProtoValue[Value] private[realize] (val value: Any)

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

/**
 * Recorded data of message type `Root` from a record of the system's that its kit declares, which
 * `Evidence.keyed` reads: the kit's own factory writes one.
 */
final class KeyedRef[Root](val recorded: Recorded)

final class RunEventRef[Root] private[realize] (val recorded: Recorded)

final class TypedOperand[Value] private[realize] (val operand: Operand)

final class ProjectedOrigin[Root] @publicInBinary private[realize] (val operand: Operand)

final class ProjectedPath[Root, Value] private[realize] (
    val origin: ProjectedOrigin[Root],
    val field: Field[Root, Value]
)

final class RunKey private[realize] (val operand: Operand)

final case class TypedAssignment[Root, Value](
    target: Field[Root, Value],
    value: TypedOperand[Value]
)

final case class TypedResponseRead[Root, Value](
    path: Field[Root, Value],
    cardinality: Cardinality,
    targets: Vector[Target]
)

final case class TypedEvidenceField[Root, Value](
    id: String,
    path: Field[Root, Value],
    role: Option[FieldRole] = None,
    redacted: Boolean = false
)

final class Condition[Root] private[realize] (
    val field: Option[Field[Root, ?]],
    val value: Option[TypedOperand[?]],
    val children: Vector[Condition[Root]]
)

object Condition:
  def present[Root, Value](field: Field[Root, Value]): Condition[Root] =
    new Condition(Some(field), None, Vector.empty)

  def equal[Root, Value](
      field: Field[Root, Value],
      value: TypedOperand[Value]
  ): Condition[Root] =
    new Condition(Some(field), Some(value), Vector.empty)

  def greater[Root, Value <: Int | Long](
      field: Field[Root, Value],
      value: TypedOperand[Value]
  ): Condition[Root] =
    new Condition(Some(field), Some(value), Vector.empty)

  def not[Root](of: Condition[Root]): Condition[Root] =
    new Condition(None, None, Vector(of))

  /** Holds when every operand does, read left to right up to the first that does not. */
  def all[Root](
      first: Condition[Root],
      rest: Condition[Root]*
  ): Condition[Root] =
    new Condition(None, None, (first +: rest).toVector)
