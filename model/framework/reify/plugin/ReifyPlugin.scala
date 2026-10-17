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
import dotty.tools.dotc.core.Flags.{
  Accessor,
  Case,
  Enum,
  Given,
  Method,
  Module,
  Package,
  Param,
  ParamAccessor,
  Synthetic,
  Trait
}
import dotty.tools.dotc.core.Types.{MethodOrPoly, MethodType}
import dotty.tools.dotc.core.Names.termName
import dotty.tools.dotc.core.Symbols.*
import dotty.tools.dotc.plugins.{PluginPhase, StandardPlugin}
import dotty.tools.dotc.report
import dotty.tools.dotc.transform.Pickler
import io.temporal.server.api.umpire.v1 as ir

final class ReifyPlugin extends StandardPlugin:
  val name = "umpire-reify"
  override val description =
    "lifts Model functions at their definitions and captures"
  override def initialize(options: List[String])(using
      Context
  ): List[PluginPhase] =
    val (mode, _, _) = options.foldLeft(
      ("full", false, false)
    ):
      case ((current, modeSeen, cacheSeen), option) if option.startsWith("mode=") =>
        val value = option.stripPrefix("mode=")
        if modeSeen || !Set("full", "realization", "signature")(value) then
          report.error(s"invalid umpire-reify mode: $value")
          (current, true, cacheSeen)
        else (value, true, cacheSeen)
      case ((current, modeSeen, cacheSeen), option) if option.startsWith("cache-key=") =>
        val value = option.stripPrefix("cache-key=")
        if cacheSeen || !value.matches("[0-9a-f]{64}") then
          report.error(s"invalid umpire-reify cache key: $value")
        (current, modeSeen, true)
      case ((current, modeSeen, cacheSeen), option) =>
        report.error(s"unknown umpire-reify option: $option")
        (current, modeSeen, cacheSeen)
    val store = new DefinitionStore
    val effects = new EffectsPhase
    if mode == "realization" then
      List(
        effects,
        new StorePhase(store),
        new NamesPhase("reifyStore"),
        new SchemaPhase(store),
        new RealizationNamesPhase
      )
    else if mode == "signature" then
      List(
        effects,
        new StorePhase(store),
        new CapturePhase(store),
        new NamesPhase,
        new SchemaPhase(store),
        new RuntimeCapturePhase,
        new SchemaPhase(store, signatures = true)
      )
    else
      List(
        effects,
        new StorePhase(store),
        new EndpointsPhase(store),
        new RulesPhase(store),
        new CapturePhase(store),
        new NamesPhase,
        new SchemaPhase(store),
        new RuntimeCapturePhase,
        new EndpointsPhase(store, retention = true),
        new RulesPhase(store, retention = true),
        new SchemaPhase(store, signatures = true)
      )

// The carriers a definition's lifted IR crosses compiler runs in. Every Model and everything it
// lifts from compile in one compilation (model/README.md), so a carrier was written by this plugin
// before pickling in this build: it holds data only, with no version, checksum or position table.
private object Carrier:
  val marker = "framework.dsl.lifted"
  val stored = "framework.dsl.liftedIr"
  val capture = "framework.dsl.capture"
  val section = "framework.dsl.sectionIr"
  val facet = "framework.dsl.facetIr"
  val status = "framework.dsl.recordedStatusIr"
  val outcome = "framework.dsl.declaredOutcomeIr"
  val callGroups = "framework.dsl.ruleCallGroups"
  val namedResult = "framework.dsl.namedResult"

  // A stored model keeps a position on its root function only.
  def rooted(model: ir.Model, root: Option[String]): ir.Model =
    model.copy(
      functions =
        model.functions.map(f => if root.contains(f.name) then f else unpositioned(f, ir.Function)),
      types = model.types.map(_.clearPosition)
    )

  def rootedGroups(
      proofs: Seq[CallGroupEvidence],
      root: Option[String]
  ): Vector[CallGroupEvidence] = proofs.toVector.map(p =>
    p.copy(
      call = if p.owner == root then p.call else unpositioned(p.call, ir.Call),
      function =
        if root.contains(p.function.name) then p.function
        else unpositioned(p.function, ir.Function)
    )
  )

  def rootedSites(
      sites: Seq[ContextualPhaseSite],
      root: Option[String]
  ): Vector[ContextualPhaseSite] = sites.toVector.map(site =>
    if site.owner == root then site else site.copy(call = unpositioned(site.call, ir.Call))
  )

  def unpositioned[A <: scalapb.GeneratedMessage](
      value: A,
      companion: scalapb.GeneratedMessageCompanion[A]
  ): A =
    import scalapb.descriptors.{PEmpty, PMessage, PRepeated, PValue, ScalaType}
    def strip(value: PValue): PValue = value match
      case PMessage(fields) =>
        PMessage(fields.map { (field, child) =>
          field -> (field.scalaType match
            case ScalaType.Message(descriptor)
                if !field.isRepeated &&
                  descriptor.fullName == ir.Position.scalaDescriptor.fullName =>
              PEmpty
            case _ => strip(child))
        })
      case PRepeated(values) => PRepeated(values.map(strip))
      case _                 => value
    companion.messageReads.read(strip(value.toPMessage))

  def callGroupChunks(proof: CallGroupEvidence): List[String] = List(
    proof.owner.getOrElse(""),
    Base64.getEncoder.encodeToString(proof.call.toByteArray),
    Base64.getEncoder.encodeToString(proof.function.toByteArray),
    proof.groups.mkString(",")
  )

  private def readGroups(data: List[String]): Vector[CallGroupEvidence] =
    data
      .grouped(4)
      .collect { case List(owner, call, function, groups) =>
        CallGroupEvidence(
          Option(owner).filter(_.nonEmpty),
          ir.Call.parseFrom(Base64.getDecoder.decode(call)),
          ir.Function.parseFrom(Base64.getDecoder.decode(function)),
          groups.split(',').map(_.toInt).toVector
        )
      }
      .toVector

  def saveCallGroups(sym: Symbol, proofs: Seq[CallGroupEvidence], at: tpd.Tree)(using
      Context
  ): Unit =
    val retained = proofs.distinct
    if retained.nonEmpty then
      sym.addAnnotation(
        Annotation(
          requiredClass(callGroups),
          List(literals(retained.toList.flatMap(callGroupChunks))),
          at.span
        )
      )

  def readCallGroups(sym: Symbol)(using Context): Vector[CallGroupEvidence] =
    annotation(sym, callGroups).fold(Vector.empty)(readGroups)

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

  private def annotation(sym: Symbol, name: String)(using Context): Option[List[String]] =
    val cls = getClassIfDefined(name)
    if !cls.exists then None else sym.getAnnotation(cls).map(_.arguments.flatMap(strings))

  private def model(chunks: Seq[String]): ir.Model =
    ir.Model.parseFrom(Base64.getDecoder.decode(chunks.mkString))

  private def text(chunks: Seq[String]): String =
    new String(Base64.getDecoder.decode(chunks.mkString), java.nio.charset.StandardCharsets.UTF_8)

  private def refusal(position: ir.Position, message: String): List[String] =
    List("refused", position.file, position.line.toString) ++
      chunks(message.getBytes(java.nio.charset.StandardCharsets.UTF_8))

  def read(sym: Symbol)(using Context): Option[ir.Model] =
    annotation(sym, stored).map(model)

  def hasSection(sym: Symbol)(using Context): Boolean =
    val cls = getClassIfDefined(section)
    cls.exists && sym.hasAnnotation(cls)

  def sectionChunks(outcome: SectionOutcome): List[String] =
    List(outcome.root, outcome.form) ++ outcome.refusal.fold(
      "ok" :: chunks(outcome.model.get)
    )(refusal)

  def readSection(sym: Symbol)(using Context): Option[SectionOutcome] =
    annotation(sym, section).map { data =>
      val root :: form :: state :: body = data: @unchecked
      if state == "ok" then SectionOutcome(root, form, Some(model(body)), None)
      else
        val file :: line :: message = body: @unchecked
        SectionOutcome(root, form, None, Some(ir.Position(file, line.toInt) -> text(message)))
    }

  def siteChunks(site: ContextualPhaseSite): List[String] =
    def bytes(value: scalapb.GeneratedMessage): String =
      Base64.getEncoder.encodeToString(value.toByteArray)
    List(
      site.kind,
      site.owner.getOrElse(""),
      bytes(site.call),
      bytes(site.state),
      bytes(site.phase),
      bytes(site.result),
      site.placement.fold("")(bytes)
    )

  def facetChunks(outcome: FacetOutcome): List[String] =
    List(outcome.facet, outcome.namespace, outcome.root) ++ outcome.refusal.fold {
      val model = chunks(outcome.model.get)
      List("ok", model.size.toString) ++ model ++
        List(outcome.sites.size.toString) ++ outcome.sites.toList.flatMap(siteChunks)
    }(refusal)

  def readFacet(sym: Symbol)(using Context): Option[FacetOutcome] =
    annotation(sym, facet).map { all =>
      val owner :: namespace :: root :: state :: data = all: @unchecked
      if state != "ok" then
        val file :: line :: message = data: @unchecked
        FacetOutcome(
          owner,
          namespace,
          root,
          None,
          Vector.empty,
          Some(ir.Position(file, line.toInt) -> text(message))
        )
      else
        val count = data.head.toInt
        val sites = data
          .drop(2 + count)
          .grouped(7)
          .map { values =>
            def decode(value: String) = Base64.getDecoder.decode(value)
            ContextualPhaseSite(
              values(0),
              Option(values(1)).filter(_.nonEmpty),
              ir.Call.parseFrom(decode(values(2))),
              ir.TypeRef.parseFrom(decode(values(3))),
              ir.TypeRef.parseFrom(decode(values(4))),
              ir.TypeRef.parseFrom(decode(values(5))),
              Option(values(6)).filter(_.nonEmpty).map(v => ir.Position.parseFrom(decode(v)))
            )
          }
          .toVector
        FacetOutcome(owner, namespace, root, Some(model(data.slice(1, 1 + count))), sites, None)
    }

  def statusChunks(outcome: StatusOutcome): List[String] =
    List(outcome.root, outcome.phase, outcome.name) ++ outcome.refusal.fold {
      val model = chunks(outcome.model.get)
      List(
        "ok",
        Base64.getEncoder.encodeToString(outcome.value.get.toByteArray),
        model.size.toString
      ) ++ model ++ outcome.proofs.toList.flatMap(callGroupChunks)
    }(refusal)

  def readStatus(sym: Symbol)(using Context): Option[StatusOutcome] =
    readDeclared(sym, status)

  def readOutcome(sym: Symbol)(using Context): Option[StatusOutcome] =
    readDeclared(sym, outcome)

  private def readDeclared(sym: Symbol, name: String)(using
      Context
  ): Option[StatusOutcome] =
    annotation(sym, name).map { all =>
      val root :: phase :: named :: state :: data = all: @unchecked
      if state != "ok" then
        val file :: line :: message = data: @unchecked
        StatusOutcome(
          root,
          phase,
          named,
          None,
          None,
          Some(ir.Position(file, line.toInt) -> text(message))
        )
      else
        val value :: size :: rest = data: @unchecked
        val count = size.toInt
        StatusOutcome(
          root,
          phase,
          named,
          Some(ir.Expr.parseFrom(Base64.getDecoder.decode(value))),
          Some(model(rest.take(count))),
          None,
          readGroups(rest.drop(count))
        )
    }

  def literals(chunks: List[String])(using Context): tpd.Tree =
    SeqLiteral(chunks.map(s => Literal(Constant(s))), TypeTree(defn.StringType))

