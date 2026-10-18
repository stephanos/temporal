package framework.lint

import framework.check.Tools
import framework.inspect.Index
import java.nio.file.Files
import scala.collection.mutable
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import scala.quoted.Quotes
import scala.tasty.inspector.{Inspector, Tasty, TastyInspector}

class NativeClassAdmissionControlsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes

  private lazy val scratch =
    Files.createTempDirectory(
      Files.createDirectories(Tools.here.directory.resolve(".flow/tmp/native-class-admission")),
      "case-"
    )

  private case class Diagnostics(
      order: Vector[String],
      structure: Vector[String],
      style: Vector[String]
  ) derives CanEqual

  private val sections = """object transitions extends Transitions
    |object effects:
    |  def enable() =
    |    active = true
    |object invariants:
    |  val reflexive =
    |    check(active == active)
    |""".stripMargin

  private def indent(source: String): String =
    source.linesIterator.map("  " + _).mkString("\n")

  private def source(
      form: String,
      body: String = sections,
      outside: String = "",
      nested: Boolean = false
  ): String =
    val header = if form == "class" then "class Model[D: Finite](initial: D, enabled: Boolean)"
    else "object Model"
    val vars = if form == "class" then "var selected = initial\nvar active = enabled\n"
    else "var active = false\n"
    val model = header + " extends Machine[Outcome, Nothing]:\n" + indent(vars + body) + "\n"
    val pkg = if nested then "fixture.nativeclass" else "fixture.features.parametric"
    "package " + pkg + "\nimport framework.dsl.*\nimport framework.dsl.Finite.given\n" +
      "import framework.dsl.outcomes.Outcome\n" +
      (if nested then "object Vocabulary:\n" + indent(model) + "\n" else model) + outside

  private def lint(sourceText: String): Diagnostics =
    val directory = Files.createTempDirectory(scratch, "case-")
    val file = Files.createDirectories(directory.resolve("parametric")).resolve("Parametric.scala")
    Files.writeString(file, sourceText): Unit
    val classes = directory.resolve("classes")
    val classpath = System.getProperty("java.class.path")
    val flags = Seq(
      "-Werror",
      "-deprecation",
      "-feature",
      "-unchecked",
      "-Wunused:all",
      "-Wvalue-discard",
      "-Wnonunit-statement",
      "-Wsafe-init",
      "-Wimplausible-patterns",
      "-Yretain-trees",
      "-language:strictEquality"
    )
    val compiled = Tools.here.scalaCli(
      Seq(
        "compile",
        "--server=false",
        "--workspace",
        directory.resolve("workspace").toString,
        "--scala",
        "3.9.0",
        "--jvm",
        "27",
        "--classpath",
        classpath,
        "--compilation-output",
        classes.toString,
        file.toString
      ) ++ flags.flatMap(flag => Seq("--scalac-option", flag))
    )
    assertEquals(compiled.exit, 0, compiled.diagnostics)
    assert(!compiled.failed, compiled.diagnostics)
    val stream = Files.walk(classes)
    val paths = try
      stream.iterator.asScala.filter(_.toString.endsWith(".tasty")).map(_.toString).toList
    finally stream.close()
    assert(paths.nonEmpty)
    val batches = mutable.ArrayBuffer.empty[Diagnostics]
    val inspector = new Inspector:
      def inspect(using quotes: Quotes)(trees: List[Tasty[quotes.type]]): Unit =
        val index = Index(trees, Map.empty)
        batches += Diagnostics(
          Order(index).refusals.map(_.getMessage).toVector,
          Structure(index).refusals(_ => false).map(_._3.getMessage).toVector,
          DeclarationStyle(index).refusals(_ => false).map(_._3.getMessage).toVector
        ): Unit
    assert(
      TastyInspector.inspectAllTastyFiles(
        paths,
        Nil,
        classes.toString :: classpath.split(java.io.File.pathSeparator).toList
      )(inspector)
    )
    assertEquals(batches.size, 1)
    batches.head

  private def clean(diagnostics: Diagnostics): Unit =
    assertEquals(diagnostics.order, Vector.empty)
    assertEquals(diagnostics.structure, Vector.empty)
    assertEquals(diagnostics.style, Vector.empty)

  private def contains(errors: Vector[String], fragments: String*): Unit =
    assert(errors.exists(error => fragments.forall(error.contains)), errors.mkString("\n"))

  test("direct class members and section members retain declaration style controls"):
    for form <- List("class", "object") do
      val body = "def inlineHelper = true\n" + sections.replace(
        "object effects:\n",
        "object effects:\n  val inlineConstant = true\n"
      )
      val diagnostics = lint(source(form, body))
      assertEquals(diagnostics.order, Vector.empty)
      assertEquals(diagnostics.structure, Vector.empty)
      contains(diagnostics.style, "declaration-multiline", "inlineHelper")
      contains(diagnostics.style, "declaration-multiline", "inlineConstant")
      assertEquals(diagnostics.style.size, 2)

  test("native vars helpers types actors and named environments remain legal"):
    val helpers = """enum Mode:
      |  case idle, running
      |type Flag = Boolean
      |def helper(input: Boolean = true) =
      |  val local = input
      |  def nested = local
      |  nested
      |object client extends Actor:
      |  lazy val advance =
      |    action
      |object clock extends Internal:
      |  lazy val advance =
      |    step
      |  lazy val expire =
      |    timer
      |""".stripMargin
    val environment = """lazy val environment =
      |  without(client.advance)
      |object progress:
      |  val eventual =
      |    leadsTo(active, active, within = 1) in environment
      |object queries:
      |  val asked =
      |    verify(invariants.reflexive)
      |""".stripMargin
    val companion = """final case class Token(value: Boolean)
      |object Token:
      |  def same(value: Token) =
      |    value
      |""".stripMargin
    for form <- List("class", "object") do
      clean(lint(source(form, helpers + companion + sections + environment)))
      val borrowed = companion + """  lazy val companionQuery =
        |    queries.asked
        |  lazy val companionScenario =
        |    without(client.advance)
        |""".stripMargin
      val misplaced = lint(source(form, helpers + borrowed + sections + environment))
      contains(misplaced.order, "companionQuery", "Query", "belongs")
      contains(misplaced.order, "companionScenario", "Scenario", "belongs")
      assertEquals(misplaced.structure, Vector.empty)
      assertEquals(misplaced.style, Vector.empty)

  test("unknown nested and external sections are refused by the responsible passes"):
    for form <- List("class", "object") do
      val unknown = lint(source(form, sections + "object extras:\n  val value =\n    true\n"))
      contains(unknown.order, "extras", "none of its sections")
      contains(unknown.structure, "extras", "none of its sections")
      assertEquals(unknown.style, Vector.empty)
      val nested = lint(source(form, sections + "object progress:\n  object queries\n"))
      contains(nested.order, "queries", "section", "another section")
      assertEquals(nested.structure, Vector.empty)
      assertEquals(nested.style, Vector.empty)
      val external = lint(source(form, outside = "object queries\n"))
      contains(external.order, "queries", "top level")
      assertEquals(external.structure, Vector.empty)
      assertEquals(external.style, Vector.empty)
      val lateTransitions = sections.replace("object transitions extends Transitions\n", "") +
        "object transitions extends Transitions\n"
      val reversed = lint(source(form, lateTransitions))
      contains(reversed.order, "transitions", "belongs before")
      assertEquals(reversed.structure, Vector.empty)
      assertEquals(reversed.style, Vector.empty)

  test("native progress and query values retain kind placement checks"):
    val suffix = """object progress:
      |  val eventual =
      |    leadsTo(active, active, within = 1)
      |object queries:
      |  val asked =
      |    verify(invariants.reflexive)
      |""".stripMargin
    for form <- List("class", "object") do
      clean(lint(source(form, sections + suffix)))
      val body = sections
        .replace(
          "object effects:\n",
          "object effects:\n  lazy val misplacedQuery =\n    queries.asked\n"
        )
        .replace(
          "object invariants:\n",
          "object invariants:\n  lazy val misplacedProgress =\n    progress.eventual\n"
        ) + suffix.replace(
        "object progress:\n",
        "object progress:\n  lazy val misplacedProgressQuery =\n    queries.asked\n"
      )
      val diagnostics = lint(source(form, body))
      contains(diagnostics.order, "misplacedQuery", "Query", "belongs")
      contains(diagnostics.order, "misplacedProgress", "progress", "belongs")
      contains(diagnostics.order, "misplacedProgressQuery", "Query", "belongs")
      assertEquals(diagnostics.structure, Vector.empty)
      assertEquals(diagnostics.style, Vector.empty)

  test("nested native classes and objects receive the same checks as top-level forms"):
    for form <- List("class", "object") do
      clean(lint(source(form, nested = true)))
      val diagnostics = lint(
        source(
          form,
          sections + "val misplaced =\n  true\n",
          nested = true
        )
      )
      contains(diagnostics.order, "misplaced", "outside its sections", "class M", "object M")
      assertEquals(diagnostics.structure, Vector.empty)
      assertEquals(diagnostics.style, Vector.empty)

  test("constructor parameters and Finite evidence remain outside authored member checks"):
    clean(lint(source("class")))
    clean(lint(source("object")))
    val composition = """object Pair extends Composition[Nothing]:
      |  val left =
      |    Vocabulary.Model(initial = false, enabled = false)
      |  val right =
      |    Vocabulary.Model(initial = true, enabled = true)
      |  object syncs
      |  object postconditions:
      |    val stable =
      |      check(before.left.active == left.active)
      |""".stripMargin
    clean(lint(source("class", outside = composition, nested = true)))
