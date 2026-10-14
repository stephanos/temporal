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
