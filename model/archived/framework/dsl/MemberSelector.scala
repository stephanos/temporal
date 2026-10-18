package framework.dsl

import scala.quoted.*

final private[framework] class memberSelectors(val parameters: String)
    extends scala.annotation.StaticAnnotation

@scala.annotation.publicInBinary
private[framework] object MemberSelector:
  private[framework] def selectionParts(using
      quotes: Quotes
  )(
      term: quotes.reflect.Term
  ): Option[(quotes.reflect.Term, quotes.reflect.Term)] =
    import quotes.reflect.*
    def unwrap(value: Term): Term = value match
      case Inlined(_, _, body) => unwrap(body)
      case Typed(body, _)      => unwrap(body)
      case TypeApply(body, _)  => unwrap(body)
      case other               => other
    unwrap(term) match
      case Apply(function, List(machine)) if machine.tpe.widen <:< TypeRepr.of[Model] =>
        unwrap(function) match
          case Select(receiver, "->") =>
            unwrap(receiver) match
              case Apply(_, List(state)) => Some(state -> machine)
              case _                     => None
          case _ => None
      case Apply(function, List(state, machine))
          if function.symbol.owner.fullName.stripSuffix("$") == "scala.Tuple2" &&
            machine.tpe.widen <:< TypeRepr.of[Model] =>
        Some(state -> machine)
      case _ => None

  private[framework] def directField(using
      quotes: Quotes
  )(
      term: quotes.reflect.Term,
      rootOwner: Option[quotes.reflect.Symbol] = None
  ): String =
    import quotes.reflect.*
    def field(value: Term, parameters: Option[Set[Symbol]]): Option[String] = value match
      case Inlined(_, _, body)  => field(body, parameters)
      case Typed(body, _)       => field(body, parameters)
      case Lambda(params, body) => field(body, Some(params.map(_.symbol).toSet))
      case Select(parameter: Ident, name)
          if parameter.symbol.flags.is(Flags.Param) && parameters.forall(_(parameter.symbol)) &&
            rootOwner.forall(parameter.symbol.owner.equals) &&
            parameter.tpe.widen.dealias.typeSymbol.caseFields.exists(_.name == name) =>
        Some(name)
      case other => selectionParts(other).flatMap((state, _) => field(state, parameters))
    field(term, None).getOrElse(
      report.errorAndAbort("a member selector must name exactly one direct state field", term.pos)
    )

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
      val value = NativeRuntime.installed.invoke(run, state)
      evaluated = Some(value)
      value
    def value(state: => S): M = evaluated.getOrElse(apply(state))

  def retain[S, M](member: S => M, field: String): S => M =
    new Retained(member, field, None)

  inline def retainDirect[S, M](inline member: S => M): S => M = ${ retained('member) }

  private def retained[S: Type, M: Type](member: Expr[S => M])(using Quotes): Expr[S => M] =
    import quotes.reflect.*
    val field = directField(member.asTerm)
    '{ MemberSelector.retain($member, ${ Expr(field) }) }

  def retainMove[S, M](member: S => M, field: String, action: () => Class | Action[?]): S => M =
    new Retained(member, field, Some(action))

  def retainedField(member: Any): Option[String] = member match
    case retained: Retained[?, ?] => Some(retained.field)
    case _                        => None

  def selected(member: Any): Option[Class | Action[?]] = member match
    case retained: Retained[?, ?] => retained.selected
    case _                        => None

  def value[S, M](member: S => M, state: => S): M = member match
    case retained: Retained[S, M] @unchecked => retained.value(state)
    case _                                   => NativeRuntime.installed.invoke(member, state)
