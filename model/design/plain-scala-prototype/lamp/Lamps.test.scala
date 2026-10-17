package proto

import plain.*

class LampsTest extends munit.FunSuite:
  import Lamps.*

  private def table(native: Native) =
    for
      a <- List(hand.press, hand.smash, hand.fix)
      l <- Lit.values.toList
      p <- 0 to 3
    yield
      val s = native.state("light" -> l, "presses" -> p)
      (a.toString, s, native.steps(s, a))

  test("a machine assembled from facets steps as the machine written as one object"):
    val faceted = table(Runtime(Faceted))
    assertEquals(faceted, table(Runtime(Whole)))
    assertEquals(table(Runtime(Reordered)), faceted)
    assert(faceted.exists(_._3.nonEmpty))

  test("facet order is the extends clause's: pressing then breaking, or the reverse"):
    assertEquals(
      Runtime(Faceted).bindings.map(b => s"${b.facet}:${b.action}"),
      List("pressing:hand.press", "breaking:hand.smash", "breaking:hand.fix")
    )
    assertEquals(
      Runtime(Reordered).bindings.map(b => s"${b.facet}:${b.action}"),
      List("breaking:hand.smash", "breaking:hand.fix", "pressing:hand.press")
    )

  test("an action bound by two facets is refused, naming both"):
    val refused = intercept[IllegalArgumentException](Runtime(Twice).bindings)
    assertEquals(
      refused.getMessage,
      "press is bound by the facets pressing and pressingAgain: an action belongs to one facet, " +
        "which holds all its transitions"
    )

  test("the phase table: a | b if g, _: Role with reject, _ => () as disabled"):
    val n = Runtime(Whole)
    def s(l: Lit, p: Int) = n.state("light" -> l, "presses" -> p)
    assertEquals(
      n.steps(s(Lit.off, 0), hand.press),
      List(Result.Fired(List("toggle"), Nil, s(Lit.on, 1)))
    )
    assertEquals(n.steps(s(Lit.on, 3), hand.press), Nil) // worn out: case _ => ()
    assertEquals(n.steps(s(Lit.broken, 1), hand.press), List(Result.Rejected(Said.broken)))

  test("a choose with blocks gives one step per alternative"):
    val n = Runtime(Whole)
    def s(l: Lit, p: Int) = n.state("light" -> l, "presses" -> p)
    assertEquals(
      n.steps(s(Lit.on, 1), hand.smash),
      List(
        Result.Fired(List("smash"), List("0"), s(Lit.broken, 1)),
        Result.Fired(List("smash"), List("1"), s(Lit.off, 2))
      )
    )

  test("sequential reads: a read after a write in one effect sees the write"):
    val n = Runtime(Whole)
    assertEquals(
      n.steps(n.state("light" -> Lit.broken, "presses" -> 3), hand.fix),
      List(Result.Fired(List("repair"), Nil, n.state("light" -> Lit.off, "presses" -> 0)))
    )

  test("UpTo[3] erases to Int in Scala; the schema keeps the bound and the runtime enforces it"):
    assertEquals(
      Runtime(Whole).schema.map(v => (v.toString, v.bound)),
      List(("light: Lit", None), ("presses: UpTo[3]", Some(3)))
    )
    // the alias survives in the symbol's info too, not only in the definition's tree
    assertEquals(Runtime(Whole).schema.map(_.info), List("Lit", "UpTo[3]"))
    val refused = intercept[IllegalArgumentException](
      Runtime(Unbounded).steps(Runtime(Unbounded).state(), hand.press)
    )
    assertEquals(refused.getMessage, "requirement failed: presses: 4 is out of UpTo[3]")

  test("the if/match transitions' arms, as first-match nested ifs"):
    val arms = Runtime(Whole).bindings.map(b => b.action.toString -> b.arms)
    arms.foreach((a, as) => println(s"$a\n  ${as.mkString("\n  ")}"))
    assertEquals(
      arms.head._2,
      List(
        "light matches (Lit.off | Lit.on) if presses.<(3) => effects.toggle",
        "!(light matches (Lit.off | Lit.on) if presses.<(3)) && " +
          "light matches (_: Terminated) => reject(Said.broken)",
        "!(light matches (Lit.off | Lit.on) if presses.<(3)) && " +
          "!(light matches (_: Terminated)) && light matches _ => disabled"
      )
    )

  test("the default checks run on a Machine with a choose"):
    assertEquals(
      Runtime(Whole).verifyAll(),
      Map("invariants.brokenOnlyFromUse" -> Verdict.Verified)
    )