private object SectionDiagnostics:
  def error(error: SectionRefusal)(using Context): Unit =
    val position = error.position
    if position.file.isEmpty || position.line < 1 then report.error(error.getMessage)
    else
      val text = "\n" * (position.line - 1) + " "
      val source = dotty.tools.dotc.util.SourceFile.virtual(position.file, text)
      val span = dotty.tools.dotc.util.Spans.Span(text.length - 1, text.length)
      report.error(
        error.getMessage,
        dotty.tools.dotc.util.SourcePosition(source, span)
      )

private object FacetDeclaration:
  def namespace(sym: Symbol)(using Context): Boolean =
    val facet = getClassIfDefined("framework.dsl.Facet")
    sym.exists && sym.isClass && sym.is(
      Module
    ) && sym.owner.isClass && sym.owner.is(Trait) &&
    facet.exists && sym.owner.asClass.typeRef.derivesFrom(facet.asClass)

  def member(sym: Symbol)(using Context): Boolean =
    val section = sym.owner
    section.exists && section.isClass && section.is(Module) && namespace(
      section.owner
    )

final private class DefinitionStore:
  private val active = mutable.Set.empty[Symbol]
  private val sites = mutable.Map.empty[Symbol, (CompilationUnit, tpd.DefDef)]
  private val sectionSites =
    mutable.Map.empty[Symbol, (CompilationUnit, tpd.ValDef)]
  private val statusSites =
    mutable.Map.empty[Symbol, (CompilationUnit, tpd.ValDef)]
  private val outcomeSites =
    mutable.Map.empty[Symbol, (CompilationUnit, tpd.ValDef)]
  private val declared = mutable.Set.empty[Symbol]
  private var indexed = false
  private var compilerRun = Option.empty[dotty.tools.dotc.Run]

  private def enterRun()(using ctx: Context): Unit =
    if !compilerRun.exists(_ eq ctx.run) then
      active.clear()
      sites.clear()
      sectionSites.clear()
      statusSites.clear()
      outcomeSites.clear()
      declared.clear()
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
              case d: tpd.DefDef                            => sites(d.symbol) = unit -> d
              case v: tpd.ValDef if sectionForm(v).nonEmpty =>
                sectionSites(v.symbol) = unit -> v
              case _ => ()
            t match
              case v: tpd.ValDef if okDeclaration(v) =>
                outcomeSites(v.symbol) = unit -> v
              case _ => ()
            t match
              case v: tpd.ValDef if v.symbol.is(Enum) && v.symbol.is(Case) =>
                statusSites(v.symbol) = unit -> v
              case _ => ()
            t match
              case d: tpd.MemberDef => declared += d.symbol
              case _                => ()
            traverseChildren(t)
        .traverse(unit.tpdTree)

  def inRun(sym: Symbol)(using Context): Boolean =
    index()
    declared(sym)

  private def sectionForm(v: tpd.ValDef)(using Context): Option[String] =
    def form(t: tpd.Tree): Option[String] = t match
      case tpd.Typed(value, _)        => form(value)
      case tpd.Block(Nil, value)      => form(value)
      case tpd.Inlined(_, Nil, value) => form(value)
      case t
          if Set("is", "effect")(t.symbol.name.toString) &&
            t.symbol.owner.fullName.toString == "framework.dsl.Syntax$package$" =>
        Some(t.symbol.name.toString)
      case _ => None
    form(v.rhs).filter { _ =>
      val section = v.symbol.owner
      val owner = section.owner
      val machine = getClassIfDefined("framework.dsl.Machine")
      section.isClass && section.is(Module) && (
        FacetDeclaration.namespace(owner) ||
          (owner.isClass && owner.is(Module) && machine.exists &&
            owner.asClass.typeRef.derivesFrom(machine.asClass))
      )
    }

  def sectionPresent(sym: Symbol)(using Context): Boolean =
    index()
    Carrier.hasSection(sym) || sectionSites.contains(sym)

  def section(sym: Symbol)(using ctx: Context): Option[SectionOutcome] =
    index()
    Carrier
      .readSection(sym)
      .orElse(sectionSites.get(sym).flatMap { (unit, value) =>
        given Context =
          ctx.fresh.setCompilationUnit(unit).setOwner(value.symbol)
        saveSection(value)
      })

  def facet(sym: Symbol)(using ctx: Context): Option[FacetOutcome] =
    index()
    val marker = getClassIfDefined(Carrier.marker)
    if marker.exists && sym.hasAnnotation(marker) then None
    else
      Carrier.readFacet(sym).orElse {
        if !FacetDeclaration.member(sym) then None
        else
          sectionSites
            .get(sym)
            .flatMap { (unit, value) =>
              given Context =
                ctx.fresh.setCompilationUnit(unit).setOwner(value.symbol)
              saveSection(value): Unit
              Carrier.readFacet(sym)
            }
            .orElse {
              sites.get(sym).flatMap { (unit, definition) =>
                given Context =
                  ctx.fresh.setCompilationUnit(unit).setOwner(definition.symbol)
                saveFacet(definition)
              }
            }
      }

  def status(sym: Symbol)(using ctx: Context): Option[StatusOutcome] =
    index()
    Carrier
      .readStatus(sym)
      .orElse(statusSites.get(sym).flatMap { (unit, definition) =>
        given Context =
          ctx.fresh.setCompilationUnit(unit).setOwner(definition.symbol)
        saveStatus(definition)
      })

  private def okDeclaration(definition: tpd.ValDef)(using Context): Boolean =
    val ok = getClassIfDefined("framework.dsl.Ok")
    definition.symbol.is(Given) && ok.exists && definition.tpt.tpe.derivesFrom(
      ok.asClass
    )

  def outcome(sym: Symbol)(using ctx: Context): Option[StatusOutcome] =
    index()
    Carrier
      .readOutcome(sym)
      .orElse(outcomeSites.get(sym).flatMap { (unit, definition) =>
        given Context =
          ctx.fresh.setCompilationUnit(unit).setOwner(definition.symbol)
        saveOutcome(definition)
      })

  def saveOutcome(
      definition: tpd.ValDef
  )(using ctx: Context): Option[StatusOutcome] =
    if !okDeclaration(definition) then None
    else
      Carrier.readOutcome(definition.symbol).orElse {
        val symbol = definition.symbol
        val root = symbol.fullName.toString
        val at = definition.srcPos.sourcePos
        val position = ir.Position(at.source.path, at.line + 1)
        if !active.add(symbol) then
          Some(
            StatusOutcome(
              root,
              "framework.dsl.Ok",
              "outcome",
              None,
              None,
              Some(position -> s"$root recursively declares its Ok outcome")
            )
          )
        else
          val e = engine(ctx.compilationUnit.tpdTree)
          val result = try
            e.readFinites()
            val reference =
              e.ctx.quotes.reflect
                .Ref(symbol.asInstanceOf[e.ctx.quotes.reflect.Symbol])
            val value = e.outcomeOf(reference, "captured Ok")
            StatusOutcome(
              root,
              "framework.dsl.Ok",
              "outcome",
              Some(value),
              Some(
                Carrier.rooted(
                  ir.Model(
                    functions = e.ctx.reachableFunctions(Seq(value)),
                    types = e.ctx.types.values.toVector
                  ),
                  None
                )
              ),
              None,
              Carrier.rootedGroups(e.ctx.retainedGroups(Seq(value)), None)
            )
          catch
            case error: LiftError =>
              StatusOutcome(
                root,
                "framework.dsl.Ok",
                "outcome",
                None,
                None,
                Some(e.ctx.pos(e.ctx.refusedAt) -> error.message)
              )
            case error: SectionRefusal =>
              StatusOutcome(
                root,
                "framework.dsl.Ok",
                "outcome",
                None,
                None,
                Some(error.position -> error.getMessage)
              )
            case error: CompilerApiError =>
              StatusOutcome(
                root,
                "framework.dsl.Ok",
                "outcome",
                None,
                None,
                Some(position -> error.getMessage)
              )
          finally active.remove(symbol): Unit
          symbol.addAnnotation(
            Annotation(
              requiredClass(Carrier.outcome),
              List(
                Carrier.literals(Carrier.statusChunks(result))
              ),
              definition.span
            )
          )
          Some(result)
      }

  def saveStatus(definition: tpd.ValDef)(using
      ctx: Context
  ): Option[StatusOutcome] =
    val symbol = definition.symbol
    val recorded = getClassIfDefined("framework.dsl.Recorded")
    if !symbol.is(Enum) || !symbol.is(Case) || !recorded.exists then None
    else
      val e = engine(ctx.compilationUnit.tpdTree)
      import e.ctx.quotes.reflect.Ref
      val current = symbol.asInstanceOf[e.ctx.quotes.reflect.Symbol]
      val phase = e.enumOf(current)
      if !phase.typeRef.derivesFrom(
          recorded.asInstanceOf[e.ctx.quotes.reflect.Symbol]
        )
      then None
      else
        Carrier.readStatus(symbol).orElse {
          val root = symbol.fullName.toString
          val at = definition.srcPos.sourcePos
          val position = ir.Position(at.source.path, at.line + 1)
          if !active.add(symbol) then
            Some(
              StatusOutcome(
                root,
                phase.fullName,
                "",
                None,
                None,
                Some(position -> s"$root recursively declares its status")
              )
            )
          else
            val result = try
              e.readFinites()
              val declared =
                e.declaredStatus(
                  current,
                  definition.asInstanceOf[e.ctx.quotes.reflect.Tree]
                )
              val name = e.resolve(declared) match
                case r: Ref => r.symbol.name
                case other  => other.show
              val fact = phase.typeRef
                .baseType(recorded.asInstanceOf[e.ctx.quotes.reflect.Symbol])
                .typeArgs
                .headOption
              val value = e.lift(declared, fact)
              StatusOutcome(
                root,
                phase.fullName,
                name,
                Some(value),
                Some(
                  Carrier.rooted(
                    ir.Model(
                      functions = e.ctx.reachableFunctions(Seq(value)),
                      types = e.ctx.types.values.toVector
                    ),
                    None
                  )
                ),
                None,
                Carrier.rootedGroups(e.ctx.retainedGroups(Seq(value)), None)
              )
            catch
              case error: LiftError =>
                StatusOutcome(
                  root,
                  phase.fullName,
                  "",
                  None,
                  None,
                  Some(e.ctx.pos(e.ctx.refusedAt) -> error.message)
                )
              case error: SectionRefusal =>
                StatusOutcome(
                  root,
                  phase.fullName,
                  "",
                  None,
                  None,
                  Some(error.position -> error.getMessage)
                )
              case error: CompilerApiError =>
                StatusOutcome(
                  root,
                  phase.fullName,
                  "",
                  None,
                  None,
                  Some(position -> error.getMessage)
                )
            finally active.remove(symbol): Unit
            symbol.addAnnotation(
              Annotation(
                requiredClass(Carrier.status),
                List(Carrier.literals(Carrier.statusChunks(result))),
                definition.span
              )
            )
            Some(result)
        }

  def saveFacet(definition: tpd.DefDef)(using
      ctx: Context
  ): Option[FacetOutcome] =
    val symbol = definition.symbol
    val marker = getClassIfDefined(Carrier.marker)
    if !FacetDeclaration.member(symbol) || symbol.is(Synthetic) || symbol.is(
        Accessor
      ) ||
      symbol.name.toString == "<init>" || definition.rhs.isEmpty ||
      (marker.exists && symbol.hasAnnotation(marker))
    then None
    else
      Carrier.readFacet(symbol).orElse {
        val root = symbol.fullName.toString
        val namespace = symbol.owner.owner
        val at = definition.srcPos.sourcePos
        val position = ir.Position(at.source.path, at.line + 1)
        if !active.add(symbol) then
          Some(
            FacetOutcome(
              namespace.owner.fullName.toString,
              namespace.fullName.toString,
              root,
              None,
              Vector.empty,
              Some(
                position -> s"$root calls itself: a recursive function has no IR form"
              )
            )
          )
        else
          val e = engine(ctx.compilationUnit.tpdTree)
          import e.ctx.quotes.reflect.DefDef
          e.ctx.contextualFacet = Some(namespace.owner.asInstanceOf[e.ctx.quotes.reflect.Symbol])
          val result = try
            e.readFinites()
            val d = definition.asInstanceOf[DefDef]
            e.ctx.functions(root) = e.function(root, d.termParamss.flatMap(_.params), d.rhs.get, d)
            val model = ir.Model(
              functions = e.ctx.reachableFunctions(Nil, Seq(root)),
              types = e.ctx.types.values.toVector
            )
            FacetOutcome(
              namespace.owner.fullName.toString,
              namespace.fullName.toString,
              root,
              Some(Carrier.rooted(model, Some(root))),
              Carrier.rootedSites(
                e.ctx.retainedContextualSites(roots = Seq(root)),
                Some(root)
              ),
              None
            )
          catch
            case error: LiftError =>
              FacetOutcome(
                namespace.owner.fullName.toString,
                namespace.fullName.toString,
                root,
                None,
                Vector.empty,
                Some(e.ctx.pos(e.ctx.refusedAt) -> error.message)
              )
            case error: SectionRefusal =>
              FacetOutcome(
                namespace.owner.fullName.toString,
                namespace.fullName.toString,
                root,
                None,
                Vector.empty,
                Some(error.position -> error.getMessage)
              )
            case error: CompilerApiError =>
              FacetOutcome(
                namespace.owner.fullName.toString,
                namespace.fullName.toString,
                root,
                None,
                Vector.empty,
                Some(position -> error.getMessage)
              )
          finally active.remove(symbol): Unit
          symbol.addAnnotation(
            Annotation(
              requiredClass(Carrier.facet),
              List(Carrier.literals(Carrier.facetChunks(result))),
              definition.span
            )
          )
          if result.model.nonEmpty then
            Carrier.saveCallGroups(
              symbol,
              Carrier.rootedGroups(e.ctx.retainedGroups(Nil, Seq(root)), Some(root)),
              definition
            )
          Some(result)
      }

  def saveSection(value: tpd.ValDef)(using
      ctx: Context
  ): Option[SectionOutcome] =
    enterRun()
    sectionForm(value).map { form =>
      Carrier.readSection(value.symbol).getOrElse {
        val root = value.symbol.fullName.toString
        val at = value.srcPos.sourcePos
        val position = ir.Position(at.source.path, at.line + 1)
        if !active.add(value.symbol) then
          SectionOutcome(
            root,
            form,
            None,
            Some(
              position ->
                s"${value.name} calls itself, directly or through another function: a recursive function has no IR form"
            )
          )
        else
          val e = engine(ctx.compilationUnit.tpdTree)
          import e.ctx.quotes.reflect.Symbol
          val namespace = value.symbol.owner.owner
          val contextual = FacetDeclaration.namespace(namespace)
          if contextual then e.ctx.contextualFacet = Some(namespace.owner.asInstanceOf[Symbol])
          val result = try
            e.readFinites()
            val block = e.ctx.blockVal(value.symbol.asInstanceOf[Symbol]).get
            e.ctx.functions(root) = e.blockFunction(root, block)
            val model = ir.Model(
              functions = e.ctx.reachableFunctions(Nil, Seq(root)),
              types = e.ctx.types.values.toVector
            )
            SectionOutcome(
              root,
              form,
              Some(Carrier.rooted(model, Some(root))),
              None
            )
          catch
            case error: LiftError =>
              SectionOutcome(
                root,
                form,
                None,
                Some(e.ctx.pos(e.ctx.refusedAt) -> error.message)
              )
            case error: SectionRefusal =>
              SectionOutcome(
                root,
                form,
                None,
                Some(error.position -> error.getMessage)
              )
            case error: CompilerApiError =>
              SectionOutcome(
                root,
                form,
                None,
                Some(position -> error.getMessage)
              )
          finally active.remove(value.symbol): Unit
          value.symbol.addAnnotation(
            Annotation(
              requiredClass(Carrier.section),
              List(Carrier.literals(Carrier.sectionChunks(result))),
              value.span
            )
          )
          if result.model.nonEmpty then
            Carrier.saveCallGroups(
              value.symbol,
              Carrier.rootedGroups(e.ctx.retainedGroups(Nil, Seq(root)), Some(root)),
              value
            )
          if contextual then
            value.symbol.addAnnotation(
              Annotation(
                requiredClass(Carrier.facet),
                List(
                  Carrier.literals(
                    Carrier.facetChunks(
                      FacetOutcome(
                        namespace.owner.fullName.toString,
                        namespace.fullName.toString,
                        root,
                        result.model,
                        Carrier.rootedSites(
                          e.ctx.retainedContextualSites(roots = Seq(root)),
                          Some(root)
                        ),
                        result.refusal
                      )
                    )
                  )
                ),
                value.span
              )
            )
          result
      }
    }

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
    val outcomes = mutable.ListBuffer.empty[tpd.Tree]
    val ok = getClassIfDefined("framework.dsl.Ok")
    if ok.exists then
      for unit <- ctx.run.units do
        new tpd.TreeTraverser:
          def traverse(t: tpd.Tree)(using Context): Unit =
            t match
              case d: tpd.ValDef if d.symbol.is(Given) && d.tpt.tpe.derivesFrom(ok.asClass) =>
                outcomes += d
              case _ => ()
            traverseChildren(t)
        .traverse(unit.tpdTree)
    val pc = new PluginContext(using q)(
      root.asInstanceOf[q.reflect.Tree],
      outcomes.toList.map(_.asInstanceOf[q.reflect.Tree]),
      s => load(s.asInstanceOf[Symbol]),
      s => sectionPresent(s.asInstanceOf[Symbol]),
      s => section(s.asInstanceOf[Symbol]),
      s => facet(s.asInstanceOf[Symbol]),
      s => status(s.asInstanceOf[Symbol]),
      s => outcome(s.asInstanceOf[Symbol]),
      s => Carrier.readCallGroups(s.asInstanceOf[Symbol])
    )
    new PluginEngine(pc)

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
          val model = Carrier.rooted(
            ir.Model(
              functions = e.ctx.functions.values.toVector,
              types = e.ctx.types.values.toVector
            ),
            Some(name)
          )
          d.symbol.addAnnotation(
            Annotation(
              requiredClass(Carrier.stored),
              List(Carrier.literals(Carrier.chunks(model))),
              d.span
            )
          )
          Carrier.saveCallGroups(
            d.symbol,
            Carrier.rootedGroups(e.ctx.callGroups.toVector, Some(name)),
            d
          )
          Some(model)
        catch
          case err: SectionRefusal =>
            SectionDiagnostics.error(err)
            None
          case err: LiftError =>
            report.error(
              err.message,
              e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos
            )
            None
          case err: CompilerApiError =>
            report.error(err.getMessage, d.srcPos)
            None
        finally active.remove(d.symbol): Unit
    }

