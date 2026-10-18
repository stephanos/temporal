package checkercontrol

import framework.dsl.*
import framework.dsl.Finite.given
import framework.checker.{Checker, Result, Verdict}

object CheckerFixture:
  enum Answer:
    case ok
  given Ok[Answer] = Ok(Answer.ok)

  object actor extends Actor:
    val advance = action

  object Counter extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        if value < 2 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

    object invariants:
      val bounded = check(value < 2)
      val initialFailure = check(false) in actions()

    object postconditions:
      val pinned = check(value < 2) in actions(actor.advance(), actor.advance())

    object monitors:
      val limited = monitor(0)(n => n + 1)(_ >= 2)
      val immediate = monitor(false)(_ => true)(identity)

    object progress:
      val impossible = leadsTo(from = value == 0, to = value == 3, within = 5)
      val slow = leadsTo(from = value == 0, to = value == 2, within = 1)

  object Loop extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit = value = 1

    object progress:
      val cycling = leadsTo(from = value == 0, to = value == 2, within = 5)

class CheckerTest extends munit.FunSuite:
  import CheckerFixture.*

  test("public exploration and claims share Result and preserve concrete counterexample steps"):
    val graph: Result = Checker.explore(Counter, Counter.initialStates, budget = 3)
    assertEquals(graph.verdict, Verdict.verified)
    assertEquals(graph.budget, 3)
    assert(graph.complete)
    val broken = Checker.verifyAll(graph).result("bounded")
    assertEquals(broken.verdict, Verdict.refuted)
    assertEquals(
      broken.path.states,
      Vector(Counter.init, Counter.init.copy(value = 1), Counter.init.copy(value = 2))
    )
    assertEquals(broken.path.steps.map(_.outcome), Vector(Answer.ok, Answer.ok))

    assertEquals(
      broken.path.steps.map(_.action.decl),
      Vector(actor.advance.decl, actor.advance.decl)
    )
    val scoped = Checker.verifyAll(graph)
    assertEquals(scoped.result("pinned").path.states, broken.path.states)
    assertEquals(scoped.result("initialFailure").path.states, Vector(Counter.init))
    assertEquals(Counter.value, 0)

  test("public budget refusal and a cut preserve the unknown answer"):
    for budget <- Vector(0, -1) do
      val questions: Vector[() => Any] = Vector(
        () => Checker.explore(Counter, Counter.initialStates, budget),
        () => Checker.verifyAll(Counter, budget = budget),
        () => Checker.monitor(Counter, Counter.monitors.limited, Counter.initialStates, budget),
        () => Checker.progress(Counter, Counter.progress.impossible, Counter.initialStates, budget)
      )
      questions.foreach: question =>
        val refused = intercept[IllegalArgumentException](question())
        assert(refused.getMessage.contains("budget"))
    val graph = Checker.explore(Counter, Counter.initialStates, budget = 1)
    assertEquals(graph.verdict, Verdict.unknown)
    assert(!graph.complete)
    assertEquals(Checker.verifyAll(graph).result("bounded").verdict, Verdict.unknown)

  test("public monitor refutation retains the path's states and outcomes"):
    val broken: Result =
      Checker.monitor(Counter, Counter.monitors.limited, Counter.initialStates, budget = 3)
    assertEquals(broken.verdict, Verdict.refuted)
    assertEquals(
      broken.path.states,
      Vector(Counter.init, Counter.init.copy(value = 1), Counter.init.copy(value = 2))
    )
    assertEquals(broken.path.steps.map(_.outcome), Vector(Answer.ok, Answer.ok))

    val beforeCut =
      Checker.monitor(Counter, Counter.monitors.immediate, Counter.initialStates, budget = 1)
    assertEquals(beforeCut.verdict, Verdict.refuted)
    assert(!beforeCut.complete)
    assertEquals(beforeCut.path.states, Vector(Counter.init, Counter.init.copy(value = 1)))

  test("public progress keeps deadlock, cycle and deadline reasons and concrete paths"):
    val cases = Vector(
      (
        Counter,
        Counter.progress.impossible,
        "deadlock",
        Vector(Counter.init, Counter.init.copy(value = 1), Counter.init.copy(value = 2))
      ),
      (
        Counter,
        Counter.progress.slow,
        "path of",
        Vector(Counter.init, Counter.init.copy(value = 1))
      ),
      (
        Loop,
        Loop.progress.cycling,
        "cycle",
        Vector(Loop.init, Loop.init.copy(value = 1), Loop.init.copy(value = 1))
      )
    )
    for (machine, claim, reason, states) <- cases do
      val broken: Result = Checker.progress(machine, claim, machine.initialStates, budget = 3)
      assertEquals(broken.verdict, Verdict.refuted)
      assertEquals(broken.path.states, states)
      assert(broken.path.steps.forall(step => java.util.Objects.equals(step.outcome, Answer.ok)))
      assert(broken.detail.mkString.contains(reason), broken.toString)
