package framework.`export`

import java.nio.file.{Files, Path}
import java.nio.file.attribute.PosixFilePermissions
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Ran, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedDerivationsSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 30.minutes
  private val tools = Tools.here
  private val mapper = new ObjectMapper()
  private lazy val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "constructed-derivations.")
  private val stored =
    tools.directory.resolve("model/framework/export/testdata/constructedDerivations")
  private lazy val primitiveCorrections = mapper
    .readTree(
      Files.readString(stored.resolve("primitive-state-corrections.json"))
    )
    .path("entries")
    .elements()
    .asScala
    .toVector
  private val booleanType = ir.TypeRef(ir.TypeRef.Ref.Bool(com.google.protobuf.empty.Empty()))
  private val malformedPrimitive = ir.TypeRef(ir.TypeRef.Ref.Named(""))

  private def correctedPrimitiveState(model: ir.Model, label: String): ir.Model =
    val entries = primitiveCorrections.filter(_.path("model").asText() == s"original-$label.json")
    val inventoried = entries.map(_.path("function").asText())
    assertEquals(inventoried.distinct.size, inventoried.size, label)
    assertEquals(
      model.functions
        .filter(_.params.headOption.exists(_.getType.equals(malformedPrimitive)))
        .map(_.name)
        .toSet,
      inventoried.toSet,
      s"uninventoried primitive-state defect in $label"
    )
    entries.foreach { entry =>
      assertEquals(entry.path("parameter").asInt(), 0)
      assertEquals(
        scalapb.json4s.JsonFormat.fromJsonString[ir.TypeRef](entry.path("oldType").toString),
        malformedPrimitive
      )
      assertEquals(
        scalapb.json4s.JsonFormat.fromJsonString[ir.TypeRef](entry.path("correctType").toString),
        booleanType
      )
      val source = entry.path("source")
      val declaration = Files
        .readString(stored.resolve(source.path("file").asText()))
        .linesIterator
        .toVector(source.path("line").asInt() - 1)
      assertEquals(declaration, source.path("declaration").asText())
      assert(declaration.contains("extends Machine[Boolean, Boolean, Nothing]"), declaration)
      val typed = model.functions.find(_.name == source.path("typedFunction").asText()).get
      assertEquals(
        typed.params.head.getType,
        booleanType,
        s"independent source type witness for $label"
      )
    }
    model.copy(functions =
      model.functions.map(f =>
        if inventoried.contains(f.name) then
          f.copy(params = f.params.updated(0, f.params.head.copy(`type` = Some(booleanType))))
        else f
      )
    )
  private lazy val library =
    val jar = scratch.resolve("framework.jar")
    recorded(
      "framework",
      tools.scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/dsl",
          "model/framework/realize",
          "--server=false",
          "--library",
          "-f",
          "-o",
          jar.toString
        )
      )
    ).orFail(): Unit
    jar
  private lazy val plugin =
    val jar = scratch.resolve("reify.jar")
    recorded(
      "plugin",
      tools.scalaCli(
        Seq(
          "--power",
          "package",
          "model/framework/reify/plugin",
          "--server=false",
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "-f",
          "-o",
          jar.toString
        )
      )
    ).orFail(): Unit
    jar
  private def settings(withPlugin: Boolean): Seq[String] = Seq(
    "--scala",
    "3.9.0",
    "--jvm",
    "27",
    "--server=false",
    "--dependency",
    "com.thesamet.scalapb::scalapb-runtime-grpc:0.11.20",
    "--jar",
    library.toString,
    "--jar",
    tools.directory.resolve("model/build/ir-scalapb.jar").toString
  ) ++ (if withPlugin then Seq("--scalac-option", s"-Xplugin:$plugin") else Nil) ++
    Seq(
      "-Werror",
      "-deprecation",
      "-feature",
      "-unchecked",
      "-Wunused:all",
      "-Wvalue-discard",
      "-Wnonunit-statement",
      "-Wsafe-init",
      "-Wimplausible-patterns",
      "-language:strictEquality"
    )
      .flatMap(o => Seq("--scalac-option", o))
  private def recorded(name: String, result: Ran): Ran =
    Files.writeString(
      scratch.resolve(s"$name.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    Files.writeString(scratch.resolve(s"$name.stdout"), result.output): Unit
    Files.writeString(scratch.resolve(s"$name.stderr"), result.standardError): Unit
    Files.writeString(scratch.resolve(s"$name.command"), result.command.mkString(" ")): Unit
    result
  private def compile(
      names: Seq[String],
      name: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil,
      changes: Map[String, String] = Map.empty
  ): (Path, Path, String) =
    val dir = Files.createDirectories(scratch.resolve(name))
    names.foreach { n =>
      changes.get(n) match
        case Some(source) => Files.writeString(dir.resolve(n), source): Unit
        case None         => Files.copy(stored.resolve(n), dir.resolve(n)): Unit
    }
    val jar = scratch.resolve(s"$name.jar")
    val result = recorded(
      s"$name.compile",
      tools.scalaCli(
        Seq(
          "compile",
          dir.toString,
          "--print-class-path",
          "--compilation-output",
          jar.toString
        ) ++ settings(withPlugin) ++ extra,
        Output.KeptApart
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    result.orFail(): Unit
    Files.writeString(scratch.resolve(s"$name.classpath"), result.output.trim): Unit
    (dir, jar, result.output.trim)
  private val sources = Seq(
    "Shared.scala",
    "Rules.scala",
    "External.scala",
    "Faults.scala",
    "FacetSignature.scala",
    "FacetDefinition.scala",
    "FacetClients.scala"
  )
  private val roots = Vector(
    "SameNamed",
    "external.Rebound",
    "external.Group",
    "faults.Rebound",
    "faults.Extended",
    "Base",
    "Classes",
    "Disabled",
    "Core",
    "Forwarded",
    "Distinct",
    "SameLambda",
    "DisabledRebound",
    "ClassRebound",
    "MixedClassRebound",
    "Partial",
    "Replaced",
    "Extended",
    "EmptyOperations",
    "RestrictedEmpty",
    "Repeated",
    "GroupThenBare",
    "BareThenGroup",
    "DerivedAgain",
    "RestrictedReplaced",
    "ExtendRestrict",
    "RestrictedExtend",
    "CoreReplaced",
    "CoreExtended",
    "Mixed",
    "Arities",
    "Curried",
    "CoreAnonymous"
  ) ++ Vector(
    "A",
    "B",
    "ARebound",
    "BRebound",
    "AReplaced",
    "BReplaced",
    "AExtended",
    "BExtended",
    "ADisabledRebound",
    "BDisabledRebound",
    "ATimerRebound",
    "BTimerRebound"
  ).map("facets." + _)
  private def qualified(root: String): String =
    if root.startsWith("facets.") then "fixture.derivationfacets." + root.stripPrefix("facets.")
    else if root.startsWith("external.") then
      "fixture.derivationexternal." + root.stripPrefix("external.")
    else if root.startsWith("faults.") then "fixture.derivationfault." + root.stripPrefix("faults.")
    else "fixture.constructedderivations." + root
  private val refused = Vector(
    "EmptyGroup" -> "not a case",
    "ClassMerge" -> "different effects",
    "facets.PhaseBearing" -> "a derivation's case",
    "DerivedAfterReplaced" -> "different effects",
    "DistinctNamed" -> "different effects",
    "HiddenInvalid" -> "different effects",
    "OverwrittenInvalid" -> "different effects",
    "DisabledExtended" -> "bare binding",
    "RestrictedBareExtended" -> "bare binding",
    "DuplicateReplace" -> "twice",
    "DuplicateExtend" -> "twice",
    "UnknownReplace" -> "does not bind",
    "AlreadyBound" -> "binds already",
    "HiddenUnknown" -> "does not bind",
    "Anonymous" -> "call of a def"
  )
  private def lifted(
      dir: Path,
      jar: Path,
      cp: String,
      root: String,
      label: String,
      extraSources: String = ""
  ): ir.Model =
    val path = scratch.resolve(s"$label.json")
    val classpath = Files.writeString(scratch.resolve(s"$label.classpath"), cp)
    recorded(
      label,
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/,$library=model/framework/$extraSources",
          classpath.toString,
          path.toString,
          qualified(root)
        )
      )
    ).orFail(): Unit
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(path))
  private lazy val oldResult = scala.util.Try {
    val (dir, jar, cp) =
      compile(sources ++ Seq("Refusals.scala", "FacetRefusals.scala"), "old", false)
    val models = roots.map(root => root -> lifted(dir, jar, cp, root, s"old-$root")).toMap
    refused.foreach { (root, message) =>
      val path = scratch.resolve(s"old-$root.json")
      val classpath = Files.writeString(scratch.resolve(s"old-$root.classpath"), cp)
      val result = recorded(
        s"old-$root",
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$jar=$dir/,$library=model/framework/",
            classpath.toString,
            path.toString,
            qualified(root)
          )
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(!Files.exists(path))
      val file = if root.startsWith("facets.") then "FacetRefusals.scala" else "Refusals.scala"
      val lines = Files.readString(dir.resolve(file)).linesIterator.toVector
      val start = lines.indexWhere(_.startsWith("object " + root.split('.').last))
      val body =
        lines.drop(start).takeWhile(line => line == lines(start) || !line.startsWith("object "))
      val target = if root == "EmptyGroup" then "on(" else "~>"
      val line = start + body.indexWhere(_.contains(target)) + 1
      assert(result.diagnostics.contains(s"$file:$line:"), result.diagnostics)
      assert(result.diagnostics.contains(message), result.diagnostics)
    }
    println(s"derivations-old-artifacts:$scratch")
    models
  }
  private def old = oldResult.get
  private def combinedSources: String =
    sources
      .map(n => Files.readString(stored.resolve(n)))
      .groupBy(_.takeWhile(_ != '\n'))
      .toVector
      .sortBy(_._1)
      .map { (pkg, bodies) =>
        val lines = bodies.flatMap(_.linesIterator.drop(1))
        val imports = lines.filter(_.startsWith("import ")).distinct
        val body = lines
          .filterNot(_.startsWith("import "))
          .mkString("\n")
          .replace("val open = is(", "val open: State => Boolean = is(")
        pkg + " {\n" + imports.mkString("\n") + "\n" + body + "\n}\n"
      }
      .mkString
  private lazy val variants =
    val combined = combinedSources
    val (unitDir, unitJar, unitCp) =
      compile(Seq("Combined.scala"), "old-unit", false, changes = Map("Combined.scala" -> combined))
    val unit =
      roots.map(root => root -> lifted(unitDir, unitJar, unitCp, root, s"old-unit-$root")).toMap
    val facetRoots = roots.filter(_.startsWith("facets."))
    val (definitionDir, definitionJar, _) =
      compile(Seq("FacetSignature.scala", "FacetDefinition.scala"), "old-definitions", false)
    val (clientDir, clientJar, clientCp) =
      compile(Seq("FacetClients.scala"), "old-clients", false, Seq("--jar", definitionJar.toString))
    val jar = facetRoots
      .map(root =>
        root -> lifted(
          clientDir,
          clientJar,
          clientCp,
          root,
          s"old-jar-$root",
          s",$definitionJar=$definitionDir/"
        )
      )
      .toMap
    val (helperDir, helperJar, _) = compile(Seq("Shared.scala"), "old-helpers", false)
    val (ordinaryDir, ordinaryJar, ordinaryCp) =
      compile(Seq("Rules.scala"), "old-ordinary", false, Seq("--jar", helperJar.toString))
    val helpers = roots
      .filterNot(_.contains('.'))
      .filterNot(root => Set("Base", "Classes", "Disabled", "Core", "Forwarded", "Distinct")(root))
      .map(root =>
        root -> lifted(
          ordinaryDir,
          ordinaryJar,
          ordinaryCp,
          root,
          s"old-helper-$root",
          s",$helperJar=$helperDir/"
        )
      )
      .toMap
    val edited = Files
      .readString(stored.resolve("FacetDefinition.scala"))
      .replace("enters(Phase.done)", "enters(Phase.open)")
    val (editDir, editJar, editCp) =
      compile(sources, "old-edited", false, changes = Map("FacetDefinition.scala" -> edited))
    val warm = facetRoots
      .map(root => root -> lifted(editDir, editJar, editCp, root, s"old-edited-$root"))
      .toMap
    Map("unit" -> unit, "jar" -> jar, "helper" -> helpers, "edited" -> warm)
  private def withoutPositions(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      for field <- node.fields().asScala if field.getKey != "position" do
        out.set[JsonNode](field.getKey, withoutPositions(field.getValue)): Unit
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(withoutPositions(n)): Unit)
      out
    else node
  private def comparable(model: ir.Model): JsonNode =
    withoutPositions(mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model)))
  private def projected(model: ir.Model, root: String): ir.Model =
    val simple = root.split('.').last
    val selected = model.machines.filter(_.name == s"${simple.head.toLower}${simple.tail}")
    assertEquals(selected.size, 1, root)
    val functions = scala.collection.mutable.Set.empty[String]
    val types = scala.collection.mutable.Set.empty[String]
    def read(value: scalapb.GeneratedMessage): Unit =
      scan(mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(value)))
    def function(name: String): Unit =
      if functions.add(name) then read(model.functions.find(_.name == name).get)
    def tpe(name: String): Unit =
      if name != "scala.Nothing" && name != "framework.dsl.Step" && types.add(name) then
        read(model.types.find(_.name == name).get)
    def scan(node: JsonNode): Unit =
      if node.isObject then
        if node.has("call") then function(node.path("call").path("function").asText())
        if node.has("named") then tpe(node.path("named").asText())
        for kind <- Seq("construct", "enum", "record", "case") if node.path(kind).has("type") do
          tpe(node.path(kind).path("type").asText())
        node.elements().asScala.foreach(scan)
      else if node.isArray then node.elements().asScala.foreach(scan)
    selected.flatMap(_.steps).foreach(s => function(s.function))
    ir.Model(
      machines = selected.map(m => ir.Machine(name = m.name, steps = m.steps)).sortBy(_.name),
      functions = model.functions.filter(f => functions(f.name)).sortBy(_.name),
      types = model.types.filter(t => types(t.name)).sortBy(_.name)
    )
  private def positions(model: ir.Model): Map[String, (String, Int)] =
    val result = scala.collection.mutable.Map.empty[String, (String, Int)]
    def scan(node: JsonNode, path: String): Unit =
      if node.isObject then
        if node.has("position") then
          val p = node.path("position")
          result(path) =
            Path.of(p.path("file").asText()).getFileName.toString -> p.path("line").asInt()
        node
          .fields()
          .asScala
          .filter(_.getKey != "position")
          .foreach(f => scan(f.getValue, s"$path.${f.getKey}"))
      else if node.isArray then
        node
          .elements()
          .asScala
          .zipWithIndex
          .foreach((n, i) => scan(n, s"$path[$i]"))
    scan(mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model)), "model")
    result.toMap
  private def references(model: ir.Model): (Vector[String], Vector[String]) =
    val channels = scala.collection.mutable.Set.empty[String]
    val holes = scala.collection.mutable.Set.empty[String]
    def scan(node: JsonNode): Unit =
      if node.isObject then
        if node.path("channel").isTextual then channels += node.path("channel").asText(): Unit
        if node.path("hole").isTextual then holes += node.path("hole").asText(): Unit
        node.elements().asScala.foreach(scan)
      else if node.isArray then node.elements().asScala.foreach(scan)
    scan(mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model)))
    channels.toVector.sorted -> holes.toVector.sorted

  test("independent plugin-free derivation Models and located refusals are frozen before capture"):
    assertEquals(old.size, roots.size)
    def pinned(label: String, model: ir.Model): Unit =
      val frozen = stored.resolve(s"original-$label.json")
      assert(Files.isRegularFile(frozen), s"missing frozen plugin-free authority $label")
      val value = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(frozen))
      assertEquals(comparable(model), comparable(value), label)
      assertEquals(positions(model), positions(value), label)
      correctedPrimitiveState(model, label): Unit
    assertEquals(primitiveCorrections.size, 79)
    assertEquals(primitiveCorrections.map(_.path("model").asText()).distinct.size, 61)
    assertEquals(primitiveCorrections.map(_.path("function").asText()).distinct.size, 26)
    roots.foreach { root =>
      pinned(root, old(root))
    }
    assertEquals(variants.keySet, Set("unit", "jar", "helper", "edited"))
    variants.foreach { (variant, models) =>
      models.foreach((root, model) => pinned(s"$variant-$root", model))
    }
    assert(
      !comparable(variants("edited")("facets.ARebound")).equals(comparable(old("facets.ARebound")))
    )

  private def checkNative(
      built: (Path, Path, String),
      selected: Vector[String],
      label: String,
      authority: Map[String, ir.Model],
      oraclePrefix: String = "",
      extraSources: String = "",
      callbacksFile: String = "original-runtime.json"
  ): Map[String, ir.Model] =
    val (dir, jar, cp) = built
    selected.map { root =>
      val artifact = s"$label-$root"
      val oracle = s"$oraclePrefix$root"
      val first = recorded(
        s"$artifact.first",
        tools.run(
          "java",
          Seq("-cp", cp, "framework.derivationsnative.Native", qualified(root)),
          Output.KeptApart
        )
      )
      val second = recorded(
        s"$artifact.second",
        tools.run(
          "java",
          Seq("-cp", cp, "framework.derivationsnative.Native", qualified(root)),
          Output.KeptApart
        )
      )
      assertEquals(first.exit, 0, first.diagnostics)
      assertEquals(second.exit, 0, second.diagnostics)
      assertEquals(first.output, second.output)
      val value = ir.Model.parseFrom(
        Base64.getDecoder.decode(
          first.output.linesIterator
            .find(_.startsWith("derivations:"))
            .get
            .stripPrefix("derivations:")
        )
      )
      Files.writeString(
        scratch.resolve(s"$artifact.json"),
        scalapb.json4s.JsonFormat.toJsonString(value)
      ): Unit
      val expected = projected(correctedPrimitiveState(authority(root), oracle), root)
      val corrections = primitiveCorrections.filter(e =>
        e.path("model").asText() == s"original-$oracle.json" &&
          expected.functions.exists(_.name == e.path("function").asText())
      )
      corrections.foreach(e =>
        assertEquals(
          value.functions.find(_.name == e.path("function").asText()).get.params.head.getType,
          booleanType,
          s"explicit primitive-state repair $root"
        )
      )
      Files.writeString(
        scratch.resolve(s"$artifact.primitive-correction.receipt.json"),
        mapper.writeValueAsString(corrections.asJava)
      ): Unit
      val (channels, holes) = references(expected)
      assert(first.output.linesIterator.contains("channels:" + channels.mkString(",")), root)
      assert(first.output.linesIterator.contains("holes:" + holes.mkString(",")), root)
      if root.startsWith("external.") then assert(channels.nonEmpty && holes.nonEmpty, root)
      assertEquals(comparable(value), comparable(expected), root)
      assertEquals(positions(value), positions(expected), root)
      val replay = lifted(dir, jar, cp, root, s"$artifact.replay", extraSources)
      assertEquals(comparable(replay), comparable(authority(root)), root)
      assertEquals(positions(replay), positions(authority(root)), root)
      if !root.startsWith("external.") then
        val observed = recorded(
          s"$artifact.runtime",
          tools.run(
            "java",
            Seq("-cp", cp, "framework.derivationsnative.Runtime", qualified(root)),
            Output.KeptApart
          )
        )
        assertEquals(observed.exit, 0, observed.diagnostics)
        val callbacks = mapper.readTree(Files.readString(stored.resolve(callbacksFile)))
        assert(callbacks.has(root), s"missing independent native callback authority $root")
        assertEquals(observed.output, callbacks.path(root).asText(), root)
      root -> value
    }.toMap

  test("old-admitted ordinary derivations construct complete contributions in two native JVMs"):
    checkNative(
      compile(
        sources.filterNot(_.startsWith("Facet")) ++ Seq("Native.scala", "Runtime.scala"),
        "native-ordinary",
        true
      ),
      roots.filterNot(_.startsWith("facets.")),
      "native-ordinary",
      old
    ): Unit

  test("old-admitted Facets derivations construct complete contributions in two native JVMs"):
    checkNative(
      compile(
        sources.filter(_.startsWith("Facet")) ++ Seq("Native.scala", "Runtime.scala"),
        "native-facets",
        true
      ),
      roots.filter(_.startsWith("facets.")),
      "native-facets",
      old
    ): Unit

  test("same-unit and precompiled facet/helper derivations retain complete isolated closure"):
    checkNative(
      compile(
        Seq("Combined.scala", "Native.scala", "Runtime.scala"),
        "native-unit",
        true,
        changes = Map("Combined.scala" -> combinedSources)
      ),
      roots,
      "native-unit",
      variants("unit"),
      "unit-"
    ): Unit
    val (definitionDir, definitionJar, _) =
      compile(Seq("FacetSignature.scala", "FacetDefinition.scala"), "native-definitions", true)
    checkNative(
      compile(
        Seq("FacetClients.scala", "Native.scala", "Runtime.scala"),
        "native-clients",
        true,
        Seq("--jar", definitionJar.toString)
      ),
      roots.filter(variants("jar").contains),
      "native-jar",
      variants("jar"),
      "jar-",
      s",$definitionJar=$definitionDir/"
    ): Unit
    val (helperDir, helperJar, _) = compile(Seq("Shared.scala"), "native-helpers", true)
    checkNative(
      compile(
        Seq("Rules.scala", "Native.scala", "Runtime.scala"),
        "native-helper-clients",
        true,
        Seq("--jar", helperJar.toString)
      ),
      roots.filter(variants("helper").contains),
      "native-helper",
      variants("helper"),
      "helper-",
      s",$helperJar=$helperDir/"
    ): Unit

  test("refused derivation histories remain reached before later overwrite or restriction"):
    val originalRefusals = Files.readString(stored.resolve("Refusals.scala"))
    val runtimeRefusals =
      originalRefusals.linesIterator.takeWhile(_ != "object Anonymous").mkString("\n") + "\n"
    val (_, _, cp) = compile(
      sources.filterNot(_.startsWith("Facet")) ++
        Seq("Refusals.scala", "Native.scala", "Runtime.scala"),
      "native-refusals",
      true,
      changes = Map("Refusals.scala" -> runtimeRefusals)
    )
    refused.filterNot(p => p._1.startsWith("facets.") || p._1 == "Anonymous").foreach {
      (root, oldReason) =>
        val result = recorded(
          s"native-refused-$root",
          tools.run(
            "java",
            Seq("-cp", cp, "framework.derivationsnative.Native", qualified(root)),
            Output.KeptApart
          )
        )
        assertNotEquals(result.exit, 0, result.diagnostics)
        val reason = if root == "EmptyGroup" then "missing captured group provenance" else oldReason
        assert(result.diagnostics.contains(reason), result.diagnostics)
        assert(result.diagnostics.contains("Machine derivation Rules"), result.diagnostics)
    }
    Seq(
      (
        "native-phase-refusal",
        sources.filter(_.startsWith("Facet")) :+ "FacetRefusals.scala",
        "FacetRefusals.scala",
        "when(",
        "a derivation's case"
      ),
      (
        "native-anonymous-refusal",
        sources.filterNot(_.startsWith("Facet")) :+ "Refusals.scala",
        "Refusals.scala",
        "List(Step(false",
        "call of a def"
      )
    ).foreach { (label, names, file, token, reason) =>
      val dir = Files.createDirectories(scratch.resolve(label))
      names.foreach(n => Files.copy(stored.resolve(n), dir.resolve(n)): Unit)
      val result = recorded(
        label,
        tools.scalaCli(Seq("compile", dir.toString) ++ settings(true), Output.KeptApart)
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains(reason), result.diagnostics)
      val line = Files
        .readString(stored.resolve(file))
        .linesIterator
        .toVector
        .indexWhere(_.contains(token)) + 1
      assert(result.diagnostics.contains(s"$file:$line:"), result.diagnostics)
    }

  test("warm facet body edits and restoration invalidate derived helper captures"):
    val dir = Files.createDirectories(scratch.resolve("warm-native"))
    val daemon = Files.createTempDirectory(Path.of("/tmp"), "derivations-warm.")
    Files.setPosixFilePermissions(daemon, PosixFilePermissions.fromString("rwx------")): Unit
    (sources.filter(_.startsWith("Facet")) ++ Seq("Native.scala", "Runtime.scala")).foreach(n =>
      Files.copy(stored.resolve(n), dir.resolve(n)): Unit
    )
    val flags = settings(true).filterNot(_ == "--server=false") ++ Seq(
      "--bloop-daemon-dir",
      daemon.toString,
      "--verbose",
      "--scalac-option",
      "-Ycheck:typer"
    )
    val jar = scratch.resolve("warm-current.jar")
    var original = Map.empty[String, ir.Model]
    try
      Seq(false, true, false).zipWithIndex.foreach { (edited, index) =>
        val source = Files.readString(stored.resolve("FacetDefinition.scala"))
        val text =
          if edited then source.replace("enters(Phase.done)", "enters(Phase.open)") else source
        Files.writeString(dir.resolve("FacetDefinition.scala"), text): Unit
        Files.writeString(scratch.resolve(s"warm-$index.FacetDefinition.scala"), text): Unit
        val result = recorded(
          s"warm-$index.compile",
          tools.scalaCli(
            Seq(
              "compile",
              dir.toString,
              "--print-class-path",
              "--compilation-output",
              jar.toString
            ) ++ flags,
            Output.KeptApart
          )
        )
        assertEquals(result.exit, 0, result.diagnostics)
        result.orFail(): Unit
        val cp = result.output.linesIterator.filter(_.nonEmpty).toVector.last
        val retainedJar = Files.copy(jar, scratch.resolve(s"warm-$index.jar"))
        val observed = checkNative(
          (dir, retainedJar, cp),
          roots.filter(_.startsWith("facets.")),
          s"warm-$index",
          if edited then variants("edited") else old,
          if edited then "edited-" else "",
          callbacksFile =
            if edited then "original-edited-runtime.json" else "original-runtime.json"
        )
        if index == 0 then original = observed
        if index == 1 then
          Seq("facets.ARebound", "facets.BRebound").foreach(root =>
            assertNotEquals(comparable(observed(root)), comparable(original(root)), root)
          )
        if index == 2 then
          observed.foreach { (root, model) =>
            assertEquals(comparable(model), comparable(original(root)), root)
            assertEquals(positions(model), positions(original(root)), root)
          }
        if index > 0 then
          assert(
            result.diagnostics.contains("FacetDefinition.scala after phase typer"),
            result.diagnostics
          )
      }
    finally
      recorded(
        "warm-exit",
        tools
          .run("scala-cli", Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString))
      ).orFail(): Unit
