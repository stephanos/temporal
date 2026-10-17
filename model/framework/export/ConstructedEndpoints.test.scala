package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Ran, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedEndpointsSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes
  private val tools = Tools.here
  private lazy val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "constructed-endpoints.")
  private val stored =
    tools.directory.resolve("model/framework/export/testdata/constructedEndpoints")
  private lazy val mapper = new ObjectMapper()
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
    )
      .flatMap(o => Seq("--scalac-option", o))
  private def recorded(name: String, result: Ran): Ran =
    Files.writeString(
      scratch.resolve(s"$name.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    Files.writeString(scratch.resolve(s"$name.command"), result.command.mkString(" ")): Unit
    result
  private def compile(
      names: Seq[String],
      name: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil
  ): (Path, Path, String) =
    val dir = Files.createDirectories(scratch.resolve(name))
    names.foreach(n => Files.copy(stored.resolve(n), dir.resolve(n)): Unit)
    val jar = scratch.resolve(s"$name.jar")
    val built = recorded(
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
    assertEquals(built.exit, 0, built.diagnostics)
    Files.writeString(scratch.resolve(s"$name.classpath"), built.output.trim): Unit
    (dir, jar, built.output.trim)
  private val roots = Seq(
    "DirectVal",
    "DirectDef",
    "Default",
    "Explicit",
    "OpenOverride",
    "Forwarded",
    "Outer$.Nested",
    "HelperRequire",
    "HelperEnsuring",
    "ChannelEnd",
    "HoleEnd"
  )
    .map("fixture.endpoints." + _)
  private def lift(sources: String, cp: String, name: String, selected: Seq[String]): ir.Model =
    val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
    val output = scratch.resolve(s"$name.json")
    val result = recorded(
      name,
      InProcess.frameworkLift(
        tools.directory,
        Seq(sources, classpath.toString, output.toString) ++ selected
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
  private lazy val original =
    val (dir, jar, cp) = compile(Seq("Endpoints.scala"), "original", false)
    val model = lift(s"$jar=$dir/", cp, "original", roots)
    assertEquals(model.machines.size, roots.size)
    assertEquals(
      model.functions.map(_.name).sorted,
      Seq("fixture.endpoints.Helpers$.assured", "fixture.endpoints.Helpers$.guarded")
    )
    assert(model.channels.nonEmpty && model.holes.nonEmpty)
    println(s"endpoint-oracle:$scratch")
    model

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
  private def frozen(name: String): ir.Model =
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(stored.resolve(name)))
  private def projected(model: ir.Model): ir.Model =
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
    val machines =
      model.machines.map(m => ir.Machine(name = m.name, starts = m.starts, ends = m.ends))
    machines.foreach(read)
    ir.Model(
      machines = machines.sortBy(_.name),
      functions = model.functions.filter(f => functions(f.name)).sortBy(_.name),
      types = model.types.filter(t => types(t.name)).sortBy(_.name)
    )
  private def native(cp: String, main: String, name: String): ir.Model =
    val first = recorded(s"$name.first", tools.run("java", Seq("-cp", cp, main)))
    val second = recorded(s"$name.second", tools.run("java", Seq("-cp", cp, main)))
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    Files.writeString(scratch.resolve(s"$name.output"), first.output): Unit
    ir.Model.parseFrom(
      Base64.getDecoder.decode(
        first.output.linesIterator.find(_.startsWith("endpoints:")).get.stripPrefix("endpoints:")
      )
    )

  test("fresh plugin-free sources retain complete endpoint output and helper contracts"):
    val old = original
    val guarded = old.functions.find(_.name.endsWith("guarded")).get
    val assured = old.functions.find(_.name.endsWith("assured")).get
    assertEquals(guarded.getRequires.getVar, "s")
    assertEquals(guarded.getBody.getVar, "s")
    assert(assured.requires.isEmpty)
    assertEquals(assured.getBody.getVar, "s")
    assertEquals(comparable(old), comparable(frozen("original.json")))

  test("fresh helper jars retain complete independent endpoint and transitive helper output"):
    val (helperDir, helperJar, _) = compile(Seq("Helpers.scala"), "helper-original", false)
    val (dir, jar, cp) =
      compile(Seq("Client.scala"), "client-original", false, Seq("--jar", helperJar.toString))
    val old = lift(
      s"$jar=$dir/,$helperJar=$helperDir/",
      cp,
      "helper-old",
      Seq("fixture.endpointclient.HelperJar")
    )
    assertEquals(
      old.functions.map(_.name).sorted,
      Seq("fixture.endpointlibrary.Helpers$.forward", "fixture.endpointlibrary.Helpers$.leaf")
    )
    assert(old.functions.forall(_.position.nonEmpty))
    assertEquals(comparable(old), comparable(frozen("helper-old.json")))

  test(
    "isolated endpoint expressions and complete dependency closure match old output in two JVMs and saved trees"
  ):
    val (dir, jar, cp) = compile(
      Seq("Endpoints.scala", "Native.scala", "Timing.scala", "Support.scala"),
      "native",
      true
    )
    val value = native(cp, "framework.endpointnative.Native", "native")
    assertEquals(comparable(value), comparable(projected(frozen("original.json"))))
    val replay = projected(lift(s"$jar=$dir/", cp, "replay", roots))
    assertEquals(comparable(replay), comparable(value))
    assertEquals(
      value.functions.map(f => f.name -> f.position).toMap,
      replay.functions.map(f => f.name -> f.position).toMap
    )
    val first =
      recorded("timing.first", tools.run("java", Seq("-cp", cp, "framework.endpointtiming.Timing")))
    val second = recorded(
      "timing.second",
      tools.run("java", Seq("-cp", cp, "framework.endpointtiming.Timing"))
    )
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    assert(first.output.contains("endpoint-timing:passed"))
    println(s"native-endpoint-artifacts:$scratch")

  test("endpoint helper imports use definition-site IR across source files and jars"):
    val (helperDir, helperJar, _) = compile(Seq("Helpers.scala"), "helper", true)
    val (dir, jar, cp) = compile(
      Seq("Client.scala", "HelperNative.scala"),
      "client",
      true,
      Seq("--jar", helperJar.toString)
    )
    val value = native(cp, "framework.endpointhelpernative.Native", "helper-native")
    assertEquals(comparable(value), comparable(projected(frozen("helper-old.json"))))
    val replay = projected(
      lift(
        s"$jar=$dir/,$helperJar=$helperDir/",
        cp,
        "helper-replay",
        Seq("fixture.endpointclient.HelperJar")
      )
    )
    assertEquals(comparable(replay), comparable(value))
    assertEquals(
      value.functions.map(f => f.name -> f.position).toMap,
      replay.functions.map(f => f.name -> f.position).toMap
    )
    val (_, _, localCp) =
      compile(Seq("Helpers.scala", "Client.scala", "HelperNative.scala"), "cross-file", true)
    assertEquals(
      comparable(native(localCp, "framework.endpointhelpernative.Native", "cross-file")),
      comparable(value)
    )
    val foreign = Files.createDirectories(scratch.resolve("unmarked"))
    Files.writeString(
      foreign.resolve("Unmarked.scala"),
      """package fixture.unmarkedendpoint
import framework.dsl.*
object Refused extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = fixture.endpointlibrary.Unmarked.unmarked(s)
  object rules extends Bindings()
"""
    ): Unit
    val refusal = recorded(
      "unmarked",
      tools.scalaCli(
        Seq("compile", foreign.toString) ++ settings(true) ++ Seq("--jar", helperJar.toString)
      )
    )
    assertEquals(refusal.exit, 1, refusal.diagnostics)
    assert(
      refusal.diagnostics.contains("Unmarked$.unmarked") && refusal.diagnostics.contains(
        "is reached from another file: define it in an object that extends Lifted"
      ),
      refusal.diagnostics
    )
    assert(refusal.diagnostics.contains("Unmarked.scala:6"), refusal.diagnostics)

  test("inherited support endpoints retain native evaluation and their actual old refusal"):
    val (dir, jar, cp) = compile(Seq("Support.scala"), "support", true)
    val native = recorded(
      "support-native",
      tools.run("java", Seq("-cp", cp, "framework.endpointsupport.Support"))
    )
    assertEquals(native.exit, 0, native.diagnostics)
    assert(native.output.contains("inherited-native-only:init=1,end=1; header-increment=0"))
    val classpath = Files.writeString(scratch.resolve("support.classpath"), cp)
    val old = recorded(
      "support-old",
      InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/",
          classpath.toString,
          scratch.resolve("support-old.json").toString,
          "framework.endpointsupport.Inherited"
        )
      )
    )
    assertEquals(old.exit, 1, old.diagnostics)
    assert(old.diagnostics.contains("inherited declares no init"), old.diagnostics)

  private def negatives = Seq(
    (
      "InitRequire",
      "Boolean",
      "def init: Boolean = { require(true); false }",
      "def end(s: Boolean) = s",
      "",
      ""
    ),
    (
      "InitEnsuring",
      "Boolean",
      "val init = false.ensuring((result: Boolean) => !result)",
      "def end(s: Boolean) = s",
      "",
      ""
    ),
    (
      "EndRequire",
      "Boolean",
      "val init = false",
      "def end(s: Boolean): Boolean = { require(s); s }",
      "",
      ""
    ),
    (
      "EndEnsuring",
      "Boolean",
      "val init = false",
      "def end(s: Boolean) = s.ensuring((result: Boolean) => result == s)",
      "",
      ""
    ),
    (
      "ThrowInit",
      "Boolean",
      "def init: Boolean = throw AssertionError(\"init\")",
      "def end(s: Boolean) = s",
      "",
      ""
    ),
    (
      "ThrowEnd",
      "Boolean",
      "val init = false",
      "def end(s: Boolean): Boolean = throw AssertionError(\"end\")",
      "",
      ""
    ),
    (
      "OpenDefault",
      "Snapshot",
      "val init = Snapshot(Phase.open)",
      "",
      ", Phased[Snapshot, Phase](_.phase)",
      "enum Phase { case open }; final case class Snapshot(phase: Phase)"
    ),
    (
      "FieldDefault",
      "Snapshot",
      "val init = Snapshot(Phase.open)",
      "",
      ", Phased[Snapshot, Phase](_.phase)",
      "enum Phase { case open; case done(flag: Boolean) extends Phase, Closed }; final case class Snapshot(phase: Phase)"
    )
  )
  test("direct contracts unsupported bodies and default role sets retain located old refusals"):
    for (name, state, init, end, phased, declarations) <- negatives do
      val dir = Files.createDirectories(scratch.resolve(s"old-$name"))
      Files.writeString(
        dir.resolve("Refused.scala"),
        s"""package fixture.endpointnegative
import framework.dsl.*
$declarations
object $name extends Machine[$state, Boolean, Nothing]$phased:
  $init
  $end
  object rules extends Bindings()
"""
      ): Unit
      val jar = scratch.resolve(s"old-$name.jar")
      val built = recorded(
        s"old-$name.compile",
        tools.scalaCli(
          Seq(
            "compile",
            dir.toString,
            "--print-class-path",
            "--compilation-output",
            jar.toString
          ) ++ settings(false),
          Output.KeptApart
        )
      )
      assertEquals(built.exit, 0, built.diagnostics)
      val classpath = Files.writeString(scratch.resolve(s"old-$name.classpath"), built.output.trim)
      val old = recorded(
        s"old-$name",
        InProcess.frameworkLift(
          tools.directory,
          Seq(
            s"$jar=$dir/",
            classpath.toString,
            scratch.resolve(s"old-$name.json").toString,
            s"fixture.endpointnegative.$name"
          )
        )
      )
      val reason = if name == "OpenDefault" then "has no Closed case"
      else if name == "FieldDefault" then "is Closed and has fields"
      else "outside the liftable subset"
      assertEquals(old.exit, 1, old.diagnostics)
      assert(old.diagnostics.contains(reason), old.diagnostics)
      assert(old.diagnostics.contains("Refused.scala:"), old.diagnostics)
      val captured =
        recorded(s"new-$name", tools.scalaCli(Seq("compile", dir.toString) ++ settings(true)))
      assertEquals(captured.exit, 1, captured.diagnostics)
      assert(captured.diagnostics.contains(reason), captured.diagnostics)
      assert(captured.diagnostics.contains("Refused.scala:"), captured.diagnostics)

  test("own declaration endpoint bodies refuse outside the liftable subset at their source"):
    val dir = Files.createDirectories(scratch.resolve("unsupported"))
    Files.copy(stored.resolve("Unsupported.scala"), dir.resolve("Unsupported.scala")): Unit
    val result = recorded(
      "unsupported.compile",
      tools.scalaCli(Seq("compile", dir.toString) ++ settings(true))
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(result.diagnostics.contains("outside the liftable subset"), result.diagnostics)
    assert(result.diagnostics.contains("Unsupported.scala:6"), result.diagnostics)
