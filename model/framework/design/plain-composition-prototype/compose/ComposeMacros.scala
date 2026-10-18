package framework.dsl
package compose

import scala.quoted.*

// The compile-time half of the probe: the member schema of a composition object, its typed
// `before`, the line of a `records` and the claim a `find` names.
@scala.annotation.publicInBinary
private[compose] object ComposeMacros:
  inline def line: Int = ${ lineImpl }

  private def lineImpl(using q: Quotes): Expr[Int] =
    Expr(q.reflect.Position.ofMacroExpansion.startLine + 1)

  // The composition class an expansion sits in, or whose parents ask for its schema.
  private def enclosing(using q: Quotes): Option[q.reflect.Symbol] =
    import q.reflect.*
    val base = TypeRepr.of[Composition[?]].typeSymbol
    Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))

  // The member vals of a composition class, in declaration order, with their declared types. A
  // member is a val whose type is a Machine or a Composition; a val naming any other object is
  // refused, and so is a Machine object named twice, since its vars hold one state.
  private def memberVals(using
      q: Quotes
  )(cls: q.reflect.Symbol): List[(q.reflect.Symbol, q.reflect.TypeRepr)] =
    import q.reflect.*
    val machine = TypeRepr.of[RecordMachine[?, ?, ?]]
    val composition = TypeRepr.of[Composition[?]]
    val vals = cls.declarations.filter(s =>
      s.isValDef && !s.flags.is(Flags.Mutable) && !s.flags.is(Flags.Synthetic) &&
        !s.flags.is(Flags.Module) && !s.name.contains("$")
    )
    val typed = vals.map(v => v -> cls.typeRef.memberType(v).widenByName)
    val members = typed.filter: (v, tpe) =>
      if tpe <:< machine || tpe <:< composition then true
      else
        tpe match
          case ref: TermRef if ref.termSymbol.flags.is(Flags.Module) =>
            report.error(
              s"${v.name} names ${ref.termSymbol.name}, which is neither a Machine nor a Composition",
              v.pos.getOrElse(Position.ofMacroExpansion)
            )
            false
          case _ => false
    for
      ((v, tpe), at) <- members.zipWithIndex
      (earlier, _) <- members.take(at).find((_, t) => t =:= tpe && t.termSymbol.exists)
    do
      report.error(
        s"${v.name} names ${tpe.termSymbol.name}, which ${earlier.name} names already: a Machine " +
          "object's vars hold one state, so it is a member once",
        v.pos.getOrElse(Position.ofMacroExpansion)
      )
    members

  def schema[F: Type](using q: Quotes): Expr[Schema[F]] =
    import q.reflect.*
    val cls = enclosing.getOrElse(
      report.errorAndAbort("a composition's members are read where its object extends Composition")
    )
    val members = memberVals(cls)
    if members.isEmpty then report.error("a composition declares at least one member val")
    val fact = TypeRepr.of[F].dealias
    val cases = fact.typeSymbol.children
    val machineBase = TypeRepr.of[RecordMachine[?, ?, ?]].typeSymbol
    val tags = members.map: (v, tpe) =>
      val memberFact = tpe.baseType(machineBase).typeArgs.lift(2).getOrElse(TypeRepr.of[Nothing])
      if memberFact =:= TypeRepr.of[Nothing] then '{ None }
      else
        cases.find(_.name == v.name) match
          case None =>
            report.error(
              s"${v.name} records ${memberFact.show(using Printer.TypeReprShortCode)}: " +
                s"${fact.show(using Printer.TypeReprShortCode)} declares a case for it, " +
                s"`case ${v.name}(value: ${memberFact.show(using Printer.TypeReprShortCode)})`",
              v.pos.getOrElse(Position.ofMacroExpansion)
            )
            '{ None }
          case Some(member) =>
            val field = member.caseFields.headOption
            val accepts = field.exists(f => member.typeRef.memberType(f) =:= memberFact)
            if member.caseFields.sizeIs != 1 || !accepts then
              report.error(
                s"${fact.show(using Printer.TypeReprShortCode)}.${v.name} carries " +
                  s"${v.name}'s fact, one value of ${memberFact.show(using Printer.TypeReprShortCode)}",
                v.pos.getOrElse(Position.ofMacroExpansion)
              )
              '{ None }
            else
              memberFact.asType match
                case '[t] =>
                  '{
                    Some((x: Any) =>
                      ${
                        Select
                          .overloaded(
                            Ref(member.companionModule),
                            "apply",
                            Nil,
                            List('{ x.asInstanceOf[t] }.asTerm)
                          )
                          .asExprOf[F]
                      }
                    )
                  }
    '{ Schema.of[F](${ Expr(members.map(_._1.name)) }, ${ Expr.ofList(tags) }) }

  // `before`: a record with one field per member, each the member's State record refined with its
  // vars, so `before.left.phase` type-checks and a misspelt member or var is a type error.
  def before(using q: Quotes): Expr[Any] =
    import q.reflect.*
    val inPostconditions = Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .exists(s => s.isClassDef && Set("postconditions$", "queries$")(s.name))
    if !inPostconditions then
      report.errorAndAbort("before is the state a step started from; use it in a postcondition")
    val cls = enclosing.getOrElse(report.errorAndAbort("before reads a composition's step"))
    val record = memberVals(cls).foldLeft(TypeRepr.of[Before]): (record, member) =>
      val (v, tpe) = member
      val machine = tpe.termSymbol.moduleClass
      val state =
        if !machine.exists then TypeRepr.of[State]
        else
          VarSchemaMacro
            .stateVars(machine)
            .foldLeft(TypeRepr.of[State]): (state, variable) =>
              Refinement(state, variable.name, VarSchemaMacro.declaredType(machine, variable))
      Refinement(record, v.name, state)
    record.asType match
      case '[r] => '{ ComposeRuntime.start.asInstanceOf[r] }

  def find(target: Expr[Unit])(using q: Quotes): Expr[Find] =
    import q.reflect.*
    def named(term: Term): Option[String] = term match
      case Inlined(_, _, body) => named(body)
      case Typed(body, _)      => named(body)
      case Ident(n)            => Some(n)
      case Select(_, n)        => Some(n)
      case _                   => None
    named(target.asTerm) match
      case Some(n) => '{ Find(${ Expr(n) }) }
      case None    => report.errorAndAbort("find names a claim of queries: find(reopened)")

// The members' states a step started from, which `before` types by the members' vars.
final class Before private[compose] (states: Map[String, State]) extends Selectable:
  def selectDynamic(member: String): Any = states(member)
