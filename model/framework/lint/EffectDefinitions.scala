package framework.lint

import framework.inspect.{Index, LiftError}

final private[framework] class EffectDefinitions(index: Index):
  import index.quotes.reflect.*
  import index.given

  private val machine = Symbol.requiredClass("framework.dsl.Machine")
  private val record = Symbol.requiredClass("framework.dsl.RecordMachine")
  private val state = Symbol.requiredClass("framework.dsl.State")
  private val facet = Symbol.requiredClass("framework.dsl.Facet")

  private def native(owner: Symbol): Boolean =
    !owner.isNoSymbol && owner.isClassDef && (
      owner.typeRef.derivesFrom(machine) || (owner.typeRef.baseType(record) match
        case AppliedType(_, first :: _) => first.dealias.typeSymbol == state
        case _                           => false)
    )

  private def effects(owner: Symbol): Boolean =
    !owner.isNoSymbol && owner.flags.is(Flags.Module) && owner.name.stripSuffix("$") == "effects" && (
      native(owner.maybeOwner) || (
        owner.maybeOwner.flags.is(Flags.Module) &&
          !owner.maybeOwner.maybeOwner.isNoSymbol &&
          owner.maybeOwner.maybeOwner.isClassDef &&
          owner.maybeOwner.maybeOwner.typeRef.derivesFrom(facet)
      )
    )

  private def first(tree: Tree): Tree = tree match
    case Block(statements, result)     => first(statements.headOption.getOrElse(result))
    case Inlined(_, bindings, result) => first(bindings.headOption.getOrElse(result))
    case Typed(value, _)               => first(value)
    case _                             => tree

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    index.defs.values.toSeq.collect {
      case method: DefDef
          if !method.symbol.flags.is(Flags.Synthetic) &&
            !method.symbol.flags.is(Flags.FieldAccessor) && method.name != "<init>" &&
            effects(method.symbol.maybeOwner) => method
    }.flatMap { method =>
      val source = method.pos.sourceFile.path
      val file = index.sourceRoots.getOrElse(source, "") + source
      if exempt(file) then Nil
      else
        val explicit = !method.returnTpt.isInstanceOf[Inferred]
        val header = (List(method.pos.startLine) ++
          method.termParamss.flatMap(_.params.map(_.pos.endLine)) ++
          Option.when(explicit)(method.returnTpt.pos.endLine)).max
        val line = index.sourceLine(file, method.pos.start)
        def refuse(rule: String, message: String) =
          (file, line, LiftError(s"$file:$line", s"$rule: ${method.name} $message"))
        Option.when(explicit)(refuse(
          "R4.effect-return-type",
          "declares a return type: let its Unit result be inferred"
        )).toList ++
          Option.when(method.rhs.exists(body => first(body).pos.startLine <= header))(refuse(
            "R4.effect-multiline",
            "starts its body on the definition line: put the effect body on the next line"
          )).toList
    }
