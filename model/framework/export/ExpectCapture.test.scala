package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import com.fasterxml.jackson.databind.node.ObjectNode
import framework.check.{Output, Ran, Tools}
import io.temporal.server.api.umpire.v1 as ir

// Expectation predicates captured natively by the reify plugin (fn-164.3): testdata/expect compiled
// with the fresh plugin retains, on each expectation Property and standalone `expect` value, the
// predicate the source exporter lifts from the same block (one shared lowering,
// reify/body/Expressions.scala `expectationPredicate`). The capture is read in two JVMs
// (testdata/expect/capture/Captures.scala) and compared with the source export of the same jar:
// every function reachable from the root, parameters, expressions and positions, and the types it
// carries. Two things are mapped, each explicitly: a Property's root, which the exporter names after
// the Property (`<machine>.property.<name>`) and the capture after the val it is written in, and a
// position's file, which the capture records as the compiled path and the exporter as the stored one.
// Blocks the DSL refuses are still refused at compile time, at their authored line, with no output.
class ExpectCaptureSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes
  private val tools = Tools.here
  private val mapper = new ObjectMapper()
  private lazy val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "expect-capture.")
  private val stored = tools.directory.resolve("model/framework/export/testdata/expect")
  private val fixture = "framework.fixture.expect.Expect$package$"
  private lazy val library = ModelBuild.frameworkLibrary:
    val jar = scratch.resolve("framework.jar")
    tools
      .scalaCli(
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
      .orFail(): Unit
    jar
  private lazy val plugin = ModelBuild.reifyPlugin:
    val jar = scratch.resolve("reify.jar")
    tools
      .scalaCli(
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
      .orFail(): Unit
    jar
  private def settings = Seq(
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
    tools.directory.resolve("model/build/ir-scalapb.jar").toString,
    "--scalac-option",
    s"-Xplugin:$plugin"
  ) ++ Seq(
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
  private def recorded(name: String, result: Ran): Ran =
    Files.writeString(
      scratch.resolve(s"$name.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    Files.writeString(scratch.resolve(s"$name.command"), result.command.mkString(" ")): Unit
    result
  // The fixture's sources (its project.scala names the retired system jars and is left out),
  // copied byte for byte so authored positions hold, compiled with the plugin.
  private def compile(name: String, files: Seq[String]): (Path, Path, Ran) =
    val dir = Files.createDirectories(scratch.resolve(name))
    for file <- files do
      Files.copy(stored.resolve(file), dir.resolve(Path.of(file).getFileName.toString)): Unit
    val out = scratch.resolve(s"$name.jar")
    val built = recorded(
      s"$name.compile",
      tools.scalaCli(
        Seq("compile", dir.toString, "--print-class-path", "--compilation-output", out.toString) ++
          settings,
        Output.KeptApart
      )
    )
    (dir, out, built)

  private lazy val compiled: (Path, Path, String) =
    val (dir, out, built) = compile("capture", Seq("Expect.scala", "capture/Captures.scala"))
    assertEquals(built.exit, 0, built.diagnostics)
    val cp = built.output.trim
    for entry <- cp.split(java.io.File.pathSeparatorChar) do
      assert(
        !entry.endsWith("api-scalapb.jar") && !entry.endsWith("model-scala.jar") &&
          !entry.contains("model/temporal"),
        s"the fixture classpath reaches a system artifact: $entry"
      )
    Files.writeString(scratch.resolve("capture.classpath"), cp): Unit
    (dir, out, cp)

  private case class Capture(name: String, root: String, model: ir.Model)
  // The captures of two JVMs, which print the same.
  private lazy val runs: (Ran, Ran) =
    val (_, _, cp) = compiled
    def run(n: Int) = recorded(
      s"captures-$n",
      tools.run("java", Seq("-cp", cp, "framework.fixture.expectcapture.Captures"))
    )
    (run(1), run(2))
  private lazy val captures: Map[String, Capture] =
    val (first, _) = runs
    assertEquals(first.exit, 0, first.diagnostics)
    first.output.linesIterator
      .filter(_.startsWith("captured:"))
      .map { line =>
        val Array(_, name, root, model) = line.split(":", 4): @unchecked
        name -> Capture(name, root, ir.Model.parseFrom(Base64.getDecoder.decode(model)))
      }
      .toMap
  // The source export of the very jar the captures ran from.
  private lazy val exported: ir.Model =
    val (dir, out, _) = compiled
    val output = scratch.resolve("source.json")
    val ran = recorded(
      "source.lift",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$out=$dir/",
          scratch.resolve("capture.classpath").toString,
          output.toString,
          s"$fixture.queries"
        )
      )
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))

  private def json(message: scalapb.GeneratedMessage): JsonNode =
    mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(message))
  // Positions with their file reduced to its name: the one mapping of a position (both sides name
  // Expect.scala; `fileOf` asserts it), line and columns compared as they are.
  private def located(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      for row <- node.fields().asScala do
        val value =
          if row.getKey == "position" then
            val p = row.getValue.deepCopy[ObjectNode]()
            p.put("file", p.path("file").asText().split('/').last)
          else located(row.getValue)
        out.set[JsonNode](row.getKey, value): Unit
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(located(n)): Unit)
      out
    else node
  private def files(node: JsonNode): Set[String] =
    if node.isObject then
      node
        .fields()
        .asScala
        .flatMap { row =>
          if row.getKey == "position" then Set(row.getValue.path("file").asText())
          else files(row.getValue)
        }
        .toSet
    else if node.isArray then node.elements().asScala.flatMap(files).toSet
    else Set.empty

  // The functions reachable from `root` in `model`, by name, with `rename` applied to every name and
  // call (the explicit root mapping), as located JSON.
  private def closure(
      model: ir.Model,
      root: String,
      rename: Map[String, String]
  ): Map[String, JsonNode] =
    val reached = scala.collection.mutable.LinkedHashMap.empty[String, JsonNode]
    def renamed(node: JsonNode): JsonNode =
      if node.isObject then
        val out = mapper.createObjectNode()
        for row <- node.fields().asScala do
          val value =
            if (row.getKey == "function" || row.getKey == "name") && row.getValue.isTextual then
              mapper.getNodeFactory.textNode(
                rename.getOrElse(row.getValue.asText(), row.getValue.asText())
              )
            else renamed(row.getValue)
          out.set[JsonNode](row.getKey, value): Unit
        out
      else if node.isArray then
        val out = mapper.createArrayNode()
        node.elements().asScala.foreach(n => out.add(renamed(n)): Unit)
        out
      else node
    def calls(node: JsonNode): Unit =
      if node.isObject then
        if node.has("call") then visit(node.path("call").path("function").asText())
        node.elements().asScala.foreach(calls)
      else if node.isArray then node.elements().asScala.foreach(calls)
    def visit(name: String): Unit =
      if !reached.contains(name) then
        val function = model.functions
          .find(_.name == name)
          .getOrElse(fail(s"$name is not in the model of $root"))
        val node = json(function)
        reached(name) = renamed(located(node))
        calls(node)
    visit(root)
    reached.map((k, v) => rename.getOrElse(k, k) -> v).toMap

  // A Property's predicate in the source export.
  private def holds(name: String): String =
    exported.properties
      .find(_.name == name)
      .getOrElse(fail(s"no Property $name in the source export"))
      .holds

  private val properties = Seq("heartbeat", "finished", "reset", "settled", "explicit")
  // The finite root mapping: the capture names a Property's own predicate after its val, the
  // exporter after the Property. A forwarding block and a standalone value need none.
  private def rootMap(name: String): (String, String) =
    val source = if name == "beatingNow" then s"$fixture.beatingNow" else holds(name)
    val captured = captures(name).root
    if properties.contains(name) then
      assertEquals(captured, s"$fixture.$name")
      assert(source.endsWith(s".property.$name"), source)
    else assertEquals(captured, source)
    captured -> source

  private def comparison(name: String): (Map[String, JsonNode], Map[String, JsonNode]) =
    val (captured, source) = rootMap(name)
    val model = captures(name).model
    (closure(model, captured, Map(captured -> source)), closure(exported, source, Map.empty))

  test("two JVMs print the same captures, and every expectation captured its own callable"):
    val (first, second) = runs
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    assert(first.output.contains("captures:passed"), first.output)
    assertEquals(
      captures.keySet,
      (properties ++ Seq("forwarded", "beatingNow")).toSet
    )

  for name <- properties do
    test(s"the $name Property captures exactly the predicate the source exporter lifts"):
      val (captured, source) = comparison(name)
      assertEquals(captured, source)

  test(
    "a forwarding block captures the standalone expectation it calls, as the exporter retains it"
  ):
    assertEquals(captures("forwarded").root, s"$fixture.beatingNow")
    val (captured, source) = comparison("forwarded")
    assertEquals(captured, source)
    assertEquals(captured.keySet, Set(s"$fixture.beatingNow"))

  test("a standalone expectation value captures its val's predicate, positioned at the val"):
    val (captured, source) = comparison("beatingNow")
    assertEquals(captured, source)
    assertEquals(
      captures("forwarded").model.functions.find(_.name == s"$fixture.beatingNow"),
      captures("beatingNow").model.functions.find(_.name == s"$fixture.beatingNow")
    )

  test("captured parameters are the block's context names and positions are the authored file"):
    def params(name: String) =
      val c = captures(name)
      c.model.functions.find(_.name == c.root).get.params.map(_.name).toList
    assertEquals(params("heartbeat"), List("after"))
    assertEquals(params("reset"), List("before", "after"))
    assertEquals(params("explicit"), List("b", "a"))
    for (name, c) <- captures do
      val reached = closure(c.model, c.root, Map.empty)
      for function <- c.model.functions if reached.contains(function.name) do
        assert(function.position.nonEmpty, s"${function.name} of $name has no position")
        assertEquals(files(json(function)).map(_.split('/').last), Set("Expect.scala"), name)

  test("the types a capture carries are the source export's types of the same names"):
    val sourceTypes = exported.types.map(t => t.name -> located(json(t))).toMap
    // The Properties read the fixture's enums; `beatingNow` reads only a Boolean field and so
    // carries no type of its own (nor does the forwarding block that captures it).
    for name <- properties do assert(captures(name).model.types.nonEmpty, s"$name carries no types")
    for (name, c) <- captures do
      for t <- c.model.types do
        assertEquals(Some(located(json(t))), sourceTypes.get(t.name), s"${t.name} of $name")

  test("the comparison tells apart another predicate, an unmapped root and a moved position"):
    val (heartbeat, _) = comparison("heartbeat")
    val (_, finished) = comparison("finished")
    assertNotEquals(heartbeat, finished)
    val c = captures("heartbeat")
    assertNotEquals(closure(c.model, c.root, Map.empty), comparison("heartbeat")._2)
    val moved = c.model.withFunctions(c.model.functions.map { f =>
      if f.name != c.root then f
      else f.withPosition(f.getPosition.withLine(f.getPosition.line + 1))
    })
    val (_, source) = comparison("heartbeat")
    assertNotEquals(closure(moved, c.root, Map(c.root -> holds("heartbeat"))), source)

  // A refused block: the marked authored line, the refusal it reports and no compiled output. The
  // capture runs before the DSL's own validation (which expands after pickling), so a block the
  // shared lowering refuses reports the exporter's refusal, the one a source export reports; a
  // block it lowers but the DSL refuses, such as a constructed step, reports the DSL's.
  for (file, refusal) <- Seq(
      "Statement.scala" -> "an expectation states requirements with check, records, if or oneOf",
      "Stepped.scala" ->
        "expect: a predicate cannot construct a path claim, temporal declaration or step",
      "HelpersOnly.scala" ->
        "a helper val of an expectation is followed by the requirements that read it"
    )
  do
    test(s"under the plugin, $file is still refused at its authored line with no output"):
      val name = file.stripSuffix(".scala")
      val (_, out, built) = compile(s"refused-$name", Seq("Expect.scala", s"refusals/$file"))
      assertNotEquals(built.exit, 0, built.diagnostics)
      val line = Files
        .readAllLines(stored.resolve(s"refusals/$file"))
        .asScala
        .indexWhere(_.contains("// refused")) + 1
      val errors = built.diagnostics.linesIterator.filter(_.contains("Error: ")).toList
      assert(
        errors.exists(e => e.contains(s"/refused-$name/$file:$line:")),
        s"no error at $file:$line in\n${built.diagnostics}"
      )
      assert(built.diagnostics.contains(refusal), built.diagnostics)
      val written =
        if Files.exists(out) then
          Files.walk(out).iterator.asScala.filter(_.toString.endsWith(".class")).toList
        else Nil
      assertEquals(written, Nil)
