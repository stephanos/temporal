package framework.dsl

import java.util.Base64
import io.temporal.server.api.umpire.v1 as ir

private[framework] object MachineEvidence:
  final case class SourceOrigin(canonicalFile: String, rawSourceFile: String)
  final case class IndexedSource(rawSourceFile: String, resolvedFile: Option[String])
  final case class SourceIndex(entries: Vector[IndexedSource])
  final class Canonical private[MachineEvidence] (
      private val state: (
          Option[String],
          ir.Model,
          Vector[String],
          Vector[String],
          Vector[SourceOrigin]
      )
  ):
    private[MachineEvidence] def matches(
        evidence: Option[String],
        model: ir.Model,
        channels: Vector[String],
        holes: Vector[String],
        sources: Vector[SourceOrigin]
    ): Boolean = state.equals((evidence, model, channels, holes, sources))
    override def equals(other: Any): Boolean = other match
      case value: Canonical => state.equals(value.state)
      case _                => false
    override def hashCode(): Int = state.hashCode()
  final case class Contribution(
      evidence: Option[String],
      functions: Vector[ir.Function],
      types: Vector[ir.Type],
      channels: Vector[String],
      holes: Vector[String],
      sources: Vector[SourceOrigin],
      private[MachineEvidence] val canonical: Canonical
  )
  final case class Retained(
      source: RecordMachine[?, ?, ?],
      owner: String,
      family: String,
      name: String,
      factKey: String,
      fact: ir.TypeRef,
      factTypes: Vector[ir.Type],
      parameter: String,
      spelling: String,
      display: String,
      position: ir.Position,
      authored: Option[ir.Function],
      model: ir.Model,
      proofs: Vector[CoreBindings.CurriedCall],
      sources: Vector[SourceOrigin]
  )

  private def positions(model: ir.Model)(f: ir.Position => ir.Position): ir.Model =
    import scalapb.descriptors.{PMessage, PRepeated, PValue, ScalaType}
    def rewrite(value: PValue): PValue = value match
      case PMessage(fields) =>
        PMessage(fields.map { (field, child) =>
          field -> (field.scalaType match
            case ScalaType.Message(descriptor)
                if descriptor.fullName == ir.Position.scalaDescriptor.fullName =>
              child match
                case message: PMessage => f(ir.Position.messageReads.read(message)).toPMessage
                case _                 => child
            case _ => rewrite(child))
        })
      case PRepeated(values) => PRepeated(values.map(rewrite))
      case _                 => value
    ir.Model.messageReads.read(rewrite(model.toPMessage))

  private def files(model: ir.Model): Set[String] =
    val found = scala.collection.mutable.Set.empty[String]
    positions(model) { at =>
      if at.file.nonEmpty then found.add(at.file): Unit
      at
    }: Unit
    found.toSet

  private def origins(
      sources: Vector[SourceOrigin],
      required: Set[String],
      context: String
  ): Map[String, String] =
    def check(condition: Boolean, message: String): Unit =
      require(condition, s"$context: $message")
    check(
      sources.forall(s => s.canonicalFile.nonEmpty && s.rawSourceFile.nonEmpty),
      "malformed source origin"
    )
    check(
      sources.map(_.canonicalFile).distinct.size == sources.size,
      "duplicate or conflicting source origin"
    )
    val result = sources.map(s => s.canonicalFile -> s.rawSourceFile).toMap
    check(required.subsetOf(result.keySet), "missing or stale source origin")
    result

  def project(value: Contribution, index: SourceIndex): Contribution =
    val context = s"Machine evidence ${value.evidence.getOrElse("omitted")} projection"
    val model = ir.Model(functions = value.functions, types = value.types)
    val source = origins(value.sources, files(model), context)
    require(
      (value.canonical ne null) && value.canonical
        .matches(value.evidence, model, value.channels, value.holes, value.sources),
      s"$context: copied or stale canonical source provenance"
    )
    require(
      index.entries.forall(e => e.rawSourceFile.nonEmpty && e.resolvedFile.forall(_.nonEmpty)),
      s"$context: malformed source-index entry"
    )
    val selected = index.entries.map(e => e.rawSourceFile -> e.resolvedFile).toMap
    val projected = positions(model) { at =>
      if at.file.isEmpty then at
      else
        val raw = source.getOrElse(
          at.file,
          throw IllegalArgumentException(s"$context: missing source origin")
        )
        selected.getOrElse(
          raw,
          throw IllegalArgumentException(s"$context: missing source-index resolution")
        ) match
          case None       => ir.Position.defaultInstance
          case Some(file) =>
            require(file == at.file, s"$context: available different-owner source resolution")
            at
    }
    value.copy(functions = projected.functions.toVector, types = projected.types.toVector)

  final private class Cursor(data: Seq[String]):
    private var index = 0
    def next(): String =
      require(index < data.size, "incomplete retained payload")
      val value = data(index)
      index += 1
      value
    def count(): Int =
      val value = next().toInt
      require(value >= 0 && value <= data.size - index, "invalid retained payload length")
      value
    def chunks(): String = Vector.fill(count())(next()).mkString
    def done(): Unit = require(index == data.size, "unrelated retained payload fields")

  private def read[A](text: String)(decode: Array[Byte] => A): A =
    decode(Base64.getDecoder.decode(text))

  final class Metadata:
    private var retained = Option.empty[Retained]
    def value: Option[Retained] = synchronized(retained)
    def owns(value: Retained): Boolean = synchronized(retained.exists(_.eq(value)))
    def retain(source: RecordMachine[?, ?, ?], data: Seq[String]): Unit = synchronized {
      val payload = try
        val c = new Cursor(data)
        require(c.next() == "machine-evidence-v2", "unsupported evidence version")
        val owner = c.next()
        val family = c.next()
        val name = c.next()
        val key = c.next()
        val fact = read(c.next())(ir.TypeRef.parseFrom)
        val schema = read(c.chunks())(ir.Model.parseFrom)
        require(schema.copy(types = Nil).equals(ir.Model()), "unrelated Fact schema declarations")
        val parameter = c.next()
        val spelling = c.next()
        val display = c.next()
        val position = read(c.next())(ir.Position.parseFrom)
        val authored = Option(c.next()).filter(_.nonEmpty).map(read(_)(ir.Function.parseFrom))
        val model = read(c.chunks())(ir.Model.parseFrom)
        val proofs = Vector.fill(c.count()) {
          CoreBindings.CurriedCall(
            read(c.next())(ir.Call.parseFrom),
            read(c.next())(ir.Function.parseFrom),
            c.next().split(',').filter(_.nonEmpty).map(_.toInt).toVector,
            Option(c.next()).filter(_.nonEmpty)
          )
        }
        val sources = Vector.fill(c.count())(SourceOrigin(c.next(), c.next()))
        c.done()
        Retained(
          source,
          owner,
          family,
          name,
          key,
          fact,
          schema.types.toVector,
          parameter,
          spelling,
          display,
          position,
          authored,
          model,
          proofs,
          sources
        )
      catch
        case scala.util.control.NonFatal(error) =>
          throw IllegalArgumentException(
            s"Machine evidence: malformed retained payload: ${error.getMessage}",
            error
          )
      checked(source, payload)
      require(
        retained.forall(_.equals(payload)),
        s"Machine ${payload.family}.${payload.name} evidence: conflicting retained provenance or payload"
      )
      if retained.isEmpty then retained = Some(payload)
    }

  private def checked(source: RecordMachine[?, ?, ?], value: Retained): Unit =
    val context = s"Machine ${value.family}.${value.name} evidence"
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    def check(condition: Boolean, message: String): Unit = if !condition then fail(message)
    check(value.source.eq(source), "stale declaring source")
    check(value.owner.nonEmpty && value.name.nonEmpty, "missing declaring identity")
    check(source.getClass.getName == value.owner.replace("$.", "$"), "stale declaring owner")
    check(
      source.headerIdentity.contains(value.family -> value.name),
      "missing or stale declaration registration"
    )
    check(
      source.headerSchemaKeys.lift(2).flatten.contains(value.factKey),
      "stale selected Fact key"
    )
    val schema = source.ff.schemaFor(value.factKey).getOrElse(fail("missing selected Fact schema"))
    check(
      schema.identity == value.factKey && !schema.root.ref.isEmpty && schema.root.equals(
        value.fact
      ),
      "stale selected Fact identity/type"
    )
    check(
      value.factTypes.map(_.name).distinct.size == value.factTypes.size,
      "duplicate retained Fact type"
    )
    check(
      schema.types
        .map(_.clearPosition)
        .sortBy(_.name)
        .equals(value.factTypes.map(_.clearPosition).sortBy(_.name)),
      "stale selected Fact type closure"
    )
    check(
      value.parameter.nonEmpty && value.spelling.nonEmpty && value.display.nonEmpty,
      "missing typed evidence naming"
    )
    check(
      value.model.copy(functions = Nil, types = Nil).equals(ir.Model()),
      "unrelated captured declarations"
    )
    origins(
      value.sources,
      files(value.model) ++ Option(value.position.file).filter(_.nonEmpty),
      context
    ): Unit
    val root = s"${value.name}.evidence"
    value.authored match
      case Some(fn) =>
        check(fn.name == root, "stale evidence root name")
        check(
          value.model.functions.count(_.name == root) == 1 && value.model.functions
            .find(_.name == root)
            .exists(_.equals(fn)),
          "stale authored evidence signature/body"
        )
      case None =>
        check(
          value.model.functions.isEmpty && value.proofs.isEmpty,
          "unrelated omitted evidence functions"
        )
    val factScope = ir.Function(
      "$selected-evidence-fact-schema",
      params = Seq(ir.Param("fact", Some(value.fact))),
      body = Some(ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Text("")))))
    )
    CoreBindings.dependencies(
      Nil,
      Seq(factScope.name),
      Seq(factScope),
      value.factTypes,
      context
    ): Unit
    val capturedScope = ir.Function(
      "$captured-evidence-types",
      params = value.model.types.zipWithIndex.map { (tpe, index) =>
        ir.Param(s"type$index", Some(ir.TypeRef(ir.TypeRef.Ref.Named(tpe.name))))
      },
      body = Some(ir.Expr(kind = ir.Expr.Kind.Literal(ir.Value(ir.Value.Kind.Text("")))))
    )
    check(
      !value.model.functions.exists(_.name == capturedScope.name),
      "unrelated captured function"
    )
    val captured = CoreBindings.dependencies(
      Nil,
      value.authored.map(_.name).toSeq :+ capturedScope.name,
      value.model.functions :+ capturedScope,
      value.model.types,
      context,
      value.proofs
    )
    check(
      captured.functions.filterNot(_.name == capturedScope.name).map(_.name).sorted ==
        value.model.functions.map(_.name).sorted,
      "unrelated captured functions"
    )
    value.factTypes.foreach { expected =>
      check(
        value.model.types
          .find(_.name == expected.name)
          .exists(_.clearPosition.equals(expected.clearPosition)),
        s"captured Fact type ${expected.name} disagrees with selected schema"
      )
    }

  def collect(machine: RecordMachine[?, ?, ?]): Contribution =
    require(machine ne null, "Machine evidence: declaration is not initialized")
    def declaring(current: RecordMachine[?, ?, ?]): RecordMachine[?, ?, ?] = current match
      case _: Derived[?, ?, ?] | _: Built[?, ?, ?, ?] =>
        current.derivationParent.fold(current)(declaring)
      case _ => current
    val source = machine.evidenceSource
    require(source ne null, "Machine evidence: declaring source is not initialized")
    require(source.eq(declaring(machine)), "Machine evidence: stale declaring-source forwarding")
    val (family, name) = source.headerIdentity.getOrElse(
      throw IllegalArgumentException(
        "Machine evidence: declaration registration is pending or absent"
      )
    )
    val context = s"Machine $family.$name evidence"
    def fail(message: String): Nothing = throw IllegalArgumentException(s"$context: $message")
    val value = source.evidenceMetadata.getOrElse(fail("no retained evidence metadata"))
    checked(source, value)
    if !source.ownsEvidence(value) then fail("copied or unpublished evidence provenance")
    def cases: Seq[(String, Boolean)] =
      value.model.types
        .find(_.name == value.fact.getNamed)
        .filter(_.shape.isEnum)
        .map(_.getEnum.cases.map(c => c.name -> c.fields.nonEmpty))
        .getOrElse(
          fail(
            s"${value.display} is no enum, so its facts have no names to default their evidence to"
          )
        )
    val function = value.authored match
      case Some(fn) => Some(EvidenceLowering.complete(fn, value.fact, cases, value.position, fail))
      case None if value.factKey == "scala.Nothing" || value.fact.getNamed.isEmpty => None
      case None                                                                    =>
        Some(
          EvidenceLowering.defaultEvidence(
            s"$name.evidence",
            value.fact,
            value.parameter,
            value.spelling,
            value.position,
            cases,
            fail
          )
        )
    val functions =
      value.model.functions.map(f => function.filter(_.name == f.name).getOrElse(f)) ++
        function.filter(fn => !value.model.functions.exists(_.name == fn.name))
    val closure = CoreBindings.dependencies(
      Nil,
      function.map(_.name).toSeq,
      functions,
      value.model.types,
      context,
      value.proofs
    )
    Contribution(
      function.map(_.name),
      closure.functions,
      closure.types,
      closure.channels,
      closure.holes,
      value.sources,
      new Canonical(
        (
          function.map(_.name),
          ir.Model(functions = closure.functions, types = closure.types),
          closure.channels,
          closure.holes,
          value.sources
        )
      )
    )
