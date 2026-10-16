package temporal.migrate

import scala.meta.*
import scalafix.v1.SyntacticDocument
import scalafix.internal.v1.Rules

class QualifiedPhasesTest extends munit.FunSuite:
  private val rules = Rules(List(new Fn159Task3QualifiedPhases))
  private def rewrite(source: String) =
    val tree = dialects.Scala3(Input.String(source)).parse[Source].get
    rules.syntacticPatch(SyntacticDocument.fromTree(tree), suppress = false)

  private val prefix =
    "enum Stage { case idle, busy, done }\n" +
      "enum Phase { case other }\n" +
      "object Job extends Machine[State, Said, Fact], Phased[State, Stage](_.stage) {\n"

  test("wildcard, selective and renamed phase imports qualify every reference and converge"):
    for (imported, bare) <- List(
        ("import Stage.*", "idle"),
        ("import Stage.idle", "idle"),
        ("import Stage.{idle as waiting}", "waiting")
      )
    do
      val body = s"$imported\nval init = State($bare)\n" +
        s"object states { def selected(s: State) = s.stage == $bare }\n" +
        s"object effects { def update(s: State) = enter(s.copy(stage = $bare)) }\n" +
        s"object rules { on(hand.press) { when($bare) ~> effects.update } }\n" +
        s"object properties { val p = property holds (a => a.state.stage == $bare) }\n}"
      val source = prefix + body
      val result = rewrite(source)
      assertEquals(result.diagnostics, Nil)
      assertEquals(result.fixed, prefix + body.replace(imported, "").replace(bare, "Stage.idle"))
      assertEquals(rewrite(result.fixed).fixed, result.fixed)
      assertEquals(rewrite(result.fixed).diagnostics, Nil)

  test("phase type comes from Phased and unrelated enum members and Property when remain intact"):
    val source = prefix + "import Phase.other\nval unrelated = other\n" +
      "object rules { when(Stage.idle); when[Closed]; when(states.created) }\n" +
      "object properties { val p = property when hand.press holds (_.state.stage == Stage.busy) }\n}"
    val result = rewrite(source)
    assertEquals(result.fixed, source)
    assertEquals(result.diagnostics, Nil)

  test("ambiguous, shadowed and unknown wildcard members refuse atomically"):
    for body <- List(
        "import Stage.idle\nimport Other.idle\nval init = State(idle)",
        "import Stage.idle, Other.busy\nval init = State(idle)",
        "import Stage.idle\nimport Other.*\nval init = State(idle)",
        "import Stage.idle\nval idle = Other.idle\nval init = State(idle)",
        "import Stage.idle\ndef f(Stage: Any) = idle",
        "import Stage.{idle as waiting}\ndef f(waiting: Any) = waiting",
        "import Stage.{idle as _}\nval init = State(Stage.idle)"
      )
    do
      val source = prefix + body + "\n}"
      val result = rewrite(source)
      assertEquals(result.fixed, source)
      assert(result.diagnostics.nonEmpty)
    val unknown = "trait T extends Phased[S, Mode] { import Mode.*; val init = S(idle) }"
    val result = rewrite(unknown)
    assertEquals(result.fixed, unknown)
    assert(result.diagnostics.nonEmpty)

  test(
    "phase imports at file, machine, section and rules positions preserve comments and literals"
  ):
    for (before, after) <- List(
        ("import Stage.idle\n", ""),
        ("", "import Stage.idle; "),
        ("", "object states { import Stage.idle; val selected = idle }; "),
        ("", "object rules { import Stage.idle; when(idle) }; ")
      )
    do
      val source =
        before + prefix + after + "val init = State(idle)\n// idle\nval text = \"idle\"\n}"
      val result = rewrite(source)
      if after.startsWith("object") then
        assert(result.diagnostics.nonEmpty)
        assertEquals(result.fixed, source)
      else
        assertEquals(result.diagnostics, Nil)
        assert(result.fixed.contains("State(Stage.idle)"), result.fixed)
        assert(result.fixed.contains("// idle\nval text = \"idle\""), result.fixed)
