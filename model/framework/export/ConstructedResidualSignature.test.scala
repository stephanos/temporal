package framework.`export`

import java.nio.file.{Files, Path}
import java.util.Base64
import scala.concurrent.duration.*
import framework.check.{Output, RetainedScratch, Tools}
import io.temporal.server.api.umpire.v1 as ir

class ConstructedResidualSignatureSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "constructed-residual.")
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
  )
  private def compile(dir: Path, name: String, extra: Seq[String] = Nil): (Path, String) =
    val jar = scratch.resolve(s"$name.jar")
    val built = tools.scalaCli(
      Seq(
        "compile",
        dir.toString,
        "--print-class-path",
        "--compilation-output",
        jar.toString
      ) ++ settings ++ extra,
      Output.KeptApart
    )
    assertEquals(built.exit, 0, built.diagnostics)
    jar -> built.output.trim
  private def run(cp: String, main: String): String =
    val result = tools.run("java", Seq("-cp", cp, main))
    assertEquals(result.exit, 0, result.diagnostics)
    result.output
  private def payload(output: String, label: String): ir.Model = ir.Model.parseFrom(
    Base64.getDecoder.decode(
      output.linesIterator.find(_.startsWith(label + ":")).get.stripPrefix(label + ":")
    )
  )
  private def frozen(path: String): ir.Model = scalapb.json4s.JsonFormat
    .fromJsonString[ir.Model](Files.readString(tools.directory.resolve(path)))

  test(
    "complete native residual messages and exact empty names match independent old output in two JVMs"
  ):
    val stored =
      tools.directory.resolve("model/framework/export/testdata/constructedResidualSignature")
    val dir = Files.createDirectories(scratch.resolve("oracle"))
    for name <- Seq("Names.scala", "Native.scala") do
      Files.copy(stored.resolve(name), dir.resolve(name)): Unit
    for name <- Seq("Declarations.scala", "ComposedProgress.scala") do
      Files.copy(
        tools.directory.resolve("model/framework/export/testdata/lifts").resolve(name),
        dir.resolve(name)
      ): Unit
    val (jar, cp) = compile(dir, "oracle")
    val first = run(cp, "framework.residual.Native")
    val second = run(cp, "framework.residual.Native")
    assertEquals(first, second)
    assert(first.contains("controls:passed"))
    val oldNames =
      frozen("model/framework/export/testdata/constructedResidualSignature/original.json")
    val native = payload(first, "names")
    assertEquals(native.actions, oldNames.actions.map(_.clearPosition))
    assertEquals(native.assumptions, oldNames.assumptions.map(_.clearPosition))
    assertEquals(native.actions.find(_.id.endsWith("emptyAction")).get.name, "")
    assertEquals(native.actions.find(_.id.endsWith("emptyAction")).get.on, "")
    assertEquals(
      native.assumptions.find(_.id.endsWith("inferredFairness")).get.fairRefs.head.selection,
      ir.ComposedActionRef.Selection.Sync("")
    )
    assertEquals(
      payload(first, "limits").queries.map(_.getLimits).sortBy(_.name),
      oldNames.queries.map(_.getLimits).sortBy(_.name)
    )
    val oldDeclarations = frozen("model/framework/export/testdata/lifts/expected/declarations.json")
    val declarations = payload(first, "declarations")
    assertEquals(declarations.assumptions, oldDeclarations.assumptions.map(_.clearPosition))
    assertEquals(declarations.holes, oldDeclarations.holes.map(_.clearPosition))
    assertEquals(
      payload(first, "declarationLimits").queries.head.getLimits,
      oldDeclarations.queries.head.getLimits
    )
    assert(
      oldDeclarations.queries.forall(_.getLimits.equals(oldDeclarations.queries.head.getLimits))
    )
    val oldComposed = frozen("model/framework/export/testdata/lifts/expected/composedProgress.json")
    assertEquals(
      payload(first, "composed").assumptions,
      oldComposed.assumptions.map(_.clearPosition)
    )
    val classpath = Files.writeString(scratch.resolve("oracle.classpath"), cp)
    val out = scratch.resolve("current-reader.json")
    val lifted = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$dir/",
        classpath.toString,
        out.toString,
        "fixture.residualnames.Names$package$.queries",
        "fixture.residualnames.Names$package$.progress",
        "fixture.residualnames.Names$package$.pairedProgress"
      )
    )
    assertEquals(lifted.exit, 0, lifted.diagnostics)
    val current = scalapb.json4s.JsonFormat.fromJsonString[ir.Model](Files.readString(out))
    assertEquals(current.actions.map(_.clearPosition), oldNames.actions.map(_.clearPosition))
    assertEquals(
      current.assumptions.map(_.clearPosition),
      oldNames.assumptions.map(_.clearPosition)
    )
    assertEquals(current.queries.map(_.getLimits), oldNames.queries.map(_.getLimits))

  test(
    "newly supported helper jars preserve native name provenance and omitted versus empty copies"
  ):
    val stored =
      tools.directory.resolve("model/framework/export/testdata/constructedResidualSignature")
    val helper = Files.createDirectories(scratch.resolve("helper"))
    Files.copy(stored.resolve("Helpers.scala"), helper.resolve("Helpers.scala")): Unit
    val (helperJar, _) = compile(helper, "helper")
    val client = Files.createDirectories(scratch.resolve("client"))
    Files.copy(stored.resolve("Client.scala"), client.resolve("Client.scala")): Unit
    val (clientJar, cp) = compile(client, "client", Seq("--jar", helperJar.toString))
    val first = run(cp, "framework.residualclient.Client")
    val second = run(cp, "framework.residualclient.Client")
    assertEquals(first, second)
    assert(first.contains("helper-controls:passed"))
    val classpath = Files.writeString(scratch.resolve("helper.classpath"), cp)
    val refused = InProcess.lift(
      tools.directory,
      Seq(
        s"$clientJar=$client/,$helperJar=$helper/",
        classpath.toString,
        scratch.resolve("old-helper-refused.json").toString,
        "framework.residualclient.Client$package$.progress"
      )
    )
    assertEquals(refused.exit, 1, refused.diagnostics)
    assert(refused.diagnostics.contains("not a part of an action declaration"), refused.diagnostics)

  test("new identities require object owners and helper values are named at their concrete caller"):
    val stored = tools.directory.resolve(
      "model/framework/export/testdata/constructedResidualSignature/Scopes.scala"
    )
    val dir = Files.createDirectories(scratch.resolve("scopes"))
    Files.copy(stored, dir.resolve("Scopes.scala")): Unit
    val (_, cp) = compile(dir, "scopes")
    assert(run(cp, "framework.residualscopes.Scopes").contains("scope-controls:passed"))
