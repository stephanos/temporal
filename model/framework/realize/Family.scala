package framework.realize

import io.temporal.server.api.umpire.v1 as ir
import scalapb.descriptors.{PMessage, PRepeated, PString, PValue}

private[framework] object Family:
  private val contextual = "\u0000umpire-family-context:v1\u0000"
  private val unbound = "\u0000umpire-family-unbound:v1\u0000"

  def resolve(
      value: ir.Realization,
      declaration: String,
      sourcePackage: Option[String]
  ): ir.Realization =
    def rewrite(value: PValue): PValue = value match
      case PString(text) =>
        require(!text.contains(unbound), s"$declaration: family has no compiler source provenance")
        if text.contains(contextual) then
          val pkg = sourcePackage
            .filter(_.nonEmpty)
            .getOrElse(
              throw IllegalArgumentException(
                s"$declaration: family has no declaring package context"
              )
            )
          val resolved = text.replace(contextual, pkg)
          require(
            !resolved.contains("\u0000umpire-family-"),
            s"$declaration: unknown family provenance version"
          )
          PString(resolved)
        else
          require(
            !text.contains("\u0000umpire-family-"),
            s"$declaration: unknown family provenance version"
          )
          value
      case PMessage(fields)  => PMessage(fields.view.mapValues(rewrite).toMap)
      case PRepeated(values) => PRepeated(values.map(rewrite))
      case _                 => value
    ir.Realization.messageReads.read(rewrite(value.toPMessage).asInstanceOf[PMessage])
