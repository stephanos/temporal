package framework.dsl

import scala.quoted.*

private[framework] object CompositionMacros:
  def enclosing(using q: Quotes): Option[q.reflect.Symbol] =
    import q.reflect.*
    val base = TypeRepr.of[Composition[?]].typeSymbol
    Iterator
      .iterate(Symbol.spliceOwner)(_.owner)
      .takeWhile(!_.isNoSymbol)
      .find(s => s.isClassDef && !s.equals(base) && s.typeRef.baseClasses.contains(base))

  private def members(using
      q: Quotes
  )(cls: q.reflect.Symbol): List[(q.reflect.Symbol, q.reflect.TypeRepr)] =
    import q.reflect.*
    cls.declarations
      .filter(s =>
        s.isValDef && !s.flags.is(Flags.Mutable) &&
          !s.flags.is(Flags.Synthetic) && !s.flags.is(Flags.Module) && !s.name.contains("$")
      )
      .map(v => v -> cls.typeRef.memberType(v).widenByName)
      .filter: (v, tpe) =>
        if tpe <:< TypeRepr.of[NativeModel[?, ?, ?]] then true
        else
          tpe match
            case ref: TermRef if ref.termSymbol.flags.is(Flags.Module) =>
              report.error(
                s"${v.name} names ${ref.termSymbol.name}, which is neither a Machine nor a Composition",
                v.pos.getOrElse(Position.ofMacroExpansion)
              )
            case _ => ()
          false

  def schema[F: Type](using q: Quotes): Expr[CompositionSchema[F]] =
    import q.reflect.*
    val cls = enclosing.getOrElse(
      report.errorAndAbort(
        "a composition declares its members where its object extends Composition"
      )
    )
    val declared = members(cls)
    if declared.isEmpty then report.error("a composition declares at least one member val")
    for
      ((v, tpe), at) <- declared.zipWithIndex
      (earlier, _) <- declared
        .take(at)
        .find((_, previous) => previous =:= tpe && tpe.termSymbol.exists)
    do
      report.error(
        s"${v.name} names ${tpe.termSymbol.name}, which ${earlier.name} names already: a Machine is a member once",
        v.pos.getOrElse(Position.ofMacroExpansion)
      )
    val fact = TypeRepr.of[F].dealias
    val base = TypeRepr.of[NativeModel[?, ?, ?]].typeSymbol
    val tags = declared.map: (v, tpe) =>
      val memberFact = tpe.baseType(base).typeArgs(2)
      if memberFact =:= TypeRepr.of[Nothing] then '{ None }
      else
        fact.typeSymbol.children.find(_.name == v.name) match
          case None =>
            report.error(
              s"${v.name} records ${memberFact.show}: ${fact.show} declares a case for it, case ${v.name}(value: ${memberFact.show})",
              v.pos.getOrElse(Position.ofMacroExpansion)
            )
            '{ None }
          case Some(tag) =>
            if tag.caseFields.size != 1 || !(tag.typeRef.memberType(
                tag.caseFields.head
              ) =:= memberFact)
            then
              report.error(
                s"${fact.show}.${v.name} carries one value of ${memberFact.show}",
                v.pos.getOrElse(Position.ofMacroExpansion)
              )
              '{ None }
            else
              memberFact.asType match
                case '[t] =>
                  '{
                    Some((value: Any) =>
                      ${
                        Select
                          .overloaded(
                            Ref(tag.companionModule),
                            "apply",
                            Nil,
                            List('{ value.asInstanceOf[t] }.asTerm)
                          )
                          .asExprOf[F]
                      }
                    )
                  }
    '{ CompositionSchema.declared[F](${ Expr(declared.map(_._1.name)) }, ${ Expr.ofList(tags) }) }

  private def sourceClass(using q: Quotes)(cls: q.reflect.Symbol): q.reflect.Symbol =
    import q.reflect.*
    val index = cls.typeRef.baseType(TypeRepr.of[Composition[?]].typeSymbol).typeArgs.head
    if !(index =:= TypeRepr.of[Nothing]) && index <:< TypeRepr.of[Composition[?]] then
      sourceClass(
        if index.termSymbol.exists then index.termSymbol.moduleClass else index.typeSymbol
      )
    else cls

  def stateType(using q: Quotes)(cls: q.reflect.Symbol): q.reflect.TypeRepr =
    import q.reflect.*
    val composition = TypeRepr.of[Composition[?]].typeSymbol
    if cls.typeRef.baseClasses.contains(composition) then
      members(sourceClass(cls)).foldLeft(TypeRepr.of[State]): (record, member) =>
        val (v, tpe) = member
        val declared = if tpe.termSymbol.exists then tpe.termSymbol.moduleClass else tpe.typeSymbol
        Refinement(record, v.name, stateType(declared))
    else
      def declaring(tpe: TypeRepr): Symbol =
        val symbol = if tpe.termSymbol.exists then tpe.termSymbol.moduleClass else tpe.typeSymbol
        if VarSchemaMacro.stateVars(symbol).nonEmpty then symbol
        else
          tpe.baseType(TypeRepr.of[DerivedFrom[?]].typeSymbol).typeArgs.headOption match
            case Some(source) => declaring(source)
            case None         => symbol
      val declared = declaring(cls.typeRef)
      VarSchemaMacro
        .stateVars(declared)
        .foldLeft(TypeRepr.of[State]): (record, variable) =>
          Refinement(record, variable.name, VarSchemaMacro.declaredType(declared, variable))
