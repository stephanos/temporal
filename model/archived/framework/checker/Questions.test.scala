package checkercontrol

import framework.dsl.*
import framework.dsl.Finite.given
import framework.checker.{
  Checker,
  Expectation,
  ReplayExpectation,
  ReplayOperation,
  ReplayScript,
  ReplayStep,
  Verdict
}

object QuestionsFixture:
  enum Answer:
    case ok
  enum Note:
    case left, right
  given Ok[Answer] = Ok(Answer.ok)

  object actor extends Actor:
    val advance = action
    val idle = action

  object Counter extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        if value < 3 then effects.advance()
      on(actor.idle):
        effects.idle()

    object effects:
      def advance(): Unit = value += 1
      def idle(): Unit = ()

  object Missing extends Machine[Answer, Nothing]:
    var value = 0
    val gap = hole

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = if gap.reached then value += 1

  object Boundary extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit =
        if value == 2 then throw IllegalStateException("past horizon")
        value += 1

  object Branching extends Machine[Answer, Note]:
    var value = 0
    val left = choice
    val right = choice

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = choose(
        left -> { value = 1; records(Note.left) },
        right -> { value = 1; records(Note.right) }
      )

class QuestionsTest extends munit.FunSuite:
  import QuestionsFixture.*

  test("positioned replay reports the second expectation and retains executed evidence"):
    val script = ReplayScript[Counter.State](
      Vector(
        ReplayStep(actor.advance()),
        ReplayExpectation(Expectation(11, "value == 1", _ => Counter.value == 1)),
        ReplayStep(actor.advance()),
        ReplayExpectation(Expectation(22, "value == 9", _ => Counter.value == 9))
      )
    )
    val result = Checker.replay(Counter, script)
    assertEquals(result.verdict, Verdict.refuted)
    assertEquals(
      result.path.states,
      Vector(Counter.init, Counter.init.copy(value = 1), Counter.init.copy(value = 2))
    )
    assertEquals(result.expectation.map(_.line), Some(22))
    assertEquals(result.expectation.map(_.text), Some("value == 9"))
    assertEquals(result.expectation.map(_.state), Some(Counter.init.copy(value = 2)))

  test("replay refuses disabled first steps and later suffixes and preserves successful paths"):
    val completed =
      Checker.replay(Counter, ReplayScript[Counter.State](Vector(ReplayStep(actor.advance()))))
    assertEquals(completed.verdict, Verdict.verified)
    assertEquals(completed.path.steps.size, 1)
    for prefix <- Vector[Vector[ReplayOperation[Counter.State]]](
        Vector.empty,
        Vector(ReplayExpectation(Expectation[Counter.State](1, "true", _ => true)))
      )
    do
      val result = Checker.replay(
        Counter,
        ReplayScript(prefix :+ ReplayStep(actor.advance()), Seq(Counter.init.copy(value = 3)))
      )
      assertEquals(result.verdict, Verdict.refuted)
      assert(result.detail.mkString.contains("advance"))
      assert(result.detail.mkString.contains("3"))
    val suffix = Checker.replay(
      Counter,
      ReplayScript[Counter.State](
        Vector(
          ReplayStep(actor.advance()),
          ReplayExpectation(Expectation(1, "value == 3", _ => Counter.value == 3)),
          ReplayStep(actor.advance())
        ),
        Seq(Counter.init.copy(value = 2))
      )
    )
    assertEquals(suffix.verdict, Verdict.refuted)
    assertEquals(suffix.path.steps.size, 1)
    assert(suffix.detail.mkString.contains("disabled step 2"))

  test("find includes the horizon boundary and never executes beyond it"):
    val space = Checker.space(Counter, steps = 3)((_, cls) => cls.decl eq actor.advance.decl)
    val atBoundary =
      Checker.find(space, horizon = 2)(Expectation(1, "value == 2", _ => Counter.value == 2))
    assertEquals(atBoundary.verdict, Verdict.found)
    assertEquals(atBoundary.path.steps.size, 2)
    val beyond =
      Checker.find(space, horizon = 2)(Expectation(1, "value == 3", _ => Counter.value == 3))
    assertEquals(beyond.verdict, Verdict.notFound)
    assert(beyond.complete)
    assertEquals(beyond.depth, 2)
    val bounded = Checker.find(Checker.space(Boundary, steps = 3)(), horizon = 2)(
      Expectation(1, "false", _ => false)
    )
    assertEquals(bounded.verdict, Verdict.notFound)

  test("replay keeps equal-state branches with distinct positioned facts"):
    val script = ReplayScript[Branching.State](
      Vector(
        ReplayStep(actor.advance()),
        ReplayExpectation(Expectation(31, "left recorded", _ => Branching.recorded(Note.left)))
      )
    )
    val result = Checker.replay(Branching, script)
    assertEquals(result.verdict, Verdict.refuted)
    assertEquals(result.path.steps.map(_.facts), Vector(List(Note.right)))
    assertEquals(result.expectation.map(_.line), Some(31))
    val cut = Checker.replay(Branching, script, budget = 1)
    assertEquals(cut.verdict, Verdict.unknown)
    assert(!cut.complete)

  test("replay and find preserve holes as unknown and replay budgets as cuts"):
    val replay =
      Checker.replay(Missing, ReplayScript[Missing.State](Vector(ReplayStep(actor.advance()))))
    assertEquals(replay.verdict, Verdict.unknown)
    assert(!replay.complete)
    val find = Checker.find(Checker.space(Missing, steps = 2)(), horizon = 2)(
      Expectation(1, "false", _ => false)
    )
    assertEquals(find.verdict, Verdict.unknown)
    assert(!find.complete)
    val cut = Checker.replay(
      Counter,
      ReplayScript[Counter.State](Vector(ReplayStep(actor.advance()))),
      budget = 1
    )
    assertEquals(cut.verdict, Verdict.unknown)
    assert(!cut.complete)

  test("find distinguishes a state budget cut from a horizon and keeps decisive evidence"):
    val space = Checker.space(Counter, steps = 3)((_, cls) => cls.decl eq actor.advance.decl)
    val cut = Checker.find(space, horizon = 3, budget = 1)(
      Expectation(1, "value == 3", _ => Counter.value == 3)
    )
    assertEquals(cut.verdict, Verdict.unknown)
    assert(!cut.complete)
    val witness = Checker.find(space, horizon = 3, budget = 1)(
      Expectation(1, "value == 1", _ => Counter.value == 1)
    )
    assertEquals(witness.verdict, Verdict.found)
    assertEquals(witness.path.steps.size, 1)

  test("position-sensitive find revisits states before selecting later actions"):
    val space = Checker.space(Counter, steps = 3)((position, cls) =>
      cls.decl eq (if position < 2 then actor.idle.decl else actor.advance.decl)
    )
    val result = Checker.find(space, horizon = 3, budget = 3)(
      Expectation(1, "value == 1", _ => Counter.value == 1)
    )
    assertEquals(result.verdict, Verdict.found)
    assertEquals(result.path.steps.size, 3)
    val far = Checker.space(Counter, steps = 100000)((position, cls) =>
      cls.decl eq (if position < 99999 then actor.idle.decl else actor.advance.decl)
    )
    val cut = Checker.find(far, horizon = 100000, budget = 2)(
      Expectation(1, "value == 1", _ => Counter.value == 1)
    )
    assertEquals(cut.verdict, Verdict.unknown)
    assert(!cut.complete)
    assertEquals(cut.depth, 1)

  test("find refuses absent targets and nonpositive horizons"):
    val space = Checker.space(Counter, steps = 1)()
    for horizon <- Vector(0, -1) do
      intercept[IllegalArgumentException](
        Checker.find(space, horizon)(Expectation(1, "true", _ => true))
      )
    val _ =
      intercept[IllegalArgumentException](Checker.find(space, 2)(Expectation(1, "true", _ => true)))
    val _ = intercept[IllegalArgumentException](Checker.find(space, 1)(null))
    val _ = intercept[IllegalArgumentException](Checker.space(Counter, steps = 0)())
    for budget <- Vector(0, -1) do
      val _ = intercept[IllegalArgumentException](
        Checker.find(space, 1, budget)(Expectation(1, "true", _ => true))
      )
      intercept[IllegalArgumentException](
        Checker.replay(Counter, ReplayScript[Counter.State](Vector.empty), budget)
      )
