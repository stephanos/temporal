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
  Synthetic
}
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
  override val description =
    "lifts Model functions at their definitions and captures"
  override def initialize(options: List[String])(using
      Context
  ): List[PluginPhase] =
    val store = new DefinitionStore
    List(
      new FamilyPhase,
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

private object Carrier:
  val marker = "framework.dsl.lifted"
  val stored = "framework.dsl.liftedIr"
  val capture = "framework.dsl.capture"
  val section = "framework.dsl.sectionIr"
  val typePosition = "framework.dsl.ruleTypePosition"
  val callGroups = "framework.dsl.ruleCallGroups"

  private def modelFingerprint(model: ir.Model): String =
    Base64.getEncoder.encodeToString(
      java.security.MessageDigest.getInstance("SHA-256").digest(model.toByteArray)
    )

  private def callGroupsFingerprint(model: ir.Model, chunks: Seq[String]): String =
    val digest = java.security.MessageDigest.getInstance("SHA-256")
    digest.update(model.toByteArray)
    chunks.foreach { chunk =>
      val bytes = chunk.getBytes(java.nio.charset.StandardCharsets.UTF_8)
      digest.update(java.nio.ByteBuffer.allocate(4).putInt(bytes.length).array())
      digest.update(bytes)
    }
    Base64.getEncoder.encodeToString(digest.digest())

  def callGroupChunks(proof: CallGroupEvidence): List[String] = List(
    proof.owner.getOrElse(""),
    Base64.getEncoder.encodeToString(proof.call.toByteArray),
    Base64.getEncoder.encodeToString(proof.function.toByteArray),
    proof.groups.mkString(",")
  )

  def saveCallGroups(sym: Symbol, model: ir.Model, proofs: Seq[CallGroupEvidence], at: tpd.Tree)(
      using Context
  ): Unit =
    val retained = proofs.distinct
    if retained.nonEmpty then
      val chunks = retained.flatMap(callGroupChunks)
      sym.addAnnotation(
        Annotation(
          requiredClass(callGroups),
          List(
            literals(
              List(
                "rule-call-groups-v2",
                sym.fullName.toString,
                modelFingerprint(model),
                callGroupsFingerprint(model, chunks)
              ) ++ chunks
            )
          ),
          at.span
        )
      )

  def readCallGroups(sym: Symbol, model: ir.Model)(using Context): Vector[CallGroupEvidence] =
    def fail(message: String): Nothing =
      val position = model.functions
        .find(_.name == sym.fullName.toString)
        .flatMap(_.position)
        .getOrElse(ir.Position.defaultInstance)
      throw new SectionRefusal(
        position,
        sym.fullName.toString,
        s"has $message compiler-owned curried-call metadata"
      )
    val marker = getClassIfDefined(callGroups)
    if !marker.exists then Vector.empty
    else
      val annotations = sym.annotations
        .filter(_.symbol == marker)
        .map(_.arguments.flatMap(strings))
        .distinct
      if annotations.size > 1 then fail("conflicting")
      annotations.headOption.toVector.flatMap { data =>
        if data.take(3) != List(
            "rule-call-groups-v2",
            sym.fullName.toString,
            modelFingerprint(model)
          ) ||
          data.size <= 4 || (data.size - 4) % 4 != 0
        then fail("malformed or stale")
        val proofs = try
          data
            .drop(4)
            .grouped(4)
            .map { group =>
              CallGroupEvidence(
                Option(group(0)).filter(_.nonEmpty),
                ir.Call.parseFrom(Base64.getDecoder.decode(group(1))),
                ir.Function.parseFrom(Base64.getDecoder.decode(group(2))),
                group(3).split(',').map(_.toInt).toVector
              )
            }
            .toVector
        catch case scala.util.control.NonFatal(_) => fail("malformed")
        def containsCall(value: Any, call: ir.Call): Boolean = value match
          case found: ir.Call if found.equals(call) => true
          case values: Seq[?]                       => values.exists(containsCall(_, call))
          case product: Product => product.productIterator.exists(containsCall(_, call))
          case _                => false
        proofs.foreach { proof =>
          if !model.functions.exists(_.equals(proof.function)) ||
            proof.call.function != proof.function.name || proof.groups.size < 2 ||
            proof.groups.exists(_ < 0) || proof.groups
              .map(_.toLong)
              .sum != proof.function.params.size.toLong ||
            proof.call.args.size >= proof.function.params.size || proof.call.args.size != proof.groups.last ||
            !proof.owner.exists(owner =>
              model.functions.exists(f => f.name == owner && containsCall(f, proof.call))
            )
          then fail("stale")
        }
        if data(3) != callGroupsFingerprint(model, data.drop(4)) then fail("stale")
        proofs
      }

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
        ir.Model.parseFrom(
          Base64.getDecoder.decode(a.arguments.flatMap(strings).mkString)
        )
      }

  def hasSection(sym: Symbol)(using Context): Boolean =
    val cls = getClassIfDefined(section)
    cls.exists && sym.hasAnnotation(cls)

  def readTypePosition(sym: Symbol)(using Context): Option[ir.Position] =
    val marker = getClassIfDefined(typePosition)
    if !marker.exists then None
    else
      val values =
        sym.annotations.filter(_.symbol == marker).map { annotation =>
          annotation.arguments.flatMap(strings) match
            case List("rule-type-position-v1", name, file, line)
                if name == sym.fullName.toString && file.nonEmpty && line.toIntOption
                  .exists(_ > 0) =>
              ir.Position(file, line.toInt)
            case _ =>
              throw CompilerApiError(
                s"${sym.fullName} has malformed or stale type-definition position metadata"
              )
        }
      if values.distinct.size > 1 then
        throw CompilerApiError(
          s"${sym.fullName} has conflicting type-definition position metadata"
        )
      values.headOption

  private def sectionBytes(sym: Symbol, value: String)(using
      Context
  ): Array[Byte] =
    try Base64.getDecoder.decode(value)
    catch
      case error: IllegalArgumentException =>
        throw CompilerApiError(
          s"${sym.fullName} has malformed compiler-owned section IR: ${error.getMessage}"
        )

  private def sectionModel(sym: Symbol, value: String)(using
      Context
  ): ir.Model =
    try ir.Model.parseFrom(sectionBytes(sym, value))
    catch
      case error: com.google.protobuf.InvalidProtocolBufferException =>
        throw CompilerApiError(
          s"${sym.fullName} has malformed compiler-owned section IR: ${error.getMessage}"
        )

  def readSection(sym: Symbol)(using Context): Option[SectionOutcome] =
    val cls = getClassIfDefined(section)
    if !cls.exists then None
    else
      sym.getAnnotation(cls).map { annotation =>
        val data = annotation.arguments.flatMap(strings)
        data match
          case "section-v1" :: root :: form :: "ok" :: body
              if root == sym.fullName.toString && Set("is", "effect")(
                form
              ) && body.nonEmpty =>
            SectionOutcome(
              root,
              form,
              Some(sectionModel(sym, body.mkString)),
              None
            )
          case "section-v1" :: root :: form :: "refused" :: file :: line :: message
              if root == sym.fullName.toString && Set("is", "effect")(
                form
              ) && message.nonEmpty &&
                file.nonEmpty && line.toIntOption.exists(_ > 0) =>
            SectionOutcome(
              root,
              form,
              None,
              Some(
                ir.Position(file, line.toInt) ->
                  new String(
                    sectionBytes(sym, message.mkString),
                    java.nio.charset.StandardCharsets.UTF_8
                  )
              )
            )
          case _ =>
            throw CompilerApiError(
              s"${sym.fullName} has malformed or stale compiler-owned section IR"
            )
      }

  def sectionChunks(outcome: SectionOutcome): List[String] =
    val prefix = List("section-v1", outcome.root, outcome.form)
    outcome.refusal match
      case Some((position, message)) =>
        prefix ++ List("refused", position.file, position.line.toString) ++
          chunks(message.getBytes(java.nio.charset.StandardCharsets.UTF_8))
      case None => prefix ++ List("ok") ++ chunks(outcome.model.get)

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

