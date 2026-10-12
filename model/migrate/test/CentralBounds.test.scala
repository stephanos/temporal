package umpire.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class CentralBoundsTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task5CentralBounds))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  test("known budgets centralize by budget, including Wide clashes and control, and converge"):
    for (old, name, steps, search) <- List(
        ("two", "two", 2, 512),
        ("three", "three", 3, 4096),
        ("four", "four", 4, 32768),
        ("five", "five", 5, 65536),
        ("six", "six", 6, 262144),
        ("seven", "seven", 7, 262144),
        ("control", "eight", 8, 262144),
        ("twelve", "twelve", 12, 262144),
        ("four", "fourWide", 4, 1048576),
        ("five", "fiveWide", 5, 1048576),
        ("twelve", "twelveWide", 12, 4194304)
      )
    do
      val source =
        s"def ask = { val $old = Limits(steps = $steps, actions = $steps, search = $search); consume($old) }"
      val result = rewrite(source)
      assertEquals(result.diagnostics, Nil)
      assertEquals(result.fixed, s"def ask = consume(temporal.Bounds.$name)")
      assertEquals(rewrite(result.fixed).fixed, result.fixed)
      assertEquals(rewrite(result.fixed).diagnostics, Nil)

  test("visible central imports retain accepted use sites and shift literals are supported"):
    val source =
      "import temporal.Bounds.fourWide\ndef ask = { val old = Limits(steps = 4, actions = 4, search = 1 << 20); consume(old) }"
    assertEquals(
      rewrite(source).fixed,
      "import temporal.Bounds.fourWide\ndef ask = consume(fourWide)"
    )

  test("unknown budgets, public bindings, ambiguous names and unsafe uses refuse atomically"):
    for bad <- List(
        "val x = Limits(steps = 9, actions = 9, search = 512); consume(x)",
        "val x = Limits(steps = n, actions = 2, search = 512); consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); def f(x: Any) = x; consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); import Other.x; consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); consume(owner.x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); val temporal = other; consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); object x; consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); val two = other; consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); /* keep */ consume(x)",
        "val x = Limits(steps = 2, actions = 2, search = 512); val y = Limits(steps = 3, actions = 3, search = 4096); consume(x, y)"
      )
    do
      val source = s"def ask = { $bad }"
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)
      assert(result.diagnostics.head.message.contains("manually"))
    val exported = "object Exported { val two = Limits(steps = 2, actions = 2, search = 512) }"
    assertEquals(rewrite(exported).fixed, exported)
    assert(rewrite(exported).diagnostics.nonEmpty)
    val shadowed =
      "val temporal = other\ndef ask = { val x = Limits(steps = 2, actions = 2, search = 512); consume(x) }"
    assertEquals(rewrite(shadowed).fixed, shadowed)
    assert(rewrite(shadowed).diagnostics.nonEmpty)
    val shadowedImport = "import temporal.Bounds.two\nval two = other\n" +
      "def ask = { val x = Limits(steps = 2, actions = 2, search = 512); consume(x) }"
    assertEquals(rewrite(shadowedImport).fixed, shadowedImport)
    assert(rewrite(shadowedImport).diagnostics.nonEmpty)

  test("a refusal cancels supported changes and preserves unrelated strings and comments"):
    val source = "def a = { val x = Limits(steps = 2, actions = 2, search = 512); consume(x) }\n" +
      "def b = { val y = Limits(steps = 99, actions = 99, search = 512); consume(y) }"
    assertEquals(rewrite(source).fixed, source)
    assert(rewrite(source).diagnostics.nonEmpty)
    val accepted =
      "// Limits and control\nval text = \"Limits(control)\"\nval x = temporal.Bounds.eight"
    assertEquals(rewrite(accepted).fixed, accepted)
    assertEquals(rewrite(accepted).diagnostics, Nil)
    val central =
      java.nio.file.Files.readString(java.nio.file.Path.of("model/temporal/Bounds.scala"))
    assertEquals(rewrite(central).fixed, central)
    assertEquals(rewrite(central).diagnostics, Nil)
