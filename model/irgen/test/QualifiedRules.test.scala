package umpire.irgen

import java.nio.file.Files
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import umpire.check.Tools

class QualifiedRulesTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private val stored = "model/irgen/testdata/qualifiedRules/"
  private val modelJar = ModelBuild.jar
  private val scratch = umpire.check.Scratch.directory("qualified.")
  private val jar = scratch.resolve("fixture.jar")
  private val forms = Vector("Whole", "Class", "Two", "Three", "Four")
  private val scopes = Vector("Outside", "InFrom")

  override def beforeAll(): Unit =
    val source = tools.directory.resolve(stored)
    for name <- Vector("Shared.scala", "Accepted.scala", "RuleRejects.scala") do
      Files.copy(source.resolve(name), scratch.resolve(name)): Unit
    Files.writeString(
      scratch.resolve("project.scala"),
      s"//> using scala 3.9.0\n//> using jvm 27\n//> using jar $modelJar\n"
    ): Unit
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          scratch.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit

  private def lift(names: Seq[String]) =
    val output = scratch.resolve(names.head + ".json")
    val ran = InProcess.lift(
      tools.directory,
      Seq(
        s"$jar=$stored,$modelJar=model/",
        ModelBuild.classpath.toString,
        output.toString
      ) ++ names.map("fixture.qualifiedrules." + _)
    )
    (ran, output)

  test("qualified whole, class and two-to-four targets lift inside and outside from"):
    val (ran, output) = lift(for form <- forms; scope <- scopes yield form + scope)
    assertEquals(ran.exit, 0, ran.diagnostics)
    val ir = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(output))
    assertEquals(ir.path("machines").size(), 10)
    val ids = ir.path("actions").elements().asScala.map(_.path("id").asText()).toSet
    assertEquals(ids, Set("press", "turn", "stop", "reset").map("fixture.qualifiedrules.hand." + _))

  test("bare targets independently refuse every compiled form at its authored source coordinate"):
    val names = for form <- forms; scope <- scopes yield "Bare" + form + scope
    val (ran, output) = lift(names)
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(!Files.exists(output))
    val source = Files.readAllLines(tools.directory.resolve(stored + "RuleRejects.scala")).asScala
    val bareLines = source.zipWithIndex
      .collect {
        case (line, index) if line.trim.startsWith("on(press") || line.trim.startsWith("on(turn") =>
          (index + 1, if line.trim.startsWith("on(turn") then "turn" else "press")
      }
      .take(10)
    assertEquals(bareLines.size, 10)
    for (line, action) <- bareLines do
      assert(ran.diagnostics.contains(s"RuleRejects.scala:$line:"), ran.diagnostics)
      assert(ran.diagnostics.contains(s"on(hand.$action)"), ran.diagnostics)
    assertEquals(ran.diagnostics.linesIterator.count(_.contains("names its action bare")), 10)

  test("from imports, foreign ownership and bare disabled keep independent located refusals"):
    val (ran, output) = lift(
      Vector(
        "ImportedWildcard",
        "ImportedSelective",
        "Foreign",
        "BareDisabled",
        "BareDisabledInFrom"
      )
    )
    assertEquals(ran.exit, 1, ran.diagnostics)
    assert(!Files.exists(output))
    val source = Files.readAllLines(tools.directory.resolve(stored + "RuleRejects.scala")).asScala
    for
      (text, index) <- source.zipWithIndex if text.trim == "import hand.*" ||
        text.trim == "import hand.press" && index > source.indexWhere(
          _.contains("object ImportedSelective")
        ) ||
        text.trim.startsWith("on(clock.tick)") || text.trim == "disabled(tick)"
    do assert(ran.diagnostics.contains(s"RuleRejects.scala:${index + 1}:"), ran.diagnostics)
    assertEquals(ran.diagnostics.linesIterator.count(_.contains("a from only groups")), 2)
    assert(ran.diagnostics.contains("hand declares no tick"), ran.diagnostics)
    assertEquals(ran.diagnostics.linesIterator.count(_.contains("disabled(tick)")), 2)
    assert(ran.diagnostics.contains("disabled(clock.tick)"), ran.diagnostics)
