package framework.checker

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

class SuiteExecutionTest extends munit.FunSuite:
  test("suite questions preserve their shared budget, target conjunction and seeded descriptor"):
    val instance = new Counter
    class Questions extends ModelSuite(instance):
      object explorations:
        val space = explore(steps = 3):
          model.client.increment
      def target = explorations.space.find(steps = 2):
        expect:
          model.count > 0
          model.count == before.count + 1
          outcome == Outcome.accepted
          records(true)
      def absent = explorations.space.find(steps = 2):
        expect:
          model.count == 9
          true
      def sample = explorations.space.sample(3, seed = 19L)
    val suite = new Questions
    val target = suite.target
    val absent = suite.absent
    val sample = suite.sample
    val execution =
      SuiteExecution.forSuite("client.Questions", BudgetPolicy.parse(Some("default=8")))
    val found = execution.find(instance, target, None)
    val missed = execution.find(instance, absent, None)
    val first = execution.sample(instance, sample, None)
    val again = execution.sample(instance, sample, None)
    val result = found.evidence.asInstanceOf[Result]
    val missing = missed.evidence.asInstanceOf[Result]
    val sampled = first.evidence.asInstanceOf[Result]
    assert(found.verified)
    assert(!missed.verified)
    assertEquals(result.verdict, Verdict.found)
    assertEquals(result.path.steps.size, 1)
    assertEquals(missing.verdict, Verdict.notFound)
    assert(missing.complete)
    assertEquals(sampled.verdict, Verdict.verified)
    assertEquals(sampled.sampling.get.walks, again.evidence.asInstanceOf[Result].sampling.get.walks)
    assertEquals((sample.count, sample.seed, sample.space.steps), (3, 19L, 3))
    assertEquals(sampled.sampling.get.requested, 3)
    assertEquals(sampled.sampling.get.steps, 3)
    assertEquals(sampled.sampling.get.seed, 19L)
    assertEquals(sampled.sampling.get.failingIndex, None)
    assert(Vector(result, missing, sampled).forall(_.budget == 8))
    assertEquals(instance.count, 0)

  test("find preserves positioned target order, cuts, effect failures and restored nested context"):
    val instance = new Counter
    var observed = Vector.empty[String]
    def observe(label: String, condition: => Boolean): Boolean =
      observed :+= label
      condition
    class Questions extends ModelSuite(instance):
      object explorations:
        val space = explore(steps = 2):
          model.client.increment
          model.client.increment
      def query(first: Boolean) = explorations.space.find(steps = 2):
        expect:
          observe("first", first)
          observe("second", model.count > 0)
          observe("third", true)
      def secondFalse = explorations.space.find(steps = 2):
        expect:
          observe("first", true)
          observe("second", false)
          observe("third", true)
      def missing = explorations.space.find(steps = 2):
        expect(model.count == 9)
      def witness = explorations.space.find(steps = 2):
        expect(model.count == 1)
      def holeTarget = explorations.space.find(steps = 2):
        expect(model.count == 0 && model.gap.reached)
    val suite = new Questions
    val execution = SuiteExecution.forSuite("client.Context", BudgetPolicy.parse(Some("default=8")))
    val cut = SuiteExecution.forSuite("client.Cut", BudgetPolicy.parse(Some("default=1")))
    val early = suite.query(false)
    val later = suite.query(true)
    assertEquals(observed, Vector.empty)
    val failed = execution.find(instance, early, None)
    assertEquals(failed.evidence.asInstanceOf[Result].verdict, Verdict.notFound)
    assertEquals(observed, Vector("first", "first", "first"))
    assert(failed.diagnostics.contains("observe(\"first\", first)"))
    observed = Vector.empty
    val secondFailure = execution.find(instance, suite.secondFalse, None)
    assert(!secondFailure.verified)
    assert(secondFailure.diagnostics.contains("observe(\"second\", false)"))
    assertEquals(observed, Vector("first", "second", "first", "second", "first", "second"))
    observed = Vector.empty
    assert(execution.find(instance, later, None).verified)
    assertEquals(observed, Vector("first", "second", "first", "second", "third"))
    val unknown = cut.find(instance, suite.missing, None)
    assert(unknown.inconclusive)
    assert(!unknown.verified)
    assertEquals(unknown.evidence.asInstanceOf[Result].verdict, Verdict.unknown)
    assert(cut.find(instance, suite.witness, None).verified)
    assert(execution.find(instance, suite.holeTarget, None).inconclusive)
    val replay = RunBody.capture(instance):
      instance.client.increment
      expect:
        execution.find(instance, suite.witness, None).verified
        !execution.find(instance, suite.missing, None).verified
        before.count == 0
        outcome == Outcome.accepted
        records(true)
        instance.count == 1
    assertEquals(SuiteExecution.run(replay, budget = 8).verdict, Verdict.verified)
    assertEquals(instance.count, 0)
    val broken = new Failures
    class Broken extends ModelSuite(broken):
      object explorations:
        val space = explore(steps = 2):
          model.client.explode
      def target = explorations.space.find(steps = 1):
        expect(false)
    val failure = execution.find(broken, new Broken().target, None)
    val result = failure.evidence.asInstanceOf[Result]
    assertEquals(result.verdict, Verdict.refuted)
    assert(!failure.verified)
    assert(result.detail.exists(_.contains("effect failed")))
    assertEquals(broken.count, 0)

  test(
    "suite samples retain claim, monitor, progress and finite-cut results without exhaustive credit"
  ):
    class Draw[M <: Machine[?, ?]](instance: M, horizon: Int) extends ModelSuite(instance):
      object explorations:
        val space = explore(steps = horizon):
          any
      def sample = explorations.space.sample(2, seed = 19L)
    val execution =
      SuiteExecution.forSuite("client.Sampling", BudgetPolicy.parse(Some("default=8")))
    def checked[M <: Machine[?, ?]](
        instance: M,
        horizon: Int,
        verdict: Verdict,
        failing: Option[Int]
    ): Unit =
      val suite = new Draw(instance, horizon)
      val request = suite.sample
      val first = execution.sample(instance, request, None)
      val second = execution.sample(instance, request, None)
      val result = first.evidence.asInstanceOf[Result]
      val again = second.evidence.asInstanceOf[Result]
      assertEquals(result.verdict, verdict)
      assertEquals(first.verified, verdict == Verdict.verified)
      assertEquals(first.inconclusive, verdict == Verdict.unknown)
      assertEquals(result.sampling.get.walks, again.sampling.get.walks)
      assertEquals(result.sampling.get.failingIndex, failing)
      assertEquals(
        (
          result.budget,
          result.sampling.get.steps,
          result.sampling.get.requested,
          result.sampling.get.seed
        ),
        (8, horizon, 2, 19L)
      )
      assert(
        result.detail.exists(_.contains("no exhaustive safety or unbounded progress conclusion"))
      )
    checked(SamplingFixture.Counter, 3, Verdict.refuted, Some(0))
    checked(SamplingFixture.Watched, 3, Verdict.refuted, Some(0))
    checked(SamplingFixture.Deadline, 2, Verdict.refuted, Some(0))
    checked(SamplingFixture.Unbounded, 2, Verdict.unknown, None)
    checked(SamplingFixture.Uncertain, 2, Verdict.unknown, None)
    val instance = new Counter
    val suite = new Draw(instance, 3)
    val cut = SuiteExecution.forSuite("client.SmallSample", BudgetPolicy.parse(Some("default=1")))
    val result = cut.sample(instance, suite.sample, None)
    assert(!result.verified)
    assert(result.inconclusive)
    assertEquals(result.evidence.asInstanceOf[Result].budget, 1)
    assert(!result.evidence.asInstanceOf[Result].sampling.get.complete)

  test("questions refuse foreign owners and starts, missing starts and invalid authored bounds"):
    val instance = new Counter
    val foreign = new Counter
    class Questions extends ModelSuite(instance):
      object explorations:
        val space = explore(steps = 2):
          model.client.increment
      def target(horizon: Int) = explorations.space.find(steps = horizon):
        expect(model.count > 0)
      def foreignTarget = explorations.space.find(steps = 1):
        expect(foreign.count == 0)
      def hiddenForeignTarget = explorations.space.find(steps = 1):
        expect:
          false
          foreign.count == 0
      def sample(count: Int) = explorations.space.sample(count, seed = 9L)
      def missingTarget = explorations.space.find(steps = 1):
        model.client.increment
    val suite = new Questions
    val execution =
      SuiteExecution.forSuite("client.Refusals", BudgetPolicy.parse(Some("default=4")))
    val cases = Vector[(String, () => Any)](
      "find horizon" -> (() => execution.find(instance, suite.target(0), None)),
      "find horizon" -> (() => execution.find(instance, suite.target(3), None)),
      "sample count" -> (() => execution.sample(instance, suite.sample(0), None)),
      "another Model instance" -> (() => execution.find(foreign, suite.target(1), None)),
      "another Model instance" -> (() => execution.sample(foreign, suite.sample(1), None)),
      "another Model instance" -> (() => execution.find(instance, suite.foreignTarget, None)),
      "another Model instance" -> (() => execution.find(instance, suite.hiddenForeignTarget, None)),
      "another machine" -> (() => execution.find(instance, suite.target(1), Some(foreign.init))),
      "expect target" -> (() => suite.missingTarget)
    )
    cases.foreach: (message, call) =>
      val error = intercept[IllegalArgumentException](call())
      assert(error.getMessage.contains(message), error.getMessage)
    val multiple = new MultipleStarts
    class Starts extends ModelSuite(multiple):
      object explorations:
        val space = explore(steps = 1):
          any
      def target = explorations.space.find(steps = 1):
        expect(model.count == 2)
      def sample = explorations.space.sample(2, seed = 3L)
    val starts = new Starts
    assert(
      intercept[IllegalArgumentException](execution.find(multiple, starts.target, None)).getMessage
        .contains("requires from")
    )
    assert(
      intercept[IllegalArgumentException](
        execution.sample(multiple, starts.sample, None)
      ).getMessage.contains("requires from")
    )
    val initial = multiple.init.copy(count = 2)
    val found = execution.find(multiple, starts.target, Some(initial))
    val sampled = execution.sample(multiple, starts.sample, Some(initial))
    assert(found.verified)
    assert(sampled.verified)
    assertEquals(
      sampled.evidence.asInstanceOf[Result].sampling.get.walks.map(_.initial),
      Vector(Some(initial), Some(initial))
    )

  test(
    "one exact-suite execution shares its immutable caller budget between claims and replay"
  ):
    val policy = BudgetPolicy.parse(
      Some("default=7\nclient.SmallSuite=1\nclient.LargeSuite=4")
    )
    val model = new Counter
    val body = RunBody.capture(model):
      model.client.increment
      expect(model.count == 1)
    for (name, budget) <- Vector(
        "client.SmallSuite" -> 1,
        "client.LargeSuite" -> 4,
        "client.OtherSuite" -> 7
      )
    do
      val execution = SuiteExecution.forSuite(name, policy)
      val claims = execution.claims(VacuityFixture.FiniteModel)
      val replay = execution.replay(body)
      val report = claims.evidence.asInstanceOf[Report]
      val result = replay.evidence.asInstanceOf[Result]
      assertEquals(execution.budget, budget)
      assert(report.results.nonEmpty)
      assert(report.results.forall(_.budget == budget))
      assertEquals(result.budget, budget)
      assertEquals(claims.verified, budget > 1)
      assertEquals(claims.inconclusive, budget == 1)
      assertEquals(replay.verified, budget > 1)
      assertEquals(replay.inconclusive, budget == 1)

  test(
    "suite claims preserve vacuity and a counterexample alongside an unrelated cut"
  ):
    val policy = BudgetPolicy.parse(Some("default=3\nclient.CutSuite=1"))
    val complete = SuiteExecution
      .forSuite("client.CompleteSuite", policy)
      .claims(VacuityFixture.FiniteModel)
    val report = complete.evidence.asInstanceOf[Report]
    assert(complete.verified)
    assert(!complete.inconclusive)
    assert(report.warnings.exists(_.contains("postcondition is vacuous")))
    assert(report.warnings.forall(complete.diagnostics.contains))
    val cut = SuiteExecution
      .forSuite("client.CutSuite", policy)
      .claims(checkercontrol.CheckerFixture.Counter)
    val evidence = cut.evidence.asInstanceOf[Report]
    assert(!cut.verified)
    assert(cut.inconclusive)
    assertEquals(evidence.result("initialFailure").verdict, Verdict.refuted)
    assertEquals(evidence.result("bounded").verdict, Verdict.unknown)
    assert(evidence.result("initialFailure").path.initial.nonEmpty)
    assert(cut.diagnostics.contains("initialFailure: refuted"))
    assert(cut.diagnostics.contains("bounded: unknown"))
    assert(cut.diagnostics.contains("State(value = 0)"))

  test(
    "suite replay retains its native counterexample and positioned expectation diagnostics"
  ):
    val model = new Counter
    val body = RunBody.capture(model):
      model.client.increment
      expect(model.count == 9)
    val execution =
      SuiteExecution.forSuite(
        "client.ReplaySuite",
        BudgetPolicy.parse(Some("default=4"))
      )
    val check = execution.replay(body)
    val result = check.evidence.asInstanceOf[Result]
    assert(!check.verified)
    assert(!check.inconclusive)
    assertEquals(result.verdict, Verdict.refuted)
    assertEquals(result.path.steps.size, 1)
    assert(result.expectation.exists(_.text.endsWith("model.count == 9")))
    assert(check.diagnostics.contains("SuiteExecution.test.scala:"))
    assert(check.diagnostics.contains("State(count = 1)"))
    assert(check.diagnostics.contains("increment"))
    assert(check.diagnostics.contains("accepted"))
    assert(check.diagnostics.contains("true"))

  final case class Addition(amount: Int)
  class Counter extends Machine[Outcome, Boolean]:
    var count = 0
    val gap = hole

    object client extends Actor:
      lazy val increment = action
      lazy val add =
        action.inputs[Addition](
          inputField[Addition](_.amount).in(domains.oneOf(0 to 4))
        )

    object transitions extends Transitions:
      on(client.increment):
        effects.increment()
      on(client.add): in =>
        effects.add(in.amount)

    object effects:
      def increment(): Unit =
        count = count + 1
        records(true)
      def add(amount: Int): Unit =
        count = count + amount

  class MultipleStarts extends Machine[Outcome, Nothing]:
    var count = oneOf(List(0, 2))

    object transitions extends Transitions

  class Failures extends Machine[Outcome, Nothing]:
    var count = 0
    val gap = hole

    object client extends Actor:
      lazy val disabled = action
      lazy val explode = action
      lazy val missing = action
      lazy val once = action

    object transitions extends Transitions:
      on(client.disabled):
        if count > 0 then effects.increment()
      on(client.explode):
        effects.explode()
      on(client.missing):
        effects.missing()
      on(client.once):
        if count == 0 then effects.increment()

    object effects:
      def increment(): Unit = count = count + 1
      def explode(): Unit = throw IllegalStateException("effect failed")
      def missing(): Unit = if gap.reached then count = count + 1

  test("constant instance claims retain verified and refuted native evidence"):
    class Constant extends Machine[Outcome, Nothing]:
      var count = 0
      object transitions extends Transitions
      object invariants:
        val validconstanttrue =
          check(true)
        val falseclaim =
          check(false)
    val model = new Constant
    val section = model.getClass.getDeclaredMethod("invariants").getReturnType
    assertEquals(
      section.getConstructors.map(_.getParameterCount).toVector,
      Vector(0)
    )
    model.checkedHolder(Some("model"), "client.ConstantSuite")
    val checked = SuiteExecution
      .forSuite("client.ConstantSuite", BudgetPolicy.parse(Some("default=2")))
      .claims(model)
    val report = checked.evidence match
      case value: Report => value
      case other         => fail(s"expected native claim report, got $other")
    assert(!checked.verified)
    assert(!checked.inconclusive)
    assertEquals(
      report.verdicts,
      Map(
        "validconstanttrue" -> Verdict.verified,
        "falseclaim" -> Verdict.refuted
      )
    )
    assertEquals(report.results.map(_.budget), Vector(2, 2))
    assert(report.results.forall(_.complete))
    assert(
      report
        .result("falseclaim")
        .path
        .initial
        .exists:
          case state: State => state.owner.machine.eq(model)
          case _            => false
    )

  test("constant claim section aliases refuse a foreign lexical owner"):
    class Holder:
      object invariants:
        val foreign =
          check(true)
    val holder = new Holder
    class Aliased extends Machine[Outcome, Nothing]:
      var count = 0
      object transitions extends Transitions
      val invariants = holder.invariants
    val model = new Aliased
    val section = model.getClass.getDeclaredMethod("invariants").getReturnType
    assertEquals(
      section.getConstructors.map(_.getParameterCount).toVector,
      Vector(0)
    )
    assert(section.getEnclosingClass.equals(holder.getClass))
    assert(!section.getEnclosingClass.equals(model.getClass))
    assert(section.getDeclaringClass.equals(holder.getClass))
    model.checkedHolder(Some("model"), "client.ForeignConstantSuite")
    val execution = SuiteExecution.forSuite(
      "client.ForeignConstantSuite",
      BudgetPolicy.parse(Some("default=2"))
    )
    val failure = intercept[IllegalArgumentException](execution.claims(model))
    assert(
      failure.getMessage.contains("claim section of an instance"),
      failure.getMessage
    )

  test(
    "authored steps and expectations are deferred until replay restores the owner"
  ):
    val model = new Counter
    var observations = 0
    def observed: Boolean =
      observations += 1
      model.count == 2
    val body = RunBody.capture(model):
      repeat(2)(model.client.increment)
      expect:
        observed
        before.count == 1
        outcome == Outcome.accepted
    assertEquals(observations, 0)
    assertEquals(model.count, 0)
    val result = SuiteExecution.run(body, budget = 8)
    assertEquals(result.verdict, Verdict.verified)
    assertEquals(result.path.steps.size, 2)
    assertEquals(observations, 1)
    assertEquals(model.count, 0)

  test("named inputs read the actual restored step state once per repetition"):
    val model = new Counter
    val start = model.init
    var inputs = Vector.empty[Int]
    def amount: Int =
      inputs = inputs :+ model.count
      model.count
    val body = RunBody.capture(model, from = start):
      model.client.increment
      repeat(2)(model.client.add(amount = amount))
      expect:
        model.count == 4
        before.count == 2
    assertEquals(inputs, Vector.empty)
    model.count = 2
    val result = SuiteExecution.run(body, budget = 8)
    assertEquals(result.verdict, Verdict.verified)
    assertEquals(inputs, Vector(1, 2))
    assertEquals(model.count, 0)

  test("an expectation cannot read a foreign instance of the same class"):
    val model = new Counter
    val other = new Counter
    val body = RunBody.capture(model):
      model.client.increment
      expect:
        other.count == 0
    val failure =
      intercept[IllegalArgumentException](SuiteExecution.run(body, budget = 8))
    assert(failure.getMessage.contains("another Model instance"))

  test(
    "a budget cut inside a long repetition cannot reach its trailing expectation"
  ):
    val model = new Counter
    var evaluated = 0
    def unreachable: Boolean =
      evaluated += 1
      false
    val body = RunBody.capture(model):
      repeat(2147483647)(model.client.increment)
      expect:
        unreachable
    val result = SuiteExecution.run(body, budget = 2)
    assertEquals(result.verdict, Verdict.unknown)
    assertEquals(result.budget, 2)
    assertEquals(result.complete, false)
    assertEquals(evaluated, 0)

  test(
    "alternatives retain each target and evaluate following inputs in every frontier state"
  ):
    import framework.dsl.{oneOf as either, repeat as times}
    val model = new Counter
    val start = model.init
    var inputs = Vector.empty[Int]
    var seen = Vector.empty[Int]
    def amount: Int =
      inputs = inputs :+ model.count
      model.count
    def observed: Boolean =
      seen = seen :+ model.count
      true
    val body = RunBody.capture(model, from = start):
      either(model.client.add(amount = 1), model.client.add(amount = 2))
      times(1)(model.client.add(amount = amount))
      expect:
        observed
        model.count == before.count * 2
    val alternatives = body.operations.head.asInstanceOf[RunStep].targets
    assertEquals(alternatives.size, 2)
    assert(alternatives.head.position.text.contains("amount = 1"))
    assert(alternatives.last.position.text.contains("amount = 2"))
    assertEquals(inputs, Vector.empty)
    model.count = 3
    val result = SuiteExecution.run(body, budget = 8)
    assertEquals(
      result.verdict,
      Verdict.verified,
      result.detail.mkString + s" inputs=$inputs seen=$seen"
    )
    assertEquals(inputs, Vector(1, 2))
    assertEquals(seen, Vector(2, 4))
    assertEquals(result.path.steps.size, 2)
    assertEquals(model.count, 0)

  test(
    "multiple starts require a located explicit from and start evaluation remains deferred"
  ):
    val model = new MultipleStarts
    val default = RunBody.capture(model):
      expect(true)
    val missing = intercept[IllegalArgumentException](
      SuiteExecution.run(default, budget = 4)
    )
    assert(missing.getMessage.contains("SuiteExecution.test.scala:"))
    assert(missing.getMessage.contains("requires from"))
    var starts = 0
    def selected: State =
      starts += 1
      model.init.copy(count = 2)
    val explicit = RunBody.capture(model, from = selected):
      expect(model.count == 2)
    assertEquals(starts, 0)
    assertEquals(
      SuiteExecution.run(explicit, budget = 4).verdict,
      Verdict.verified
    )
    assertEquals(starts, 1)

  test(
    "zero-step expectations supply state and empty facts but no invented preceding step"
  ):
    val model = new Counter
    val state = RunBody.capture(model):
      expect:
        model.count == 0
        !records(true)
    assertEquals(
      SuiteExecution.run(state, budget = 4).verdict,
      Verdict.verified
    )
    val previous = RunBody.capture(model):
      expect(before.count == 0)
    val answer = RunBody.capture(model):
      expect(outcome == Outcome.accepted)
    Vector(previous, answer).foreach: body =>
      val failure =
        intercept[IllegalStateException](SuiteExecution.run(body, budget = 4))
      assert(failure.getMessage.contains("requires a preceding replay step"))

  test(
    "the lazy adapter preserves disabled effect exception and reached-hole results"
  ):
    val model = new Failures
    val disabled = RunBody.capture(model):
      model.client.disabled
    val thrown = RunBody.capture(model):
      model.client.explode
    val missing = RunBody.capture(model):
      model.client.missing
    val suffix = RunBody.capture(model):
      repeat(2)(model.client.once)
    val expectation = RunBody.capture(model):
      expect(model.gap.reached[Boolean])
    Vector(
      (
        disabled,
        Vector(ReplayStep[State](model.client.disabled)),
        Verdict.refuted,
        0
      ),
      (
        thrown,
        Vector(ReplayStep[State](model.client.explode)),
        Verdict.refuted,
        0
      ),
      (
        missing,
        Vector(ReplayStep[State](model.client.missing)),
        Verdict.unknown,
        0
      ),
      (
        suffix,
        Vector(
          ReplayStep[State](model.client.once),
          ReplayStep[State](model.client.once)
        ),
        Verdict.refuted,
        1
      )
    ).foreach: (body, steps, verdict, depth) =>
      val eager =
        framework.checker.Checker.replay(model, ReplayScript(steps), budget = 4)
      val result = SuiteExecution.run(body, budget = 4)
      assertEquals(result.verdict, verdict)
      assertEquals(result.detail, eager.detail)
      assertEquals(result.path, eager.path)
      assertEquals(result.path.steps.size, depth)
      assertEquals(model.count, 0)
    val reached = SuiteExecution.run(expectation, budget = 4)
    assertEquals(reached.verdict, Verdict.unknown)
    assert(
      reached.detail.exists(
        _.startsWith("expectation at State(count = 0) reaches ")
      )
    )
    val counter = new Counter
    def broken: Int = throw IllegalStateException("input failed")
    val input = RunBody.capture(counter):
      counter.client.add(amount = broken)
    val result = SuiteExecution.run(input, budget = 4)
    assertEquals(result.verdict, Verdict.refuted)
    assert(result.detail.mkString.contains("input failed"))
    assert(result.detail.mkString.contains("SuiteExecution.test.scala:"))
    assertEquals(counter.count, 0)
    def missingAmount: Int = counter.gap.reached[Int]
    val missingInput = RunBody.capture(counter):
      counter.client.add(amount = missingAmount)
    val unknown = SuiteExecution.run(missingInput, budget = 4)
    assertEquals(unknown.verdict, Verdict.unknown)
    assert(
      unknown.detail.exists(_.contains("inputs at State(count = 0) reach "))
    )
    assertEquals(counter.count, 0)

  test(
    "nested and exceptional replay restores outer context and retained evidence is immutable"
  ):
    val model = new Counter
    val inner = new Counter
    val nested = RunBody.capture(inner):
      repeat(2)(inner.client.increment)
      expect(inner.count == 2)
    def thrown: Boolean = throw IllegalStateException("predicate failed")
    val exceptional = RunBody.capture(inner):
      inner.client.increment
      expect(thrown)
    def reenter: Boolean =
      assertEquals(
        SuiteExecution.run(nested, budget = 8).verdict,
        Verdict.verified
      )
      val failure = intercept[IllegalStateException](
        SuiteExecution.run(exceptional, budget = 8)
      )
      failure.getMessage == "predicate failed"
    val outer = RunBody.capture(model):
      model.client.increment
      expect:
        reenter
        model.count == 1
        before.count == 0
        outcome == Outcome.accepted
        records(true)
        model.count == 9
    val first = SuiteExecution.run(outer, budget = 8)
    assertEquals(first.verdict, Verdict.refuted)
    val retained = (first.path, first.expectation, first.detail)
    assertEquals(
      SuiteExecution.run(nested, budget = 8).verdict,
      Verdict.verified
    )
    assertEquals((first.path, first.expectation, first.detail), retained)
    assert(first.expectation.exists(_.text.endsWith("model.count == 9")))
    assertEquals(model.count, 0)
    assertEquals(inner.count, 0)

  test(
    "an input hole preserves known sibling alternatives and their decisive evidence"
  ):
    val model = new Counter
    val start = model.init
    var inputs = Vector.empty[(String, Int)]
    def known: Int =
      inputs = inputs :+ ("known", model.count)
      model.count
    def missing: Int =
      inputs = inputs :+ ("hole", model.count)
      model.gap.reached[Int]
    val knownFirst = RunBody.capture(model, from = start):
      model.client.increment
      oneOf(
        model.client.add(amount = known),
        model.client.add(amount = missing)
      )
      expect(model.count == 0)
    val holeFirst = RunBody.capture(model, from = start):
      model.client.increment
      oneOf(
        model.client.add(amount = missing),
        model.client.add(amount = known)
      )
      expect(model.count == 0)
    val passing = RunBody.capture(model, from = start):
      model.client.increment
      oneOf(
        model.client.add(amount = missing),
        model.client.add(amount = known)
      )
      expect(model.count == 2)
    assertEquals(inputs, Vector.empty)
    val cases = Vector(
      (knownFirst, Vector("known", "hole"), Verdict.refuted),
      (holeFirst, Vector("hole", "known"), Verdict.refuted),
      (passing, Vector("hole", "known"), Verdict.unknown)
    )
    val evaluated = cases.map: (body, order, verdict) =>
      inputs = Vector.empty
      model.count = 3
      val result = SuiteExecution.run(body, budget = 8)
      (body, order, verdict, result, inputs)
    assertEquals(
      evaluated.map(_._4.verdict),
      cases.map(_._3),
      evaluated.map(value => value._5).toString
    )
    evaluated.foreach: (body, order, verdict, result, observed) =>
      assertEquals(observed, order.map(_ -> 1))
      assertEquals(result.path.steps.size, 2)
      assert(
        result.detail.exists(_.contains("inputs at State(count = 1) reach "))
      )
      if verdict == Verdict.refuted then
        val position = body.operations.collect:
          case checks: RunAssertions => checks.assertions.head.position
        assertEquals(result.expectation.map(_.line), Some(position.head.line))
        assertEquals(
          result.expectation.map(_.text),
          Some("SuiteExecution.test.scala: model.count == 0")
        )
        assertEquals(
          result.expectation.map(_.state),
          Some(model.init.copy(count = 2))
        )
      else assertEquals(result.expectation, None)
      assertEquals(model.count, 0)
