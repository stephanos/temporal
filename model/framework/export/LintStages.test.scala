package framework.`export`

import java.nio.file.{Files, Path}
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture

class LintStagesSuite extends munit.FunSuite with RetainedScratch:
  private val tools = Tools.here

  test("the gate checks sources, lifts and checks markers before publishing IR"):
    val out = retained(tools.directory.resolve("model/build/history"), "lint-stages.")
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.checkAndLift",
        "--ir",
        ModelBuild.jarArgument,
        ModelBuild.classpath.toString,
        out.toString,
        "activity-mini"
      )
    )
    assertEquals(result.exit, 0, result.diagnostics)
    assertEquals(
      result.output.linesIterator.filter(_.startsWith("== ")).toList,
      List("== lint admitted sources", "== lift IR", "== check lifted markers")
    )
    assert(Files.isRegularFile(out.resolve("activity-mini.json")), result.diagnostics)

  test("a gate source refusal stops before lifting and creates no output directory"):
    val scratch = retained(tools.directory.resolve("model/build/history"), "lint-stopped.")
    val stored = "model/framework/export/testdata/initOrder/"
    val sources = ModelBuild.copied(tools.directory.resolve(stored), scratch.resolve("sources"))
    val jar = scratch.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          sources.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val out = scratch.resolve("ir")
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.checkAndLift",
        "--ir",
        s"$jar=$stored,${ModelBuild.jarArgument}",
        ModelBuild.classpath.toString,
        out.toString
      )
    )
    assertEquals(result.exit, 1, result.diagnostics)
    assert(!Files.exists(out), result.diagnostics)
    assertEquals(
      result.output.linesIterator.filter(_.startsWith("== ")).toList,
      List("== lint admitted sources")
    )
    assert(
      result.diagnostics.contains(stored + "Forward.scala:7: late is read"),
      result.diagnostics
    )
