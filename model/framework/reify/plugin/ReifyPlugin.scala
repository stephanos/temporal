package framework.reify

import java.util.Base64
import scala.collection.mutable
import scala.quoted.Quotes
import scala.quoted.runtime.impl.QuotesImpl
import dotty.tools.dotc.CompilationUnit
import dotty.tools.dotc.ast.tpd
import dotty.tools.dotc.ast.tpd.*
import dotty.tools.dotc.core.Annotations.Annotation
import dotty.tools.dotc.core.Constants.Constant
import dotty.tools.dotc.core.Contexts.Context
import dotty.tools.dotc.core.Flags.{Accessor, Case, Enum, Method, Param, ParamAccessor, Synthetic}
import dotty.tools.dotc.core.Types.MethodType
import dotty.tools.dotc.core.Names.termName
import dotty.tools.dotc.core.tasty.DottyUnpickler
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.plugins.{PluginPhase, StandardPlugin}
import dotty.tools.dotc.report
import dotty.tools.dotc.transform.Pickler
import io.temporal.server.api.umpire.v1 as ir

final class ReifyPlugin extends StandardPlugin:
  val name = "umpire-reify"
  override val description = "lifts Model functions at their definitions and captures"
  override def initialize(options: List[String])(using Context): List[PluginPhase] =
    val store = new DefinitionStore
    List(
      new FamilyPhase,
      new StorePhase(store),
      new CapturePhase(store),
      new NamesPhase,
      new SchemaPhase(store),
      new SchemaPhase(store, signatures = true)
    )

private object Carrier:
  val marker = "framework.dsl.lifted"
  val stored = "framework.dsl.liftedIr"
  val capture = "framework.dsl.capture"

  def chunks(model: ir.Model): List[String] =
    chunks(model.toByteArray)

  def chunks(bytes: Array[Byte]): List[String] =
    Base64.getEncoder.encodeToString(bytes).grouped(24000).toList

  def strings(tree: tpd.Tree): List[String] = tree match
    case Literal(c) if c.value.isInstanceOf[String] => List(c.stringValue)
    case Typed(t, _)                                => strings(t)
    case SeqLiteral(items, _)                       => items.flatMap(strings)
    case Apply(_, args)                             => args.flatMap(strings)
    case _                                          => Nil

  def read(sym: Symbol)(using Context): Option[ir.Model] =
    val cls = getClassIfDefined(stored)
    if !cls.exists then None
    else
      sym.getAnnotation(cls).map { a =>
        ir.Model.parseFrom(Base64.getDecoder.decode(a.arguments.flatMap(strings).mkString))
      }

  def literals(chunks: List[String])(using Context): tpd.Tree =
    SeqLiteral(chunks.map(s => Literal(Constant(s))), TypeTree(defn.StringType))

