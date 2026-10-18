package framework.checker

import framework.dsl.{Checker as _, *}
import framework.dsl.Finite.given

object SamplingFixture:
  enum Answer:
    case ok
  given Ok[Answer] = Ok(Answer.ok)

  object actor extends Actor:
    val advance = action
    val set = action.input[Boolean](name = "active")
    val explode = action

  object Counter extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = value = (value + 1) % 3

    object invariants:
      val bounded = check(value < 2)

  object Watched extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = value = value

    object monitors:
      val second = monitor(0)(n => n + 1)(_ >= 2)

  object Branch extends Machine[Answer, Nothing]:
    var active = false

    object transitions extends Transitions:
      on(actor.set): value =>
        effects.set(value)
      on(actor.explode):
        effects.explode()

    object effects:
      def set(value: Boolean): Unit = active = value
      def explode(): Unit = throw IllegalStateException("unselected effect")

    object invariants:
      val inactive = check(!active)

  object Scoped extends Machine[Answer, Nothing]:
    var active = false

    object transitions extends Transitions:
      on(actor.set): value =>
        effects.set(value)

    object effects:
      def set(value: Boolean): Unit = active = value

    object postconditions:
      val off = check(!active) in only(actor.set(false))
      val pinned = check(!active) in actions(actor.set(false), actor.set(false))
      val anotherStart = check(false) in only(actor.set).starts(active = true)

  object Deadline extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = value = value

    object progress:
      val pending = leadsTo(from = value == 0, to = value == 1, within = 2)

  object Unbounded extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = value = value

    object progress:
      val pending = whenever(value == 0):
        eventually(value == 1)

  object Terminal extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        if value == 0 then effects.advance()

    object effects:
      def advance(): Unit = value = 1

    object progress:
      val pending = whenever(value == 0):
        eventually(value == 2)

  object Uncertain extends Machine[Answer, Nothing]:
    var value = 0
    val missing = hole

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit =
        if value == 0 then value = 1
        else if missing.reached then value += 1

    object invariants:
      val bounded = check(value < 2)

