package framework.lint

import framework.inspect.{Index, LiftableSubset, LiftError}
import scala.collection.mutable

final private[framework] class Subset(index: Index):
  import index.quotes.reflect.*
  import index.given

  def refusals(exempt: String => Boolean): Seq[(String, Int, LiftError)] =
    val refused = mutable.ArrayBuffer.empty[(String, Int, LiftError)]
    val seen = mutable.Set.empty[Symbol]
    var sugar = Option.empty[Term] // scalafix:ok DisableSyntax.var
    def fileOf(t: Tree): String =
      val path = t.pos.sourceFile.path
      index.sourceRoots.getOrElse(path, "") + path
    def semantic(t: TypeRepr): Boolean =
      val result = t.widen.dealias
      result =:= defn.BooleanClass.typeRef ||
      result.typeSymbol.fullName == "framework.dsl.Step" ||
      result.typeArgs.exists(a => a.widen.dealias.typeSymbol.fullName == "framework.dsl.Step")
    def generatedValue(sym: Symbol): Boolean =
      sym.isClassConstructor || sym.flags.is(Flags.FieldAccessor) ||
        (sym.name == "apply" && sym.maybeOwner.companionClass.flags.is(Flags.Case)) ||
        ((sym.name == "copy" || sym.name.startsWith("copy$default$")) && sym.maybeOwner.flags.is(
          Flags.Case
        ))
    def refuse(t: Tree, construct: String): Unit =
      val at = sugar.getOrElse(t)
      val file = fileOf(at)
      val line =
        scala.util.Try(index.sourceLine(file, at.pos.start)).getOrElse(at.pos.startLine + 1)
      val origin = sugar.fold("")(call => s"sugar ${call.symbol.fullName} cannot be lifted: ")
      refused += ((
        file,
        line,
        LiftError(
          s"$file:$line",
          s"${origin}liftable subset: $construct has no function-body IR form; see ${LiftableSubset.subsetTable}"
        )
      ))
    def inspect(body: Tree): Unit = inspectBody(body, draft = false)
    def inspectBody(body: Tree, draft: Boolean): Unit =
      def metadata(tpe: TypeRepr): Boolean = Set(
        "scala.reflect.ClassTag",
        "scala.reflect.TypeTest",
        "scala.CanEqual",
        "scala.ValueOf",
        "framework.dsl.Finite"
      )(tpe.widen.dealias.typeSymbol.fullName)
      def family(t: Term): Boolean = t match
        case Apply(Select(value: Ref, "forall"), _) if value.symbol.flags.is(Flags.Param) =>
          value.symbol.maybeOwner.tree match
            case d: DefDef =>
              d.returnTpt.tpe.widen.dealias.typeSymbol.fullName == "framework.dsl.Property" &&
              value.tpe.widen.dealias.typeArgs.exists(_.typeSymbol.name == "Fact")
            case _ => false
        case _ => false
      object expressions extends TreeTraverser:
        override def traverseTree(t: Tree)(owner: Symbol): Unit = t match
          case Inlined(call, bindings, body) =>
            val saved = sugar
            sugar = saved.orElse(call.collect { case term: Term => term })
            try
              bindings.foreach(inspectBody(_, draft))
              inspectBody(body, draft)
            finally sugar = saved
          case Literal(UnitConstant()) if draft                                        => ()
          case a: Apply if a.symbol.fullName == "framework.dsl.Syntax$package$.effect" =>
            a.args.foreach(arg => inspectBody(arg, draft = true))
          case Unapply(_, _, fields) => fields.foreach(inspect)
          case TypedOrTest(inner, _) => inspectBody(inner, draft)
          case _ if LiftableSubset.unsupported(using index.quotes)(t).nonEmpty =>
            refuse(t, LiftableSubset.unsupported(using index.quotes)(t).get)
          case a: Apply =>
            val sym = a.symbol
            val ownerName = sym.maybeOwner.fullName.stripSuffix("$")
            val contextual = a.fun.tpe match
              case m: MethodType if m.isImplicit => Some(m.paramTypes)
              case _                             => None
            if LiftableSubset
                .library(ownerName) && !LiftableSubset.permitted(using index.quotes)(a, family(a))
            then refuse(a, s"library operation ${sym.name} of $ownerName")
            else if index.defs.contains(sym) && !sym.flags.is(Flags.Synthetic) && !generatedValue(
                sym
              ) && !seen(sym) &&
              index.authorSource(fileOf(index.defs(sym)))
            then
              seen += sym: Unit
              index.defs(sym) match
                case d: DefDef => d.rhs.foreach(inspectBody(_, draft))
                case _         => ()
            if contextual.nonEmpty then
              traverseTree(a.fun)(owner)
              a.args
                .zip(contextual.get)
                .filterNot((_, tpe) => metadata(tpe))
                .foreach((argument, _) => inspect(argument))
            else if sym.fullName == "framework.dsl.Machine$package$.choose" then
              a.args.foreach { argument =>
                def alternatives(t: Term): Unit = t match
                  case Typed(value, _)    => alternatives(value)
                  case Repeated(items, _) => items.foreach(alternatives)
                  case Apply(TypeApply(Select(Apply(_, List(token)), "->"), _), List(value)) =>
                    inspect(token)
                    inspect(value)
                  case other => inspect(other)
                alternatives(argument)
              }
            else super.traverseTree(t)(owner)
          case r: Ref
              if index.defs.contains(r.symbol) && !seen(r.symbol) &&
                !generatedValue(r.symbol) &&
                !r.symbol.flags.is(Flags.Param) && !r.symbol.flags.is(
                  Flags.Module
                ) && !r.symbol.flags.is(Flags.Enum) &&
                index.authorSource(fileOf(index.defs(r.symbol))) =>
            seen += r.symbol: Unit
            index.defs(r.symbol) match
              case v: ValDef => v.rhs.foreach(inspectBody(_, draft))
              case d: DefDef => d.rhs.foreach(inspectBody(_, draft))
              case _         => ()
            super.traverseTree(t)(owner)
          case _ => super.traverseTree(t)(owner)
      expressions.traverseTree(body)(Symbol.spliceOwner)
    object authored extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit = t match
        case d: DefDef
            if !d.symbol.flags.is(Flags.Synthetic) && !generatedValue(d.symbol) && !d.name
              .startsWith("$") &&
              (d.returnTpt.tpe.widen.dealias.typeArgs
                .exists(_.typeSymbol.fullName == "framework.dsl.Step") ||
                (d.name == "end" && Seq(
                  "framework.dsl.Machine",
                  "framework.dsl.Composition",
                  "framework.dsl.Facet"
                )
                  .exists(name =>
                    d.symbol.maybeOwner.typeRef.baseClasses.exists(_.fullName == name)
                  ))) && !seen(d.symbol) =>
          seen += d.symbol: Unit
          d.rhs.foreach(inspect)
        case a: Apply if a.symbol.maybeOwner.fullName.startsWith("framework.dsl.") =>
          a.args
            .filter(arg =>
              arg.tpe.widen.dealias.isFunctionType && semantic(arg.tpe.widen.dealias.typeArgs.last)
            )
            .foreach(inspect)
          super.traverseTree(t)(owner)
        case _ => super.traverseTree(t)(owner)
    index.trees
      .filter(t => index.authored(t) && !exempt(fileOf(t)) && !fileOf(t).endsWith(".test.scala"))
      .foreach(authored.traverseTree(_)(Symbol.spliceOwner))
    refused.toSeq.distinct
