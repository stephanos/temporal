package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import com.fasterxml.jackson.databind.{JsonNode, ObjectMapper}
import framework.check.{Output, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedBindingsSuite extends munit.FunSuite:
  override val munitTimeout: Duration = 20.minutes
  private val tools = Tools.here
  private lazy val scratch =
    Files.createTempDirectory(tools.directory.resolve(".flow/tmp"), "constructed-bindings.")
  private val stored =
    tools.directory.resolve("model/framework/export/testdata/constructedBindings")
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
    ).flatMap(o => Seq("--scalac-option", o))
  private def compile(
      names: Seq[String],
      name: String,
      withPlugin: Boolean,
      extra: Seq[String] = Nil
  ): (Path, Path, String) =
    val dir = Files.createDirectories(scratch.resolve(name))
    names.foreach(n => Files.copy(stored.resolve(n), dir.resolve(n)): Unit)
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
  private def lift(dir: Path, jar: Path, cp: String, name: String, roots: Seq[String]): ir.Model =
    val classpath = Files.writeString(scratch.resolve(s"$name.classpath"), cp)
    val output = scratch.resolve(s"$name.json")
    val result = InProcess.frameworkLift(
      tools.directory,
      Seq(s"$jar=$dir/", classpath.toString, output.toString) ++ roots.map(
        "fixture.bindingdiscovery." + _
      )
    )
    Files.writeString(
      scratch.resolve(s"$name.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    Files.writeString(scratch.resolve(s"$name.command"), result.command.mkString(" ")): Unit
    assertEquals(result.exit, 0, result.diagnostics)
    scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
  private val roots = Seq(
    "Base",
    "Named",
    "Eta",
    "Forwarding",
    "Explicit",
    "ActionAlias",
    "Replaced",
    "Added",
    "Restricted",
    "Unmonitored",
    "Chained",
    "DuplicateInitial",
    "DuplicateAnonymous"
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
    val reached = scala.collection.mutable.Set.empty[String]
    def calls(node: JsonNode): Unit =
      if node.isObject then
        if node.has("call") then visit(node.path("call").path("function").asText())
        node.elements().asScala.foreach(calls)
      else if node.isArray then node.elements().asScala.foreach(calls)
    def visit(name: String): Unit =
      if reached.add(name) then
        calls(
          mapper.readTree(
            scalapb.json4s.JsonFormat.toJsonString(model.functions.find(_.name == name).get)
          )
        )
    model.machines.flatMap(_.steps).foreach(s => visit(s.function))
    ir.Model(
      machines = model.machines.map(m => ir.Machine(name = m.name, steps = m.steps)),
      functions = model.functions.filter(f => reached(f.name)).sortBy(_.name),
      types = model.types.sortBy(_.name)
    )

  test("strict original sources retain complete old binding outputs and actual refusal controls"):
    val (dir, jar, cp) = compile(Seq("Bindings.scala"), "original", false)
    val old = lift(dir, jar, cp, "original", roots)
    assertEquals(old.machines.size, roots.size)
    assertEquals(old.machines.find(_.name == "duplicateAnonymous").get.steps.size, 2)
    val frozen = scalapb.json4s.JsonFormat
      .fromJsonString[ir.Model](Files.readString(stored.resolve("original.json")))
    assertEquals(comparable(projected(old)), comparable(projected(frozen)))
    val classpath = scratch.resolve("original.classpath")
    for root <- Seq(
        "ValAlias",
        "UnchangedCopy",
        "ChangedCallableCopy",
        "ChangedDeclarationCopy",
        "DuplicateReplace",
        "DuplicateAdd",
        "UnknownReplace",
        "AlreadyBound",
        "InvalidThenRestrict"
      )
    do
      val result = InProcess.frameworkLift(
        tools.directory,
        Seq(
          s"$jar=$dir/",
          classpath.toString,
          scratch.resolve(s"refused-$root.json").toString,
          s"fixture.bindingdiscovery.$root"
        )
      )
      Files.writeString(
        scratch.resolve(s"refused-$root.log"),
        s"exit=${result.exit}\n${result.diagnostics}"
      ): Unit
      assertEquals(result.exit, 1, result.diagnostics)
    println(s"binding-oracle:$scratch")

  test("native table propagation retains capture without changing its callable"):
    val (_, _, cp) = compile(Seq("Bindings.scala", "Retention.scala"), "retention", true)
    val result = tools.run("java", Seq("-cp", cp, "framework.bindingretention.Retention"))
    Files.writeString(
      scratch.resolve("retention.log"),
      s"exit=${result.exit}\n${result.diagnostics}"
    ): Unit
    assertEquals(result.exit, 0, result.diagnostics)
    assert(result.output.contains("retention:passed"))

  test(
    "complete ordered native binding contributions match the frozen old projection in two JVMs and saved trees"
  ):
    val (dir, jar, cp) = compile(Seq("Bindings.scala", "Native.scala"), "native", true)
    val first = tools.run("java", Seq("-cp", cp, "framework.bindingnative.Native"))
    val second = tools.run("java", Seq("-cp", cp, "framework.bindingnative.Native"))
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    Files.writeString(
      scratch.resolve("native.log"),
      s"first=${first.exit},second=${second.exit}\n${first.output}\n${first.diagnostics}"
    ): Unit
    val native = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        first.output.linesIterator.find(_.startsWith("bindings:")).get.stripPrefix("bindings:")
      )
    )
    val frozen = scalapb.json4s.JsonFormat
      .fromJsonString[ir.Model](Files.readString(stored.resolve("original.json")))
    assertEquals(comparable(native), comparable(projected(frozen)))
    val replay = lift(dir, jar, cp, "replay", roots)
    assertEquals(comparable(projected(replay)), comparable(projected(frozen)))
    assert(first.output.contains("controls:passed"))
    assert(native.functions.forall(_.position.nonEmpty))
    println(s"native-binding-artifacts:$scratch")

  test("named helper jars preserve transitive closure beside a caller anonymous root"):
    val (baseDir, baseJar, _) = compile(Seq("Bindings.scala"), "helper-base", true)
    val (helperDir, helperJar, _) =
      compile(Seq("Helpers.scala"), "helper-library", true, Seq("--jar", baseJar.toString))
    val (clientDir, clientJar, cp) = compile(
      Seq("Client.scala", "HelperNative.scala"),
      "helper-client",
      true,
      Seq("--jar", baseJar.toString, "--jar", helperJar.toString)
    )
    val first = tools.run("java", Seq("-cp", cp, "framework.bindinghelper.HelperNative"))
    val second = tools.run("java", Seq("-cp", cp, "framework.bindinghelper.HelperNative"))
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    Files.writeString(
      scratch.resolve("helper-native.log"),
      s"first=${first.exit},second=${second.exit}\n${first.output}\n${first.diagnostics}"
    ): Unit
    val native = ir.Model.parseFrom(
      Base64.getDecoder.decode(
        first.output.linesIterator.find(_.startsWith("helper:")).get.stripPrefix("helper:")
      )
    )
    val classpath = Files.writeString(scratch.resolve("helper.classpath"), cp)
    val output = scratch.resolve("helper-old.json")
    val lifted = InProcess.frameworkLift(
      tools.directory,
      Seq(
        s"$clientJar=$clientDir/,$helperJar=$helperDir/,$baseJar=$baseDir/",
        classpath.toString,
        output.toString,
        "fixture.bindingdiscovery.HelperMachine"
      )
    )
    assertEquals(lifted.exit, 0, lifted.diagnostics)
    val old = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(output))
    assertEquals(comparable(native), comparable(projected(old)))

  test("all binding arities and catalog init refinement timing survive native conversion"):
    val (_, _, cp) = compile(Seq("Arities.scala"), "arities", true)
    val first = tools.run("java", Seq("-cp", cp, "framework.bindingarity.ArityNative"))
    val second = tools.run("java", Seq("-cp", cp, "framework.bindingarity.ArityNative"))
    assertEquals(first.exit, 0, first.diagnostics)
    assertEquals(second.exit, 0, second.diagnostics)
    assertEquals(first.output, second.output)
    assert(first.output.contains("arity-evaluation:0,1,2,3,4,6"))
    Files.writeString(
      scratch.resolve("arity-native.log"),
      s"first=${first.exit},second=${second.exit}\n${first.output}\n${first.diagnostics}"
    ): Unit