final private class StorePhase(store: DefinitionStore) extends PluginPhase:
  val phaseName = "reifyStore"
  override val runsAfter = Set("reifyEffects")
  override val runsBefore = Set("sbt-api", Pickler.name)

  override def transformValDef(tree: tpd.ValDef)(using Context): tpd.Tree =
    store.saveSection(tree): Unit
    store.saveStatus(tree): Unit
    store.saveOutcome(tree): Unit
    tree

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    val section = tree.symbol.owner
    val declaring = section.owner
    val machine = getClassIfDefined("framework.dsl.Machine")
    if section.isClass && section.is(Module) && section.name.toString
        .stripSuffix(
          "$"
        ) == "effects" &&
      (FacetDeclaration.namespace(declaring) ||
        (declaring.isClass && declaring.is(Module) && machine.exists &&
          declaring.asClass.typeRef.derivesFrom(machine.asClass)))
    then
      val e = store.engine(summon[Context].compilationUnit.tpdTree)
      import e.ctx.quotes.reflect.DefDef
      val definition = tree.asInstanceOf[DefDef]
      try if e.makesSteps(definition.returnTpt.tpe) then e.givesNoEmpty(definition)
      catch
        case err: LiftError =>
          report.error(
            err.message,
            e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos
          )
    val marker = getClassIfDefined(Carrier.marker)
    if marker.exists && tree.symbol.hasAnnotation(marker) then store.save(tree): Unit
    else store.saveFacet(tree): Unit
    tree

