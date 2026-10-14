package temporal.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class QualifiedActionsTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task1QualifiedActions))
  private case class Rewritten(fixed: String, diagnostics: List[String])
  private def rewrite(source: String) =
    val prefix = "object Fixture {\n"
    val tree = dialects.Scala3(Input.String(prefix + source + "\n}")).parse[Source].get
    val result = rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)
    Rewritten(
      result.fixed.stripPrefix(prefix).stripSuffix("\n}"),
      result.diagnostics.map(_.message)
    )

  test("whole, class and multi targets qualify from explicit imports and converge"):
    for target <- List(
        "poll",
        "failed(Failure.fatal)",
        "poll, failed(Failure.fatal)",
        "poll, failed(Failure.fatal), pause",
        "poll, failed(Failure.fatal), pause, stop"
      )
    do
      val source =
        "import worker.{poll, failed, pause, stop}\non(" + target + ") { always ~> effects.keep }"
      val result = rewrite(source)
      assertEquals(result.diagnostics, Nil)
      for name <- List("poll", "failed", "pause", "stop") if target.contains(name) do
        assert(result.fixed.contains("worker." + name), result.fixed)
      assert(result.fixed.contains("Failure.fatal") == target.contains("Failure.fatal"))
      assertEquals(rewrite(result.fixed).fixed, result.fixed)

  test(
    "from wildcard imports qualify only their grouped actions, leaving qualified targets intact"
  ):
    val source =
      "from(worker) { import worker.*; on(poll, failed(Failure.fatal)) { always ~> effects.keep } }"
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assert(result.fixed.contains("on(worker.poll, worker.failed(Failure.fatal))"))
    assert(!result.fixed.contains("import worker"))
    assertEquals(rewrite(result.fixed).fixed, result.fixed)
    val qualified = "on(worker.poll) { always ~> effects.keep }\ndisabled(worker.stop)"
    assertEquals(rewrite(qualified).fixed, qualified)

  test("ambiguous imports and shadowed targets refuse the entire document"):
    for source <- List(
        "import worker.poll\nimport other.poll\non(poll) { always ~> effects.keep }",
        "import worker.*\non(poll) { always ~> effects.keep }",
        "def elsewhere = { import worker.poll; poll }\non(poll) { always ~> effects.keep }",
        "import worker.poll\nval poll = other.poll\non(poll) { always ~> effects.keep }",
        "import worker.poll\ndef f(worker: Any) = { on(poll) { always ~> effects.keep } }",
        "from(worker) { import other.*; on(worker.poll) { always ~> effects.keep } }"
      )
    do
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)

  test(
    "disabled and selective renamed imports use the declaring member without rewriting literals"
  ):
    val source = "import worker.{stop as halt}\n// halt\nval text = \"halt\"\ndisabled(halt)"
    val result = rewrite(source)
    assertEquals(result.diagnostics, Nil)
    assertEquals(result.fixed, "\n// halt\nval text = \"halt\"\ndisabled(worker.stop)")
