package framework.dsl

import framework.dsl.Finite.given

object ExpectFixture:
  import WitnessFixture.*

  object Reordered extends Composition[Joined](_.right -> Other, _.left -> Subject)

  object Repeated extends Composition[Joined](_.right -> Subject, _.left -> Subject)
  def left(using view: View[Joined]): State = view.get(_.left)

class ExpectTest extends munit.FunSuite:
  val _ = () => summon[CanEqual[EffectsFixture.Phase, EffectsFixture.Phase]]
  test("builder and ordinary predicate heads infer typed contextual bindings"):
    assertEquals(
      compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      import framework.dsl.EffectsFixture.*
      val same: Step[Job, Said, Note] => Boolean = expect:
        (s: Expectation[Job, Said, Note]) ?=>
          check(s.step.state.phase == Phase.busy)
          records(Note.started)
      val transition: (Job, Step[Job, Said, Note]) => Boolean = expect.across:
        (b: Before[Job], a: Expectation[Job, Said, Note]) ?=>
          check(before.phase == b.state.phase)
          check(after.state == a.step.state)
          check(phase == Phase.busy)
      val inferred: Step[Job, Said, Note] => Boolean = expect:
        s ?=> check(s.step.state.phase == Phase.busy)
      def generic[S](m: Declares[S]) = m.property("generic").expect:
        check(after.state.equals(after.state))
      val plain = Subject.property.expect:
        check(phase == Phase.busy)
      val restricted = Subject.property.after(hand.press()).expect:
        records(Note.started)
      val infix = Subject.property after(hand.press()) expect:
        check(phase == Phase.busy)
      val whole = Subject.property.afterAction("press").across.expect:
        check(before.phase == Phase.idle)
      val forwarded = Subject.property.expect:
        check(same(after))
      val forwardedAcross = Subject.property.across.expect:
        check(transition(before, after))
      generic(Subject)
    """),
      ""
    )

  test("same-step and transition blocks read the resulting state and typed facts"):
    import EffectsFixture.*
    val predicate: Step[Job, Said, Note] => Boolean = expect:
      records(Note.started)
      check(phase == Phase.busy)
    val old = Job(Phase.idle, UpTo(0))
    val next = Job(Phase.busy, UpTo(1))
    val transition = Subject.property.across.expect:
      (b: Before[Job], a: Expectation[Job, Said, Note]) ?=>
        check(before == b.state)
        check(after.equals(a.step))
        check(phase == Phase.busy)
        check(before.phase == Phase.idle)
    val holds = transition.decl.holds2.get.asInstanceOf[(Job, Step[Job, Said, Note]) => Boolean]
    for (state, facts, wanted) <- List(
        (old, List(Note.started), false),
        (next, Nil, false),
        (next, List(Note.started), true)
      )
    do assertEquals(predicate(Step(Said.ok, state, facts)), wanted)
    assert(holds(old, Step(Said.ok, next)))
    assert(!holds(next, Step(Said.ok, next)))

  test("ordinary predicates retain authored contextual parameter names"):
    import EffectsFixture.*
    val same: Step[Job, Said, Note] => Boolean = expect: sourceStep ?=>
      check(sourceStep.step.state.equals(after.state))
    val transition: (Job, Step[Job, Said, Note]) => Boolean = expect.across:
      (oldState, resultStep) ?=> check(oldState.state.equals(resultStep.step.state))
    val unused: (Job, Step[Job, Said, Note]) => Boolean = expect.across:
      (unusedBefore, resulting) ?=> check(resulting.step.state.equals(after.state))
    assertEquals(same.asInstanceOf[PredicateBindings].sourceParameters, Vector("sourceStep"))
    assertEquals(
      transition.asInstanceOf[PredicateBindings].sourceParameters,
      Vector("oldState", "resultStep")
    )
    assertEquals(
      unused.asInstanceOf[PredicateBindings].sourceParameters,
      Vector("unusedBefore", "resulting")
    )

  test("requirements and alternatives short circuit without swallowing holes or errors"):
    import EffectsFixture.*
    val missing = hole
    val step = Step(Said.ok, Subject.init, List.empty[Note])
    val failure: Step[Job, Said, Note] => Boolean = expect:
      check(false)
      if missing.reached then check(true)
    assert(!failure(step))
    val alternative: Step[Job, Said, Note] => Boolean = expect:
      oneOf:
        branch:
          check(false)
          check(missing.reached)
        branch:
          check(true)
        branch:
          check(missing.reached)
    assert(alternative(step))
    val allFalse: Step[Job, Said, Note] => Boolean = expect:
      oneOf:
        branch:
          check(false)
        branch:
          check(false)
    assert(!allFalse(step))
    val nested: Step[Job, Said, Note] => Boolean = expect:
      oneOf:
        branch:
          oneOf:
            branch:
              check(false)
            branch:
              check(true)
          check(false)
        branch:
          check(true)
    assert(nested(step))
    val reached: Step[Job, Said, Note] => Boolean = expect:
      oneOf:
        branch:
          check(false)
        branch:
          check(missing.reached)
    assertEquals(intercept[HoleReached](reached(step)).hole, missing)
    def evaluated(condition: () => Boolean): Step[Job, Said, Note] => Boolean = expect:
      check(condition())
    val unrelated = new IllegalStateException("unrelated")
    assert(intercept[IllegalStateException](evaluated(() => throw unrelated)(step)) eq unrelated)
    val unrelatedSignal = new scala.util.control.ControlThrowable {}
    val propagated = try
      val _ = evaluated(() => throw unrelatedSignal)(step)
      false
    catch case signal: scala.util.control.ControlThrowable => signal eq unrelatedSignal
    assert(propagated)
    def alternativeEvaluation(condition: () => Boolean): Step[Job, Said, Note] => Boolean = expect:
      oneOf:
        branch:
          check(condition())
        branch:
          check(true)
    assert(
      intercept[IllegalStateException](
        alternativeEvaluation(() => throw unrelated)(step)
      ) eq unrelated
    )
    val guard: Step[Job, Said, Note] => Boolean = expect:
      if missing.reached then check(true)
    assertEquals(intercept[HoleReached](guard(step)).hole, missing)

  test("a conditional-only block succeeds when no requirement applies"):
    import EffectsFixture.*
    val predicate: Step[Job, Said, Note] => Boolean = expect:
      if phase == Phase.busy then records(Note.started)
    assert(predicate(Step(Said.ok, Subject.init)))
    assert(!predicate(Step(Said.ok, Job(Phase.busy, UpTo(0)))))

  test("ordinary expectation bridges preserve independent core pattern and monitor behavior"):
    import EffectsFixture.*
    val named: Step[Job, Said, Note] => Boolean = expect:
      records(Note.started)
      check(phase == Phase.busy)
    val core: Step[Job, Said, Note] => Boolean =
      step => step.records(Note.started) && step.state.phase == Phase.busy
    val never = Subject.property.never(named).from(_.phase == Phase.idle)
    val original = Property[Job](
      PropertyDecl(
        "",
        Subject,
        None,
        None,
        Some((old: Job, step: Step[Job, Said, Note]) => old.phase != Phase.idle || !core(step))
      )
    )
    val actual = never.decl.holds2.get.asInstanceOf[(Job, Step[Job, Said, Note]) => Boolean]
    val oracle = original.decl.holds2.get.asInstanceOf[(Job, Step[Job, Said, Note]) => Boolean]
    val watched = sticky(named).readAfter(expect:
      check(after.outcome == Said.ok))
    val coreMonitor =
      monitor[Job, Said, Note, Boolean](false)((broken, _, step) => broken || !core(step))(identity)
        .readAfter(_.outcome == Said.ok)
    val transition: (Job, Step[Job, Said, Note]) => Boolean = expect.across:
      if before.phase == Phase.idle then check(core(after))
    val acrossMonitor = stickyAcross(transition)
    for old <- List(Subject.init, Job(Phase.busy, UpTo(0))) do
      for state <- List(Subject.init, Job(Phase.busy, UpTo(0))) do
        for facts <- List(Nil, List(Note.started)) do
          val step = Step(Said.ok, state, facts)
          assertEquals(actual(old, step), oracle(old, step))
          for broken <- List(false, true) do
            assertEquals(watched.next(broken, old, step), coreMonitor.next(broken, old, step))
            assertEquals(
              acrossMonitor.next(broken, old, step),
              broken || !(old.phase != Phase.idle || core(step))
            )
          assertEquals(watched.readAt.get(step), coreMonitor.readAt.get(step))
    val missing = hole
    val promise: Step[Job, Said, Note] => Boolean = expect:
      check(after.state.phase == Phase.busy)
      check(missing.reached)
    val watch = sticky(promise)
    assert(watch.next(false, Subject.init, Step(Said.ok, Subject.init)))
    assert(watch.next(true, Subject.init, Step(Said.ok, Job(Phase.busy, UpTo(0)))))
    assertEquals(
      intercept[HoleReached](
        watch.next(false, Subject.init, Step(Said.ok, Job(Phase.busy, UpTo(0))))
      ).hole,
      missing
    )

  test("if-else requirements and eager immutable helpers preserve their evaluation order"):
    import EffectsFixture.*
    val conditional: Step[Job, Said, Note] => Boolean = expect:
      if phase == Phase.busy then records(Note.started)
      else check(tries == UpTo[2](0))
    for (state, facts, wanted) <- List(
        (Job(Phase.busy, UpTo[2](1)), List(Note.started), true),
        (Job(Phase.busy, UpTo[2](1)), Nil, false),
        (Job(Phase.idle, UpTo[2](0)), Nil, true),
        (Job(Phase.idle, UpTo[2](1)), List(Note.started), false)
      )
    do assertEquals(conditional(Step(Said.ok, state, facts)), wanted)
    var evaluations = 0
    def predicate(read: () => Boolean): Step[Job, Said, Note] => Boolean = expect:
      val eager = read()
      check(false)
      check(eager)
    assert(!predicate { () => evaluations += 1; true }(Step(Said.ok, Subject.init)))
    assertEquals(evaluations, 1)

  test("exact-class and whole-action heads retain distinct native restrictions"):
    import EffectsFixture.*
    val flag = input[Boolean](default = false)
    val move = action("move", Actor("tester")).input(flag)
    val exact = Subject.property
      .after(move(false))
      .expect:
        check(true)
    val whole = Subject.property
      .afterAction(move.decl.name)
      .expect:
        check(true)
    assertEquals(exact.decl.when, Some(move(false)))
    assertNotEquals(exact.decl.when, Some(move(true)))
    assertEquals(whole.decl.when, Some("move"))
    val composed = ExpectFixture.Reordered.own(_.left, WitnessFixture.start)
    val allMemberClasses = ExpectFixture.Reordered.property
      .afterAction(composed)
      .expect:
        check(true)
    assertEquals(allMemberClasses.decl.when, Some(composed))

  test("generic Declares builders supply transition and composition state contexts"):
    import WitnessFixture.*
    import ExpectFixture.*
    def generic[S](owner: Declares[S], equal: (S, S) => Boolean): Property[S] =
      owner.property.across.expect:
        check(equal(before, after.state))
    def composed(owner: Declares[Joined]): Property[Joined] = owner.property.expect:
      check(left.equals(State.open))
    val predicate = generic(Reordered, (a, b) => a.equals(b)).decl.holds2.get
      .asInstanceOf[(Joined, Step[Joined, String, String]) => Boolean]
    val old = Joined(State.open, State.closed)
    assert(predicate(old, Step("accepted", old)))
    assert(!predicate(old, Step("accepted", old.copy(left = State.closed))))
    val reads = composed(Reordered).decl.holds.get
      .asInstanceOf[Step[Joined, String, String] => Boolean]
    assert(reads(Step("accepted", old)))
    assert(!reads(Step("accepted", old.copy(left = State.closed))))

  test("predicate evaluations isolate concurrent contexts and nested predicate failures"):
    import EffectsFixture.*
    val barrier = new java.util.concurrent.CyclicBarrier(2)
    def predicate(ready: () => Boolean): Step[Job, Said, Note] => Boolean = expect:
      check(ready())
      check(phase == Phase.busy)
    val shared = predicate { () =>
      val _ = barrier.await(10, java.util.concurrent.TimeUnit.SECONDS); true
    }
    val pool = java.util.concurrent.Executors.newFixedThreadPool(2)
    try
      val results = List(Phase.idle, Phase.busy).map: phase =>
        pool.submit(new java.util.concurrent.Callable[Boolean]:
          def call(): Boolean = shared(Step(Said.ok, Job(phase, UpTo(0)))))
      assertEquals(results.map(_.get(15, java.util.concurrent.TimeUnit.SECONDS)), List(false, true))
    finally pool.shutdown()
    val inner: Step[Job, Said, Note] => Boolean = expect:
      check(false)
    val outer: Step[Job, Said, Note] => Boolean = expect:
      oneOf:
        branch:
          check(!inner(after))
        branch:
          check(false)
    assert(outer(Step(Said.ok, Subject.init)))

  test("witness selectors bind exact fields in reordered and repeated-machine compositions"):
    import WitnessFixture.*
    import ExpectFixture.*
    val step = Step("accepted", Joined(State.open, State.closed), List("left_completed"))
    def recorded(q: Query): Boolean =
      q.property.holds.get.asInstanceOf[Step[Joined, String, String] => Boolean](step)
    assert(recorded(Reordered.witness(start()).records(_.left -> Subject, Fact.completed)))
    assert(!recorded(Reordered.witness(start()).records(_.right -> Other, Foreign.completed)))
    assert(recorded(Repeated.witness(start()).records(_.left -> Subject, Fact.completed)))
    assert(!recorded(Repeated.witness(start()).records(_.right -> Subject, Fact.completed)))
    intercept[IllegalArgumentException]:
      Reordered.witness(start()).records(_.right -> Subject, Fact.completed)

  test("expectation validation refuses empty, discarded and wrong-level statements"):
    val invalid = List(
      compileErrors("""import EffectsFixture.*; Subject.property.expect { () }"""),
      compileErrors("""import EffectsFixture.*; Subject.property.expect { val only = true; () }"""),
      compileErrors("""import EffectsFixture.*; Subject.property.expect { check(true); false }"""),
      compileErrors("""import EffectsFixture.*; Subject.property.expect { false; check(true) }"""),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { val path = Subject.witness(hand.press()); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { val step = Step(Said.ok, Subject.init); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { oneOf { branch { check(true) } } }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { oneOf { branch {}; branch { check(true) } } }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { println("not a requirement"); check(true) }"""
      )
    )
    assertEquals(
      invalid.zipWithIndex.filterNot((diagnostic, _) => diagnostic.contains("expect:")),
      Nil
    )

  test("scope and fact typing refuse before, assignment, foreign facts and stray branches"):
    val invalid = List(
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { check(before.equals(Subject.init)) }"""
      ),
      compileErrors("""import EffectsFixture.*; Subject.property.expect { phase = Phase.busy }"""),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { records(WitnessFixture.Foreign.completed) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { check(after.records(WitnessFixture.Foreign.completed)) }"""
      ),
      compileErrors(
        """import WitnessFixture.*; Together.property.expect { records(_.left -> Subject, Foreign.completed) }"""
      ),
      compileErrors(
        """import WitnessFixture.*; Together.property.expect { check(after.records(_.left -> Subject, Foreign.completed)) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { branch { check(true) } }"""
      )
    )
    assertEquals(invalid.zipWithIndex.filter((diagnostic, _) => diagnostic.isEmpty), Nil)

  test("member metadata requires an authored direct field rather than a computed selector"):
    val invalid = List(
      compileErrors(
        """import WitnessFixture.*; object Computed extends Composition[Joined](s => identity(s.left) -> Subject, _.right -> Other)"""
      ),
      compileErrors(
        """import WitnessFixture.*; Together.property.expect { records(s => identity(s.left) -> Subject, Fact.completed) }"""
      )
    )
    assert(invalid.head.contains("Required: framework.dsl.MemberSelection"), invalid.head)
    assert(invalid(1).contains("direct"), invalid(1))
    val foreignRoot = compileErrors(
      """import WitnessFixture.*; def build(foreign: Joined) = { object Wrong extends Composition[Joined](_ => foreign.left -> Subject, _.right -> Other); Wrong }"""
    )
    assert(foreignRoot.nonEmpty, "a constructor member must select its own state parameter")

  test("structural validation inspects nested expressions and transitive source helpers"):
    val invalid = List(
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { if true then { false; check(true) } }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { oneOf { branch { check(true); false }; branch { check(true) } } }"""
      ),
      compileErrors("""import EffectsFixture.*; Subject.property.expect { oneOf {} }"""),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { Subject.witness(hand.press()); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { Step(Said.ok, Subject.init); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { val okay = { val made = Subject.property; true }; check(okay) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { if { val made = Step(Said.ok, Subject.init); true } then check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; def hidden(): Boolean = { val made = Step(Said.ok, Subject.init); true }; Subject.property.expect { check(hidden()) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; val path = Subject.witness(hand.press()).records(Note.started); Subject.property.expect { check(path.name.nonEmpty) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; given Owner[Job, Said, Note] = Owner(Subject); Subject.property.expect { val step = effect { record(Note.started) }; check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; object Bad extends Machine[Job, Said, Note] { def init = Subject.init; def end(s: Job) = true; object rules extends Rules; val bad = property.expect { val path = witness(hand.press()); check(true) } }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { val temporal = Subject.leadsTo("p")(_ => true, _ => true, 1); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { val watch = monitor[Job, Said, Note, Boolean](false)((m, b, a) => m)(_ => false); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; Subject.property.expect { val q = query.verify(Subject.property.expect { check(true) }).in(Subject.scenario.free).limits(Limits(steps = 1, actions = 1, search = 1)); check(true) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; given Before[Job] = new Before(Subject.init); Subject.property.expect { check(before.equals(Subject.init)) }"""
      ),
      compileErrors(
        """import EffectsFixture.*; given Draft[Job, Said, Note] = new Draft(Subject.init); Subject.property.expect { phase = Phase.busy; check(true) }"""
      )
    )
    assertEquals(
      invalid.zipWithIndex.filterNot((diagnostic, _) => diagnostic.contains("expect:")),
      Nil
    )

  test("composed predicates retain exact field-machine requirements until attachment"):
    import WitnessFixture.*
    import ExpectFixture.*
    val named: Step[Joined, String, String] => Boolean = expect:
      records(_.left -> Subject, Fact.completed)
    val transition: (Joined, Step[Joined, String, String]) => Boolean = expect.across:
      records(_.right -> Other, Foreign.completed)
      check(before.left.equals(after.state.left))
    val joined = Joined(State.open, State.closed)
    def holds(p: Property[Joined], facts: List[String]): Boolean =
      p.decl.holds.get
        .asInstanceOf[Step[Joined, String, String] => Boolean](Step("accepted", joined, facts))
    val reordered = Reordered.property.expect:
      check(named(after))
    val repeated = Repeated.property.expect:
      check(named(after))
    assert(holds(reordered, List("left_completed")))
    assert(!holds(repeated, List("right_completed")))
    assert(holds(repeated, List("left_completed")))
    val across = Reordered.property.across.expect:
      check(transition(before, after))
    assert(across.decl.holds2.nonEmpty)
    val _ = intercept[IllegalArgumentException]:
      Repeated.property.across.expect:
        check(transition(before, after))
    val wrong: Step[Joined, String, String] => Boolean = expect:
      records(_.right -> Subject, Fact.completed)
    val _ = intercept[IllegalArgumentException]:
      Reordered.property.expect:
        check(wrong(after))
    val _ = intercept[IllegalArgumentException](Reordered.property.never(wrong))
    val _ = intercept[IllegalArgumentException](Reordered.property.stays(_ => true).unless(wrong))
    assert(!holds(Repeated.property.never(named), List("left_completed")))
    val _ = intercept[IllegalArgumentException]:
      Reordered.property.expect:
        if false then records(_.right -> Subject, Fact.completed)
    val conditional = Reordered.property.expect:
      if after.records(_.right -> Other, Foreign.completed) then
        records(_.left -> Subject, Fact.completed)
    assert(!holds(conditional, List("right_completed")))
    assert(holds(conditional, List("left_completed", "right_completed")))