final private class DefinitionStore:
  private val active = mutable.Set.empty[Symbol]
  private val sites = mutable.Map.empty[Symbol, (CompilationUnit, tpd.DefDef)]
  private val declared = mutable.Set.empty[Symbol]
  private val sugarTrees = mutable.Map.empty[Symbol, tpd.Tree]
  private val sugarRoots = new java.util.IdentityHashMap[DottyUnpickler, List[tpd.Tree]]
  private var indexed = false
  private var compilerRun = Option.empty[dotty.tools.dotc.Run]

  private def enterRun()(using ctx: Context): Unit =
    if !compilerRun.exists(_ eq ctx.run) then
      active.clear()
      sites.clear()
      declared.clear()
      sugarTrees.clear()
      sugarRoots.clear()
      indexed = false
      compilerRun = Some(ctx.run)

  private def index()(using ctx: Context): Unit =
    enterRun()
    if !indexed then
      indexed = true
      for unit <- ctx.run.units do
        new tpd.TreeTraverser:
          def traverse(t: tpd.Tree)(using Context): Unit =
            t match
              case d: tpd.DefDef => sites(d.symbol) = unit -> d
              case _             => ()
            t match
              case d: tpd.MemberDef => declared += d.symbol
              case _                => ()
            traverseChildren(t)
        .traverse(unit.tpdTree)

  def inRun(sym: Symbol)(using Context): Boolean =
    index()
    declared(sym)

  def load(sym: Symbol)(using ctx: Context): Option[ir.Model] =
    index()
    Carrier.read(sym).orElse {
      sites
        .get(sym)
        .filter { (_, d) =>
          val marker = getClassIfDefined(Carrier.marker)
          marker.exists && d.symbol.hasAnnotation(marker)
        }
        .flatMap { (unit, d) =>
          given Context = ctx.fresh.setCompilationUnit(unit).setOwner(d.symbol)
          save(d)
        }
    }

  def engine(root: tpd.Tree)(using ctx: Context): PluginEngine =
    enterRun()
    val q: Quotes = QuotesImpl()
    val pc = new PluginContext(using q)(
      root.asInstanceOf[q.reflect.Tree],
      s => load(s.asInstanceOf[Symbol]),
      s => inRun(s.asInstanceOf[Symbol]),
      s => inJar(s.asInstanceOf[Symbol]),
      s => jarSugar(s.asInstanceOf[Symbol]).map(_.asInstanceOf[q.reflect.Tree])
    )
    new PluginEngine(pc)

  private def jarSugar(sym: Symbol)(using Context): Option[tpd.Tree] =
    enterRun()
    sugarTrees.get(sym).orElse {
      var found = Option.empty[tpd.Tree]
      val walk = new tpd.TreeTraverser:
        def traverse(t: tpd.Tree)(using Context): Unit =
          if t.symbol == sym then found = Some(t)
          else
            t match
              case _: tpd.DefDef | _: tpd.ValDef => ()
              case _ if found.isEmpty            => traverseChildren(t)
              case _                             => ()
      val provider = JarSugar.provider(sym)
      val cached = sugarRoots.get(provider)
      val roots = if cached != null then cached
      else
        val roots = provider.rootTrees
        sugarRoots.put(provider, roots): Unit
        roots
      roots.foreach(walk.traverse)
      found.foreach(t => sugarTrees(sym) = t)
      found
    }

  private def inJar(sym: Symbol)(using Context): Boolean =
    val file = sym.binaryFile
    file != null && file.underlyingSource.exists(_.path.endsWith(".jar"))

  def save(d: tpd.DefDef)(using ctx: Context): Option[ir.Model] =
    enterRun()
    Carrier.read(d.symbol).orElse {
      if !active.add(d.symbol) then
        report.error(
          s"${d.name} calls itself, directly or through another function: " +
            "a recursive function has no IR form, so write the bounded computation out",
          d.srcPos
        )
        None
      else
        val e = engine(ctx.compilationUnit.tpdTree)
        import e.ctx.quotes.reflect.DefDef
        try
          e.readFinites()
          val dd = d.asInstanceOf[DefDef]
          val name = d.symbol.fullName.toString
          e.ctx.functions(name) = e.function(name, dd.termParamss.flatMap(_.params), dd.rhs.get, dd)
          val model = ir.Model(
            functions = e.ctx.functions.values.toVector,
            types = e.ctx.types.values.toVector
          )
          d.symbol.addAnnotation(
            Annotation(
              requiredClass(Carrier.stored),
              List(Carrier.literals(Carrier.chunks(model))),
              d.span
            )
          )
          Some(model)
        catch
          case err: LiftError =>
            report.error(err.message, e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos)
            None
          case err: CompilerApiError =>
            report.error(err.getMessage, d.srcPos)
            None
        finally active.remove(d.symbol): Unit
    }

final private class StorePhase(store: DefinitionStore) extends PluginPhase:
  val phaseName = "reifyStore"
  override val runsAfter = Set("reifyFamily")
  override val runsBefore = Set("sbt-api", Pickler.name)

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    val marker = getClassIfDefined(Carrier.marker)
    if marker.exists && tree.symbol.hasAnnotation(marker) then store.save(tree): Unit
    tree