private object EndpointPayload:
  final case class Payload(
      owner: String,
      family: String,
      name: String,
      starts: List[String],
      end: List[String],
      closure: List[String]
  )
  val key = new dotty.tools.dotc.util.Property.StickyKey[Payload]

private object EvidencePayload:
  val key = new dotty.tools.dotc.util.Property.StickyKey[List[String]]

private object FacetContextPayload:
  val key = new dotty.tools.dotc.util.Property.StickyKey[List[String]]

private object FacetNamespacePayload:
  final case class Payload(owner: Symbol, namespace: String)
  val key = new dotty.tools.dotc.util.Property.StickyKey[Payload]

private object FacetOriginPayload:
  final case class Payload(owner: Symbol, namespace: Symbol, rules: String)
  val key = new dotty.tools.dotc.util.Property.StickyKey[Payload]

final private class EndpointsPhase(
    store: DefinitionStore,
    retention: Boolean = false
) extends PluginPhase:
  val phaseName =
    if retention then "reifyRuntimeEndpoints" else "reifyEndpoints"
  override val runsAfter =
    if retention then Set("reifyRuntimeCapture") else Set("reifyStore")
  override val runsBefore =
    if retention then Set("reifySignature")
    else Set("reifyCapture", Pickler.name)

  override def transformTemplate(
      tree: tpd.Template
  )(using ctx: Context): tpd.Tree =
    if retention then
      val evidence =
        tree.getAttachment(EvidencePayload.key).fold(tree) { data =>
          val receiver = This(ctx.owner.asClass).withSpan(tree.span)
          val registration = receiver
            .select(receiver.tpe.member(termName("retainEvidence")).symbol)
            .appliedToArgs(List(Carrier.literals(data)))
          cpy.Template(tree)(body = registration :: tree.body)
        }
      val endpoints =
        tree.getAttachment(EndpointPayload.key).fold(evidence) { payload =>
          val receiver = This(ctx.owner.asClass).withSpan(tree.span)
          val registration = receiver
            .select(receiver.tpe.member(termName("retainEndpoints")).symbol)
            .appliedToArgs(
              List(
                Literal(Constant(payload.owner)),
                Literal(Constant(payload.family)),
                Literal(Constant(payload.name)),
                Carrier.literals(payload.starts),
                Carrier.literals(payload.end),
                Carrier.literals(payload.closure)
              )
            )
          cpy.Template(evidence)(body = registration :: evidence.body)
        }
      val context =
        tree.getAttachment(FacetContextPayload.key).fold(endpoints) { data =>
          val receiver = This(ctx.owner.asClass).withSpan(tree.span)
          val registration = receiver
            .select(receiver.tpe.member(termName("retainFacetContext")).symbol)
            .appliedToArgs(List(Carrier.literals(data)))
          cpy.Template(endpoints)(body = registration :: endpoints.body)
        }
      val namespace =
        tree.getAttachment(FacetNamespacePayload.key).fold(context) { payload =>
          val receiver = This(payload.owner.asClass).withSpan(tree.span)
          val registration = receiver
            .select(
              receiver.tpe.member(termName("retainFacetNamespace")).symbol
            )
            .appliedToArgs(
              List(
                This(ctx.owner.asClass),
                Literal(Constant(payload.owner.fullName.toString)),
                Literal(Constant(payload.namespace)),
                Literal(Constant(payload.owner.asClass.typeRef))
              )
            )
          cpy.Template(context)(body = registration :: context.body)
        }
      tree.getAttachment(FacetOriginPayload.key).fold(namespace) { payload =>
        val receiver = This(ctx.owner.asClass).withSpan(tree.span)
        val registration = receiver
          .select(receiver.tpe.member(termName("retainFacetOrigin")).symbol)
          .appliedToArgs(
            List(
              This(payload.owner.asClass),
              This(payload.namespace.asClass),
              Literal(Constant(payload.owner.asClass.typeRef)),
              Literal(Constant(payload.owner.fullName.toString)),
              Literal(Constant(payload.namespace.fullName.toString)),
              Literal(Constant(payload.rules))
            )
          )
        cpy.Template(namespace)(body = registration :: namespace.body)
      }
    else
      if FacetDeclaration.namespace(ctx.owner) then
        tree.putAttachment(
          FacetNamespacePayload.key,
          FacetNamespacePayload.Payload(
            ctx.owner.owner,
            ctx.owner.fullName.toString
          )
        ): Unit
        return tree
      val rules = getClassIfDefined("framework.dsl.Rules")
      if ctx.owner.isClass && ctx.owner.is(Module) && rules.exists &&
        ctx.owner.asClass.typeRef.derivesFrom(rules.asClass) &&
        FacetDeclaration.namespace(ctx.owner.owner)
      then
        tree.putAttachment(
          FacetOriginPayload.key,
          FacetOriginPayload.Payload(
            ctx.owner.owner.owner,
            ctx.owner.owner,
            ctx.owner.fullName.toString
          )
        ): Unit
        return tree
      val machine = getClassIfDefined("framework.dsl.Machine")
      val declared = ctx.owner.isClass && ctx.owner.is(Module) && Iterator
        .iterate(ctx.owner)(_.owner)
        .takeWhile(s => s.exists && !s.is(Package))
        .forall(s => s.isClass && s.is(Module))
      if !machine.exists || !declared || !ctx.owner.asClass.typeRef
          .derivesFrom(machine.asClass)
      then tree
      else
        val e = store.engine(ctx.compilationUnit.tpdTree)
        import e.ctx.quotes.reflect.*
        val cls = ctx.owner.asInstanceOf[e.ctx.quotes.reflect.Symbol]
        try
          val c = e.ctx
            .sourceTree(cls)
            .collect { case c: ClassDef => c }
            .getOrElse(
              e.ctx.fail(
                tree.asInstanceOf[e.ctx.quotes.reflect.Tree],
                "Machine endpoints have no local typed declaration body"
              )
            )
          val members = c.body.collect {
            case d: Definition
                if !d.symbol.flags.is(Flags.Synthetic) &&
                  !d.symbol.flags.is(
                    Flags.Module
                  ) && !d.symbol.isClassConstructor =>
              d
          }
          def member(name: String): Option[Definition] =
            members.find(_.name == name)
          e.readFinites()
          val name = e.ctx.objectFormName(cls)
          if cls.typeRef.derivesFrom(e.ctx.facetClass) then
            val state = cls.typeRef.baseType(e.ctx.machineClass).typeArgs.head
            val stateRef = e.typeRef(state, c)
            val phase = cls.typeRef
              .baseType(Symbol.requiredClass("framework.dsl.Phased"))
              .typeArgs
              .lift(1)
              .map(e.typeRef(_, c))
            def captured(
                form: String,
                term: Option[Term],
                arity: Int
            ): Option[ir.Function] =
              term.flatMap(e.lambda).collect {
                case (params, body) if params.size == arity =>
                  val root = s"$$facet-context-v1.${cls.fullName}.$form"
                  val lifted = e.function(root, params, body, term.get)
                  val function = if form == "projection" then
                    c.body
                      .collectFirst {
                        case definition: Definition
                            if definition.name.stripSuffix("$") == "rules" =>
                          lifted.copy(position = Some(e.ctx.pos(definition)))
                      }
                      .getOrElse(lifted)
                  else lifted
                  e.ctx.functions(root) = function
                  function
              }
            val projection = captured("projection", e.phaseProjection(cls), 1)
            val writer = captured("writer", e.phaseWriter(cls, c), 2)
            val roots = (projection.toList ++ writer.toList).map(_.name)
            val model = Carrier.chunks(
              ir.Model(
                functions = e.ctx.reachableFunctions(Nil, roots),
                types = e.ctx.types.values.toVector
              )
            )
            val positions = Carrier.chunks(ir.Model())
            val proofs = e.ctx.retainedGroups(Nil, roots)
            def bytes(value: scalapb.GeneratedMessage): String =
              Base64.getEncoder.encodeToString(value.toByteArray)
            val data = List(
              "facet-context-v1",
              cls.fullName,
              e.ctx.familyOf(cls),
              name,
              projection.fold("")(_.name),
              writer.fold("")(_.name),
              bytes(stateRef),
              phase.fold("")(bytes),
              projection.flatMap(_ => phase).fold("")(bytes),
              writer.fold("")(_ => bytes(stateRef)),
              model.size.toString
            ) ++ model ++
              List(positions.size.toString) ++ positions ++ List(
                proofs.size.toString
              ) ++
              proofs.toList.flatMap(p =>
                List(
                  bytes(p.call),
                  bytes(p.function),
                  p.groups.mkString(","),
                  p.owner.getOrElse("")
                )
              )
            tree.putAttachment(FacetContextPayload.key, data): Unit
          if !cls.typeRef.derivesFrom(e.ctx.derivedClass) then
            val fact = cls.typeRef.baseType(e.ctx.machineClass).typeArgs(2)
            val factRef = e.typeRef(fact, c)
            val authored = member("evidence").map {
              case v: ValDef if v.rhs.nonEmpty =>
                e.lambda(v.rhs.get) match
                  case Some((params, body)) =>
                    val fn =
                      e.function(s"$name.evidence", params, body, v.rhs.get)
                    e.ctx.functions(fn.name) = fn
                    fn
                  case None =>
                    e.ctx.fail(v, "evidence is a function of the fact")
              case other =>
                e.ctx.fail(
                  other,
                  "evidence is `val evidence: PartialFunction[F, String] = ...`"
                )
            }
            val roots = authored.map(_.name).toSeq
            val model = ir.Model(
              functions = e.ctx.reachableFunctions(Nil, roots),
              types = e.ctx.types.values.toVector
            )
            val proofs = e.ctx.retainedGroups(Nil, roots)
            val sources = ctx.run.units
              .map(unit => unit.source.path -> unit.source.file.name)
              .distinct
              .sortBy(_._1)
            val schema = new SchemaEngine(
              store.engine(ctx.compilationUnit.tpdTree).ctx
            )
            schema.readFinites()
            val schemaFact =
              fact.asInstanceOf[schema.ctx.quotes.reflect.TypeRepr]
            val schemaAt = c.asInstanceOf[schema.ctx.quotes.reflect.Tree]
            val ref = schema.root(schemaFact, schemaAt)
            val key = schema.identity(schemaFact, ref, schemaAt)
            def bytes(value: scalapb.GeneratedMessage): String =
              Base64.getEncoder.encodeToString(value.toByteArray)
            val factTypes = Carrier.chunks(
              ir.Model(types = schema.ctx.types.values.map(_.clearPosition).toVector)
            )
            val body = Carrier.chunks(model)
            val at = member("evidence").getOrElse(c)
            val data = List(
              "machine-evidence-v2",
              cls.fullName,
              e.ctx.familyOf(cls),
              name,
              key,
              bytes(ref),
              factTypes.size.toString
            ) ++ factTypes ++
              List(
                e.typeName(fact),
                fact.typeSymbol.name,
                fact.show,
                bytes(e.ctx.pos(at)),
                authored.fold("")(fn => bytes(model.functions.find(_.name == fn.name).get)),
                body.size.toString
              ) ++ body ++
              List(proofs.size.toString) ++ proofs.toList.flatMap(p =>
                List(
                  bytes(p.call),
                  bytes(p.function),
                  p.groups.mkString(","),
                  p.owner.getOrElse("")
                )
              ) ++ List(sources.size.toString) ++
              sources.flatMap((file, raw) => List(file, raw))
            require(
              factRef.equals(ref),
              "evidence Fact capture and selected schema disagree"
            )
            tree.putAttachment(EvidencePayload.key, data): Unit
          val start = member("init").map {
            case v: ValDef =>
              e.lift(v.rhs.getOrElse(e.ctx.fail(v, "init has no body")))
            case d: DefDef if d.termParamss.isEmpty =>
              e.lift(d.rhs.getOrElse(e.ctx.fail(d, "init has no body")))
            case other =>
              e.ctx.fail(other, "init is a val or a parameterless def")
          }
          val end = member("end")
            .map {
              case d: DefDef => e.endOf(d)
              case other     =>
                e.ctx.fail(other, "end reads one state: `def end(s: S) = ...`")
            }
            .orElse {
              if start.isEmpty then None
              else
                e.phaseProjection(cls).map { projection =>
                  val state =
                    cls.typeRef.baseType(e.ctx.machineClass).typeArgs.head
                  e.defaultEnd(projection, state, name, c)
                }
            }
          for initial <- start; ends <- end do
            tree.putAttachment(
              EndpointPayload.key,
              EndpointPayload.Payload(
                ctx.owner.fullName.toString,
                e.ctx.familyOf(cls),
                name,
                Carrier.chunks(ir.ListOf(Seq(initial)).toByteArray),
                Carrier.chunks(ends.toByteArray),
                Carrier.chunks(
                  ir.Model(
                    functions = e.ctx.functions.values.toVector,
                    types = e.ctx.types.values.toVector
                  )
                )
              )
            ): Unit
          tree
        catch
          case err: SectionRefusal =>
            SectionDiagnostics.error(err)
            tree
          case err: LiftError =>
            dotty.tools.dotc.report
              .error(err.message, e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos)
            tree
          case err: CompilerApiError =>
            dotty.tools.dotc.report.error(err.getMessage, tree.srcPos)
            tree

