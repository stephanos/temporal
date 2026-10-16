package framework.dsl

import scala.language.implicitConversions
import scala.quoted.*

final class MemberSelection @scala.annotation.publicInBinary private[framework] (
    private[framework] val field: String,
    private[framework] val value: Any,
    private[framework] val model: Model
)

object MemberSelection:
  implicit transparent inline def capture(inline selected: (Any, Model)): MemberSelection =
    ${ captured('selected) }

  private def captured(selected: Expr[(Any, Model)])(using Quotes): Expr[MemberSelection] =
    import quotes.reflect.*
    val field = MemberPath.directField(selected.asTerm, Some(Symbol.spliceOwner.owner))
    '{
      val pair = $selected
      new MemberSelection(${ Expr(field) }, pair._1, pair._2)
    }
