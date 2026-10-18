package framework.checker

import framework.dsl.*
import scala.quoted.*

@scala.annotation.publicInBinary
private[framework] object CheckerNaming:
  inline def bind[S, O, F](inline instance: NativeModel[S, O, F]): NativeModel[S, O, F] =
    ${ capture[S, O, F]('instance) }

  def held[S, O, F](
      instance: NativeModel[S, O, F],
      holder: Option[String],
      location: String
  ): NativeModel[S, O, F] =
    instance match
      case machine: Machine[?, ?] if !machine.getClass.getName.endsWith("$") =>
        machine.checkedHolder(holder, location)
      case _ => ()
    instance

  private def capture[S: Type, O: Type, F: Type](instance: Expr[NativeModel[S, O, F]])(using
      Quotes
  ): Expr[NativeModel[S, O, F]] =
    import quotes.reflect.*
    def underlying(term: Term): Term = term match
      case Inlined(_, Nil, value) => underlying(value)
      case Typed(value, _)        => underlying(value)
      case Block(Nil, value)      => underlying(value)
      case _                      => term
    val term = underlying(instance.asTerm)
    val symbol = term.symbol
    def stable(term: Term): Boolean = underlying(term) match
      case _: This          => true
      case reference: Ident =>
        reference.symbol.flags.is(Flags.Module) ||
        (reference.symbol.isValDef && !reference.symbol.flags.is(Flags.Mutable))
      case selected: Select =>
        stable(selected.qualifier) &&
        (selected.symbol.flags.is(
          Flags.Module
        ) || (selected.symbol.isValDef && !selected.symbol.flags.is(Flags.Mutable)))
      case _ => false
    def parameterAlias(term: Term, seen: Set[Symbol] = Set.empty): Boolean = underlying(term) match
      case reference: Ref if reference.symbol.flags.is(Flags.Param) => true
      case reference: Ref
          if reference.symbol.isValDef && reference.symbol.owner.isDefDef && !seen(
            reference.symbol
          ) =>
        scala.util.Try(reference.symbol.tree).toOption match
          case Some(value: ValDef) => value.rhs.exists(parameterAlias(_, seen + reference.symbol))
          case _                   => false
      case _ => false
    val holder = term match
      case _: Ref
          if symbol.isValDef && !symbol.flags.is(Flags.Param) &&
            !symbol.flags.is(Flags.Mutable) && !symbol.name
              .contains("$") && stable(term) && !parameterAlias(term) =>
        Some(symbol.name)
      case _ => None
    val construction = symbol.isClassConstructor
    if construction && holder.isEmpty && term.tpe.widen <:< TypeRepr.of[Machine[?, ?]] &&
      !symbol.flags.is(Flags.Module) && !symbol.flags.is(Flags.Param)
    then
      report.errorAndAbort(
        s"${term.tpe.widen.typeSymbol.name}: a checked Model instance needs a stable val holder",
        term.pos
      )
    val position = term.pos
    val location = Expr(s"${position.sourceFile.name}:${position.startLine + 1}")
    '{ held($instance, ${ Expr(holder) }, $location) }
