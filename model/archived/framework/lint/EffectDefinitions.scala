package framework.lint

import framework.inspect.{Index, LiftError}

final private[framework] class EffectDefinitions(index: Index):
  import index.quotes.reflect.*
  import index.given

  private val model = ModelDeclarations[index.type](index)

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    index.defs.values.toSeq
      .collect {
        case method: DefDef
            if !method.symbol.flags.is(Flags.Synthetic) &&
              !method.symbol.flags.is(
                Flags.FieldAccessor
              ) && method.name != "<init>" && model.effects(method.symbol.maybeOwner) =>
          method
      }
      .flatMap { method =>
        val file = model.file(method)
        if exempt(file) || !model.authored(file) || model.defaultGetter(method) then Nil
        else
          val explicit = method.returnTpt match
            case Inferred() => false
            case _          => true
          val line = index.sourceLine(file, method.pos.start)
          def refuse(rule: String, message: String) =
            (file, line, LiftError(s"$file:$line", s"$rule: ${method.name} $message"))
          Option
            .when(explicit)(
              refuse(
                "R4.effect-return-type",
                "declares a return type: let its Unit result be inferred"
              )
            )
            .toList
      }