private object RulesPayload:
  final case class Payload(
      target: Symbol,
      data: List[String],
      disabled: Boolean
  )
  val key = new dotty.tools.dotc.util.Property.StickyKey[Payload]

final private class RulesPhase(
    store: DefinitionStore,
    retention: Boolean = false
) extends PluginPhase:
  val phaseName = if retention then "reifyRuntimeRules" else "reifyRules"
  override val runsAfter =
    if retention then Set("reifyRuntimeEndpoints") else Set("reifyEndpoints")
  override val runsBefore =
    if retention then Set("reifySignature", "inlining")
    else Set("reifyCapture", Pickler.name)

  override def transformApply(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    if retention then
      tree.getAttachment(RulesPayload.key).fold(tree) { payload =>
        def retarget(fun: tpd.Tree): tpd.Tree = fun match
          case tpd.TypeApply(fn, args)          => retarget(fn).appliedToTypeTrees(args)
          case tpd.Select(receiver, _)          => receiver.select(payload.target)
          case _: tpd.Ident if payload.disabled =>
            This(ctx.owner.enclosingClass.asClass).select(payload.target)
          case _ =>
            throw CompilerApiError(
              "a retained ordinary Rule has no selected receiver"
            )
        val applied =
          retarget(tree.fun).appliedToArgs(if payload.disabled then tree.args
          else tree.args :+ Carrier.literals(payload.data))
        (if payload.disabled then applied.appliedToArgs(List(Carrier.literals(payload.data)))
         else applied)
          .withSpan(tree.span)
      }
    else
      val caseClass = getClassIfDefined("framework.dsl.Case")
      val rulesClass = getClassIfDefined("framework.dsl.Rules")
      val machineClass = getClassIfDefined("framework.dsl.Machine")
      val rule = caseClass.exists && tree.fun.symbol.owner == caseClass &&
        tree.fun.symbol.name.toString == "~>" && !tree.fun
          .isInstanceOf[tpd.Apply]
      val disabled = rulesClass.exists && tree.fun.symbol.owner == rulesClass &&
        tree.fun.symbol.name.toString == "disabled"
      if !rule && !disabled then tree
      else
        val section = Iterator
          .iterate(ctx.owner)(_.owner)
          .takeWhile(_.exists)
          .find(s =>
            s.isClass && s.is(Module) && s.asClass.typeRef
              .derivesFrom(rulesClass.asClass)
          )
        val facet = section.map(_.owner).filter(FacetDeclaration.namespace)
        val derivedClass = getClassIfDefined("framework.dsl.Derived")
        val derived = if section.nonEmpty || disabled then None
        else
          Iterator
            .iterate(ctx.owner)(_.owner)
            .takeWhile(_.exists)
            .find(s =>
              s.isClass && s
                .is(Module) && derivedClass.exists && s.asClass.typeRef
                .derivesFrom(derivedClass.asClass)
            )
        val owner = section
          .map(_.owner)
          .filter(s =>
            s.isClass && s.is(Module) &&
              machineClass.exists && s.asClass.typeRef
                .derivesFrom(machineClass.asClass)
          )
          .orElse(facet.map(_.owner))
          .orElse(derived)
        if owner.isEmpty then tree
        else
          val e = store.engine(ctx.compilationUnit.tpdTree)
          import e.ctx.quotes.reflect.*
          def method(
              symbol: dotty.tools.dotc.core.Symbols.Symbol,
              name: String
          ) =
            symbol.info
              .decl(termName(name))
              .alternatives
              .map(_.symbol)
              .filter(_.is(Method)) match
              case List(method) => method
              case _            =>
                e.ctx.fail(
                  tree.asInstanceOf[Tree],
                  s"missing retained Rule implementation $name"
                )
          def bytes(value: scalapb.GeneratedMessage): String =
            Base64.getEncoder.encodeToString(value.toByteArray)
          try
            e.readFinites()
            val cls = owner.get.asInstanceOf[Symbol]
            val contextual = facet.nonEmpty || derived.nonEmpty
            if contextual then e.ctx.contextualFacet = Some(cls)
            val family = e.ctx.familyOf(cls)
            val machine = e.ctx.objectFormName(cls)
            val provenance = List(
              owner.get.fullName.toString,
              if facet.nonEmpty then "" else family,
              if facet.nonEmpty then "" else machine
            )
            def payload(
                ordinary: List[String],
                fragments: Seq[ir.Expr] = Nil
            ): List[String] =
              if derived.nonEmpty then
                val sites = e.ctx.retainedContextualSites(fragments)
                List("derived-rule-v1", ordinary.size.toString) ++ ordinary ++
                  List(sites.size.toString) ++ sites.toList.flatMap(
                    Carrier.siteChunks
                  )
              else
                facet.fold(ordinary) { namespace =>
                  val sites = e.ctx.retainedContextualSites(fragments)
                  List(
                    if disabled then "facet-disabled-v1" else "facet-rule-v1",
                    owner.get.fullName.toString,
                    namespace.fullName.toString,
                    section.get.fullName.toString,
                    ordinary.size.toString
                  ) ++ ordinary ++
                    List(sites.size.toString) ++ sites.toList.flatMap(
                      Carrier.siteChunks
                    )
                }
            val typePositions = Carrier.chunks(ir.Model())
            if disabled then
              val targets =
                tree.args.flatMap(a => e.varargs(a.asInstanceOf[Term]))
              val entries = targets.flatMap { target =>
                e.qualifiedRuleTarget(target, machine, None, "disabled")
                val name = e
                  .ruleActionReference(target)
                  .map(_.symbol.name)
                  .getOrElse(
                    e.ctx.fail(target, "disabled names a declared Action")
                  )
                List(name, bytes(e.ctx.pos(target)))
              }
              tree.putAttachment(
                RulesPayload.key,
                RulesPayload.Payload(
                  method(rulesClass, "capturedDisabledRules"),
                  payload(
                    "ordinary-disabled-v1" :: (provenance ++ List(
                      typePositions.size.toString
                    ) ++ typePositions ++ entries)
                  ),
                  true
                )
              ): Unit
            else
              def receiver(t: tpd.Tree): tpd.Tree = t match
                case tpd.TypeApply(fn, _) => receiver(fn)
                case tpd.Select(value, _) => value
                case _                    =>
                  e.ctx.fail(
                    tree.asInstanceOf[Tree],
                    "a Rule has no Case receiver"
                  )
              val c = receiver(tree.fun).asInstanceOf[Term]
              val effect = tree.args.head.asInstanceOf[Term]
              val rules = e.ctx
                .sourceTree(section.orElse(derived).get.asInstanceOf[Symbol])
                .collect { case rules: ClassDef => rules }
                .getOrElse(
                  e.ctx.fail(c, "ordinary Rules have no local declaration body")
                )
              val taken = mutable.Set.from(e.ctx.defs.keys.map(_.name))
              object bindings extends TreeTraverser:
                override def traverseTree(t: Tree)(
                    owner: e.ctx.quotes.reflect.Symbol
                ): Unit =
                  t match
                    case Bind(name, _) => taken += name: Unit
                    case _             => ()
                  super.traverseTree(t)(owner)
              bindings.traverseTree(
                ctx.compilationUnit.tpdTree.asInstanceOf[Tree]
              )(
                e.ctx.quotes.reflect.Symbol.spliceOwner
              )
              def unique(base: String): String =
                (base #:: LazyList.from(2).map(i => s"$base$i"))
                  .find(n => !taken(n))
                  .get
              val stateName = unique("$ruleState")
              val inputNames = e
                .lambda(effect)
                .toList
                .flatMap(_._1.drop(1))
                .indices
                .map(i => unique(s"$$ruleInput$i"))
                .toList
              val state = c.tpe.widen
                .baseType(caseClass.asInstanceOf[Symbol])
                .typeArgs
                .head
              val params = ir.Param(stateName, Some(e.typeRef(state, c))) +:
                e.lambda(effect)
                  .toList
                  .flatMap(_._1.drop(1))
                  .zip(inputNames)
                  .map((p, name) => ir.Param(name, Some(e.typeRef(p.tpt.tpe, p))))
              val heading =
                e.projectRuleHeading(
                  e.ruleHeading(c, machine),
                  e.phaseProjection(cls),
                  c,
                  machine
                )
              if derived.nonEmpty then
                heading match
                  case e.Heading.When(_) | e.Heading.Always => ()
                  case _                                    =>
                    e.ctx.fail(
                      c,
                      "a derivation's case is where(g) or always; it names no phase"
                    )
              val capturedGuard = e.ruleGuard(
                heading,
                stateName,
                tree.asInstanceOf[Tree],
                _ => Some(e.ctx.pos(rules))
              )
              def guardPlacements(
                  h: e.Heading,
                  path: String = ""
              ): List[String] = h match
                case e.Heading.Always        => List(path)
                case e.Heading.In(_, _)      => List(path, path + "r")
                case e.Heading.And(inner, _) =>
                  path :: guardPlacements(inner, path + "l")
                case _ => Nil
              // A native record action's effect reads its one record input, whose fields are
              // the action's inputs: `in.field` lowers to that input.
              val recordClass = Symbol.requiredClass("framework.dsl.RecordAction")
              val recordAction = e
                .ruleCall(e.ruleBlock(rules, tree.asInstanceOf[Tree]))
                .toList
                .flatMap(_._2.flatten)
                .flatMap(e.ruleActionReference)
                .headOption
                .exists(_.tpe.widen.derivesFrom(recordClass))
              val effectInputs = e.lambda(effect).toList.flatMap(_._1.drop(1)) match
                case List(in) if recordAction =>
                  in.tpt.tpe.widen.dealias.typeSymbol.caseFields.map(_.name)
                case _ => inputNames
              val capturedResult = e.ruleEffect(
                effect,
                stateName,
                effectInputs,
                machine,
                "capture",
                tree.asInstanceOf[Tree]
              )
              val fragments =
                Carrier.chunks(ir.ListOf(Seq(capturedGuard, capturedResult)).toByteArray)
              val curried =
                e.ctx.retainedGroups(Seq(capturedGuard, capturedResult)).map { proof =>
                  List(
                    bytes(proof.call),
                    bytes(proof.function),
                    proof.groups.mkString(","),
                    proof.owner.getOrElse("")
                  )
                }
              val model = ir.Model(
                functions = e.ctx.reachableFunctions(Seq(capturedGuard, capturedResult)),
                types = e.ctx.types.values.toVector
              )
              val closure = Carrier.chunks(model)
              val ordinary = List(
                "ordinary-rule-v3",
                e.forwardedDef(effect).fold(effect.show)(_.fullName)
              ) ++ provenance ++
                List(
                  bytes(e.ruleSite(rules, tree.asInstanceOf[Tree])),
                  bytes(e.ctx.pos(tree.asInstanceOf[Tree])),
                  params.size.toString
                ) ++
                params.map(bytes) ++ List(
                  fragments.size.toString
                ) ++ fragments ++
                List(closure.size.toString) ++ closure ++ List(
                  typePositions.size.toString
                ) ++
                typePositions ++ curried.toList.flatten
              val placements = guardPlacements(heading)
              val data = List(
                "rule-guard-placements-v1",
                ordinary.size.toString
              ) ++ ordinary ++
                List(placements.size.toString) ++ placements
              tree.putAttachment(
                RulesPayload.key,
                RulesPayload.Payload(
                  method(caseClass, s"capturedRule${params.size - 1}"),
                  payload(data, Seq(capturedGuard, capturedResult)),
                  false
                )
              ): Unit
            tree
          catch
            case error: SectionRefusal => SectionDiagnostics.error(error); tree
            case error: LiftError      =>
              dotty.tools.dotc.report
                .error(
                  error.message,
                  e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos
                )
              tree
            case error: CompilerApiError =>
              dotty.tools.dotc.report.error(error.getMessage, tree.srcPos); tree

final private class CapturePhase(store: DefinitionStore) extends PluginPhase:
  val phaseName = "reifyCapture"
  override val runsAfter = Set("reifyStore")
  override val runsBefore = Set(Pickler.name)

  private def operation(tree: tpd.Apply)(using ctx: Context): Boolean =
    val machine = getClassIfDefined("framework.dsl.Machine")
    val derived = getClassIfDefined("framework.dsl.Derived")
    val kind = tree.fun.symbol.name.toString
    val owner =
      if machine.exists && derived.exists && tree.fun.symbol.owner == machine &&
        Set("rebind", "extend")(kind) && !tree.fun.isInstanceOf[tpd.Apply]
      then
        Iterator
          .iterate(ctx.owner)(_.owner)
          .takeWhile(_.exists)
          .find(s =>
            s.isClass && s.is(Module) && s.asClass.typeRef
              .derivesFrom(derived.asClass)
          )
      else None
    owner match
      case None              => false
      case Some(declaration) =>
        val e = store.engine(ctx.compilationUnit.tpdTree)
        import e.ctx.quotes.reflect.Term
        try
          val target = machine.info
            .decl(termName(if kind == "rebind" then "capturedRebind"
            else "capturedExtend"))
            .alternatives
            .map(_.symbol)
            .filter(_.is(Method)) match
            case List(method) => method
            case _            =>
              throw CompilerApiError(
                "missing retained derivation operation implementation"
              )
          val symbol = declaration.asInstanceOf[e.ctx.quotes.reflect.Symbol]
          val count =
            tree.args.flatMap(a => e.varargs(a.asInstanceOf[Term])).size
          val position = Base64.getEncoder.encodeToString(
            e.ctx.pos(tree.asInstanceOf[e.ctx.quotes.reflect.Tree]).toByteArray
          )
          tree.putAttachment(
            RuntimeCapture.operationPayload,
            RuntimeCapture.OperationPayload(
              target,
              List(
                "derivation-operation-v1",
                declaration.fullName.toString,
                e.ctx.familyOf(symbol),
                e.ctx.objectFormName(symbol),
                kind,
                position,
                count.toString
              )
            )
          ): Unit
        catch
          case error: LiftError        => report.error(error.message, tree.srcPos)
          case error: CompilerApiError =>
            report.error(error.getMessage, tree.srcPos)
        true

  override def transformApply(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    val marker = getClassIfDefined(Carrier.capture)
    if operation(tree) || !marker.exists then tree
    else
      def appliedClauses(fun: tpd.Tree): Int = fun match
        case tpd.Apply(previous, _)     => appliedClauses(previous) + 1
        case tpd.TypeApply(previous, _) => appliedClauses(previous)
        case _                          => 0
      val clauses =
        tree.fun.symbol.paramSymss.filter(_.headOption.exists(_.isTerm))
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
            val binding = getClassIfDefined("framework.dsl.StepBinding")
            val machineClass = getClassIfDefined("framework.dsl.Machine")
            // A binding whose call has a later clause, the state-only `~>`'s evidence, is a
            // binding once that clause is applied.
            val bindingResult =
              binding.exists && tree.tpe.widen.finalResultType.derivesFrom(binding.asClass)
            val bindingOwner =
              if bindingResult
              then
                Iterator
                  .iterate(ctx.owner)(_.owner)
                  .takeWhile(_.exists)
                  .find(s =>
                    s.isClass && machineClass.exists && s.asClass.typeRef
                      .derivesFrom(machineClass.asClass)
                  )
              else None
            val derivedClass = getClassIfDefined("framework.dsl.Derived")
            val contextual = bindingOwner.exists(s =>
              derivedClass.exists && s.asClass.typeRef.derivesFrom(
                derivedClass.asClass
              )
            )
            if contextual then
              e.ctx.contextualFacet = bindingOwner.map(_.asInstanceOf[e.ctx.quotes.reflect.Symbol])
            // An expectation block is captured as the predicate the exporter lifts it to.
            val (root, named) = e
              .capturedExpectation(
                tree.asInstanceOf[Term],
                ctx.owner.asInstanceOf[e.ctx.quotes.reflect.Symbol]
              )
              .fold(
                e.stepFunctionOrigin(
                  e.resolve(arg),
                  tree.fun.symbol.fullName.toString,
                  "capture"
                )
              )(_ -> false)
            val origin =
              if bindingResult
              then
                val owner = bindingOwner
                Some(
                  (
                    owner.fold("")(_.fullName.toString),
                    owner.fold("") { s =>
                      val name = s.name.toString.stripSuffix("$")
                      name.take(1).toLowerCase + name.drop(1)
                    },
                    named,
                    Base64.getEncoder.encodeToString(
                      e.ctx
                        .pos(tree.asInstanceOf[e.ctx.quotes.reflect.Tree])
                        .toByteArray
                    )
                  )
                )
              else None
            val model = ir.Model(
              functions = e.ctx.functions.values.toVector,
              types = e.ctx.types.values.toVector
            )
            val ruleData = bindingOwner.map { lexicalOwner =>
              val r = store.engine(ctx.compilationUnit.tpdTree)
              import r.ctx.quotes.reflect.*
              try
                r.readFinites()
                val owner = lexicalOwner.asInstanceOf[Symbol]
                if contextual then r.ctx.contextualFacet = Some(owner)
                val effect =
                  tree.args(i).asInstanceOf[r.ctx.quotes.reflect.Term]
                val function = model.functions.find(_.name == root).get
                val names = function.params.indices
                  .map(n => if n == 0 then "$ruleState" else s"$$ruleInput${n - 1}")
                  .toVector
                val params =
                  function.params.zip(names).map((p, n) => p.copy(name = n))
                val result = r.ruleEffect(
                  effect,
                  names.head,
                  names.tail.toList,
                  r.ctx.objectFormName(owner),
                  "capture",
                  tree.asInstanceOf[Tree]
                )
                val guard = ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Bool(true))))
                val captured = ir.Model(
                  functions = r.ctx.reachableFunctions(Seq(result)),
                  types = r.ctx.types.values.toVector
                )
                val fragments = Carrier.chunks(ir.ListOf(Seq(guard, result)).toByteArray)
                val closure = Carrier.chunks(captured)
                val positions = Carrier.chunks(ir.Model())
                def bytes(value: scalapb.GeneratedMessage): String =
                  Base64.getEncoder.encodeToString(value.toByteArray)
                val proofs = r.ctx.retainedGroups(Seq(result))
                val groups =
                  proofs.toList.flatMap(p =>
                    List(
                      bytes(p.call),
                      bytes(p.function),
                      p.groups.mkString(","),
                      p.owner.getOrElse("")
                    )
                  )
                val position = bytes(r.ctx.pos(tree.asInstanceOf[Tree]))
                val ordinary = List(
                  "ordinary-rule-v3",
                  r.forwardedDef(effect).fold(effect.show)(_.fullName),
                  lexicalOwner.fullName.toString,
                  r.ctx.familyOf(owner),
                  r.ctx.objectFormName(owner),
                  position,
                  position,
                  params.size.toString
                ) ++ params.map(bytes) ++ List(
                  fragments.size.toString
                ) ++ fragments ++ List(
                  closure.size.toString
                ) ++ closure ++ List(
                  positions.size.toString
                ) ++ positions ++ groups
                if contextual then
                  val sites = r.ctx.retainedContextualSites(Seq(result))
                  List(
                    "derived-rule-v1",
                    ordinary.size.toString
                  ) ++ ordinary ++ List(
                    sites.size.toString
                  ) ++ sites.toList.flatMap(Carrier.siteChunks)
                else ordinary
              catch
                case error: LiftError =>
                  List("refused-rule-effect-v1", error.message)
                case error: SectionRefusal =>
                  List("refused-rule-effect-v1", error.getMessage)
            }
            def implementation(owner: Symbol): Symbol =
              val methods =
                owner.info
                  .decl(termName(target))
                  .alternatives
                  .map(_.symbol)
                  .filter(_.is(Method))
              methods match
                case List(method) => method
                case _            =>
                  e.ctx.fail(
                    tree.asInstanceOf[e.ctx.quotes.reflect.Tree],
                    s"capture implementation ${owner.fullName}.$target is missing or not a unique method"
                  )
            tree.putAttachment(
              RuntimeCapture.payload,
              RuntimeCapture.Payload(
                implementation(tree.fun.symbol.owner),
                root,
                Carrier.chunks(model),
                origin,
                ruleData
              )
            ): Unit
            tree
          catch
            case err: SectionRefusal =>
              SectionDiagnostics.error(err)
              tree
            case err: LiftError =>
              report.error(
                err.message,
                e.ctx.refusedAt.asInstanceOf[tpd.Tree].srcPos
              )
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

