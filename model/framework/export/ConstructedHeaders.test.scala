package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedHeadersSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes
  private val tools = Tools.here
  private lazy val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "constructed-headers.")
  private val stored = tools.directory.resolve("model/framework/export/testdata/constructedHeaders")
  private lazy val library =
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
  private lazy val plugin =
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
  private def compile(
      names: Seq[String],
      name: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil
  ): (Path, Path, String) =
    val dir = Files.createDirectories(scratch.resolve(name))
    names.foreach { n =>
      val selected = if withPlugin && n == "Headers.scala" then "ProspectiveHeaders.scala" else n
      Files.copy(stored.resolve(selected), dir.resolve(selected)): Unit
    }
    if withPlugin && names.contains("Headers.scala") then
      Files.writeString(
        dir.resolve("EndpointControls.scala"),
        """package framework.headerendpointcontrols
          |import framework.dsl.*
          |import fixture.constructedheader.*
          |object EndpointControls:
          |  def main(args: Array[String]): Unit =
          |    for machine <- Seq(Scalar, Opaque, OpaqueSeparate) do
          |      val failure = try
          |        MachineEndpoints.collect(machine): Unit
          |        throw AssertionError("support endpoint metadata unexpectedly retained")
          |      catch
          |        case error: IllegalArgumentException => error
          |      assert(failure.getMessage.contains("no retained endpoint metadata"))
          |    assert(Scalar.init == UpTo[2](0))
          |    assert(Opaque.init.equals(Domains.zero))
          |    assert(OpaqueSeparate.init.equals(Independent.zero))
          |    assert(Scalar.end(Scalar.init) && Opaque.end(Opaque.init) && OpaqueSeparate.end(OpaqueSeparate.init))
          |    println("support-endpoint-controls:passed")
          |""".stripMargin
      ): Unit
    val jar = scratch.resolve(s"$name.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        dir.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings(withPlugin) ++ extra,
      Output.KeptApart
    )
    Files.writeString(
      scratch.resolve(s"$name.compile.log"),
      s"exit=${built.exit}\n${built.diagnostics}"
    ): Unit
    Files.writeString(scratch.resolve(s"$name.compile.command"), built.command.mkString(" ")): Unit
    assertEquals(built.exit, 0, built.diagnostics)
    (dir, jar, built.output.trim)
  private val roots = Seq(
    "Named",
    "Aliased",
    "BooleanMachine",
    "Replaced",
    "OpaqueOutcome",
    "InboxMachine",
    "NestedMachine"
  )
  private val mapper = new ObjectMapper()
  private def withoutPositions(node: JsonNode): JsonNode =
    if node.isObject then
      val out = mapper.createObjectNode()
      for row <- node.fields().asScala if row.getKey != "position" do
        out.set[JsonNode](row.getKey, withoutPositions(row.getValue)): Unit
      out
    else if node.isArray then
      val out = mapper.createArrayNode()
      node.elements().asScala.foreach(n => out.add(withoutPositions(n)): Unit)
      out
    else node
  private def comparable(model: ir.Model): JsonNode =
    withoutPositions(mapper.readTree(scalapb.json4s.JsonFormat.toJsonString(model)))
  private def projected(model: ir.Model): ir.Model =
    ir.Model(
      machines = model.machines.map(m =>
        ir.Machine(
          family = m.family,
          name = m.name,
          stateType = m.stateType,
          outcomeType = m.outcomeType,
          factType = m.factType
        )
      ),
      types = model.types
    )
  private def frozen: ir.Model =
    scalapb.json4s.JsonFormat
      .fromJsonString[ir.Model](Files.readString(stored.resolve("original.json")))
  private def lift(
      dir: Path,
      jar: Path,
      cp: String,
      name: String,
      selected: Seq[String] = roots
  ): ir.Model =
    val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
    val output = scratch.resolve(s"$name.json")
    val result = InProcess.lift(
      tools.directory,
      Seq(s"$jar=$dir/", classpath.toString, output.toString) ++ selected.map(
        "fixture.constructedheader." + _
      )
    )
    Files.writeString(
      scratch.resolve(s"$name.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    assertEquals(result.exit, 0, result.diagnostics)
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))

  test("independent original sources retain complete old header output"):
    val (dir, jar, cp) = compile(Seq("Headers.scala"), "original", false)
    val old = lift(dir, jar, cp, "original")
    assertEquals(old.machines.size, roots.size)
    assertEquals(comparable(projected(old)), comparable(projected(frozen)))
    for (root, expected) <- Seq(
        "Scalar" -> "neither an enum nor a case class",
        "OpaqueSeparate" -> "neither an enum nor a case class",
        "Opaque" -> "has no range"
      )
    do
      val result = InProcess.lift(
        tools.directory,
        Seq(
          s"$jar=$dir/",
          scratch.resolve("original.classpath").toString,
          scratch.resolve(s"refused-$root.json").toString,
          "fixture.constructedheader." + root
        )
      )
      Files.writeString(
        scratch.resolve(s"refused-$root.log"),
        s"exit=${result.exit}\n${result.diagnostics}"
      ): Unit
      assertEquals(result.exit, 1, result.diagnostics)
      assert(result.diagnostics.contains(expected), result.diagnostics)
    println(s"header-oracle:$scratch")

  test(
    "five-field contributions and complete selected Types match the old projection in two JVMs and saved trees"
  ):
    val (dir, jar, cp) = compile(Seq("Headers.scala", "Native.scala"), "native", true)
    val first = tools.run("java", Seq("-cp", cp, "framework.headernative.Native"))
    val second = tools.run("java", Seq("-cp", cp, "framework.headernative.Native"))
    Files.writeString(
      scratch.resolve("native.log"),
      s"first=${first.exit},second=${second.exit}\n${first.output}\n${first.diagnostics}"
    ): Unit
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    val controls =
      tools.run("java", Seq("-cp", cp, "framework.headerendpointcontrols.EndpointControls"))
    Files.writeString(
      scratch.resolve("support-endpoint-controls.log"),
      s"exit=${controls.exit}\n${controls.output}\n${controls.diagnostics}"
    ): Unit
    assertEquals(controls.exit, 0, controls.diagnostics)
    assert(controls.output.contains("support-endpoint-controls:passed"))
    val native = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        first.output.linesIterator
          .find(_.startsWith("headers:"))
          .get
          .stripPrefix("headers:")
      )
    )
    assertEquals(comparable(native), comparable(projected(frozen)))
    assertEquals(comparable(projected(lift(dir, jar, cp, "replay"))), comparable(projected(frozen)))
    assert(first.output.contains("native-controls:passed"))
    println(s"native-header-artifacts:$scratch")

  test(
    "catalogs and getters stay inert while native Built construction and premature provenance keep their timing"
  ):
    val (_, _, cp) =
      compile(Seq("Headers.scala", "Selection.scala", "Timing.scala"), "timing", true)
    val first = tools.run("java", Seq("-cp", cp, "framework.headertiming.Timing"))
    val second = tools.run("java", Seq("-cp", cp, "framework.headertiming.Timing"))
    Files.writeString(
      scratch.resolve("timing.log"),
      s"first=${first.exit},second=${second.exit}\n${first.output}\n${first.diagnostics}"
    ): Unit
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    assert(first.output.contains("timing:catalogs=0,getters=0,native-init=2,native-refinement=2"))

  test(
    "concrete generic helper jars retain their own module identity and unchanged native counters"
  ):
    val (baseDir, baseJar, _) = compile(Seq("Headers.scala"), "helper-base", true)
    val (helperDir, helperJar, _) =
      compile(Seq("Helpers.scala"), "helper-library", true, Seq("--jar", baseJar.toString))
    val (clientDir, clientJar, cp) = compile(
      Seq("Client.scala", "HelperNative.scala"),
      "helper-client",
      true,
      Seq("--jar", baseJar.toString, "--jar", helperJar.toString)
    )
    val first = tools.run("java", Seq("-cp", cp, "framework.headerclient.Client"))
    val second = tools.run("java", Seq("-cp", cp, "framework.headerclient.Client"))
    Files.writeString(
      scratch.resolve("helper-native.log"),
      s"first=${first.exit},second=${second.exit}\n${first.output}\n${first.diagnostics}"
    ): Unit
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    assert(first.output.contains("helper-controls:passed,init=1,name=0"))
    val classpath = Files.writeString(scratch.resolve("helper.classpath"), cp)
    val output = scratch.resolve("helper-old.json")
    val sources = s"$clientJar=$clientDir/,$helperJar=$helperDir/,$baseJar=$baseDir/"
    val old = InProcess.lift(
      tools.directory,
      Seq(
        sources,
        classpath.toString,
        output.toString,
        "fixture.headerclient.Inherited",
        "fixture.headerclient.DerivedInherited"
      )
    )
    Files.writeString(
      scratch.resolve("helper-old.log"),
      s"exit=${old.exit}\n${old.diagnostics}"
    ): Unit
    assertEquals(old.exit, 0, old.diagnostics)
    val oldModel = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
    val native = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        first.output.linesIterator
          .find(_.startsWith("helper:"))
          .get
          .stripPrefix("helper:")
      )
    )
    assertEquals(comparable(native), comparable(projected(oldModel)))
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        sources,
        classpath.toString,
        scratch.resolve("helper-counter-refused.json").toString,
        "fixture.headerlibrary.LibraryMachine"
      )
    )
    Files.writeString(
      scratch.resolve("helper-counter-refused.log"),
      s"exit=${refused.exit}\n${refused.diagnostics}"
    ): Unit
    assertEquals(refused.exit, 1, refused.diagnostics)