final private class DefinitionStore:
  private val active = mutable.Set.empty[Symbol]
  private val sites = mutable.Map.empty[Symbol, (CompilationUnit, tpd.DefDef)]
  private val sectionSites =
    mutable.Map.empty[Symbol, (CompilationUnit, tpd.ValDef)]
  private val declared = mutable.Set.empty[Symbol]
  private val locations = mutable.Map.empty[String, ir.Position]
  private val sugarTrees = mutable.Map.empty[Symbol, tpd.Tree]
  private val sugarRoots =
    new java.util.IdentityHashMap[DottyUnpickler, List[tpd.Tree]]
  private var indexed = false
  private var compilerRun = Option.empty[dotty.tools.dotc.Run]

  private def enterRun()(using ctx: Context): Unit =
    if !compilerRun.exists(_ eq ctx.run) then
      active.clear()
      sites.clear()
      sectionSites.clear()
      declared.clear()
      locations.clear()
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
              case d: tpd.DefDef                            => sites(d.symbol) = unit -> d
              case v: tpd.ValDef if sectionForm(v).nonEmpty =>
                sectionSites(v.symbol) = unit -> v
              case _ => ()
            t match
              case d: tpd.TypeDef
                  if d.symbol.isClass && !d.symbol.is(Module) &&
                    (d.symbol.is(Enum) || d.symbol.is(Case)) =>
                val position = d.srcPos.sourcePos
                if position.exists then
                  locations(d.symbol.fullName.toString) =
                    ir.Position(position.source.path, position.line + 1)
              case _ => ()
            t match
              case d: tpd.MemberDef => declared += d.symbol
              case _                => ()
            traverseChildren(t)
        .traverse(unit.tpdTree)

  def inRun(sym: Symbol)(using Context): Boolean =
    index()
    declared(sym)

  def typeLocations(using Context): Map[String, ir.Position] =
    index()
    locations.toMap

  def saveTypeLocation(tree: tpd.TypeDef)(using Context): Unit =
    typeLocations.get(tree.symbol.fullName.toString).foreach { position =>
      Carrier.readTypePosition(tree.symbol) match
        case Some(prior) if !prior.equals(position) =>
          throw CompilerApiError(
            s"${tree.symbol.fullName} has stale type-definition position metadata"
          )
        case Some(_) => ()
        case None    =>
          tree.symbol.addAnnotation(
            Annotation(
              getClassIfDefined(Carrier.typePosition).asClass,
              List(
                Carrier.literals(
                  List(
                    "rule-type-position-v1",
                    tree.symbol.fullName.toString,
                    position.file,
                    position.line.toString
                  )
                )
              ),
              tree.span
            )
          )
    }

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
      section.isClass && section.is(Module) && owner.isClass && owner.is(
        Module
      ) &&
      machine.exists && owner.asClass.typeRef.derivesFrom(machine.asClass)
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
          val result = try
            e.readFinites()
            val block = e.ctx.blockVal(value.symbol.asInstanceOf[Symbol]).get
            e.ctx.functions(root) = e.blockFunction(root, block)
            SectionOutcome(
              root,
              form,
              Some(
                ir.Model(
                  functions = e.ctx.functions.values.toVector,
                  types = e.ctx.types.values.toVector
                )
              ),
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
          result.model.foreach(model =>
            Carrier.saveCallGroups(value.symbol, model, e.ctx.callGroups.toVector, value)
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
      s => inRun(s.asInstanceOf[Symbol]),
      s => inJar(s.asInstanceOf[Symbol]),
      s => jarSugar(s.asInstanceOf[Symbol]).map(_.asInstanceOf[q.reflect.Tree]),
      s => sectionPresent(s.asInstanceOf[Symbol]),
      s => section(s.asInstanceOf[Symbol]),
      (s, model) => Carrier.readCallGroups(s.asInstanceOf[Symbol], model)
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
          Carrier.saveCallGroups(d.symbol, model, e.ctx.callGroups.toVector, d)
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
  override val runsAfter = Set("reifyFamily")
  override val runsBefore = Set("sbt-api", Pickler.name)

  override def transformTypeDef(tree: tpd.TypeDef)(using Context): tpd.Tree =
    try store.saveTypeLocation(tree)
    catch
      case error: CompilerApiError =>
        report.error(error.getMessage, tree.srcPos)
    tree

  override def transformValDef(tree: tpd.ValDef)(using Context): tpd.Tree =
    store.saveSection(tree): Unit
    tree

  override def transformDefDef(tree: tpd.DefDef)(using Context): tpd.Tree =
    val section = tree.symbol.owner
    val declaring = section.owner
    val machine = getClassIfDefined("framework.dsl.Machine")
    if section.isClass && section.is(Module) && section.name.toString
        .stripSuffix(
          "$"
        ) == "effects" &&
      declaring.isClass && declaring.is(Module) && machine.exists &&
      declaring.asClass.typeRef.derivesFrom(machine.asClass)
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
      tree.getAttachment(EndpointPayload.key).fold(tree) { payload =>
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
        cpy.Template(tree)(body = registration :: tree.body)
      }
    else
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
          val start = member("init").map {
            case v: ValDef =>
              e.lift(v.rhs.getOrElse(e.ctx.fail(v, "init has no body")))
            case d: DefDef if d.termParamss.isEmpty =>
              e.lift(d.rhs.getOrElse(e.ctx.fail(d, "init has no body")))
            case other =>
              e.ctx.fail(other, "init is a val or a parameterless def")
          }
          val name = e.ctx.objectFormName(cls)
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
        val owner = section
          .map(_.owner)
          .filter(s =>
            s.isClass && s.is(Module) &&
              machineClass.exists && s.asClass.typeRef
                .derivesFrom(machineClass.asClass)
          )
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
            val family = e.ctx.familyOf(cls)
            val machine = e.ctx.objectFormName(cls)
            val provenance = List(owner.get.fullName.toString, family, machine)
            val frameworkJar =
              Option(machineClass.binaryFile)
                .flatMap(_.underlyingSource)
                .map(_.path)
            def positionChunks(reference: Term): List[String] =
              val sourcePositions = store.typeLocations
              val positions = mutable.Map.empty[String, ir.Position]
              val seen = mutable.Set.empty[String]
              def location(symbol: e.ctx.quotes.reflect.Symbol): Unit =
                if !symbol.isNoSymbol && symbol.isClassDef && !symbol.flags.is(
                    Flags.Module
                  ) &&
                  (symbol.flags.is(Flags.Enum) || symbol.flags.is(
                    Flags.Case
                  )) &&
                  !positions.contains(e.ctx.irTypeName(symbol))
                then
                  val name = e.ctx.irTypeName(symbol)
                  val raw =
                    symbol.asInstanceOf[dotty.tools.dotc.core.Symbols.Symbol]
                  val jar = Option(raw.binaryFile)
                    .flatMap(_.underlyingSource)
                    .map(_.path)
                  positions(name) = sourcePositions
                    .get(symbol.fullName)
                    .getOrElse(if jar.nonEmpty && jar == frameworkJar then
                      ir.Position.defaultInstance
                    else
                      Carrier
                        .readTypePosition(raw)
                        .getOrElse(
                          e.ctx.fail(
                            reference,
                            s"${symbol.fullName} has no retained type-definition position metadata"
                          )
                        ))
              def domain(tpe: TypeRepr): Unit =
                val t = tpe.dealias
                val symbol = t.typeSymbol
                if !symbol.isNoSymbol && seen.add(t.show) then
                  location(symbol)
                  t.typeArgs.foreach(domain)
                  symbol.caseFields.foreach(field => domain(t.memberType(field)))
                  symbol.children.foreach(child =>
                    child.caseFields.foreach(field => domain(child.typeRef.memberType(field)))
                  )
              def inputs(tpe: TypeRepr): Unit =
                val t = tpe.dealias
                if t.typeSymbol.fullName == "scala.*:" then
                  t.typeArgs.headOption.foreach(domain)
                  t.typeArgs.drop(1).headOption.foreach(inputs)
                else t.typeArgs.foreach(domain)
              val action =
                getClassIfDefined("framework.dsl.Action").asInstanceOf[Symbol]
              object actions extends TreeTraverser:
                override def traverseTree(t: Tree)(
                    owner: e.ctx.quotes.reflect.Symbol
                ): Unit =
                  t match
                    case term: Term =>
                      term.tpe.widen
                        .baseType(action)
                        .typeArgs
                        .headOption
                        .foreach(inputs)
                    case _ => ()
                  super.traverseTree(t)(owner)
              cls.typeRef
                .baseType(machineClass.asInstanceOf[Symbol])
                .typeArgs
                .headOption
                .foreach(domain)
              actions.traverseTree(reference)(
                e.ctx.quotes.reflect.Symbol.spliceOwner
              )
              e.ctx.namedTypeSymbols.foreach(location)
              Carrier.chunks(
                ir.Model(types = positions.toVector.sortBy(_._1).map { (name, position) =>
                  ir.Type(name = name, position = Some(position))
                })
              )
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
              val typePositions = positionChunks(tree.asInstanceOf[Term])
              tree.putAttachment(
                RulesPayload.key,
                RulesPayload.Payload(
                  method(rulesClass, "capturedDisabledRules"),
                  "ordinary-disabled-v1" :: (provenance ++ List(
                    typePositions.size.toString
                  ) ++ typePositions ++ entries),
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
                .sourceTree(section.get.asInstanceOf[Symbol])
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
              val guard = e.ruleGuard(
                heading,
                stateName,
                tree.asInstanceOf[Tree],
                _ => Some(e.ctx.pos(rules))
              )
              val result = e.ruleEffect(
                effect,
                stateName,
                inputNames,
                machine,
                "capture",
                tree.asInstanceOf[Tree]
              )
              val fragments =
                Carrier.chunks(ir.ListOf(Seq(guard, result)).toByteArray)
              val curried = e.ctx.callGroups.toVector.distinct.map { proof =>
                List(
                  bytes(proof.call),
                  bytes(proof.function),
                  proof.groups.mkString(","),
                  proof.owner.getOrElse("")
                )
              }
              val closure = Carrier.chunks(
                ir.Model(
                  functions = e.ctx.functions.values.toVector,
                  types = e.ctx.types.values.toVector
                )
              )
              val typePositions = positionChunks(
                e.ruleBlock(rules, tree.asInstanceOf[Tree])
              )
              val data = List("ordinary-rule-v2") ++ provenance ++
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
              tree.putAttachment(
                RulesPayload.key,
                RulesPayload.Payload(
                  method(caseClass, s"capturedRule${params.size - 1}"),
                  data,
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

  override def transformApply(tree: tpd.Apply)(using ctx: Context): tpd.Tree =
    val marker = getClassIfDefined(Carrier.capture)
    if !marker.exists then tree
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
            val (root, named) =
              e.stepFunctionOrigin(
                e.resolve(arg),
                tree.fun.symbol.fullName.toString,
                "capture"
              )
            val binding = getClassIfDefined("framework.dsl.StepBinding")
            val origin =
              if binding.exists && tree.tpe.widen.derivesFrom(binding.asClass)
              then
                val machine = getClassIfDefined("framework.dsl.Machine")
                val owner = Iterator
                  .iterate(ctx.owner)(_.owner)
                  .takeWhile(_.exists)
                  .find(s =>
                    s.isClass && machine.exists && s.asClass.typeRef
                      .derivesFrom(machine.asClass)
                  )
                Some(
                  (
                    owner.fold("")(_.fullName.toString),
                    owner.fold("") { s =>
                      val name = s.name.toString.stripSuffix("$")
                      name.take(1).toLowerCase + name.drop(1)
                    },
                    named
                  )
                )
              else None
            val model = ir.Model(
              functions = e.ctx.functions.values.toVector,
              types = e.ctx.types.values.toVector
            )
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
                origin
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
  final case class Payload(
      target: Symbol,
      root: String,
      chunks: List[String],
      origin: Option[(String, String, Boolean)]
  )
  val payload = new dotty.tools.dotc.util.Property.StickyKey[Payload]

final private class RuntimeCapturePhase extends PluginPhase:
  val phaseName = "reifyRuntimeCapture"
  override val runsAfter = Set(Pickler.name)
  override val runsBefore = Set("reifySignature", "inlining")

  override def transformApply(tree: tpd.Apply)(using Context): tpd.Tree =
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
        payload.origin.fold(captured) { (owner, machine, named) =>
          captured
            .select(captured.tpe.widen.member(termName("retainOrigin")).symbol)
            .appliedToArgs(
              List(
                Literal(Constant(owner)),
                Literal(Constant(machine)),
                Literal(Constant(named))
              )
            )
            .withSpan(tree.span)
        }

private object ChannelNames:
  val generated = new dotty.tools.dotc.util.Property.StickyKey[Unit]

final private class NamesPhase extends PluginPhase:
  val phaseName = "reifyNames"
  override val runsAfter = Set("reifyCapture")
  override val runsBefore = Set(Pickler.name)

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
    if declared &&
      !tree.symbol.is(Param) && !tree.symbol.is(Synthetic) && !tree.symbol.owner
        .is(
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
          case Some(param) if !top(param.widenDealias.typeSymbol) =>
            name(arg, named)
          case _ => arg
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

  private def name(arg: tpd.Tree, named: Symbol)(using Context): tpd.Tree =
    arg match
      case NamedArg(n, value) => cpy.NamedArg(arg)(n, name(value, named))
      case Typed(value, tpt)  => cpy.Typed(arg)(name(value, named), tpt)
      case SeqLiteral(_, tpt) if top(tpt.tpe.widenDealias.typeSymbol) => arg
      case SeqLiteral(values, tpt)                                    =>
        cpy.SeqLiteral(arg)(values.map(name(_, named)), tpt)
      case _ if arg.tpe.widenDealias.derivesFrom(named.asClass) =>
        val sym = arg.symbol
        arg match
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