private object RuntimeCapture:
  final case class OperationPayload(target: Symbol, data: List[String])
  val operationPayload =
    new dotty.tools.dotc.util.Property.StickyKey[OperationPayload]
  final case class Payload(
      target: Symbol,
      root: String,
      chunks: List[String],
      origin: Option[(String, String, Boolean, String)],
      ruleData: Option[List[String]]
  )
  val payload = new dotty.tools.dotc.util.Property.StickyKey[Payload]
  // A captured call that still takes a later clause: its origin and derivation evidence are
  // retained on the call that applies that clause.
  val pending = new dotty.tools.dotc.util.Property.StickyKey[Payload]

final private class RuntimeCapturePhase extends PluginPhase:
  val phaseName = "reifyRuntimeCapture"
  override val runsAfter = Set(Pickler.name)
  override val runsBefore = Set("reifySignature", "inlining")

  override def transformApply(tree: tpd.Apply)(using Context): tpd.Tree =
    val operation = tree.getAttachment(RuntimeCapture.operationPayload)
    if operation.nonEmpty then
      val payload = operation.get
      def retarget(fun: tpd.Tree): tpd.Tree = fun match
        case tpd.TypeApply(previous, args) =>
          retarget(previous).appliedToTypeTrees(args)
        case tpd.Select(receiver, _) => receiver.select(payload.target)
        case _                       =>
          throw CompilerApiError(
            "a retained derivation operation has no receiver"
          )
      return retarget(tree.fun)
        .appliedToArgs(tree.args)
        .appliedToArgs(List(Carrier.literals(payload.data)))
        .withSpan(tree.span)
    def retained(located: tpd.Tree, payload: RuntimeCapture.Payload): tpd.Tree =
      val origined = payload.origin.fold(located) { (owner, machine, named, position) =>
        located
          .select(
            located.tpe.widen.member(termName("retainOrigin")).symbol
          )
          .appliedToArgs(
            List(
              Literal(Constant(owner)),
              Literal(Constant(machine)),
              Literal(Constant(named)),
              Literal(Constant(position))
            )
          )
          .withSpan(tree.span)
      }
      payload.ruleData.fold(origined)(data =>
        origined
          .select(
            origined.tpe.widen.member(termName("retainDerivation")).symbol
          )
          .appliedToArgs(List(Carrier.literals(data)))
          .withSpan(tree.span)
      )
    tree.fun.getAttachment(RuntimeCapture.pending) match
      case Some(payload) if !tree.tpe.widen.isInstanceOf[MethodOrPoly] =>
        return retained(tree, payload)
      case _ => ()
    tree.getAttachment(RuntimeCapture.payload) match
      case None          => tree
      case Some(payload) =>
        def retarget(fun: tpd.Tree): tpd.Tree = fun match
          case tpd.Apply(previous, args) =>
            retarget(previous).appliedToArgs(args)
          case tpd.TypeApply(previous, args) =>
            retarget(previous).appliedToTypeTrees(args)
          case tpd.Select(receiver, _) => receiver.select(payload.target)
          case _                       =>
            val owner = fun.symbol.owner
            val receiver = if owner.isClass then owner.sourceModule else owner
            tpd.ref(receiver).select(payload.target)
        val captured = retarget(tree.fun)
          .appliedToArgs(
            tree.args ++ List(
              tpd.Literal(Constant(payload.root)),
              Carrier.literals(payload.chunks)
            )
          )
          .withSpan(tree.span)
        if captured.tpe.widen.isInstanceOf[MethodOrPoly] then
          captured.putAttachment(RuntimeCapture.pending, payload): Unit
          captured
        else retained(captured, payload)

