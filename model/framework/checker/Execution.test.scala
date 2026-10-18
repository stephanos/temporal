package checkercontrol

import framework.dsl.*
import framework.checker.{BudgetPolicy, Checker, Expectation, ReplayScript, ReplayStep, Verdict}

object ExecutionFixture:
  enum Answer:
    case ok
  given Ok[Answer] = Ok(Answer.ok)

  object actor extends Actor:
    val advance = action

  object Broken extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(actor.advance):
        effects.advance()

    object effects:
      def advance(): Unit =
        value += 1
        throw IllegalStateException("effect exploded")

    object invariants:
      val bounded = check(value < 2)

  object Watched extends Machine[Answer, Nothing]:
    var value = 0
    val missing = hole

    object transitions extends Transitions:
      on(actor.advance):
        if value < 2 then effects.advance()

    object effects:
      def advance(): Unit = value += 1

    object monitors:
      val broken = monitor(0)(n => n + 1)(_ >= 2)
      val safe = monitor(0)(n => n + 1)(_ > 3)
      val uncertain = monitor(false)(_ => missing.reached)(identity)

class ExecutionTest extends munit.FunSuite:
  import ExecutionFixture.*

  test("a model effect failure is refuted with its pre-step state and action"):
    val result = Checker.explore(Broken, budget = 3)
    assertEquals(result.verdict, Verdict.refuted)
    assertEquals(result.budget, 3)
    assert(!result.complete)
    assertEquals(result.path.states, Vector(Broken.init))
    assert(result.detail.exists(_.contains("effect exploded")))
    assert(result.detail.exists(_.contains("advance")))
    assert(result.detail.exists(_.contains(Broken.init.toString)))
    val report = Checker.verifyAll(Broken, budget = 3)
    assert(report.results.exists(_.verdict == Verdict.refuted))
    assertEquals(Broken.value, 0)

  test("two suite boundaries in one JVM pass their resolved budget to every question"):
    val model = CheckerFixture.Counter
    val policy = BudgetPolicy.parse(Some("default=7\nclient.SmallSuite=1\nclient.LargeSuite=4"))
    for (name, budget) <- Vector(
        "client.SmallSuite" -> 1,
        "client.LargeSuite" -> 4,
        "client.OtherSuite" -> 7
      )
    do
      val run = Checker.forSuite(policy, name)
      val space = Checker.space(model, steps = 2)()
      val answers = Vector(
        run.explore(model),
        run.replay(
          model,
          ReplayScript[model.State](Vector(ReplayStep(CheckerFixture.actor.advance())))
        ),
        run.find(space, 1)(Expectation(1, "false", _ => false)),
        run.sample(space, 1, 42L),
        run.refuted(model, model.invariants.bounded),
        run.monitor(model, model.monitors.limited, model.initialStates),
        run.progress(model, model.progress.impossible, model.initialStates)
      ) ++ run.verifyAll(model).results
      assert(answers.nonEmpty)
      assert(answers.forall(_.budget == budget), answers.map(_.toString).mkString("\n"))
      assertEquals(answers(1).verdict, if budget == 1 then Verdict.unknown else Verdict.verified)

  test("refuted returns ordinary verified and unknown claim answers without assertion inversion"):
    val verified = Checker.refuted(
      framework.checker.VacuityFixture.FiniteModel,
      framework.checker.VacuityFixture.FiniteModel.invariants.safe,
      budget = 2
    )
    assertEquals(verified.verdict, Verdict.verified)
    val unknown = Checker.refuted(
      framework.checker.VacuityFixture.FiniteModel,
      framework.checker.VacuityFixture.FiniteModel.invariants.safe,
      budget = 1
    )
    assertEquals(unknown.verdict, Verdict.unknown)
    assert(!unknown.complete)

  test("monitor negative controls return ordinary results through raw and suite execution"):
    val cases = Vector(
      (Watched.monitors.broken, Verdict.refuted),
      (Watched.monitors.safe, Verdict.verified),
      (Watched.monitors.uncertain, Verdict.unknown)
    )
    val execution =
      Checker.forSuite(BudgetPolicy.parse(Some("client.MonitorSuite=3")), "client.MonitorSuite")
    for (claim, verdict) <- cases do
      val raw = Checker.refuted(Watched, claim, budget = 3)
      val configured = execution.refuted(Watched, claim)
      assertEquals(raw.verdict, verdict)
      assertEquals(configured.verdict, verdict)
      assertEquals(raw.budget, 3)
      assertEquals(configured.budget, 3)
      assertEquals(raw.path, configured.path)
    val cut = Checker.refuted(Watched, Watched.monitors.safe, budget = 1)
    assertEquals(cut.verdict, Verdict.unknown)
    assert(!cut.complete)
    val from = Checker.refuted(Watched, Watched.monitors.broken, starts = Watched.initialStates)
    assertEquals(from.verdict, Verdict.refuted)
