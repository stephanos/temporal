package framework.lint

import framework.check.Tools
import framework.inspect.{Index, LiftError}
import java.nio.file.Files
import scala.collection.mutable
import scala.concurrent.duration.*
import scala.jdk.CollectionConverters.*
import scala.quoted.Quotes
import scala.tasty.inspector.{Inspector, Tasty, TastyInspector}

class NativeClassFormsTest extends munit.FunSuite:
  override val munitTimeout: Duration = 10.minutes

  private def lint(form: String, extra: String): Vector[String] =
    val root = Files.createDirectories(Tools.here.directory.resolve(".flow/tmp/native-class-lint"))
    val scratch = Files.createTempDirectory(root, "case-")
    val source = Files.createDirectories(scratch.resolve("parametric")).resolve("Parametric.scala")
    val header = if form == "class" then "class Model[D: Finite](initial: D, enabled: Boolean)"
    else "object Model"
    val state = if form == "class" then "  var selected = initial\n  var active = enabled\n"
    else "  var active = false\n"
    Files.writeString(
      source,
      "package fixture.features.parametric\nimport framework.dsl.*\nimport framework.dsl.Finite.given\n" +
        "import framework.dsl.outcomes.Outcome\n" + header + " extends Machine[Outcome, Nothing]:\n" + state +
        "  object transitions extends Transitions\n" +
        "  object effects:\n    def enable() =\n      active = true\n" +
        "  object invariants:\n    val reflexive =\n      check(active == active)\n" + extra
    ): Unit
    val classes = scratch.resolve("classes")
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
        scratch.resolve("workspace").toString,
        "--scala",
        "3.9.0",
        "--jvm",
        "27",
        "--classpath",
        classpath,
        "--compilation-output",
        classes.toString,
        source.toString
      ) ++ flags.flatMap(flag => Seq("--scalac-option", flag))
    )
    assertEquals(compiled.exit, 0, compiled.diagnostics)
    assert(!compiled.failed, compiled.diagnostics)
    val stream = Files.walk(classes)
    val paths = try
      stream.iterator.asScala.filter(_.toString.endsWith(".tasty")).map(_.toString).toList
    finally stream.close()
    assert(paths.nonEmpty)
    val errors = mutable.ArrayBuffer.empty[LiftError]
    val inspector = new Inspector:
      def inspect(using quotes: Quotes)(trees: List[Tasty[quotes.type]]): Unit =
        errors ++= Order(Index(trees, Map.empty)).refusals
    assert(
      TastyInspector.inspectAllTastyFiles(
        paths,
        Nil,
        classes.toString :: classpath.split(java.io.File.pathSeparator).toList
      )(inspector)
    )
    errors.map(_.getMessage).toVector

  test("native class and object sections have the same structure rules"):
    for form <- List("class", "object") do
      assertEquals(lint(form, ""), Vector.empty)
      val refused = lint(form, "  val misplaced =\n    true\n")
      assert(
        refused.exists(error =>
          error.contains("misplaced") && error.contains("outside its sections")
        ),
        refused.mkString("\n")
      )