final private class CapturePhase(store: DefinitionStore) extends PluginPhase:
  val phaseName = "reifyCapture"
  override val runsAfter = Set("reifyStore")
  override val runsBefore = Set(Pickler.name)

  override def transformApply(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    val marker = getClassIfDefined(Carrier.capture)
    if !marker.exists then tree
    else
      def appliedClauses(fun: tpd.Tree): Int = fun match
        case tpd.Apply(previous, _)     => appliedClauses(previous) + 1
        case tpd.TypeApply(previous, _) => appliedClauses(previous)
        case _                          => 0
      val clauses = tree.fun.symbol.paramSymss.filter(_.headOption.exists(_.isTerm))
      val params = clauses.lift(appliedClauses(tree.fun)).getOrElse(Nil)
      val captures = params.zipWithIndex.flatMap { (p, i) =>
        p.getAnnotation(marker).map(a => (i, a.argumentConstantString(0).get))
      }
      captures match
        case Nil                                     => tree
        case List((i, target)) if i < tree.args.size =>
          val e = store.engine(ctx.compilationUnit.tpdTree)
          import e.ctx.quotes.reflect.Term
          try
            e.readFinites()
            val arg = tree.args(i).asInstanceOf[Term]
            val root = e.stepFunction(e.resolve(arg), tree.fun.symbol.fullName.toString, "capture")
            val model = ir.Model(
              functions = e.ctx.functions.values.toVector,
              types = e.ctx.types.values.toVector
            )
            def implementation(owner: Symbol): Symbol =
              val methods =
                owner.info.decl(termName(target)).alternatives.map(_.symbol).filter(_.is(Method))
              methods match
                case List(method) => method
                case _            =>
                  e.ctx.fail(
                    tree.asInstanceOf[e.ctx.quotes.reflect.Tree],
                    s"capture implementation ${owner.fullName}.$target is missing or not a unique method"
                  )
            def retarget(fun: tpd.Tree): tpd.Tree = fun match
              case tpd.Apply(previous, args)     => retarget(previous).appliedToArgs(args)
              case tpd.TypeApply(previous, args) => retarget(previous).appliedToTypeTrees(args)
              case tpd.Select(receiver, _)       =>
                receiver.select(implementation(fun.symbol.owner))
              case _ =>
                val owner = fun.symbol.owner
                val receiver = if owner.isClass then owner.sourceModule else owner
                tpd.ref(receiver).select(implementation(owner))
            val selected = retarget(tree.fun)
            selected
              .appliedToArgs(
                tree.args ++ List(
                  tpd.Literal(Constant(root)),
                  Carrier.literals(Carrier.chunks(model))
                )
              )
              .withSpan(tree.span)
          catch
            case err: LiftError =>
              report.error(err.message, e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos)
              tree
            case err: CompilerApiError =>
              report.error(err.getMessage, tree.args(i).srcPos)
              tree
        case _ =>
          report.error(
            "a capture applies one marked function parameter at its call site",
            tree.srcPos
          )
          tree

private object ChannelNames:
  val generated = new dotty.tools.dotc.util.Property.StickyKey[Unit]

final private class NamesPhase extends PluginPhase:
  val phaseName = "reifyNames"
  override val runsAfter = Set("reifyCapture")
  override val runsBefore = Set(Pickler.name)

  override def transformTemplate(tree: tpd.Template)(using ctx: Context): tpd.Tree =
    val machine = getClassIfDefined("framework.dsl.Machine")
    if !machine.exists || !ctx.owner.isClass then tree
    else
      val args = ctx.owner.asClass.typeRef.baseType(machine.asClass).argInfos
      args.lift(2) match
        case Some(fact) if fact.dealias.typeSymbol.is(Enum) =>
          val cases =
            fact.dealias.typeSymbol.children.filter(s => s.isClass && s.is(Enum) && s.is(Case))
          val registrations = cases.map: symbol =>
            val receiver = This(ctx.owner.asClass)
            val method = receiver.tpe.member(termName("declaredFact")).symbol
            receiver
              .select(method)
              .appliedToArgs(
                List(ref(symbol.companionModule), Literal(Constant(symbol.name.toString)))
              )
          cpy.Template(tree)(body = registrations ++ tree.body)
        case _ => tree

  private def entityName(tree: tpd.Select)(using ctx: Context): tpd.Tree =
    val entity = getClassIfDefined("framework.dsl.Entity")
    if entity.exists && ctx.owner.enclosingClass != entity &&
      tree.symbol == entity.asClass.typeRef.member(termName("name")).symbol
    then
      tree.qualifier
        .select(entity.asClass.typeRef.member(termName("sourceName")).symbol)
        .withSpan(tree.span)
    else tree

  override def transformSelect(tree: tpd.Select)(using Context): tpd.Tree = entityName(tree)

  override def transformValDef(tree: tpd.ValDef)(using Context): tpd.Tree =
    val action = getClassIfDefined("framework.dsl.Action")
    val channel = getClassIfDefined("framework.dsl.Channel")
    val entity = getClassIfDefined("framework.dsl.Entity")
    def alias(t: tpd.Tree): Boolean = t match
      case _: Ident | _: Select => !t.symbol.is(Method) || t.symbol.is(Accessor)
      case Apply(Select(receiver, operation), _) if Set("on", "creates")(operation.toString) =>
        alias(receiver)
      case Typed(value, _) => alias(value)
      case _               => false
    val isAction = action.exists && tree.tpt.tpe.widenDealias.derivesFrom(action.asClass)
    val isEntity = entity.exists && tree.tpt.tpe.widenDealias.derivesFrom(entity.asClass)
    val declared = isAction || isEntity ||
      (channel.exists && tree.tpt.tpe.widenDealias.derivesFrom(channel.asClass))
    val bound = isAction && (tree.rhs match
      case Apply(Select(receiver, operation), _) if Set("on", "creates")(operation.toString) =>
        alias(receiver)
      case _ => false)
    if declared &&
      !tree.symbol.is(Param) && !tree.symbol.is(Synthetic) && !tree.symbol.owner.is(
        Method
      ) && (!alias(tree.rhs) || bound)
    then
      val id = tree.symbol.fullName.toString
        .split('.')
        .filterNot(_.endsWith("$package$"))
        .map(_.stripSuffix("$"))
        .mkString(".")
      val method = tree.rhs.tpe.widenDealias
        .member(termName(if isAction then "sourceAt" else "declaredAt"))
        .symbol
      val selected = tree.rhs.select(method)
      if !isAction && !isEntity then selected.putAttachment(ChannelNames.generated, ()): Unit
      cpy.ValDef(tree)(rhs =
        selected
          .appliedTo(Literal(Constant(if isEntity then tree.name.toString else id)))
          .withSpan(tree.rhs.span)
      )
    else tree

  override def transformApply(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    val sourceNames = new tpd.TreeMap:
      override def transform(tree: tpd.Tree)(using Context): tpd.Tree = tree match
        case selected: tpd.Select => super.transform(entityName(selected))
        case _                    => super.transform(tree)
    val retained = tree.args.map:
      case arg: tpd.NamedArg => cpy.NamedArg(arg)(arg.name, sourceNames.transform(arg.arg))
      case arg               => arg
    val named = getClassIfDefined("framework.realize.Named")
    if !named.exists || (ctx.owner.is(Synthetic) && ctx.owner.is(
        Method
      ) && ctx.owner.name.toString != "$anonfun") ||
      tree.fun.symbol.owner.fullName.toString
        .stripSuffix("$") == "framework.realize.Conversion$package"
    then cpy.Apply(tree)(tree.fun, retained)
    else
      val params = tree.fun.tpe.widen match
        case mt: MethodType => mt.paramInfos
        case _              => Nil
      val args = retained.zipWithIndex.map { (arg, index) =>
        params.lift(index) match
          case Some(param) if !top(param.widenDealias.typeSymbol) => name(arg, named)
          case _                                                  => arg
      }
      cpy.Apply(tree)(tree.fun, args)

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    val named = getClassIfDefined("framework.realize.Named")
    if named.exists && tree.name.toString.contains("$default$") && !top(
        tree.tpt.tpe.widenDealias.typeSymbol
      )
    then cpy.DefDef(tree)(rhs = name(tree.rhs, named))
    else tree

  private def top(sym: Symbol)(using Context): Boolean =
    sym == defn.AnyClass || sym == defn.ObjectClass || sym == defn.MatchableClass || sym == defn.AnyValClass

  private def name(arg: tpd.Tree, named: Symbol)(using Context): tpd.Tree = arg match
    case NamedArg(n, value) => cpy.NamedArg(arg)(n, name(value, named))
    case Typed(value, tpt)  => cpy.Typed(arg)(name(value, named), tpt)
    case SeqLiteral(_, tpt) if top(tpt.tpe.widenDealias.typeSymbol) => arg
    case SeqLiteral(values, tpt) => cpy.SeqLiteral(arg)(values.map(name(_, named)), tpt)
    case _ if arg.tpe.widenDealias.derivesFrom(named.asClass) =>
      val sym = arg.symbol
      arg match
        case _ if sym.exists && sym.name.toString.contains("$default$")                     => arg
        case _: Ident | _: Select if sym.exists && (sym.is(Param) || sym.is(ParamAccessor)) => arg
        case _: Ident | _: Select
            if sym.exists && sym.isTerm && (!sym.is(Method) || sym.is(Accessor)) && !sym.is(
              Synthetic
            ) =>
          val location =
            s"${arg.srcPos.sourcePos.source.file.path}:${arg.srcPos.sourcePos.line + 1}"
          arg
            .select(termName("named"))
            .appliedToArgs(
              List(
                Literal(Constant(sym.name.toString)),
                Literal(Constant(sym.fullName.toString)),
                Literal(Constant(location))
              )
            )
            .withSpan(arg.span)
        case _: Apply if sym.name.toString.contains("$default$") => arg
        case _                                                   =>
          report.error(
            "a command is named after the val that declares it: declare it as a val and refer to it by value",
            arg.srcPos
          )
          arg
    case _ => arg
