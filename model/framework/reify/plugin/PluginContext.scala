package framework.reify

import scala.collection.mutable
import scala.quoted.Quotes
import io.temporal.server.api.umpire.v1 as ir

final private[reify] case class SectionOutcome(
    root: String,
    form: String,
    model: Option[ir.Model],
    refusal: Option[(ir.Position, String)]
)

final private[reify] class SectionRefusal(val position: ir.Position, root: String, message: String)
    extends Exception(s"${position.file}:${position.line}: $root: $message")

final private[reify] class PluginContext(using val quotes: Quotes)(
    root: quotes.reflect.Tree,
    outcomes: List[quotes.reflect.Tree],
    load: quotes.reflect.Symbol => Option[ir.Model],
    inRun: quotes.reflect.Symbol => Boolean,
    inJar: quotes.reflect.Symbol => Boolean,
    jarSugar: quotes.reflect.Symbol => Option[quotes.reflect.Tree],
    sectionPresent: quotes.reflect.Symbol => Boolean,
    section: quotes.reflect.Symbol => Option[SectionOutcome],
    retainedCallGroups: (quotes.reflect.Symbol, ir.Model) => Vector[CallGroupEvidence]
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
  outcomes.foreach {
    case v: ValDef =>
      defs(v.symbol) = v
      trees(v.symbol) = v
    case _ => ()
  }

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
    super.isFunction(sym) || sectionPresent(sym) || (sym.isDefDef && !sym.fullName.startsWith(
      "framework.dsl."
    ) &&
      !sym.fullName.startsWith("scala.") && !sym.fullName.startsWith("java.") &&
      sym.maybeOwner.fullName != "<special-ops>")

  override def sectionValue(sym: Symbol): Boolean = super.sectionValue(sym) || sectionPresent(sym)

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
      importModel(sym, at, model)
      Some(sym.fullName)

  def importedSection(sym: Symbol, at: Tree): Option[String] =
    section(sym).map { outcome =>
      if outcome.root != sym.fullName || !Set("is", "effect")(outcome.form) then
        fail(at, s"${sym.fullName} has stale section-value provenance")
      outcome.refusal.foreach((position, message) =>
        throw SectionRefusal(position, outcome.root, message)
      )
      val model =
        outcome.model.getOrElse(fail(at, s"${sym.fullName} has no stored section outcome"))
      importModel(sym, at, model)
      outcome.root
    }

  private def importModel(sym: Symbol, at: Tree, model: ir.Model): Unit =
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
    callGroups ++= retainedCallGroups(sym, model)

private[reify] class PluginEngine(val ctx: PluginContext) extends BodyEngine:
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  override def storedFunction(sym: Symbol, at: Tree): Option[String] = imported(sym, at)
  override def storedSectionFunction(sym: Symbol, at: Tree): Option[String] =
    importedSection(sym, at)

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