class SamplingTest extends munit.FunSuite:
  import SamplingFixture.*

  test("sample retains seeded ordered walks and the selected counterexample"):
    val space = Checker.space(Counter, steps = 3)()
    val first = Checker.sample(space, count = 3, seed = 19L, budget = 4)
    val again = Checker.sample(space, count = 3, seed = 19L, budget = 4)
    assertEquals(first.verdict, Verdict.refuted)
    assertEquals(first.sampling.get.walks, again.sampling.get.walks)
    assertEquals(first.sampling.get.walks.size, 3)
    assertEquals(first.sampling.get.seed, 19L)
    assertEquals(first.sampling.get.failingIndex, Some(0))
    assertEquals(
      first.path.states,
      Vector(Counter.init, Counter.init.copy(value = 1), Counter.init.copy(value = 2))
    )

  test("sample carries monitor memory across repeated model states"):
    val result =
      Checker.sample(Checker.space(Watched, steps = 3)(), count = 1, seed = 1L, budget = 3)
    assertEquals(result.verdict, Verdict.refuted)
    assertEquals(result.path.steps.size, 2)

  test("sample refuses an empty experiment"):
    val error = intercept[IllegalArgumentException](
      Checker.sample(Checker.space(Counter, steps = 3)(), count = 0, seed = 1L, budget = 3)
    )
    assert(error.getMessage.contains("count"))

  test("sampling judges only admitted selected paths and never executes an excluded effect"):
    val space =
      Checker.space(Branch, steps = 2)((_, cls) => ClaimEvaluation.selects(actor.set(false), cls))
    val sampled = Checker.sample(space, count = 2, seed = 1L, budget = 3)
    assertEquals(sampled.verdict, Verdict.verified)
    assert(sampled.sampling.get.complete)
    assert(
      sampled.sampling.get.walks
        .forall(_.steps.forall(s => java.util.Objects.equals(s.after, Branch.init)))
    )
    assert(sampled.toString.contains("finite sample complete"))
    assert(
      sampled.detail.exists(_.contains("no exhaustive safety or unbounded progress conclusion"))
    )
    val unsafe = Checker.sample(
      Checker.space(Branch, steps = 1)((_, cls) => ClaimEvaluation.selects(actor.set(true), cls)),
      count = 1,
      seed = 1L,
      budget = 3
    )
    assertEquals(unsafe.verdict, Verdict.refuted)
    assertEquals(unsafe.path.states, Vector(Branch.init, Branch.init.copy(active = true)))
    val branching =
      Checker.space(Branch, steps = 3)((_, cls) => ClaimEvaluation.selects(actor.set, cls))
    val random = Checker.sample(branching, count = 5, seed = 7L, budget = 3)
    val repeated = Checker.sample(branching, count = 5, seed = 7L, budget = 3)
    assertEquals(random.sampling.get.walks, repeated.sampling.get.walks)
    assert(random.sampling.get.walks.distinct.size > 1)

  test("scoped safety follows exact starts admission and pinned positions on the sampled prefix"):
    for active <- Vector(false, true) do
      val space = Checker.space(Scoped, steps = 2)((position, cls) =>
        ClaimEvaluation.selects(actor.set(position == 0 && active), cls)
      )
      val result = Checker.sample(space, count = 1, seed = 2L, budget = 3)
      assertEquals(result.verdict, Verdict.verified)
      assertEquals(result.sampling.get.walks.head.steps.size, 2)

  test("sampled progress distinguishes bounded violation from finite unfinished obligations"):
    for (steps, verdict) <- Vector(1 -> Verdict.unknown, 2 -> Verdict.refuted) do
      val result =
        Checker.sample(Checker.space(Deadline, steps = steps)(), count = 1, seed = 1L, budget = 3)
      assertEquals(result.verdict, verdict)
      assertEquals(result.sampling.get.complete, true)
      if verdict == Verdict.refuted then assertEquals(result.path.steps.size, 2)
    val unbounded =
      Checker.sample(Checker.space(Unbounded, steps = 3)(), count = 1, seed = 1L, budget = 3)
    assertEquals(unbounded.verdict, Verdict.unknown)
    assert(unbounded.detail.exists(_.contains("unmet progress obligation")))
    for allow <- Vector(true, false) do
      val terminal = Checker.sample(
        Checker.space(Terminal, steps = 3)((_, _) => allow),
        count = 1,
        seed = 1L,
        budget = 3
      )
      assertEquals(terminal.verdict, if allow then Verdict.refuted else Verdict.unknown)
      if allow then
        assertEquals(terminal.path.states, Vector(Terminal.init, Terminal.init.copy(value = 1)))
        assert(terminal.detail.exists(_.contains("deadlock")))

  test("sampling keeps holes and cuts unknown and preserves decisive evidence before either"):
    val cases = Vector(
      (Uncertain.init, 1, 1, Verdict.unknown),
      (Uncertain.init, 2, 3, Verdict.unknown),
      (Uncertain.init.copy(value = 2), 2, 3, Verdict.refuted)
    )
    for (start, steps, budget, verdict) <- cases do
      val result = Checker.sample(
        Checker.space(Uncertain, steps = steps, starts = Seq(start))(),
        count = 1,
        seed = 1L,
        budget = budget
      )
      assertEquals(result.verdict, verdict)
      assert(!result.complete)
      assert(!result.sampling.get.complete)
      if verdict == Verdict.refuted then assertEquals(result.path.states, Vector(start))
    val cutFailure =
      Checker.sample(Checker.space(Counter, steps = 2)(), count = 1, seed = 1L, budget = 2)
    assertEquals(cutFailure.verdict, Verdict.refuted)
    assertEquals(cutFailure.path.steps.size, 2)
    assert(!cutFailure.complete)
