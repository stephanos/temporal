package framework.reify

import scala.collection.mutable
import scala.quoted.Quotes
import scala.quoted.runtime.impl.QuotesImpl
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Phases.inliningPhase
import dotty.tools.dotc.core.Symbols.Symbol as CompilerSymbol
import dotty.tools.dotc.inlines.Inlines
import io.temporal.server.api.umpire.v1 as ir

// A compiler tree the plugin's own rewrite does not find in the shape it wrote.
final private[reify] class CompilerApiError(detail: String) extends Exception(detail)

final private[reify] class PluginContext(using val quotes: Quotes)(
    root: quotes.reflect.Tree,
    dependencies: List[quotes.reflect.Tree],
    covered: quotes.reflect.Symbol => Boolean,
    load: quotes.reflect.Symbol => Option[ir.Model]
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
  dependencies.foreach(collect.traverseTree(_)(Symbol.spliceOwner))

  def sourceTree(sym: Symbol): Option[Tree] = trees.get(sym)

  private def currentDefinition: Option[Definition] =
    callOwner.flatMap(name => defs.valuesIterator.find(_.symbol.fullName == name))

  override def inlineCall(t: Term): Option[Term] =
    val compiler = quotes.asInstanceOf[QuotesImpl].ctx
    val inlining = compiler.fresh.setPhase(inliningPhase(using compiler))
    given Context = currentDefinition
      .fold(inlining)(owner => inlining.setOwner(owner.symbol.asInstanceOf[CompilerSymbol]))
    val tree = t.asInstanceOf[tpd.Tree]
    Option.when(Inlines.needsInlining(tree))(
      Inlines.inlineCall(tree).asInstanceOf[Term]
    )

  def definition(sym: Symbol): Option[Definition] =
    defs.get(sym).orElse(sourceTree(sym).collect { case d: Definition => d })

  override def opaqueRange(sym: Symbol): Option[(Long, Long)] =
    opaqueRange(sym, Set.empty)

  private def opaqueRange(sym: Symbol, seen: Set[Symbol]): Option[(Long, Long)] =
    super
      .opaqueRange(sym)
      .orElse {
        sym.annotations.collectFirst {
          case Apply(fn, List(Literal(LongConstant(low)), Literal(LongConstant(high))))
              if fn.symbol.owner.fullName == "framework.dsl.finiteRange" =>
            (low, high)
        }
      }
      .orElse {
        if seen(sym) then None
        else
          defs.values.iterator
            .flatMap {
              case ValDef(_, tpt, Some(rhs: Ref))
                  if tpt.tpe.typeSymbol.fullName == "framework.dsl.Finite" &&
                    tpt.tpe.typeArgs.headOption.exists(_.typeSymbol == sym) =>
                rhs.tpe.widen.typeArgs.headOption
                  .flatMap(t => opaqueRange(t.typeSymbol, seen + sym))
              case _ => None
            }
            .nextOption()
      }

  override def isFunction(sym: Symbol): Boolean =
    super.isFunction(sym) || (sym.isDefDef && !sym.fullName.startsWith(
      "framework.dsl."
    ) &&
      !sym.fullName.startsWith("scala.") && !sym.fullName.startsWith("java.") &&
      sym.maybeOwner.fullName != "<special-ops>")

  protected def written(t: Tree): ir.Position =
    scala.util
      .Try(ir.Position(t.pos.sourceFile.path, t.pos.startLine + 1))
      .getOrElse(ir.Position.defaultInstance)

  def refusedAt: Tree = refused.getOrElse(root)

  def refuse(t: Tree, message: String): Nothing =
    refused = Some(t)
    throw LiftError(where(t), message)

  def imported(sym: Symbol, at: Tree): Option[String] =
    val file = written(currentDefinition.getOrElse(root)).file
    if defs.get(sym).exists(d => written(d).file == file || covered(sym)) then None
    else
      val model = load(sym).getOrElse(
        fail(
          at,
          s"${sym.fullName} is reached from another file: define it in an object that extends Lifted"
        )
      )
      importClosure(at, model)
      Some(sym.fullName)

  // A stored model keeps a position on its root function only, so the same function can arrive
  // with and without positions; the positioned one is kept.
  private def importClosure(at: Tree, model: ir.Model): Unit =
    for f <- model.functions do
      functions.get(f.name) match
        case Some(known)
            if Carrier.unpositioned(known, ir.Function) != Carrier.unpositioned(f, ir.Function) =>
          fail(at, s"stored function ${f.name} has two different definitions")
        case Some(known) if known.position.nonEmpty => ()
        case _                                      => functions(f.name) = f
    for t <- model.types do
      types
        .get(t.name)
        .filter(_.clearPosition != t.clearPosition)
        .foreach(_ => fail(at, s"stored type ${t.name} has two different definitions"))
      types(t.name) = t

private[reify] class PluginEngine(val ctx: PluginContext) extends BodyEngine:
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  override def storedFunction(sym: Symbol, at: Tree): Option[String] =
    imported(sym, at)

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

  private def phasedArguments(cls: Symbol): Option[List[Term]] =
    sourceTree(cls).collect { case c: ClassDef => c }.flatMap { c =>
      c.parents.collectFirst {
        case Apply(Apply(TypeApply(fn, _), args), _)
            if fn.symbol.maybeOwner.fullName == "framework.dsl.Phased" =>
          args
        case Apply(TypeApply(fn, _), args)
            if fn.symbol.maybeOwner.fullName == "framework.dsl.Phased" =>
          args
      }
    }

  def phaseProjection(cls: Symbol): Option[Term] = phasedArguments(cls).flatMap(_.headOption)
  def phaseWriter(cls: Symbol, at: Tree): Option[Term] = phasedArguments(cls).flatMap(_.lift(1))
