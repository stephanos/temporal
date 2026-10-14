package framework.dsl

import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

final class lifted extends scala.annotation.StaticAnnotation
final class liftedIr(val chunks: String*) extends scala.annotation.StaticAnnotation
private[framework] final class capture(val target: String) extends scala.annotation.StaticAnnotation

private[framework] final case class Captured[F](callable: F, root: String, model: ir.Model):
  require(model.functions.map(_.name).distinct.size == model.functions.size,
    "captured function names are not unique")
  require(model.types.map(_.name).distinct.size == model.types.size,
    "captured type names are not unique")
  require(model.functions.exists(_.name == root), s"captured function $root is missing")
  def function: ir.Function = model.functions.find(_.name == root)
    .getOrElse(throw IllegalArgumentException(s"captured function $root is missing"))

object Reify:
  def function[F](@capture("functionCaptured") f: F): ir.Function =
    throw IllegalStateException("function capture requires the reify compiler plugin")

  private[framework] def capture[F](@capture("captured") f: F): Captured[F] =
    throw IllegalStateException("function capture requires the reify compiler plugin")

  private[framework] def captured[F](f: F, root: String, chunks: Seq[String]): Captured[F] =
    Captured(f, root, ir.Model.parseFrom(Base64.getDecoder.decode(chunks.mkString)))

  private[framework] def functionCaptured[F](f: F, root: String, chunks: Seq[String]): ir.Function =
    captured(f, root, chunks).function
