package temporal.migrate

import java.nio.file.{Files, Path}
import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules
import scala.jdk.CollectionConverters.*

class DslPackageTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn169Task1DslPackage))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  test("every import and reference of the DSL through framework gains .dsl, and it converges"):
    val legacy =
      """package temporal.features.lamp
        |
        |import framework.*
        |import framework.Finite.given
        |import framework.outcomes.{Outcome, Rejection}
        |import framework.{Observation as Obs, Machine}
        |import framework.realize.*
        |import framework.realize.{Reason, RunExpectation}
        |
        |val _ = () => summon[CanEqual[framework.outcomes.Outcome, framework.outcomes.Outcome]]
        |type Closed = framework.Closed
        |val laws = _root_.framework.IrFile.discover("temporal")
        |val wait = framework.realize.Elapsed.milliseconds(250)
        |// framework.Machine in a comment, and in "framework.Machine" a literal
        |""".stripMargin
    val accepted =
      """package temporal.features.lamp
        |
        |import framework.dsl.*
        |import framework.dsl.Finite.given
        |import framework.dsl.outcomes.{Outcome, Rejection}
        |import framework.dsl.{Observation as Obs, Machine}
        |import framework.realize.*
        |import framework.realize.{Reason, RunExpectation}
        |
        |val _ = () => summon[CanEqual[framework.dsl.outcomes.Outcome, framework.dsl.outcomes.Outcome]]
        |type Closed = framework.dsl.Closed
        |val laws = _root_.framework.dsl.IrFile.discover("temporal")
        |val wait = framework.realize.Elapsed.milliseconds(250)
        |// framework.Machine in a comment, and in "framework.Machine" a literal
        |""".stripMargin
    val result = rewrite(legacy)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, accepted)
    // Each line keeps its place: positions in the IR are by line.
    assertEquals(result.fixed.linesIterator.size, legacy.linesIterator.size)
    assertEquals(rewrite(result.fixed).fixed, result.fixed)

  test("a test in package framework moves into framework.dsl, naming the others from the root"):
    val legacy =
      "package framework\n\nimport realize.*\nclass ChoicesTest extends munit.FunSuite:\n" +
        "  val e: realize.Elapsed = ???\n  val m: Machine[?, ?, ?] = ???\n"
    assertEquals(
      rewrite(legacy).fixed,
      "package framework.dsl\n\nimport framework.realize.*\nclass ChoicesTest extends " +
        "munit.FunSuite:\n  val e: framework.realize.Elapsed = ???\n  val m: Machine[?, ?, ?] = ???\n"
    )
    // `import framework.*` brought `realize` into scope as well.
    val imported = "package temporal\n\nimport framework.*\nval run = realize.RunExpectation\n"
    assertEquals(
      rewrite(imported).fixed,
      "package temporal\n\nimport framework.dsl.*\nval run = framework.realize.RunExpectation\n"
    )
    // Outside package framework, a name `realize` is someone else's.
    val other = "package temporal\n\nval x = realize.Elapsed\n"
    assertEquals(rewrite(other).fixed, other)

  test("the framework's other packages, and other roots named framework, are left alone"):
    for source <- List(
        "import framework.realize.Instruction.*\nimport framework.check.Tools\n",
        "import framework.`export`.lift\nimport framework.migrate.x\n",
        "val a = other.framework.Machine\nval b = framework.dsl.Machine\n"
      )
    do assertEquals(rewrite(source).fixed, source)

  test("an import that names a framework package beside a DSL name refuses the document"):
    for source <- List(
        "import framework.{realize, Machine}\nval m = framework.Machine\n",
        "import framework.{realize, *}\n"
      )
    do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assertEquals(result.diagnostics.size, 1)
      assert(result.diagnostics.head.message.contains("beside a name of its DSL"))

  test("the committed mini and shared sources are the rule's fixed point"):
    val roots = List("model/temporal/features/activity_mini", "model/temporal/foundations")
    val sources = roots.flatMap: root =>
      val stream = Files.walk(Path.of(root))
      try
        stream.iterator.asScala
          .filter(p => p.toString.endsWith(".scala") && !p.toString.contains("/generated/"))
          .toList
      finally stream.close()
    assert(sources.nonEmpty)
    for path <- sources do
      val text = Files.readString(path)
      assertEquals(rewrite(text).fixed, text, path.toString)
