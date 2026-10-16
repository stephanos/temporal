package framework.reify

import java.util.Base64
import java.util.zip.ZipFile
import scala.collection.mutable
import scala.jdk.CollectionConverters.*
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
    val family = new FamilyPhase
    if mode == "realization" then
      List(
        family,
        new StorePhase(store),
        new NamesPhase("reifyStore"),
        new SchemaPhase(store),
        new RealizationNamesPhase
      )
    else if mode == "signature" then
      List(
        family,
        new StorePhase(store),
        new CapturePhase(store),
        new NamesPhase,
        new SchemaPhase(store),
        new RuntimeCapturePhase,
        new SchemaPhase(store, signatures = true)
      )
    else
      List(
        family,
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
  val facet = "framework.dsl.facetIr"
  val status = "framework.dsl.recordedStatusIr"
  val outcome = "framework.dsl.declaredOutcomeIr"
  val typePosition = "framework.dsl.ruleTypePosition"
  val callGroups = "framework.dsl.ruleCallGroups"
  val familyFlow = "framework.dsl.familyFlow"
  val namedResult = "framework.dsl.namedResult"

  private def modelFingerprint(model: ir.Model): String =
    Base64.getEncoder.encodeToString(
      java.security.MessageDigest
        .getInstance("SHA-256")
        .digest(model.toByteArray)
    )

  private def callGroupsFingerprint(
      model: ir.Model,
      chunks: Seq[String]
  ): String =
    val digest = java.security.MessageDigest.getInstance("SHA-256")
    digest.update(model.toByteArray)
    chunks.foreach { chunk =>
      val bytes = chunk.getBytes(java.nio.charset.StandardCharsets.UTF_8)
      digest.update(
        java.nio.ByteBuffer.allocate(4).putInt(bytes.length).array()
      )
      digest.update(bytes)
    }
    Base64.getEncoder.encodeToString(digest.digest())

  def callGroupChunks(proof: CallGroupEvidence): List[String] = List(
    proof.owner.getOrElse(""),
    Base64.getEncoder.encodeToString(proof.call.toByteArray),
    Base64.getEncoder.encodeToString(proof.function.toByteArray),
    proof.groups.mkString(",")
  )

  def saveCallGroups(
      sym: Symbol,
      model: ir.Model,
      proofs: Seq[CallGroupEvidence],
      at: tpd.Tree
  )(using
      Context
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

  def readCallGroups(sym: Symbol, model: ir.Model)(using
      Context
  ): Vector[CallGroupEvidence] =
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
          case product: Product                     =>
            product.productIterator.exists(containsCall(_, call))
          case _ => false
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
    val prefix =
      List("facet-helper-v1", outcome.facet, outcome.namespace, outcome.root)
    val data = outcome.refusal match
      case Some((position, message)) =>
        prefix ++ List("refused", position.file, position.line.toString) ++
          chunks(message.getBytes(java.nio.charset.StandardCharsets.UTF_8))
      case None =>
        val model = chunks(outcome.model.get)
        prefix ++ List("ok", model.size.toString) ++ model ++
          List(outcome.sites.size.toString) ++ outcome.sites.toList.flatMap(
            siteChunks
          )
    data :+ callGroupsFingerprint(ir.Model.defaultInstance, data)

  def readFacet(sym: Symbol)(using Context): Option[FacetOutcome] =
    val marker = getClassIfDefined(facet)
    if !marker.exists then None
    else
      val annotations = sym.annotations
        .filter(_.symbol == marker)
        .map(_.arguments.flatMap(strings))
        .distinct
      def fail(): Nothing = throw CompilerApiError(
        s"${sym.fullName} has malformed, conflicting or stale compiler-owned facet template"
      )
      if annotations.size > 1 then fail()
      annotations.headOption.map { all =>
        try
          if all.size < 7 || all(0) != "facet-helper-v1" || all(
              3
            ) != sym.fullName.toString ||
            all.last != callGroupsFingerprint(
              ir.Model.defaultInstance,
              all.init
            )
          then fail()
          val data = all.init
          data(4) match
            case "refused" =>
              if data.size < 8 || data(5).isEmpty || data(6).toInt < 1 then fail()
              FacetOutcome(
                data(1),
                data(2),
                data(3),
                None,
                Vector.empty,
                Some(
                  ir.Position(data(5), data(6).toInt) -> new String(
                    Base64.getDecoder.decode(data.drop(7).mkString),
                    java.nio.charset.StandardCharsets.UTF_8
                  )
                )
              )
            case "ok" =>
              val count = data(5).toInt
              if count < 1 || count > data.size - 7 then fail()
              val model = sectionModel(sym, data.slice(6, 6 + count).mkString)
              val sites = data(6 + count).toInt
              if sites < 0 || (data.size - 7 - count).toLong != sites.toLong * 7
              then fail()
              val found = data
                .drop(7 + count)
                .grouped(7)
                .map { values =>
                  ContextualPhaseSite(
                    values(0),
                    Option(values(1)).filter(_.nonEmpty),
                    ir.Call.parseFrom(Base64.getDecoder.decode(values(2))),
                    ir.TypeRef.parseFrom(Base64.getDecoder.decode(values(3))),
                    ir.TypeRef.parseFrom(Base64.getDecoder.decode(values(4))),
                    ir.TypeRef.parseFrom(Base64.getDecoder.decode(values(5))),
                    Option(values(6))
                      .filter(_.nonEmpty)
                      .map(v => ir.Position.parseFrom(Base64.getDecoder.decode(v)))
                  )
                }
                .toVector
              def contains(value: Any, call: ir.Call): Boolean = value match
                case current: ir.Call if current.equals(call) => true
                case values: Seq[?]                           => values.exists(contains(_, call))
                case product: Product                         =>
                  product.productIterator.exists(contains(_, call))
                case _ => false
              if !model.functions
                  .exists(_.name == data(3)) || found.exists(site =>
                  !Set("read", "write")(site.kind) ||
                    !site.call.function.startsWith("$facet-context-v1.") ||
                    !site.call.function.endsWith("." + site.kind) ||
                    site.call.function.length <= "$facet-context-v1.".length + site.kind.length + 1 ||
                    site.call.args.size != (if site.kind == "read" then 1
                                            else 2) ||
                    site.state.ref.isEmpty || site.phase.ref.isEmpty || site.result.ref.isEmpty ||
                    !site.result.equals(if site.kind == "read" then site.phase
                    else site.state) ||
                    site.placement.exists(p =>
                      !p.equals(ir.Position.defaultInstance) &&
                        (p.file.isEmpty || p.line < 1)
                    ) ||
                    (site.kind == "write" && site.placement.nonEmpty) ||
                    !site.owner.exists(owner =>
                      model.functions.exists(f => f.name == owner && contains(f, site.call))
                    )
                )
              then fail()
              FacetOutcome(data(1), data(2), data(3), Some(model), found, None)
            case _ => fail()
        catch
          case error: CompilerApiError        => throw error
          case scala.util.control.NonFatal(_) => fail()
      }

  def statusChunks(
      outcome: StatusOutcome,
      version: String = "recorded-status-v1"
  ): List[String] =
    val prefix = List(version, outcome.root, outcome.phase, outcome.name)
    val data = outcome.refusal match
      case Some((position, message)) =>
        prefix ++ List("refused", position.file, position.line.toString) ++
          chunks(message.getBytes(java.nio.charset.StandardCharsets.UTF_8))
      case None =>
        val model = chunks(outcome.model.get)
        prefix ++ List(
          "ok",
          Base64.getEncoder.encodeToString(outcome.value.get.toByteArray),
          model.size.toString
        ) ++ model ++ List(outcome.proofs.size.toString) ++
          outcome.proofs.toList.flatMap(callGroupChunks)
    data :+ callGroupsFingerprint(ir.Model.defaultInstance, data)

  def readStatus(sym: Symbol)(using Context): Option[StatusOutcome] =
    readDeclared(sym, status, "recorded-status-v1")

  def readOutcome(sym: Symbol)(using Context): Option[StatusOutcome] =
    readDeclared(sym, outcome, "declared-ok-v1")

  private def readDeclared(sym: Symbol, annotation: String, version: String)(using
      Context
  ): Option[StatusOutcome] =
    val marker = getClassIfDefined(annotation)
    if !marker.exists then None
    else
      def fail(): Nothing = throw CompilerApiError(
        s"${sym.fullName} has malformed, conflicting or stale Recorded status metadata"
      )
      val annotations = sym.annotations
        .filter(_.symbol == marker)
        .map(_.arguments.flatMap(strings))
        .distinct
      if annotations.size > 1 then fail()
      annotations.headOption.map { all =>
        try
          if all.size < 9 || all.head != version || all(
              1
            ) != sym.fullName.toString ||
            all.last != callGroupsFingerprint(
              ir.Model.defaultInstance,
              all.init
            )
          then fail()
          val data = all.init
          data(4) match
            case "refused" =>
              if data(5).isEmpty || data(6).toInt < 1 then fail()
              StatusOutcome(
                data(1),
                data(2),
                data(3),
                None,
                None,
                Some(
                  ir.Position(data(5), data(6).toInt) -> new String(
                    Base64.getDecoder.decode(data.drop(7).mkString),
                    java.nio.charset.StandardCharsets.UTF_8
                  )
                )
              )
            case "ok" =>
              val value = ir.Expr.parseFrom(Base64.getDecoder.decode(data(5)))
              val count = data(6).toInt
              if count < 0 || count > data.size - 8 then fail()
              val model = sectionModel(sym, data.slice(7, 7 + count).mkString)
              val size = data(7 + count).toInt
              if size < 0 || (data.size - 8 - count).toLong != size.toLong * 4
              then fail()
              val proofs = data
                .drop(8 + count)
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
              def contains(value: Any, call: ir.Call): Boolean = value match
                case found: ir.Call if found.equals(call) => true
                case values: Seq[?]                       => values.exists(contains(_, call))
                case product: Product                     =>
                  product.productIterator.exists(contains(_, call))
                case _ => false
              if value.kind.isEmpty || data(3).isEmpty ||
                !model
                  .copy(functions = Nil, types = Nil)
                  .equals(ir.Model()) || proofs.exists(p =>
                  !model.functions.exists(
                    _.equals(p.function)
                  ) || p.call.function != p.function.name ||
                    p.groups.size < 2 || p.groups.exists(_ < 0) ||
                    p.groups
                      .map(_.toLong)
                      .sum != p.function.params.size.toLong ||
                    p.call.args.size >= p.function.params.size || p.call.args.size != p.groups.last ||
                    !p.owner.fold(contains(value, p.call))(owner =>
                      model.functions.exists(f => f.name == owner && contains(f, p.call))
                    )
                )
              then fail()
              StatusOutcome(
                data(1),
                data(2),
                data(3),
                Some(value),
                Some(model),
                None,
                proofs
              )
            case _ => fail()
        catch
          case error: CompilerApiError        => throw error
          case scala.util.control.NonFatal(_) => fail()
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

private object RuleTypeLocations:
  def chunks(e: PluginEngine)(
      reference: e.ctx.quotes.reflect.Term,
      state: Option[e.ctx.quotes.reflect.TypeRepr],
      sourcePositions: Map[String, ir.Position]
  )(using Context): List[String] =
    Carrier.chunks(model(e)(reference, state, sourcePositions))

  def model(e: PluginEngine)(
      reference: e.ctx.quotes.reflect.Term,
      state: Option[e.ctx.quotes.reflect.TypeRepr],
      sourcePositions: Map[String, ir.Position]
  )(using Context): ir.Model =
    import e.ctx.quotes.reflect.*
    val machine = getClassIfDefined("framework.dsl.Machine")
    val frameworkJar =
      Option(machine.binaryFile).flatMap(_.underlyingSource).map(_.path)
    val positions = mutable.Map.empty[String, ir.Position]
    val seen = mutable.Set.empty[String]
    def location(symbol: e.ctx.quotes.reflect.Symbol): Unit =
      if !symbol.isNoSymbol && symbol.isClassDef && !symbol.flags.is(
          Flags.Module
        ) &&
        (symbol.flags.is(Flags.Enum) || symbol.flags.is(Flags.Case)) &&
        !positions.contains(e.ctx.irTypeName(symbol))
      then
        val raw = symbol.asInstanceOf[dotty.tools.dotc.core.Symbols.Symbol]
        val jar = Option(raw.binaryFile).flatMap(_.underlyingSource).map(_.path)
        val position = sourcePositions
          .get(symbol.fullName)
          .getOrElse(if jar.nonEmpty && jar == frameworkJar then ir.Position.defaultInstance
          else
            Carrier
              .readTypePosition(raw)
              .getOrElse(
                e.ctx.fail(
                  reference,
                  s"${symbol.fullName} has no retained type-definition position metadata"
                )
              ))
        e.ctx.retainTypeSource(symbol, position)
        positions(e.ctx.irTypeName(symbol)) = position
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
    val action = Symbol.requiredClass("framework.dsl.Action")
    object actions extends TreeTraverser:
      override def traverseTree(t: Tree)(owner: Symbol): Unit =
        t match
          case term: Term =>
            term.tpe.widen.baseType(action).typeArgs.headOption.foreach(inputs)
          case _ => ()
        super.traverseTree(t)(owner)
    state.foreach(domain)
    actions.traverseTree(reference)(Symbol.spliceOwner)
    e.ctx.namedTypeSymbols.foreach(location)
    ir.Model(types = positions.toVector.sortBy(_._1).map { (name, position) =>
      ir.Type(name = name, position = Some(position))
    })

private object FacetPositions:
  def qualify[A <: scalapb.GeneratedMessage](
      value: A,
      companion: scalapb.GeneratedMessageCompanion[A],
      sources: collection.Map[String, String],
      collisions: Set[String]
  ): A =
    import scalapb.descriptors.{PMessage, PRepeated, PValue, ScalaType}
    def position(value: PValue): PValue = value match
      case message: PMessage =>
        val at = ir.Position.messageReads.read(message)
        if sources.get(at.file).exists(collisions) then ir.Position.defaultInstance.toPMessage
        else value
      case _ => value
    def rewrite(value: PValue): PValue = value match
      case PMessage(fields) =>
        PMessage(fields.map { (field, child) =>
          val rewritten = field.scalaType match
            case ScalaType.Message(descriptor)
                if descriptor.fullName == ir.Position.scalaDescriptor.fullName =>
              position(child)
            case _ => rewrite(child)
          field -> rewritten
        })
      case PRepeated(values) => PRepeated(values.map(rewrite))
      case _                 => value
    companion.messageReads.read(
      if companion.scalaDescriptor.fullName == ir.Position.scalaDescriptor.fullName
      then position(value.toPMessage)
      else rewrite(value.toPMessage)
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
  private val locations = mutable.Map.empty[String, ir.Position]
  private val sugarTrees = mutable.Map.empty[Symbol, tpd.Tree]
  private val sugarRoots =
    new java.util.IdentityHashMap[DottyUnpickler, List[tpd.Tree]]
  private var indexed = false
  private var compilerRun = Option.empty[dotty.tools.dotc.Run]
  private var frameworkLabels = Option.empty[Set[String]]

  private def enterRun()(using ctx: Context): Unit =
    if !compilerRun.exists(_ eq ctx.run) then
      active.clear()
      sites.clear()
      sectionSites.clear()
      statusSites.clear()
      outcomeSites.clear()
      declared.clear()
      locations.clear()
      sugarTrees.clear()
      sugarRoots.clear()
      indexed = false
      frameworkLabels = None
      compilerRun = Some(ctx.run)

  private def sourceLabel(symbol: Symbol)(using Context): Option[String] =
    val marker = getClassIfDefined("scala.annotation.internal.SourceFile")
    if !marker.exists then None
    else
      Iterator
        .iterate(symbol)(_.owner)
        .takeWhile(s => s.exists && !s.is(Package))
        .flatMap(_.getAnnotation(marker).flatMap(_.argumentConstantString(0)))
        .nextOption()

  private def frameworkSourceLabels(using ctx: Context): Set[String] =
    frameworkLabels.getOrElse {
      val machine = getClassIfDefined("framework.dsl.Machine")
      val file = Option(machine.binaryFile)
        .flatMap(_.underlyingSource)
        .map(_.path)
        .filter(_.endsWith(".jar"))
      val labels = file.fold {
        val sources = mutable.Set.empty[String]
        ctx.run.units.foreach { unit =>
          new tpd.TreeTraverser:
            def traverse(tree: tpd.Tree)(using Context): Unit =
              tree match
                case definition: tpd.TypeDef
                    if List("framework.dsl.", "framework.realize.")
                      .exists(definition.symbol.fullName.toString.startsWith) =>
                  sources += sourceLabel(definition.symbol).getOrElse(
                    unit.source.file.name
                  ): Unit
                case _ => ()
              traverseChildren(tree)
          .traverse(unit.tpdTree)
        }
        sources.toSet
      } { path =>
        val zip = new ZipFile(path)
        try
          zip.entries.asScala
            .filter(_.getName.endsWith(".tasty"))
            .flatMap { entry =>
              val name = entry.getName.stripSuffix(".tasty").replace('/', '.')
              List(getClassIfDefined(name), getClassIfDefined(name + "$"))
                .filter(_.exists)
                .flatMap(sourceLabel)
            }
            .toSet
        finally zip.close()
      }
      frameworkLabels = Some(labels)
      labels
    }

  def qualify[A <: scalapb.GeneratedMessage](
      engine: PluginEngine,
      value: A,
      companion: scalapb.GeneratedMessageCompanion[A]
  )(using Context): A =
    FacetPositions.qualify(
      value,
      companion,
      engine.ctx.sourceLabels,
      frameworkSourceLabels
    )

  def qualifiedGroups(engine: PluginEngine, proofs: Vector[CallGroupEvidence])(using
      Context
  ): Vector[CallGroupEvidence] = proofs.map(p =>
    p.copy(
      call = qualify(engine, p.call, ir.Call),
      function = qualify(engine, p.function, ir.Function)
    )
  )

  def qualifiedSites(engine: PluginEngine, sites: Vector[ContextualPhaseSite])(using
      Context
  ): Vector[ContextualPhaseSite] =
    sites.map(s => s.copy(call = qualify(engine, s.call, ir.Call)))

  def locatedModel(engine: PluginEngine)(
      reference: engine.ctx.quotes.reflect.Term,
      model: ir.Model
  )(using Context): ir.Model =
    val positions = RuleTypeLocations
      .model(engine)(reference, None, typeLocations)
      .types
      .map(t => t.name -> t.getPosition)
      .toMap
    model.copy(types =
      model.types.map(t => positions.get(t.name).fold(t)(p => t.copy(position = Some(p))))
    )

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
                ir.Model(
                  functions = e.ctx.reachableFunctions(Seq(value)),
                  types = e.ctx.types.values.toVector
                )
              ),
              None,
              e.ctx.retainedGroups(Seq(value))
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
                Carrier.literals(Carrier.statusChunks(result, "declared-ok-v1"))
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
                  ir.Model(
                    functions = e.ctx.reachableFunctions(Seq(value)),
                    types = e.ctx.types.values.toVector
                  )
                ),
                None,
                e.ctx.retainedGroups(Seq(value))
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
            val model = locatedModel(e)(
              d.rhs.get,
              ir.Model(
                functions = e.ctx.reachableFunctions(Nil, Seq(root)),
                types = e.ctx.types.values.toVector
              )
            )
            FacetOutcome(
              namespace.owner.fullName.toString,
              namespace.fullName.toString,
              root,
              Some(qualify(e, model, ir.Model)),
              qualifiedSites(
                e,
                e.ctx.retainedContextualSites(roots = Seq(root))
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
          result.model.foreach(model =>
            Carrier.saveCallGroups(
              symbol,
              model,
              qualifiedGroups(e, e.ctx.retainedGroups(Nil, Seq(root))),
              definition
            )
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
            val emitted = if contextual then
              qualify(e, locatedModel(e)(block.body, model), ir.Model)
            else model
            SectionOutcome(
              root,
              form,
              Some(emitted),
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
            val proofs = e.ctx.retainedGroups(Nil, Seq(root))
            Carrier.saveCallGroups(
              value.symbol,
              model,
              if contextual then qualifiedGroups(e, proofs) else proofs,
              value
            )
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
                        qualifiedSites(
                          e,
                          e.ctx.retainedContextualSites(roots = Seq(root))
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
      s => inRun(s.asInstanceOf[Symbol]),
      s => inJar(s.asInstanceOf[Symbol]),
      s => jarSugar(s.asInstanceOf[Symbol]).map(_.asInstanceOf[q.reflect.Tree]),
      s => sectionPresent(s.asInstanceOf[Symbol]),
      s => section(s.asInstanceOf[Symbol]),
      s => facet(s.asInstanceOf[Symbol]),
      s => status(s.asInstanceOf[Symbol]),
      s => outcome(s.asInstanceOf[Symbol]),
      ctx.run.units
        .map(unit => unit.source.path -> unit.source.file.name)
        .toMap,
      s => sourceLabel(s.asInstanceOf[Symbol]),
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
              store.qualify(
                e,
                ir.Model(
                  functions = e.ctx.reachableFunctions(Nil, roots),
                  types = e.ctx.types.values.toVector
                ),
                ir.Model
              )
            )
            val positions = Carrier.chunks(
              store.qualify(
                e,
                RuleTypeLocations.model(e)(
                  c.constructor.rhs.getOrElse(c.parents.collectFirst { case t: Term =>
                    t
                  }.get),
                  Some(state),
                  store.typeLocations
                ),
                ir.Model
              )
            )
            val proofs =
              store.qualifiedGroups(e, e.ctx.retainedGroups(Nil, roots))
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
            val locations = RuleTypeLocations
              .model(e)(
                c.parents.collectFirst { case t: Term => t }.get,
                Some(fact),
                store.typeLocations
              )
              .types
              .map(t => t.name -> t.position)
              .toMap
            val model = ir.Model(
              functions = e.ctx.reachableFunctions(Nil, roots),
              types = e.ctx.types.values.toVector.map(t =>
                t.copy(position = locations.getOrElse(t.name, t.position))
              )
            )
            val proofs = e.ctx.retainedGroups(Nil, roots)
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
              ) ++ List(e.ctx.sourceLabels.size.toString) ++
              e.ctx.sourceLabels.toVector
                .sortBy(_._1)
                .flatMap((file, raw) => List(file, raw))
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
                val sites = store.qualifiedSites(
                  e,
                  e.ctx.retainedContextualSites(fragments)
                )
                List("derived-rule-v1", ordinary.size.toString) ++ ordinary ++
                  List(sites.size.toString) ++ sites.toList.flatMap(
                    Carrier.siteChunks
                  )
              else
                facet.fold(ordinary) { namespace =>
                  val sites = store.qualifiedSites(
                    e,
                    e.ctx.retainedContextualSites(fragments)
                  )
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
            def positionChunks(reference: Term): List[String] =
              val positions = RuleTypeLocations.model(e)(
                reference,
                cls.typeRef
                  .baseType(machineClass.asInstanceOf[Symbol])
                  .typeArgs
                  .headOption,
                store.typeLocations
              )
              Carrier.chunks(if contextual then store.qualify(e, positions, ir.Model)
              else positions)
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
              val capturedResult = e.ruleEffect(
                effect,
                stateName,
                inputNames,
                machine,
                "capture",
                tree.asInstanceOf[Tree]
              )
              val guard =
                if contextual then store.qualify(e, capturedGuard, ir.Expr)
                else capturedGuard
              val result =
                if contextual then store.qualify(e, capturedResult, ir.Expr)
                else capturedResult
              val fragments =
                Carrier.chunks(ir.ListOf(Seq(guard, result)).toByteArray)
              val capturedProofs =
                e.ctx.retainedGroups(Seq(capturedGuard, capturedResult))
              val curried =
                (if contextual then store.qualifiedGroups(e, capturedProofs)
                 else capturedProofs).map { proof =>
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
              val closure =
                Carrier.chunks(if contextual then store.qualify(e, model, ir.Model)
                else model)
              val typePositions = positionChunks(
                e.ruleBlock(rules, tree.asInstanceOf[Tree])
              )
              val ordinary = List(
                "ordinary-rule-v3",
                e.forwardedDef(effect).fold(effect.show)(_.fullName)
              ) ++ provenance ++
                List(
                  bytes(if contextual then
                    store.qualify(
                      e,
                      e.ruleSite(rules, tree.asInstanceOf[Tree]),
                      ir.Position
                    )
                  else e.ruleSite(rules, tree.asInstanceOf[Tree])),
                  bytes(if contextual then
                    store.qualify(
                      e,
                      e.ctx.pos(tree.asInstanceOf[Tree]),
                      ir.Position
                    )
                  else e.ctx.pos(tree.asInstanceOf[Tree])),
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
            val bindingOwner =
              if binding.exists && tree.tpe.widen.derivesFrom(binding.asClass)
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
            val (root, named) =
              e.stepFunctionOrigin(
                e.resolve(arg),
                tree.fun.symbol.fullName.toString,
                "capture"
              )
            val origin =
              if binding.exists && tree.tpe.widen.derivesFrom(binding.asClass)
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
                val fragments = Carrier.chunks(
                  ir.ListOf(
                    Seq(
                      guard,
                      if contextual then store.qualify(r, result, ir.Expr)
                      else result
                    )
                  ).toByteArray
                )
                val closure = Carrier.chunks(if contextual then store.qualify(r, captured, ir.Model)
                else captured)
                val positions = Carrier.chunks(
                  RuleTypeLocations.model(r)(
                    effect,
                    owner.typeRef
                      .baseType(machineClass.asInstanceOf[Symbol])
                      .typeArgs
                      .headOption,
                    store.typeLocations
                  )
                )
                def bytes(value: scalapb.GeneratedMessage): String =
                  Base64.getEncoder.encodeToString(value.toByteArray)
                val proofs = r.ctx.retainedGroups(Seq(result))
                val groups =
                  (if contextual then store.qualifiedGroups(r, proofs)
                   else proofs).toList.flatMap(p =>
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
                  val sites = store.qualifiedSites(
                    r,
                    r.ctx.retainedContextualSites(Seq(result))
                  )
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
        val located = payload.origin.fold(captured) { (owner, machine, named, position) =>
          captured
            .select(
              captured.tpe.widen.member(termName("retainOrigin")).symbol
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
        payload.ruleData.fold(located)(data =>
          located
            .select(
              located.tpe.widen.member(termName("retainDerivation")).symbol
            )
            .appliedToArgs(List(Carrier.literals(data)))
            .withSpan(tree.span)
        )

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
          "/model/framework/realize/Conversion.scala",
          "commandId",
          2,
          "java.lang.String"
        ),
        canonicalImplementation(
          "framework.realize.Conversion$package$",
          "/model/framework/realize/Conversion.scala",
          "constructedCommandToIR",
          3,
          "io.temporal.server.api.umpire.v1.Command"
        )
      )
      if framework.forall(_.nonEmpty) then conversionImplementations = framework.flatten.toSet
      val kitImplementations = Vector(
        canonicalImplementation(
          "temporal.realize.TemporalConversion",
          "/model/temporal/realize/Conversion.scala",
          "commandToIR",
          2,
          "io.temporal.server.api.umpire.v1.Command"
        ),
        canonicalImplementation(
          "temporal.realize.TemporalConversion",
          "/model/temporal/realize/Conversion.scala",
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
    describedStatusMember(tree.symbol) && tree.srcPos.sourcePos.source.file.path
      .replace('\\', '/')
      .endsWith("/model/temporal/realize/Modules.scala")

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
      source: String,
      name: String,
      parameters: Int,
      result: String
  )(using Context): Option[Symbol] =
    val methods = resultSites.iterator.collect {
      case (symbol, definition)
          if symbol.owner.fullName.toString == owner &&
            symbol.name.toString == name &&
            symbol.paramSymss.flatten.size == parameters &&
            definition.tpt.tpe.widenDealias.typeSymbol.fullName.toString == result &&
            definition.srcPos.sourcePos.source.file.path
              .replace('\\', '/')
              .endsWith(source) =>
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
    def artifact(sym: Symbol): Option[String] =
      val file = if sym.exists then sym.binaryFile else null
      if file == null then None else file.underlyingSource.map(_.path)
    val method = tree.fun.symbol
    indexResults()
    val named = getClassIfDefined("framework.realize.Named")
    val artifactOwned = named.exists && artifact(method).exists(
      artifact(named).contains
    )
    val implementationOwned = resultSites
      .get(method)
      .fold(artifactOwned): definition =>
        definition.srcPos.sourcePos.source.file.path
          .replace('\\', '/')
          .endsWith("/model/framework/realize/Scripts.scala")
    method.exists && method.is(Method) &&
    method.name.toString == "commanded" &&
    method.owner.fullName.toString == "framework.realize.Scripts$package$" &&
    method.paramSymss.flatten.size == 1 && tree.args.size == 1 &&
    implementationOwned

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
        .is(Synthetic) && !tree.symbol.owner
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
