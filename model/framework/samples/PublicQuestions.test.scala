package samples

import scala.compiletime.testing.typeCheckErrors

class PublicQuestionsTest extends munit.FunSuite:
  test("public questions preserve external holder naming and concrete execution context"):
    PublicQuestionsClient.run()
    import PublicQuestionsClient.*
    import framework.checker.{BudgetPolicy, Checker, Expectation, ReplayScript, ReplayStep, Verdict}
    import framework.dsl.State
    val outside = new Counter(Holder.one)
    val space = Checker.space(outside, steps = 2)()
    val policy = BudgetPolicy.parse(Some(s"default=7\n${this.getClass.getName}=3"))
    val execution = Checker.forSuite(policy, this.getClass.getName)
    val sampled = execution.sample(space, count = 2, seed = 17L)
    assertEquals(sampled.verdict, Verdict.refuted)
    assertEquals(sampled.budget, 3)
    assertEquals(sampled.subject, "outside.sample")
    assertEquals(sampled.sampling.get.seed, 17L)
    assertEquals(sampled.sampling.get.walks.size, 2)
    assertEquals(sampled.sampling.get.failingIndex, Some(0))
    assertEquals(
      sampled.sampling.get.walks,
      Checker.sample(space, count = 2, seed = 17L, budget = 3).sampling.get.walks
    )
    val replayed =
      execution.replay(outside, ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    assertEquals(replayed.verdict, Verdict.verified)
    val found =
      execution.find(space, 1)(Expectation(1, "outside.value == 1", _ => outside.value == 1))
    assertEquals(found.verdict, Verdict.found)
    val checked = Vector(
      replayed,
      found,
      execution.explore(outside),
      execution.refuted(outside, outside.invariants.belowOne),
      execution.refuted(outside, outside.monitors.first),
      execution.monitor(outside, outside.monitors.first, outside.initialStates)
    ) ++ execution.verifyAll(outside).results
    assert(checked.forall(_.budget == 3))

  test("public space and replay refuse anonymous class Model construction"):
    val space = typeCheckErrors("""
      import samples.PublicQuestionsClient.*
      framework.checker.Checker.space(new Counter(Holder.one), steps = 1)()
    """)
    assert(space.exists(_.message.contains("stable val holder")), space.toString)
    val replay = typeCheckErrors("""
      import samples.PublicQuestionsClient.*
      import framework.checker.{ReplayScript, ReplayStep}
      import framework.dsl.State
      framework.checker.Checker.replay(new Counter(Holder.one), ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    """)
    assert(replay.exists(_.message.contains("stable val holder")), replay.toString)
    val suite = typeCheckErrors("""
      import samples.PublicQuestionsClient.*
      import framework.checker.{BudgetPolicy, Checker, ReplayScript, ReplayStep}
      import framework.dsl.State
      val execution = Checker.forSuite(BudgetPolicy.parse(None), "samples.PublicQuestionsTest")
      execution.replay(new Counter(Holder.one), ReplayScript[State](Vector(ReplayStep(actor.advance()))))
    """)
    assert(suite.exists(_.message.contains("stable val holder")), suite.toString)
