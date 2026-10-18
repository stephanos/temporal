package framework.lint

import framework.`export`.{FixtureBuild, ModelBuild}
import framework.check.{RetainedScratch, Tools}
import FixtureBuild.buildFixture
import java.nio.file.{Files, Path}
import java.util.zip.ZipFile
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import scala.tasty.inspector.TastyInspector

class EffectDefinitionsSuite extends munit.FunSuite with RetainedScratch:
  override val munitTimeout: Duration = 10.minutes
  private val tools = Tools.here
  private lazy val scratch =
    retained(tools.directory.resolve("model/build/history"), "effect-definitions.")
  private lazy val library = ModelBuild.frameworkJarArgument.takeWhile(_ != '=')
  private lazy val plugin = ModelBuild.reifyPlugin:
    val jar = scratch.resolve("reify.jar")
    tools
      .scalaCli(
        Seq(
          "--power",
          "package",
          "--server=false",
          "--assembly",
          "--preamble=false",
          "--main-class-in-manifest=false",
          "--provided",
          "org.scala-lang:scala3-compiler_3",
          "model/framework/reify/plugin",
          "-f",
          "-o",
          jar.toString
        )
      )
      .orFail(): Unit
    jar

  private def compile(
      name: String,
      members: String,
      facet: Boolean = false,
      derived: Boolean = false,
      extra: String = "",
      inside: String = ""
  ) =
    val sources = Files.createDirectories(scratch.resolve(name))
    val base = if derived then
      "object Base extends Machine[Boolean, Nothing]:\n  var flag = false\n  object transitions extends Transitions\n"
    else ""
    val owner = if derived then "object Subject extends Derived(Base.unmonitored):"
    else if facet then "trait Subject extends Facet[Boolean, Nothing]:"
    else "object Subject extends Machine[Boolean, Nothing]:"
    val section =
      if facet then "  object behavior:\n    object effects:\n" else "  object effects:\n"
    val indent = if facet then "      " else "    "
    Files.writeString(
      sources.resolve("Fixture.scala"),
      "package framework.effectdefinitions\nimport framework.dsl.*\ngiven Ok[Boolean] = Ok(true)\n" + base + owner +
        (if derived then "\n" else "\n  var flag = false\n") + section + members.linesIterator
          .map(indent + _)
          .mkString("\n") +
        (if !facet && !derived then "\n  object transitions extends Transitions"
         else
           "") + inside + "\n  def helper: Int =\n    1\nobject Unrelated:\n  object effects:\n    def value: Int =\n      1\n" + extra
    ): Unit
    Files.writeString(
      sources.resolve("project.scala"),
      s"//> using scala 3.9.0\n//> using jvm 27\n//> using jar $library\n"
    ): Unit
    val jar = scratch.resolve(s"$name.jar")
    val ran = tools.buildFixture(
      Seq(
        "--power",
        "package",
        "--server=false",
        "--library",
        sources.toString,
        "-f",
        "-o",
        jar.toString,
        "--scalac-option",
        s"-Xplugin:$plugin",
        "--scalac-option",
        "-P:umpire-reify:mode=signature"
      )
    )
    (ran, jar)

  test("unused native Machine and Facet effect definitions must return exactly Unit"):
    for
      (facet, derived) <- Vector((false, false), (true, false), (false, true))
      (name, body) <- Vector(
        "inferred" -> "def unused = 1",
        "annotated" -> "private def unused(input: Int): Int = input",
        "bottom" -> "def unused = throw RuntimeException()"
      )
    do
      val (ran, _) = compile(
        s"$name-$facet-$derived",
        body + "\ndef authored$default$helper = 1",
        facet,
        derived
      )
      assertNotEquals(ran.exit, 0, ran.diagnostics)
      assert(ran.diagnostics.contains("effect definitions must return Unit"), ran.diagnostics)
      assert(
        ran.diagnostics.contains("authored$default$helper: effect definitions must return Unit"),
        ran.diagnostics
      )

  test("inferred and explicitly annotated Unit effects compile independently of style"):
    val (ran, _) = compile(
      "unit",
      "def inferred =\n  flag = true\ndef annotated(input: Boolean): Unit =\n  flag = input\ndef defaulted(input: Boolean = true) =\n  flag = input"
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    val (derived, _) = compile("derived-unit", "def inferred =\n  ()", derived = true)
    assertEquals(derived.exit, 0, derived.diagnostics)

  private def lint(jar: Path): String =
    val zip = ZipFile(jar.toFile)
    val paths = try
      zip.entries.asScala
        .filter(_.getName.endsWith(".tasty"))
        .map { entry =>
          val path = scratch.resolve("tasty").resolve(entry.getName)
          Files.createDirectories(path.getParent): Unit
          val input = zip.getInputStream(entry)
          try Files.copy(input, path, java.nio.file.StandardCopyOption.REPLACE_EXISTING): Unit
          finally input.close()
          path.toString
        }
        .toList
    finally zip.close()
    val prefix = jar.getParent.resolve(jar.getFileName.toString.stripSuffix(".jar")).toString + "/"
    val checker = ConventionInspector(paths.map(_ -> prefix).toMap, ModulePolicy.Generic)
    assert(
      TastyInspector.inspectAllTastyFiles(
        paths,
        Nil,
        jar.toString :: library :: ModelBuild.classpathEntries
          .split(java.io.File.pathSeparator)
          .toList
          .drop(1)
      )(checker)
    )
    checker.errors.map(_.getMessage).mkString("\n")

  test("effect style rejects return annotations and single-line bodies at native sections only"):
    for facet <- Vector(false, true) do
      val (ran, jar) = compile(
        s"style-$facet",
        "def inferred(input: Boolean) =\n  flag = input\ndef annotated(input: Boolean): _root_.scala.Unit =\n  flag = input\ndef qualified: scala.Unit =\n  flag = false\ndef styled$default$helper: Unit =\n  flag = false\ndef inline = { flag = true }\ndef inlineFirst = { flag = true\n  flag = false\n}\ndef multilineParameters(\n  input: Boolean\n) = flag = input",
        facet
      )
      assertEquals(ran.exit, 0, ran.diagnostics)
      val errors = lint(jar)
      assertEquals(errors.linesIterator.count(_.contains("effect-return-type")), 3, errors)
      assertEquals(errors.linesIterator.count(_.contains("declaration-multiline")), 3, errors)
      assert(!errors.contains(": inferred "), errors)
      assert(!errors.contains("Unrelated"), errors)

  test(
    "Model object members use multiline bodies; vars, locals and outer declarations stay inline"
  ):
    val (ran, jar) = compile(
      "declarations",
      "def effect =\n  flag = true",
      inside = """object Declarations extends Actor:
        |  val send = action
        |  val constant = 1
        |  val signed = -1
        |  val text = "value"
        |  val boolean = true
        |  val character = 'a'
        |  val unit = ()
        |  val folded = 1 + 2
        |  def helper(input: Boolean) = input
        |  def answer = 100
        |  def named$default$member = 100
        |  def locals =
        |    val fromBalance = identity(1)
        |    def nested = fromBalance
        |    nested
        |  val closure =
        |    () => { val lambdaLocal = identity(1); lambdaLocal }
        |  class Nested:
        |    val classMember = identity(1)
        |  val (inlineFirst, inlineSecond) = identity((1, 2))
        |  val (firstValue, secondValue) =
        |    identity((1, 2))
        |  def comment =
        |    /* equals = in trivia */ true
        |  def stringDefault(value: String = "/*") =
        |    value.length
        |  val expanded =
        |    1
        |  def expandedHelper =
        |    true
        |  var counter = 0
        |  def parameters(
        |    input: Boolean
        |  ) = input
        |""".stripMargin.linesIterator
        .map("  " + _)
        .mkString("\n", "\n", "\n"),
      extra =
        "object Outside extends Actor:\n  val outsideAction = action\n  val outsideComputed = identity(1)\n  val outsideConstant = 100\n  def outsideHelper = 100\n"
    )
    assertEquals(ran.exit, 0, ran.diagnostics)
    val errors = lint(jar)
    assertEquals(errors.linesIterator.count(_.contains("declaration-multiline")), 13, errors)
    for name <- Vector(
        "send",
        "constant",
        "signed",
        "text",
        "boolean",
        "character",
        "unit",
        "folded",
        "helper",
        "answer",
        "named$default$member",
        "parameters"
      )
    do assert(errors.contains(s": $name "), errors)
    assert(errors.contains(": val pattern "), errors)
    for name <- Vector(
        "expanded",
        "expandedHelper",
        "counter",
        "fromBalance",
        "nested",
        "lambdaLocal",
        "classMember",
        "firstValue",
        "secondValue",
        "comment",
        "stringDefault",
        "outsideAction",
        "outsideComputed",
        "outsideConstant",
        "outsideHelper"
      )
    do assert(!errors.contains(s": $name "), errors)
