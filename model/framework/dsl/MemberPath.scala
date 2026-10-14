package framework.dsl

import scala.quoted.*
import scala.annotation.unused

final private[framework] class memberSelectors(val parameters: String)
    extends scala.annotation.StaticAnnotation

final private[framework] case class MemberPath(field: String):
  require(
    field.nonEmpty && !field.contains('.'),
    "selector capture requires a direct field at its call site; a plain function parameter or selector reference is opaque"
  )

@scala.annotation.publicInBinary
private[framework] object MemberPath:
  final private class Retained[S, M](
      run: S => M,
      val field: String,
      action: Option[() => Class | Action[?]]
  ) extends (S => M):
    private var evaluated = Option.empty[M]
    lazy val selected: Option[Class | Action[?]] = evaluated
      .collect:
        case (_, selected: (Class | Action[?])) => selected
      .orElse(action.map(_()))
    def apply(state: S): M =
      val value = run(state)
      evaluated = Some(value)
      value
    def value(state: => S): M = evaluated.getOrElse(apply(state))

  def retain[S, M](member: S => M, field: String): S => M =
    new Retained(member, field, None)

  def retainMove[S, M](member: S => M, field: String, action: () => Class | Action[?]): S => M =
    new Retained(member, field, Some(action))

  def field(member: Any): Option[String] = member match
    case retained: Retained[?, ?] => Some(retained.field)
    case _                        => None

  def selected(member: Any): Option[Class | Action[?]] = member match
    case retained: Retained[?, ?] => retained.selected
    case _                        => None

  def value[S, M](member: S => M, state: => S): M = member match
    case retained: Retained[S, M] @unchecked => retained.value(state)
    case _                                   => member(state)

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
