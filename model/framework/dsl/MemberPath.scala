package framework.dsl

import scala.quoted.*
import scala.annotation.unused

final private[framework] case class MemberPath(field: String):
  require(
    field.nonEmpty && !field.contains('.'),
    "selector capture requires a direct field at its call site; a plain function parameter or selector reference is opaque"
  )

@scala.annotation.publicInBinary
private[framework] object MemberPath:
  inline def capture[S, M](inline member: S => M, @unused fact: Any): MemberPath = ${
    captured('member)
  }

  private def captured[S: Type, M: Type](member: Expr[S => M])(using Quotes): Expr[MemberPath] =
    import quotes.reflect.*
    def field(term: Term): Option[String] = term match
      case Inlined(_, _, body)                => field(body)
      case Block(List(d: DefDef), _: Closure) =>
        val parameters = d.termParamss.flatMap(_.params).map(_.symbol)
        def path(t: Term): Option[List[String]] = t match
          case p: Ident if parameters.contains(p.symbol) => Some(Nil)
          case Select(p, name)
              if p.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
            path(p).map(_ :+ name)
          case _ => None
        d.rhs.flatMap(path).filter(_.nonEmpty).map(_.mkString("."))
      case _ => None
    val name = field(member.asTerm).getOrElse("")
    '{ MemberPath(${ Expr(name) }) }