private object ChannelNames:
  val generated = new dotty.tools.dotc.util.Property.StickyKey[Unit]

private object EntityNames:
  val generated = new dotty.tools.dotc.util.Property.StickyKey[Unit]

final private class NamesPhase(after: String = "reifyCapture") extends PluginPhase:
  val phaseName = "reifyNames"
  override val runsAfter = Set(after)
  override val runsBefore = Set(Pickler.name)

  private val resultSites = mutable.Map.empty[Symbol, tpd.DefDef]
  private val resultBindings = mutable.Set.empty[Symbol]
  private val resultContracts = mutable.Map.empty[Symbol, Boolean]
  private var describedStatusAwait = Option.empty[tpd.DefDef]
  private var conversionImplementations = Set.empty[Symbol]
  private var temporalConversionImplementations = Set.empty[Symbol]
  private var resultRun = Option.empty[dotty.tools.dotc.Run]

  private def indexResults()(using ctx: Context): Unit =
    if !resultRun.exists(_ eq ctx.run) then
      resultSites.clear()
      resultBindings.clear()
      resultContracts.clear()
      describedStatusAwait = None
      conversionImplementations = Set.empty
      temporalConversionImplementations = Set.empty
      resultRun = Some(ctx.run)
      ctx.run.units.foreach { unit =>
        new tpd.TreeTraverser:
          def traverse(tree: tpd.Tree)(using Context): Unit =
            tree match
              case definition: tpd.DefDef =>
                resultSites(definition.symbol) = definition
              case binding: tpd.Bind => resultBindings += binding.symbol
              case _                 => ()
            traverseChildren(tree)
        .traverse(unit.tpdTree)
      }
      val sites = resultSites.valuesIterator
        .filter: definition =>
          describedStatusDefinition(definition)
        .toVector
      if sites.size == 1 then describedStatusAwait = sites.headOption
      val framework = Vector(
        canonicalImplementation(
          "framework.realize.Conversion$package$",
          "commandId",
          2,
          "java.lang.String"
        ),
        canonicalImplementation(
          "framework.realize.Conversion$package$",
          "constructedCommandToIR",
          3,
          "io.temporal.server.api.umpire.v1.Command"
        )
      )
      if framework.forall(_.nonEmpty) then conversionImplementations = framework.flatten.toSet
      val kitImplementations = Vector(
        canonicalImplementation(
          "temporal.realize.TemporalConversion",
          "commandToIR",
          2,
          "io.temporal.server.api.umpire.v1.Command"
        ),
        canonicalImplementation(
          "temporal.realize.TemporalConversion",
          "deadlineItem",
          4,
          "io.temporal.server.api.umpire.v1.Item"
        )
      )
      if kitImplementations.forall(_.nonEmpty) then
        temporalConversionImplementations = kitImplementations.flatten.toSet

  private def describedStatusMember(sym: Symbol)(using Context): Boolean =
    sym.exists && sym.owner.fullName.toString == "temporal.realize.DescribedStatus" &&
      sym.name.toString == "await" && sym.paramSymss.flatten.size == 1

  private def describedStatusDefinition(
      tree: tpd.DefDef
  )(using Context): Boolean =
    describedStatusMember(tree.symbol)

  private def defaultArgument(tree: tpd.Tree)(using Context): Boolean =
    var found = false
    new tpd.TreeTraverser:
      def traverse(tree: tpd.Tree)(using Context): Unit =
        if tree.symbol.exists && tree.symbol.name.toString.contains("$default$")
        then found = true
        else traverseChildren(tree)
    .traverse(tree)
    found

  private def namedInvocation(tree: tpd.Apply)(using Context): Boolean =
    val typedPoll = getClassIfDefined("framework.realize.Instruction.TypedPoll")
    val named =
      if typedPoll.exists then typedPoll.asClass.typeRef.member(termName("named")).symbol
      else NoSymbol
    val parameters = named.paramSymss.flatten
    named.exists && tree.fun.symbol == named && parameters.size == 3 &&
    parameters.forall(
      _.info.widenTermRefExpr =:= defn.StringType
    ) && tree.args.size == 3 &&
    !tree.args.exists(defaultArgument)

  private def namedExpression(tree: tpd.Tree)(using Context): Boolean =
    tree match
      case Typed(value, _)      => namedExpression(value)
      case NamedArg(_, value)   => namedExpression(value)
      case Inlined(_, _, value) => namedExpression(value)
      case Block(_, value)      => namedExpression(value)
      case If(_, yes, no)       => namedExpression(yes) && namedExpression(no)
      case Match(_, cases)      =>
        cases.nonEmpty && cases.forall(c => namedExpression(c.body))
      case application: Apply => namedInvocation(application)
      case _                  => false

  private def noNamedResultEscape(tree: tpd.Tree)(using Context): Boolean =
    var safe = true
    new tpd.TreeTraverser:
      def traverse(tree: tpd.Tree)(using Context): Unit = tree match
        case _: tpd.Return | _: tpd.Try => safe = false
        case _                          => traverseChildren(tree)
    .traverse(tree)
    safe

  private def namedResult(sym: Symbol)(using ctx: Context): Boolean =
    if !describedStatusMember(sym) then false
    else
      indexResults()
      describedStatusAwait match
        case None =>
          val carrier = getClassIfDefined(Carrier.namedResult)
          carrier.exists && sym.hasAnnotation(carrier)
        case Some(definition) =>
          sym == definition.symbol && resultContracts.getOrElseUpdate(
            definition.symbol,
            noNamedResultEscape(definition.rhs) && namedExpression(
              definition.rhs
            )
          )

  private def retainNamedResult(tree: tpd.DefDef)(using Context): Unit =
    if describedStatusDefinition(tree) &&
      resultContracts.getOrElseUpdate(
        tree.symbol,
        noNamedResultEscape(tree.rhs) && namedExpression(tree.rhs)
      )
    then
      val carrier = getClassIfDefined(Carrier.namedResult)
      if carrier.exists && !tree.symbol.hasAnnotation(carrier) then
        tree.symbol.addAnnotation(Annotation(carrier.asClass, Nil, tree.span))

  private def enclosingMethods(owner: Symbol)(using Context): Vector[Symbol] =
    val methods = Vector.newBuilder[Symbol]
    var current = owner
    while current.exists do
      if current.is(Method) then methods += current
      current = current.owner
    methods.result()

  private def enclosingMethod(owner: Symbol)(using Context): Option[Symbol] =
    enclosingMethods(owner).headOption

  private def canonicalImplementation(
      owner: String,
      name: String,
      parameters: Int,
      result: String
  )(using Context): Option[Symbol] =
    val methods = resultSites.iterator.collect {
      case (symbol, definition)
          if symbol.owner.fullName.toString == owner &&
            symbol.name.toString == name &&
            symbol.paramSymss.flatten.size == parameters &&
            definition.tpt.tpe.widenDealias.typeSymbol.fullName.toString == result =>
        symbol
    }.toVector
    methods match
      case Vector(method) => Some(method)
      case _              => None

  private def workerInstructionReference(tree: tpd.Apply)(using
      ctx: Context
  ): Boolean =
    val worker = getClassIfDefined("temporal.realize.WorkerInstruction")
    val referenced = getClassIfDefined(
      "temporal.realize.ReferencedWorkerInstruction"
    )
    val enclosing = enclosingMethod(ctx.owner)
    val referencedOwners =
      if referenced.exists then
        Set(referenced.companionModule, referenced.companionModule.moduleClass)
      else Set.empty[Symbol]
    def self(tree: tpd.Tree): Boolean = tree match
      case value: tpd.This  => value.tpe.widenDealias.typeSymbol == worker
      case Typed(value, _)  => self(value)
      case Inlined(_, _, v) => self(v)
      case _                => false
    val workerNamed =
      if worker.exists then worker.asClass.typeRef.member(termName("named")).symbol
      else NoSymbol
    worker.exists && referenced.exists && enclosing.contains(workerNamed) &&
    tree.fun.symbol.exists && referencedOwners(tree.fun.symbol.owner) &&
    tree.fun.symbol.name.toString == "apply" && (tree.args match
      case List(value) => self(value)
      case _           => false)

  private def conversionImplementation(using ctx: Context): Boolean =
    indexResults()
    enclosingMethods(ctx.owner).exists(conversionImplementations)

  private def temporalConversionImplementation(using ctx: Context): Boolean =
    indexResults()
    enclosingMethods(ctx.owner).exists(temporalConversionImplementations)

  private def commandedResult(tree: tpd.Apply)(using Context): Boolean =
    val method = tree.fun.symbol
    method.exists && method.is(Method) &&
    method.name.toString == "commanded" &&
    method.owner.fullName.toString == "framework.realize.Scripts$package$" &&
    method.paramSymss.flatten.size == 1 && tree.args.size == 1

  override def transformTemplate(tree: tpd.Template)(using
      ctx: Context
  ): tpd.Tree =
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
                List(
                  ref(symbol.companionModule),
                  Literal(Constant(symbol.name.toString))
                )
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

  override def transformSelect(tree: tpd.Select)(using Context): tpd.Tree =
    entityName(tree)

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
    val isAction =
      action.exists && tree.tpt.tpe.widenDealias.derivesFrom(action.asClass)
    val isEntity =
      entity.exists && tree.tpt.tpe.widenDealias.derivesFrom(entity.asClass)
    val declared = isAction || isEntity ||
      (channel.exists && tree.tpt.tpe.widenDealias.derivesFrom(channel.asClass))
    val bound = isAction && (tree.rhs match
      case Apply(Select(receiver, operation), _) if Set("on", "creates")(operation.toString) =>
        alias(receiver)
      case _ => false)
    val channelOwnedAction = isAction && channel.exists && tree.symbol.owner == channel
    if declared && !channelOwnedAction &&
      !tree.symbol.is(Param) && !tree.symbol.is(ParamAccessor) && !tree.symbol
        .is(Synthetic) && !tree.symbol.owner.is(Method) &&
      (!isEntity || (tree.symbol.owner.isClass && tree.symbol.owner.is(Module))) &&
      (!alias(tree.rhs) || bound)
    then
      val id = tree.symbol.fullName.toString
        .split('.')
        .filterNot(_.endsWith("$package$"))
        .map(_.stripSuffix("$"))
        .mkString(".")
      val method = tree.rhs.tpe.widenDealias
        .member(termName(if isEntity then "declaredAt" else "sourceAt"))
        .symbol
      val selected = tree.rhs.select(method)
      if isEntity then selected.putAttachment(EntityNames.generated, ()): Unit
      else if !isAction then selected.putAttachment(ChannelNames.generated, ()): Unit
      cpy.ValDef(tree)(rhs =
        selected
          .appliedTo(Literal(Constant(if isEntity then tree.name.toString
          else id)))
          .withSpan(tree.rhs.span)
      )
    else tree

  override def transformApply(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    val sourceNames = new tpd.TreeMap:
      override def transform(tree: tpd.Tree)(using Context): tpd.Tree =
        tree match
          case selected: tpd.Select => super.transform(entityName(selected))
          case _                    => super.transform(tree)
    val retained = tree.args.map:
      case arg: tpd.NamedArg =>
        cpy.NamedArg(arg)(arg.name, sourceNames.transform(arg.arg))
      case arg => arg
    val named = getClassIfDefined("framework.realize.Named")
    val namingContract = named.exists && tree.fun.symbol.exists &&
      tree.fun.symbol.owner == named && Set("copied", "referenced")(
        tree.fun.symbol.name.toString
      )
    if !named.exists || workerInstructionReference(tree) || (ctx.owner.is(
        Synthetic
      ) && ctx.owner
        .is(
          Method
        ) && ctx.owner.name.toString != "$anonfun") || namingContract ||
      commandedResult(tree) || conversionImplementation ||
      temporalConversionImplementation
    then cpy.Apply(tree)(tree.fun, retained)
    else
      val params = tree.fun.tpe.widen match
        case mt: MethodType => mt.paramInfos
        case _              => Nil
      val args = retained.zipWithIndex.map { (arg, index) =>
        params.lift(index) match
          case Some(param) if !top(param.widenDealias.typeSymbol) =>
            name(arg, named)
          case _ => arg
      }
      cpy.Apply(tree)(tree.fun, args)

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    retainNamedResult(tree)
    val named = getClassIfDefined("framework.realize.Named")
    if named.exists && tree.name.toString.contains("$default$") && !top(
        tree.tpt.tpe.widenDealias.typeSymbol
      )
    then cpy.DefDef(tree)(rhs = name(tree.rhs, named))
    else tree

  private def top(sym: Symbol)(using Context): Boolean =
    sym == defn.AnyClass || sym == defn.ObjectClass || sym == defn.MatchableClass || sym == defn.AnyValClass

  private def name(arg: tpd.Tree, named: Symbol)(using Context): tpd.Tree =
    arg match
      case NamedArg(n, value) => cpy.NamedArg(arg)(n, name(value, named))
      case Typed(value, tpt)  => cpy.Typed(arg)(name(value, named), tpt)
      case SeqLiteral(_, tpt) if top(tpt.tpe.widenDealias.typeSymbol) => arg
      case SeqLiteral(values, tpt)                                    =>
        cpy.SeqLiteral(arg)(values.map(name(_, named)), tpt)
      case application: Apply if commandedResult(application)        => arg
      case application: Apply if namedResult(application.fun.symbol) => arg
      case _ if arg.tpe.widenDealias.derivesFrom(named.asClass)      =>
        val sym = arg.symbol
        indexResults()
        arg match
          case _ if sym.exists && resultBindings(sym)                                         => arg
          case _ if sym.exists && sym.name.toString.contains("$default$")                     => arg
          case _: Ident | _: Select if sym.exists && (sym.is(Param) || sym.is(ParamAccessor)) =>
            arg
          case _: Ident | _: Select
              if sym.exists && sym.isTerm && (!sym
                .is(Method) || sym.is(Accessor)) && !sym.is(
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
