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

final private[reify] case class FacetOutcome(
    facet: String,
    namespace: String,
    root: String,
    model: Option[ir.Model],
    sites: Vector[ContextualPhaseSite],
    refusal: Option[(ir.Position, String)]
)

final private[reify] case class StatusOutcome(
    root: String,
    phase: String,
    name: String,
    value: Option[ir.Expr],
    model: Option[ir.Model],
    refusal: Option[(ir.Position, String)],
    proofs: Vector[CallGroupEvidence] = Vector.empty
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
    facet: quotes.reflect.Symbol => Option[FacetOutcome],
    status: quotes.reflect.Symbol => Option[StatusOutcome],
    outcome: quotes.reflect.Symbol => Option[StatusOutcome],
    sources: Map[String, String],
    sourceLabel: quotes.reflect.Symbol => Option[String],
    retainedCallGroups: (quotes.reflect.Symbol, ir.Model) => Vector[CallGroupEvidence]
) extends BodyContext:
  import quotes.reflect.*
  private given CanEqual[ir.Function, ir.Function] = CanEqual.derived
  private given CanEqual[ir.Type, ir.Type] = CanEqual.derived

  override val defs: mutable.LinkedHashMap[Symbol, Definition] =
    mutable.LinkedHashMap.empty[Symbol, Definition]
  private val trees = mutable.Map.empty[Symbol, Tree]
  private var refused = Option.empty[Tree]
  val sourceLabels = mutable.Map.from(sources)

  private def retainSource(sym: Symbol, position: Option[ir.Position]): Unit =
    for label <- sourceLabel(sym); at <- position if at.file.nonEmpty do
      sourceLabels(at.file) = label

  def retainTypeSource(sym: Symbol, position: ir.Position): Unit =
    retainSource(sym, Some(position))

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
    if contextualFacet.isEmpty && facetObject(sym.maybeOwner.maybeOwner) then None
    else
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

  def importedFacet(sym: Symbol, at: Tree): Option[String] =
    if contextualFacet.isEmpty then None
    else
      facet(sym).map { outcome =>
        val namespace = sym.maybeOwner.maybeOwner
        if !facetObject(namespace) || outcome.root != sym.fullName ||
          outcome.namespace != namespace.fullName || outcome.facet != namespace.maybeOwner.fullName
        then fail(at, s"${sym.fullName} has stale facet-template provenance")
        outcome.refusal.foreach((position, message) =>
          throw SectionRefusal(position, outcome.root, message)
        )
        val model =
          outcome.model.getOrElse(fail(at, s"${sym.fullName} has no stored facet template"))
        importModel(sym, at, model)
        contextualSites ++= outcome.sites
        outcome.root
      }

  def importedStatus(sym: Symbol, at: Tree): Option[(ir.Expr, String)] =
    if defs.contains(sym) then None
    else
      status(sym).map { outcome =>
        if outcome.root != sym.fullName || outcome.phase != sym.owner.companionClass.fullName then
          fail(at, s"${sym.fullName} has stale Recorded status provenance")
        outcome.refusal.foreach((position, message) =>
          throw SectionRefusal(position, outcome.root, message)
        )
        val model =
          outcome.model.getOrElse(fail(at, s"${sym.fullName} has no stored Recorded status"))
        retainSource(sym, outcome.value.flatMap(_.position))
        importClosure(at, model)
        callGroups ++= outcome.proofs.map(p =>
          if p.owner.isEmpty then p.copy(owner = callOwner) else p
        )
        outcome.value.getOrElse(
          fail(at, s"${sym.fullName} has no retained status fact")
        ) -> outcome.name
      }

  def importedOutcome(ok: Term): Option[ir.Expr] = ok match
    case reference: Ref if contextualFacet.nonEmpty && !defs.contains(resolveSymbol(reference)) =>
      val symbol = resolveSymbol(reference)
      outcome(symbol).map { retained =>
        if retained.root != symbol.fullName || retained.phase != "framework.dsl.Ok" then
          fail(ok, s"${symbol.fullName} has stale Ok outcome provenance")
        retained.refusal.foreach((position, message) =>
          throw SectionRefusal(position, retained.root, message)
        )
        val model =
          retained.model.getOrElse(fail(ok, s"${symbol.fullName} has no retained Ok closure"))
        retainSource(symbol, retained.value.flatMap(_.position))
        importClosure(ok, model)
        callGroups ++= retained.proofs.map(p =>
          if p.owner.isEmpty then p.copy(owner = callOwner) else p
        )
        retained.value.getOrElse(fail(ok, s"${symbol.fullName} has no retained Ok outcome"))
      }
    case _ => None

  private def importModel(sym: Symbol, at: Tree, model: ir.Model): Unit =
    retainSource(sym, model.functions.find(_.name == sym.fullName).flatMap(_.position))
    importClosure(at, model)
    if !functions.contains(sym.fullName) then
      fail(at, s"${sym.fullName}'s stored IR has no root function")
    callGroups ++= retainedCallGroups(sym, model)

  private def importClosure(at: Tree, model: ir.Model): Unit =
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

private[reify] class PluginEngine(val ctx: PluginContext) extends BodyEngine:
  import ctx.*
  import ctx.given
  import ctx.quotes.reflect.*

  override def storedFunction(sym: Symbol, at: Tree): Option[String] =
    importedFacet(sym, at).orElse(imported(sym, at))
  override def storedSectionFunction(sym: Symbol, at: Tree): Option[String] =
    importedFacet(sym, at).orElse(importedSection(sym, at))

  override def storedStatus(sym: Symbol, at: Tree): Option[(ir.Expr, String)] =
    importedStatus(sym, at)

  override def outcomeOf(ok: Term, form: String): ir.Expr =
    importedOutcome(ok).getOrElse(super.outcomeOf(ok, form))

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

  private def contextualPhase(
      kind: String,
      owner: Symbol,
      args: Seq[ir.Expr],
      at: Tree,
      placement: Option[ir.Position]
  ): ir.Expr =
    val phased = owner.typeRef.baseType(Symbol.requiredClass("framework.dsl.Phased")).typeArgs
    if phased.size != 2 then fail(at, s"${owner.fullName} has no typed contextual phase")
    val state = typeRef(phased.head, at)
    val phase = typeRef(phased(1), at)
    val result = if kind == "read" then phase else state
    val call = ir.Call(s"$$facet-context-v1.${owner.fullName}.$kind", args)
    contextualSites += ContextualPhaseSite(kind, callOwner, call, state, phase, result, placement)
    expr(at)(ir.Expr.Kind.Call(call))

  override def contextualPhaseRead(
      state: ir.Expr,
      at: Tree,
      placement: Option[ir.Position],
      owner: Option[Symbol]
  ): Option[ir.Expr] = contextualFacet.flatMap { current =>
    val declared = owner.getOrElse(current)
    Option.when(facetTrait(declared))(contextualPhase("read", declared, Seq(state), at, placement))
  }

  override def contextualPhaseWrite(
      owner: Symbol,
      state: Either[String, Term],
      phase: Term,
      at: Tree
  ): Option[ir.Expr] = contextualFacet.filter(_ => facetTrait(owner)).map { _ =>
    val argument = state.fold(name => expr(at)(ir.Expr.Kind.Var(name)), lift(_))
    contextualPhase("write", owner, Seq(argument, lift(phase)), at, None)
  }
