package framework.reify

import scala.collection.mutable
import scala.quoted.Quotes
import io.temporal.server.api.umpire.v1 as ir

final private[reify] class PluginContext(using val quotes: Quotes)(
    root: quotes.reflect.Tree,
    load: quotes.reflect.Symbol => Option[ir.Model],
    inRun: quotes.reflect.Symbol => Boolean,
    inJar: quotes.reflect.Symbol => Boolean,
    jarSugar: quotes.reflect.Symbol => Option[quotes.reflect.Tree]
) extends BodyContext:
  import quotes.reflect.*
  private given CanEqual[ir.Function, ir.Function] = CanEqual.derived
  private given CanEqual[ir.Type, ir.Type] = CanEqual.derived

  override val defs: mutable.LinkedHashMap[Symbol, Definition] =
    mutable.LinkedHashMap.empty[Symbol, Definition]
  private val trees = mutable.Map.empty[Symbol, Tree]
  private var refused = Option.empty[Tree]

  object collect extends TreeTraverser:
    override def traverseTree(tree: Tree)(owner: Symbol): Unit =
      tree match
        case d: Definition =>
          trees(d.symbol) = d
          d match
            case v: ValDef => defs(v.symbol) = v
            case d: DefDef => defs(d.symbol) = d
            case _         => ()
        case _ => ()
      super.traverseTree(tree)(owner)
  collect.traverseTree(root)(Symbol.spliceOwner)

  private def sugar(sym: Symbol): Boolean =
    sym.annotations.exists(_.tpe.typeSymbol.fullName == "framework.dsl.sugar")

  override def expansionAllowed(sym: Symbol): Boolean =
    defs.contains(sym) || (!inRun(sym) && inJar(sym))

  def sourceTree(sym: Symbol): Option[Tree] =
    trees.get(sym).orElse {
      if !sym.isNoSymbol && !inRun(sym) && inJar(sym) && (sugar(sym) || sugar(sym.owner)) then
        jarSugar(sym)
      else None
    }

  def definition(sym: Symbol): Option[Definition] =
    defs.get(sym).orElse(sourceTree(sym).collect { case d: Definition => d })

  override def isFunction(sym: Symbol): Boolean =
    super.isFunction(sym) || (sym.isDefDef && !sym.fullName.startsWith("framework.dsl.") &&
      !sym.fullName.startsWith("scala.") && !sym.fullName.startsWith("java."))

  protected def written(t: Tree): ir.Position =
    scala.util
      .Try(ir.Position(t.pos.sourceFile.path, t.pos.startLine + 1))
      .getOrElse(ir.Position.defaultInstance)

  def refusedAt: Tree = refused.getOrElse(root)

  def refuse(t: Tree, message: String): Nothing =
    refused = Some(t)
    throw LiftError(where(t), message)

  def imported(sym: Symbol, at: Tree): Option[String] =
    if defs.contains(sym) then None
    else
      val model = load(sym).getOrElse(
        fail(
          at,
          s"${sym.fullName} is reached outside this capture's compilation unit: " +
            "mark its definition @lifted so its IR is stored at the definition site"
        )
      )
      for f <- model.functions do
        functions
          .get(f.name)
          .filter(_ != f)
          .foreach(_ => fail(at, s"stored function ${f.name} has two different definitions"))
        functions(f.name) = f
      for t <- model.types do
        types
          .get(t.name)
          .filter(_.clearPosition != t.clearPosition)
          .foreach(_ => fail(at, s"stored type ${t.name} has two different definitions"))
        types(t.name) = t
      if !functions.contains(sym.fullName) then
        fail(at, s"${sym.fullName}'s stored IR has no root function")
      Some(sym.fullName)

final private[reify] class PluginEngine(val ctx: PluginContext) extends BodyEngine:
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  override def storedFunction(sym: Symbol, at: Tree): Option[String] = imported(sym, at)

  override def functionParameterTypes(sym: Symbol): List[TypeRepr] =
    val local = super.functionParameterTypes(sym)
    if local.nonEmpty then local
    else
      def parameters(t: TypeRepr): List[TypeRepr] = t match
        case p: PolyType   => parameters(p.resType)
        case m: MethodType => m.paramTypes ++ parameters(m.resType)
        case _             => Nil
      parameters(sym.termRef.widen)

  def channelOf(sym: Symbol, at: Tree): String = definitionId(sym, at)
  def holeOf(sym: Symbol, at: Tree): String = definitionId(sym, at)
  def machineNamed(name: String): Option[ir.Machine] = machines.values.find(_.name == name)

  def phaseWriter(cls: Symbol, at: Tree): Option[Term] =
    sourceTree(cls).collect { case c: ClassDef => c }.flatMap { c =>
      c.parents.collectFirst {
        case Apply(Apply(TypeApply(fn, _), List(_, writer)), _)
            if fn.symbol.maybeOwner.fullName == "framework.dsl.Phased" =>
          writer
        case Apply(TypeApply(fn, _), List(_, writer))
            if fn.symbol.maybeOwner.fullName == "framework.dsl.Phased" =>
          writer
      }
    }
