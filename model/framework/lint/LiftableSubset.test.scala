package framework.lint

import java.nio.file.{Files, Path}
import scala.concurrent.duration.*
import framework.check.{RetainedScratch, Tools}
import framework.`export`.{FixtureBuild, ModelBuild}
import framework.`export`.InProcess
import scala.jdk.CollectionConverters.*
import FixtureBuild.buildFixture

class LiftableSubsetFixtures extends munit.FunSuite with RetainedScratch:
  override val munitTimeout = 10.minutes
  private val tools = Tools.here
  private lazy val scratch = retained(tools.directory.resolve(".flow/tmp"), "liftable-subset.")

  test("each documented body row has an exercised fixture and lowering"):
    val source = Files.createDirectory(scratch.resolve("supported"))
    Files.copy(
      tools.directory.resolve("model/framework/export/testdata/lifts/Subset.scala"),
      source.resolve("Subset.scala")
    ): Unit
    Files.writeString(
      source.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = source.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          source.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.lintConventions",
        s"$jar=$source/",
        ModelBuild.classpath.toString
      )
    )
    assertEquals(result.exit, 0, result.output)
    val out = source.resolve("fixture.json")
    InProcess
      .lift(
        tools.directory,
        Seq(
          s"$jar=$source/,${ModelBuild.jarArgument}",
          ModelBuild.classpath.toString,
          out.toString,
          "fixture.subset.Checked",
          "fixture.subset.Subset$package$.predicate",
          "fixture.subset.EmptyDraft"
        )
      )
      .orFail(): Unit
    val model = new com.fasterxml.jackson.databind.ObjectMapper().readTree(Files.readString(out))
    val functions = model
      .path("functions")
      .elements()
      .asScala
      .map(f => f.path("name").asText().split('.').last -> f)
      .toMap
    for (name, kind) <- Seq(
        "booleanLiteral" -> "literal",
        "integerLiteral" -> "literal",
        "textLiteral" -> "literal",
        "reference" -> "var",
        "field" -> "field",
        "called" -> "call",
        "constructed" -> "construct",
        "wrapped" -> "call",
        "localValue" -> "let",
        "conditional" -> "if",
        "matched" -> "match",
        "listed" -> "list",
        "optional" -> "if",
        "stepRecord" -> "list",
        "equal" -> "binary",
        "unequal" -> "binary",
        "conjoined" -> "binary",
        "disjoined" -> "binary",
        "negated" -> "unary",
        "less" -> "binary",
        "lessEqual" -> "binary",
        "greater" -> "binary",
        "greaterEqual" -> "binary",
        "added" -> "binary",
        "subtracted" -> "binary",
        "contained" -> "binary",
        "copied" -> "copy",
        "minimum" -> "if",
        "maximum" -> "if",
        "concatenated" -> "binary",
        "prepended" -> "binary",
        "converted" -> "call",
        "contracted" -> "field",
        "interpolated" -> "literal",
        "predicate" -> "field"
      )
    do
      assert(functions.contains(name), s"unexercised subset fixture $name")
      assert(functions(name).path("body").has(kind), s"$name: ${functions(name)}")

  test("author lint names unsupported trees and library operations in predicates and steps"):
    val source = Files.createDirectory(scratch.resolve("refused"))
    Files.writeString(
      source.resolve("Unsupported.scala"),
      """package framework.fixture.subset
import framework.dsl.*
def mutation(s: Boolean): List[Step[Boolean, Boolean, Nothing]] =
  var value = s
  value = !value
  List(Step(true, value))
def loop(s: Boolean): Boolean =
  while s do ()
  s
def library(s: Boolean): Boolean = s.toString.startsWith("t")
def mapping(s: Boolean): Boolean = List(s).map(x => x).contains(s)
def finding(s: Boolean): Boolean = List(s).find(x => x).contains(s)
def declaration = List(true).map(x => x).toVector
class Reader(val read: Boolean)
def fresh(s: Boolean): Boolean = (new Reader(s)).read
def foreign(s: java.util.List[Boolean]): Boolean = s.contains(true)
def stringContains(s: Boolean): Boolean = s.toString.contains("t")
given observed: Boolean = System.nanoTime() > 0L
@sugar def contextual(s: Boolean)(using observed: Boolean): Boolean = observed && s
val press = action(Actor("user"))
object Checked extends Machine[Boolean, Boolean, Nothing]:
  val init = false
  def end(s: Boolean) = loop(s) || library(s) || mapping(s) || finding(s) || fresh(s) || foreign(null) || stringContains(s) || contextual(s)
  object rules extends Bindings(press ~> mutation)
"""
    ): Unit
    Files.writeString(
      source.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = source.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          source.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val result = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.lintConventions",
        s"$jar=$source/,${ModelBuild.jarArgument}",
        ModelBuild.classpath.toString
      )
    )
    assertNotEquals(result.exit, 0, result.output)
    for construct <- Seq(
        "var value",
        "while",
        "startsWith",
        "map",
        "find",
        "new",
        "contains",
        "nanoTime"
      )
    do assert(result.output.contains(construct), result.output)
    assert(result.output.contains("model/SEMANTICS.md#liftable-subset"), result.output)
    assert(!result.output.contains("Unsupported.scala:13:"), result.output)

  test("author checking and lifting preserve their distinct positioned refusal contracts"):
    val source = Files.createDirectory(scratch.resolve("legacy-refusal"))
    Files.copy(
      tools.directory.resolve("model/framework/export/testdata/unsupported/Unsupported.scala"),
      source.resolve("Unsupported.scala")
    ): Unit
    Files.writeString(
      source.resolve("project.scala"),
      "//> using scala 3.9.0\n//> using jvm 27\n"
    ): Unit
    val jar = source.resolve("fixture.jar")
    tools
      .buildFixture(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--library",
          "--classpath",
          ModelBuild.classpathEntries,
          source.toString,
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    val arguments = Seq(
      s"$jar=$source/,${ModelBuild.jarArgument}",
      ModelBuild.classpath.toString,
      source.resolve("fixture.json").toString,
      "fixture.unsupported.Unsupported"
    )
    val lifted = InProcess.lift(tools.directory, arguments)
    assertNotEquals(lifted.exit, 0)
    assert(
      lifted.diagnostics.contains("Unsupported.scala:18: `var out` has no IR form"),
      lifted.diagnostics
    )
    val checked = tools.run(
      Path.of(System.getProperty("java.home"), "bin", "java").toString,
      Seq(
        "-cp",
        System.getProperty("java.class.path"),
        "framework.export.checkAndLift"
      ) ++ arguments
    )
    assertNotEquals(checked.exit, 0)
    assert(
      checked.output.contains("Unsupported.scala:18: liftable subset: var out"),
      checked.output
    )
    assert(checked.output.contains("while"), checked.output)
    assert(checked.output.contains("model/SEMANTICS.md#liftable-subset"), checked.output)
