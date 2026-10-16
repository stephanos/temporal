package framework.`export`

import java.nio.file.{Files, Path}
import java.nio.file.attribute.PosixFilePermissions
import java.util.Base64
import java.util.zip.ZipFile
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import scala.quoted.*
import scala.tasty.inspector.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Ran, Tools}
import framework.dsl.MachineEvidence
import framework.inspect.Index
import io.temporal.server.api.umpire.v1 as ir

class ConstructedEvidenceSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 30.minutes
  private val tools = Tools.here
  private val mapper = new ObjectMapper()
  private lazy val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "constructed-evidence.")
  private val stored =
    tools.directory.resolve("model/framework/export/testdata/constructedEvidence")
  private val roots = Vector(
    "Default",
    "Partial",
    "Overlap",
    "Fields",
    "Constant",
    "Ordinary",
    "Conditional",
    "Curried",
    "External",
    "NoFacts",
    "ExplicitNothing",
    "Scalar",
    "Forwarded",
    "Unmonitored",
    "Repeated",
    "Inherited",
    "TwoParams",
    "NonRootMatch",
    "RuleSource",
    "Mixed"
  )
  private val refused = mapper
    .readTree(Files.readString(stored.resolve("original-refusals.json")))
    .path("entries")
    .elements()
    .asScala
    .toVector
  private val sources = Seq("Signature.scala", "Cases.scala")
  private def recorded(label: String, result: Ran): Ran =
    Files.writeString(
      scratch.resolve(s"$label.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    Files.writeString(scratch.resolve(s"$label.stdout"), result.output): Unit
    Files.writeString(scratch.resolve(s"$label.stderr"), result.standardError): Unit
    Files.writeString(scratch.resolve(s"$label.command"), result.command.mkString(" ")): Unit
    result
  private lazy val library = ModelBuild.frameworkLibrary:
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
  private lazy val plugin = ModelBuild.reifyPlugin:
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
  private def settings(withPlugin: Boolean) = Seq(
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
  ) ++
    (if withPlugin then Seq("--scalac-option", s"-Xplugin:$plugin") else Nil) ++
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
    ).flatMap(o => Seq("--scalac-option", o))
  private def attempt(
      names: Seq[String],
      label: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil,
      changes: Map[String, String]
  ): (Path, Path, Ran) =
    val dir = Files.createDirectories(scratch.resolve(label))
    val copied = if names.contains("Native.scala") then names :+ "Projection.scala" else names
    copied.foreach { name =>
      changes.get(name) match
        case Some(source) => Files.writeString(dir.resolve(name), source): Unit
        case None         => Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    }
    val jar = scratch.resolve(s"$label.jar")
    val result = recorded(
      s"$label.compile",
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
    (dir, jar, result)
  private def compile(
      names: Seq[String],
      label: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil,
      changes: Map[String, String] = Map.empty
  ): (Path, Path, String) =
    val (dir, jar, result) = attempt(names, label, withPlugin, extra, changes)
    assertEquals(result.exit, 0, result.diagnostics)
    result.orFail(): Unit
    Files.writeString(scratch.resolve(s"$label.classpath"), result.output.trim): Unit
    (dir, jar, result.output.trim)
  private def qualified(root: String) = "fixture.constructedevidence." + root
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
  private lazy val old =
    val (dir, jar, cp) = compile(sources :+ "Refusals.scala", "old", false)
    val models = roots.map(root => root -> lifted(dir, jar, cp, root, s"old-$root")).toMap
    refused.foreach { refusal =>
      val root = refusal.path("root").asText()
      val message = refusal.path("message").asText()
      val line = refusal.path("line").asInt()
      val output = scratch.resolve(s"old-$root.json")
      val classpath = Files.writeString(scratch.resolve(s"old-$root.classpath"), cp)
      val result = recorded(
        s"old-$root",
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$jar=$dir/,$library=model/framework/",
            classpath.toString,
            output.toString,
            qualified(root)
          )
        )
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(!Files.exists(output))
      assert(result.diagnostics.contains(message), result.diagnostics)
      assert(result.diagnostics.contains(s"Refusals.scala:$line: $message"), result.diagnostics)
    }
    println(s"evidence-old-artifacts:$scratch")
    models
  private def combinedSources: String =
    val imported =
      sources.map(n => Files.readString(stored.resolve(n))).map(_.linesIterator.drop(1).toVector)
    "package fixture.constructedevidence\n" + imported.flatten
      .filter(_.startsWith("import "))
      .distinct
      .mkString("\n") + "\n" + imported.flatten.filterNot(_.startsWith("import ")).mkString("\n")
  private def editedSource: String =
    Files.readString(stored.resolve("Signature.scala")).replace("then \"one\"", "then \"edited\"")
  private lazy val variants =
    val (unitDir, unitJar, unitCp) =
      compile(
        Seq("Combined.scala"),
        "old-unit",
        false,
        changes = Map("Combined.scala" -> combinedSources)
      )
    val unit =
      roots.map(root => root -> lifted(unitDir, unitJar, unitCp, root, s"old-unit-$root")).toMap
    val (helperDir, helperJar, _) = compile(Seq("Signature.scala"), "old-helpers", false)
    val (clientDir, clientJar, clientCp) =
      compile(Seq("Cases.scala"), "old-clients", false, Seq("--jar", helperJar.toString))
    val jar = roots
      .map(root =>
        root -> lifted(
          clientDir,
          clientJar,
          clientCp,
          root,
          s"old-jar-$root",
          s",$helperJar=$helperDir/"
        )
      )
      .toMap
    val (editDir, editJar, editCp) =
      compile(sources, "old-edited", false, changes = Map("Signature.scala" -> editedSource))
    val edited =
      roots.map(root => root -> lifted(editDir, editJar, editCp, root, s"old-edited-$root")).toMap
    Map("unit" -> unit, "jar" -> jar, "edited" -> edited)
  private def withoutPositions(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      node
        .fields()
        .asScala
        .filter(_.getKey != "position")
        .foreach(f => out.set[JsonNode](f.getKey, withoutPositions(f.getValue)): Unit)
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(withoutPositions(n)): Unit)
      out
    else node
  private def comparable(model: ir.Model) = withoutPositions(
    mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model))
  )
  private def projected(model: ir.Model, root: String): ir.Model =
    val name = s"${root.head.toLower}${root.tail}"
    val machines = model.machines.filter(_.name == name)
    assertEquals(machines.size, 1, root)
    val functions = scala.collection.mutable.Set.empty[String]
    val types = scala.collection.mutable.Set.empty[String]
    def scan(node: JsonNode): Unit =
      if node.isObject then
        if node.has("call") then function(node.path("call").path("function").asText())
        if node.has("named") then tpe(node.path("named").asText())
        for kind <- Seq("construct", "enum", "record", "case") if node.path(kind).has("type") do
          tpe(node.path(kind).path("type").asText())
        node.elements().asScala.foreach(scan)
      else if node.isArray then node.elements().asScala.foreach(scan)
    def function(name: String): Unit = if functions.add(name) then
      scan(
        mapper.readTree(
          scalapb.json4s.JsonFormat.toJsonString(model.functions.find(_.name == name).get)
        )
      )
    def tpe(name: String): Unit =
      if name != "scala.Nothing" && name != "framework.dsl.Step" && types.add(name) then
        scan(
          mapper.readTree(
            scalapb.json4s.JsonFormat.toJsonString(model.types.find(_.name == name).get)
          )
        )
    machines.head.evidence match
      case ""   => ()
      case root => function(root)
    ir.Model(
      machines = Seq(ir.Machine(evidence = machines.head.evidence)),
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
        node.elements().asScala.zipWithIndex.foreach((n, i) => scan(n, s"$path[$i]"))
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
  private def native(
      cp: String,
      root: String,
      label: String,
      expected: ir.Model
  ): (ir.Model, String) =
    val oracle = Base64.getEncoder.encodeToString(expected.toByteArray)
    val result = recorded(
      label,
      tools.run(
        "java",
        Seq("-cp", cp, "framework.evidencenative.Native", qualified(root), oracle),
        Output.KeptApart
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    val (channels, holes) = references(expected)
    assert(
      result.output.linesIterator.contains("channels:" + channels.mkString(",")),
      result.output
    )
    assert(result.output.linesIterator.contains("holes:" + holes.mkString(",")), result.output)
    val encoded =
      result.output.linesIterator.find(_.startsWith("evidence:")).get.stripPrefix("evidence:")
    ir.Model.parseFrom(Base64.getDecoder.decode(encoded)) -> result.output

  private def sourceIndex(
      inputs: Vector[(Path, String)],
      cp: String,
      label: String
  ): MachineEvidence.SourceIndex =
    val dir = Files.createDirectories(scratch.resolve(s"$label-index"))
    val prefixes = scala.collection.mutable.Map.empty[String, String]
    val tastys = inputs.zipWithIndex
      .flatMap { case ((jar, prefix), index) =>
        val zip = new ZipFile(jar.toFile)
        try
          zip.entries.asScala
            .filter(_.getName.endsWith(".tasty"))
            .map { entry =>
              val name = s"$index/${entry.getName}"
              val target = dir.resolve(name)
              Files.createDirectories(target.getParent): Unit
              Files.copy(zip.getInputStream(entry), target): Unit
              prefixes(name) = prefix
              target.toString
            }
            .toVector
        finally zip.close()
      }
      .sorted
      .toList
    var result = Option.empty[MachineEvidence.SourceIndex]
    val inspector = new Inspector:
      def inspect(using quotes: Quotes)(trees: List[Tasty[quotes.type]]): Unit =
        val index = Index(trees, prefixes.toMap)
        result = Some(
          MachineEvidence.SourceIndex(index.sourceRoots.toVector.sortBy(_._1).map { (raw, prefix) =>
            val file = Path.of(prefix + raw).toAbsolutePath.normalize()
            MachineEvidence
              .IndexedSource(raw, Option.when(Files.isRegularFile(file))(file.toString))
          })
        )
    assert(
      TastyInspector.inspectAllTastyFiles(
        tastys,
        Nil,
        (inputs.map(_._1.toString) ++ cp.split(java.io.File.pathSeparator)).toList
      )(inspector)
    )
    val context = result.get
    Files.writeString(scratch.resolve(s"$label-index.txt"), encodeContext(context)): Unit
    context

  private def encodeContext(context: MachineEvidence.SourceIndex): String =
    def encoded(value: String) =
      Base64.getEncoder.encodeToString(value.getBytes(java.nio.charset.StandardCharsets.UTF_8))
    encoded(
      context.entries
        .map(e => encoded(e.rawSourceFile) + "\t" + e.resolvedFile.fold("-")(encoded))
        .mkString("\n")
    )

  private def contextModels(
      cp: String,
      root: String,
      label: String,
      first: MachineEvidence.SourceIndex,
      second: MachineEvidence.SourceIndex
  ): (ir.Model, ir.Model, ir.Model, String) =
    val result = recorded(
      label,
      tools.run(
        "java",
        Seq(
          "-cp",
          cp,
          "framework.evidencecontext.Projection",
          qualified(root),
          encodeContext(first),
          encodeContext(second)
        ),
        Output.KeptApart
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    def model(name: String): ir.Model = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        result.output.linesIterator.find(_.startsWith(s"$name:")).get.stripPrefix(s"$name:")
      )
    )
    (model("canonical"), model("first"), model("second"), result.output)

  private def sourceInputs(built: (Path, Path, String), extra: String): Vector[(Path, String)] =
    val (dir, jar, _) = built
    Vector(jar -> s"$dir/", library -> "model/framework/") ++
      extra.stripPrefix(",").split(',').filter(_.nonEmpty).map { pair =>
        val fields = pair.split("=", 2)
        Path.of(fields(0)) -> fields(1)
      }
  test("foreign and opaque admission characterization") {
    val (dir, jar, cp) =
      compile(Seq("Signature.scala", "Foreign.scala", "ForeignClients.scala"), "old-foreign", false)
    val model = lifted(dir, jar, cp, "UnmarkedEvidence", "old-UnmarkedEvidence")
    val pin = stored.resolve("original-UnmarkedEvidence.json")
    assert(Files.isRegularFile(pin), "missing frozen unmarked-helper reader authority")
    val frozen = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(pin))
    assertEquals(comparable(model), comparable(frozen))
    assertEquals(positions(model), positions(frozen))
    assert(projected(model, "UnmarkedEvidence").functions.exists(_.name.endsWith("Foreign$.text")))
    val output = scratch.resolve("old-OpaqueEvidence.json")
    val classpath = Files.writeString(scratch.resolve("old-OpaqueEvidence.classpath"), cp)
    val opaque = recorded(
      "old-OpaqueEvidence",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/,$library=model/framework/",
          classpath.toString,
          output.toString,
          qualified("OpaqueEvidence")
        )
      )
    )
    assertEquals(opaque.exit, 1, opaque.diagnostics)
    assert(!Files.exists(output))
    assert(opaque.diagnostics.contains("evidence is a function of the fact"), opaque.diagnostics)
    assert(opaque.diagnostics.contains("ForeignClients.scala:15:"), opaque.diagnostics)
    val probe = Files.createDirectories(scratch.resolve("foreign-probe"))
    Seq("Signature.scala", "Foreign.scala", "ForeignProbe.scala").foreach(name =>
      Files.copy(stored.resolve(name), probe.resolve(name)): Unit
    )
    val failure = recorded(
      "foreign-probe.compile",
      tools.scalaCli(Seq("compile", probe.toString) ++ settings(true), Output.KeptApart)
    )
    assertEquals(failure.exit, 1, failure.diagnostics)
    assert(failure.diagnostics.contains("ForeignProbe.scala:7:"), failure.diagnostics)
    assert(
      failure.diagnostics.contains("outside this capture's compilation unit"),
      failure.diagnostics
    )
    println(s"evidence-helper-authority:$scratch")
  }
  test("complete plugin-free authority and saved-tree replay") {
    def pinned(label: String, model: ir.Model): Unit =
      val pin = stored.resolve(s"original-$label.json")
      assert(Files.isRegularFile(pin), s"missing frozen complete plugin-free authority $label")
      val frozen = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(pin))
      assertEquals(comparable(model), comparable(frozen), label)
      assertEquals(positions(model), positions(frozen), label)
    old.foreach((root, model) => pinned(root, model))
    variants.foreach((variant, models) =>
      models.foreach((root, model) => pinned(s"$variant-$root", model))
    )
    assert(!comparable(variants("edited")("Ordinary")).equals(comparable(old("Ordinary"))))
    assertEquals(projected(old("NoFacts"), "NoFacts").functions.size, 0)
    assert(projected(old("ExplicitNothing"), "ExplicitNothing").functions.nonEmpty)
    val external = references(projected(old("External"), "External"))
    assert(external._1.nonEmpty && external._2.nonEmpty)
  }
  private def checkNative(
      built: (Path, Path, String),
      selected: Vector[String],
      label: String,
      authority: Map[String, ir.Model],
      canonicalAuthority: Map[String, ir.Model],
      extraSources: String = ""
  ): Map[String, ir.Model] =
    val (dir, jar, cp) = built
    val context = sourceIndex(sourceInputs(built, extraSources), cp, label)
    selected.map { root =>
      val expected = projected(authority(root), root)
      val (first, firstOutput) = native(cp, root, s"$label-$root-1", expected)
      val (second, secondOutput) = native(cp, root, s"$label-$root-2", expected)
      assertEquals(secondOutput, firstOutput, root)
      assertEquals(
        positions(first),
        positions(projected(canonicalAuthority(root), root)),
        s"canonical $root"
      )
      assertEquals(second, first, root)
      val (canonical, resolved, again, output) =
        contextModels(cp, root, s"$label-context-$root-1", context, context)
      val secondContext = contextModels(cp, root, s"$label-context-$root-2", context, context)
      assertEquals(secondContext._4, output, root)
      assertEquals(canonical, first, root)
      assertEquals(again, resolved, root)
      assertEquals(comparable(resolved), comparable(expected), root)
      assertEquals(positions(resolved), positions(expected), root)
      val replay = lifted(dir, jar, cp, root, s"$label-replay-$root", extraSources)
      assertEquals(comparable(replay), comparable(authority(root)), root)
      assertEquals(positions(replay), positions(authority(root)), root)
      root -> resolved
    }.toMap
  test("complete old-admitted evidence in two native JVMs") {
    checkNative(
      compile(sources :+ "Native.scala", "native", true),
      roots,
      "native",
      old,
      variants("jar")
    ): Unit
  }
  test("same-unit helper jars and edited sources retain complete isolated evidence") {
    val unit = compile(
      Seq("Combined.scala", "Native.scala"),
      "native-unit",
      true,
      changes = Map("Combined.scala" -> combinedSources)
    )
    checkNative(unit, roots, "native-unit", variants("unit"), variants("unit")): Unit
    val (helperDir, helperJar, _) = compile(Seq("Signature.scala"), "native-helpers", true)
    val clients = compile(
      Seq("Cases.scala", "Native.scala"),
      "native-clients",
      true,
      Seq("--jar", helperJar.toString)
    )
    checkNative(
      clients,
      roots,
      "native-jar",
      variants("jar"),
      variants("jar"),
      s",$helperJar=$helperDir/"
    ): Unit
    val a = sourceIndex(
      sourceInputs(clients, s",$helperJar=$helperDir/"),
      clients._3,
      "same-artifact-helperlast"
    )
    val b = sourceIndex(
      Vector(
        clients._2 -> s"${clients._1}/",
        helperJar -> s"$helperDir/",
        library -> "model/framework/"
      ),
      clients._3,
      "same-artifact-frameworklast"
    )
    val firstContexts = contextModels(clients._3, "External", "same-artifact-context-1", a, b)
    val secondContexts = contextModels(clients._3, "External", "same-artifact-context-2", a, b)
    assertEquals(firstContexts._4, secondContexts._4)
    assertEquals(
      positions(firstContexts._1),
      positions(projected(variants("jar")("External"), "External"))
    )
    assertEquals(
      comparable(firstContexts._2),
      comparable(projected(variants("jar")("External"), "External"))
    )
    assertEquals(
      positions(firstContexts._2),
      positions(projected(variants("jar")("External"), "External"))
    )
    assertEquals(comparable(firstContexts._3), comparable(projected(old("External"), "External")))
    assertEquals(positions(firstContexts._3), positions(projected(old("External"), "External")))
    val edited = compile(
      sources :+ "Native.scala",
      "native-edited",
      true,
      changes = Map("Signature.scala" -> editedSource)
    )
    val observed = checkNative(edited, roots, "native-edited", variants("edited"), variants("jar"))
    assertNotEquals(
      comparable(observed("Ordinary")),
      comparable(projected(old("Ordinary"), "Ordinary"))
    )
  }
  test("warm helper body edit and restore invalidate retained evidence without stale publication") {
    val dir = Files.createDirectories(scratch.resolve("warm-native"))
    val daemon = Files.createTempDirectory(Path.of("/tmp"), "evidence-warm.")
    Files.setPosixFilePermissions(daemon, PosixFilePermissions.fromString("rwx------")): Unit
    (sources ++ Seq("Native.scala", "Projection.scala")).foreach(name =>
      Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    )
    val flags = settings(true).filterNot(_ == "--server=false") ++ Seq(
      "--bloop-daemon-dir",
      daemon.toString,
      "--verbose",
      "--scalac-option",
      "-Ycheck:typer"
    )
    val jar = scratch.resolve("warm-current.jar")
    var original = Option.empty[ir.Model]
    try
      Seq(false, true, false).zipWithIndex.foreach { (edited, index) =>
        val text =
          if edited then editedSource else Files.readString(stored.resolve("Signature.scala"))
        Files.writeString(dir.resolve("Signature.scala"), text): Unit
        Files.writeString(scratch.resolve(s"warm-$index.Signature.scala"), text): Unit
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
        val saved = Files.copy(jar, scratch.resolve(s"warm-$index.jar"))
        val value = checkNative(
          (dir, saved, cp),
          Vector("Ordinary"),
          s"warm-$index",
          if edited then variants("edited") else old,
          variants("jar")
        )("Ordinary")
        if index == 0 then original = Some(value)
        if index == 1 then assertNotEquals(comparable(value), comparable(original.get))
        if index == 2 then
          assertEquals(comparable(value), comparable(original.get))
          assertEquals(positions(value), positions(original.get))
        if index > 0 then
          assert(
            result.diagnostics.contains("Signature.scala after phase typer"),
            result.diagnostics
          )
      }
    finally
      recorded(
        "warm-exit",
        tools
          .run("scala-cli", Seq("--power", "bloop", "exit", "--bloop-daemon-dir", daemon.toString))
      ).orFail(): Unit
  }
  private def isolated(file: String, root: String): String =
    var selected = true
    Files
      .readString(stored.resolve(file))
      .linesIterator
      .map { line =>
        if line.startsWith("object ") then selected = line.startsWith(s"object $root ")
        if selected then line else ""
      }
      .mkString("\n") + "\n"
  test("historically refused evidence retains compiler or prepublication refusals") {
    val compilation = Set("RecordMatch", "Stepping", "DefEvidence")
    refused.foreach { entry =>
      val root = entry.path("root").asText()
      val message = entry.path("message").asText()
      val line = entry.path("line").asInt()
      val (_, _, result) = attempt(
        sources ++ Seq("Refusals.scala", "Native.scala"),
        s"refused-$root",
        true,
        changes = Map("Refusals.scala" -> isolated("Refusals.scala", root))
      )
      if compilation(root) then
        assertEquals(result.exit, 1, result.diagnostics)
        assert(result.diagnostics.contains(s"Refusals.scala:$line:"), result.diagnostics)
        assert(result.diagnostics.contains(message), result.diagnostics)
      else
        assertEquals(result.exit, 0, result.diagnostics)
        result.orFail(): Unit
        val failure = recorded(
          s"refused-$root.native",
          tools.run(
            "java",
            Seq("-cp", result.output.trim, "framework.evidencenative.Native", qualified(root), ""),
            Output.KeptApart
          )
        )
        assertEquals(failure.exit, 1, failure.diagnostics)
        assert(failure.diagnostics.contains(message), failure.diagnostics)
        assert(
          failure.diagnostics.contains("Machine fixture.constructedevidence."),
          failure.diagnostics
        )
        assert(!failure.output.linesIterator.exists(_.startsWith("evidence:")), failure.output)
    }
    Vector(
      "UnmarkedEvidence" -> (9, "outside this capture's compilation unit"),
      "OpaqueEvidence" -> (15, "evidence is a function of the fact")
    ).foreach { (root, expected) =>
      val (_, _, result) = attempt(
        Seq("Signature.scala", "Foreign.scala", "ForeignClients.scala"),
        s"refused-$root",
        true,
        changes = Map("ForeignClients.scala" -> isolated("ForeignClients.scala", root))
      )
      assertEquals(result.exit, 1, result.diagnostics)
      assert(
        result.diagnostics.contains(s"ForeignClients.scala:${expected._1}:"),
        result.diagnostics
      )
      assert(result.diagnostics.contains(expected._2), result.diagnostics)
    }
    val (_, helperJar, _) =
      compile(Seq("Signature.scala", "Foreign.scala"), "unmarked-helper-jar", true)
    val (_, _, precompiled) = attempt(
      Seq("ForeignClients.scala"),
      "refused-precompiled-UnmarkedEvidence",
      true,
      Seq("--jar", helperJar.toString),
      Map("ForeignClients.scala" -> isolated("ForeignClients.scala", "UnmarkedEvidence"))
    )
    assertEquals(precompiled.exit, 1, precompiled.diagnostics)
    assert(precompiled.diagnostics.contains("ForeignClients.scala:9:"), precompiled.diagnostics)
    assert(
      precompiled.diagnostics.contains(
        "Foreign$.text is reached outside this capture's compilation unit"
      ),
      precompiled.diagnostics
    )
  }
