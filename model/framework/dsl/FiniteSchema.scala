package framework.dsl

import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

// Channel references listed in unresolvedInboxes carry message identities until the signature
// binds a state field to its channel. They are metadata, never emitted channel Definition IDs.
final case class FiniteSchema private[framework] (
    identity: String,
    root: ir.TypeRef,
    types: Vector[ir.Type],
    unresolvedInboxes: Map[String, ir.TypeRef]
) derives CanEqual

object FiniteSchema:
  private[framework] def selected(
      witness: Finite[?],
      identity: String,
      context: String
  ): FiniteSchema =
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    val schema = witness.schemaFor(identity).getOrElse(fail("no retained typed schema selection"))
    if schema.identity != identity || schema.root.ref.isEmpty then
      fail("stale selected schema identity")
    val ownership = witness.inboxesFor(identity)
    val used = scala.collection.mutable.Set.empty[InboxOwnership.Field]
    def field(value: ir.Field, owner: String, caseName: String): ir.Field = value.getType.ref match
      case ir.TypeRef.Ref.Channel(message) =>
        val coordinate = InboxOwnership.Field(owner, caseName, value.name)
        val entry = ownership.getOrElse(
          coordinate,
          fail(s"field ${coordinate.context}: missing Inbox ownership")
        )
        if !schema.root.ref.named
            .contains(owner) || entry.field != coordinate || entry.message != message ||
          !schema.unresolvedInboxes.contains(message) || entry.declaration.isEmpty
        then fail(s"field ${coordinate.context}: stale Inbox owner/message/declaration")
        used += coordinate: Unit
        value.copy(`type` = Some(ir.TypeRef(ir.TypeRef.Ref.Channel(entry.declaration))))
      case _ => value
    val types = schema.types.map { t =>
      t.shape match
        case ir.Type.Shape.Record(record) =>
          t.copy(shape =
            ir.Type.Shape.Record(record.copy(fields = record.fields.map(field(_, t.name, ""))))
          )
        case ir.Type.Shape.Enum(enumeration) =>
          t.copy(shape =
            ir.Type.Shape.Enum(
              enumeration.copy(cases =
                enumeration.cases.map(c => c.copy(fields = c.fields.map(field(_, t.name, c.name))))
              )
            )
          )
        case _ => t
    }
    if ownership.keySet != used.toSet then fail("extra or foreign Inbox ownership coordinate")
    if schema.unresolvedInboxes.nonEmpty then
      val proof = ir.Function(
        "$retained-inbox-type-scope",
        params =
          (schema.root +: schema.unresolvedInboxes.toVector.sortBy(_._1).map(_._2)).zipWithIndex
            .map((t, i) => ir.Param(s"p$i", Some(t))),
        body = Some(ir.Expr(kind = ir.Expr.Kind.List(ir.ListOf())))
      )
      CoreBindings.dependencies(Nil, Seq(proof.name), Seq(proof), types, context): Unit
    schema.copy(types = types)

  private[framework] def decode(
      identity: String,
      root: Seq[String],
      closure: Seq[String],
      inboxes: Seq[String]
  ): FiniteSchema =
    val decoder = Base64.getDecoder
    FiniteSchema(
      identity,
      ir.TypeRef.parseFrom(decoder.decode(root.mkString)),
      ir.Model.parseFrom(decoder.decode(closure.mkString)).types.toVector,
      ir.Record
        .parseFrom(decoder.decode(inboxes.mkString))
        .fields
        .map(f => f.name -> f.getType)
        .toMap
    )

final private[framework] class finiteRange(val low: Long, val high: Long)
    extends scala.annotation.StaticAnnotation
