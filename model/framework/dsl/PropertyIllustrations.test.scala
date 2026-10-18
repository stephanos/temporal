package framework.dsl

object PropertyIllustrationsFixture:

  object Answer:
    opaque type Value = Int
    def apply(value: Int): Value = value
    given Data.Structural[Value] = summon[Data.Structural[Int]]
  val accepted = Answer(1)
  val excluded = Answer(2)
  final case class NativeState(count: Int, values: List[Int])
  final case class Input(first: Int, last: Int)
  val one = Input(0, 0)
  val two = Input(1, 1)
  val correlated = action("correlated", Actor("tester")).inputs[Input](Finite.of(one, two))
  val plain = action("plain", Actor("tester"))
  val foreign = action("plain", Actor("tester"))
  given Finite[NativeState] = new Finite[NativeState]:
    def values = throw AssertionError("state catalog evaluated")

  object Native
      extends RecordMachine[NativeState, Boolean, Boolean](using
        summon[Data[NativeState]],
        Finite.of(true),
        Finite.of(false)
      ):
    def init: NativeState = throw AssertionError("init evaluated")
    def end(state: NativeState): Boolean = throw AssertionError("end evaluated")

    object transitions extends Rules:
      on(PropertyIllustrationsFixture.plain):
        where((_: NativeState) => throw AssertionError("guard evaluated")) ~>
          ((_: NativeState) => throw AssertionError("effect evaluated"))
      on(PropertyIllustrationsFixture.correlated):
        always ~> ((_: NativeState, _: Input) => throw AssertionError("input effect evaluated"))

  object Opaque
      extends RecordMachine[Boolean, Answer.Value, Answer.Value](using
        summon[Data[Boolean]],
        Finite.of(accepted),
        Finite.of(accepted)
      ):
    def init = false
    def end(state: Boolean): Boolean = state

    object transitions
        extends Bindings(
          PropertyIllustrationsFixture.plain ~> ((s: Boolean) => List(Step(accepted, s)))
        )

class PropertyIllustrationsSuite extends munit.FunSuite:
  import WitnessFixture.*

  test("typed completions append ordered illustrations without changing ordinary declarations"):
    val original: SameStepProperty[State, Outcome, Fact] = Subject
      .property("completion")
      .expect:
        records(Fact.completed)
    val step = Step(Outcome.accepted, State.closed, List(Fact.completed, Fact.completed))
    val first = original.illustrate(
      "  completion  ",
      "  the completion is recorded  ",
      IllustrationKind.positive,
      start(),
      step
    )
    val second = first.illustrate(
      "missing",
      "No completion was recorded",
      IllustrationKind.counterexample,
      start(),
      Step(Outcome.accepted, State.closed)
    )
    val ordinary: Property = second
    val q = query
      .verify(ordinary)
      .in(Subject.scenario.free)
      .limits(Limits(steps = 1, actions = 1, search = 8))
    assert(original.illustrations.isEmpty)
    assertEquals(first.illustrations.size, 1)
    assertEquals(second.illustrations.map(_.label), Vector("  completion  ", "missing"))
    assertEquals(first.illustrations.head.because, "  the completion is recorded  ")
    assert(first.illustrations.head.step eq step)
    assertEquals(first.illustrations.head.step.facts, List(Fact.completed, Fact.completed))
    assert(q.property eq second.decl)
    assertEquals(q.property.illustrations.size, 2)
    assert(q.property.illustrations.head eq first.illustrations.head)
    assert(q.property.illustrations.last eq second.illustrations.last)
    assertEquals(q.property.illustrations.map(_.label), Vector("  completion  ", "missing"))
    assert(second.decl.machine eq original.decl.machine)
    assert(second.decl.when eq original.decl.when)
    assert(second.decl.holds eq original.decl.holds)
    assert(second.decl.holds2 eq original.decl.holds2)
    assert(
      first.illustrations.head.origin
        .matches("PropertyIllustrations.test.scala:[1-9][0-9]*:[1-9][0-9]*")
    )
    assertNotEquals(first.illustrations.head.origin, second.illustrations.last.origin)

  test("transition illustrations require and retain their typed hypothetical before state"):
    val original: TransitionProperty[State, Outcome, Fact] = Subject
      .property("closed")
      .across
      .expect:
        check(before.equals(after.state))
    val attached = original.illustrate(
      "hypothetical",
      "The input need not be reachable",
      IllustrationKind.counterexample,
      start(),
      State.closed,
      Step(Outcome.accepted, State.open)
    )
    assertEquals(attached.illustrations.head.before, Some(State.closed))
    assert(attached.decl.holds2 eq original.decl.holds2)

  test("metadata validation is located and leaves previous immutable copies intact"):
    val p = Subject
      .property("metadata")
      .expect:
        check(true)
    val step = Step[State, Outcome, Fact](Outcome.accepted, State.open)
    val first = p.illustrate("one", "reason", IllustrationKind.positive, start(), step)
    for (label, because, expected) <- List(
        (" ", "reason", "label"),
        ("two", " \n", "because"),
        (null, "reason", "label"),
        ("two", null, "because"),
        ("one", "reason", "duplicate")
      )
    do
      val error = intercept[IllegalArgumentException]:
        first.illustrate(label, because, IllustrationKind.positive, start(), step)
      assert(error.getMessage.contains(expected), error.getMessage)
      assert(error.getMessage.contains("metadata"), error.getMessage)
      assert(error.getMessage.contains("PropertyIllustrations.test.scala:"), error.getMessage)
    val kindError = intercept[IllegalArgumentException]:
      first.illustrate("kind", "Explicit kind required", null, start(), step)
    assert(kindError.getMessage.contains("kind must be explicitly"), kindError.getMessage)
    assertEquals(first.illustrations.size, 1)

  test("a same-spelled foreign action is refused while a bound unselected action is retained"):
    val p = Subject
      .property("unselected")
      .afterAction("not-start")
      .expect:
        check(true)
    val step = Step[State, Outcome, Fact](Outcome.accepted, State.open)
    assertEquals(
      p.illustrate("allowed", "Unclassified", IllustrationKind.positive, start(), step)
        .illustrations
        .size,
      1
    )
    val foreign = action("start", actor)
    val error = intercept[IllegalArgumentException]:
      p.illustrate("foreign", "wrong declaration", IllustrationKind.positive, foreign(), step)
    assert(error.getMessage.contains("not bound"), error.getMessage)

  test("ordinary property predicates remain the independent authority"):
    val p = Subject
      .property("independent")
      .expect:
        records(Fact.completed)
    val predicate = p.decl.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(predicate(Step(Outcome.accepted, State.open, List(Fact.completed))))
    assert(!predicate(Step(Outcome.accepted, State.open)))
    assert(p.decl.illustrations.isEmpty)

  test(
    "admission uses actual selected domains and never reads modeled callbacks or state catalogs"
  ):
    import PropertyIllustrationsFixture.*
    def claim(read: () => Boolean) = Native
      .property("native")
      .after(plain())
      .expect:
        check(read())
    val p = claim(() => throw AssertionError("predicate evaluated"))
    val state = NativeState(Int.MaxValue, List(Int.MinValue, 1000000))
    val step = Step(true, state, List(false, false))
    val supplied = correlated(one)
    val originalInput = supplied.supplied.get.value
    val attached = p.illustrate(
      "unreachable",
      "Unselected, truth unclaimed",
      IllustrationKind.counterexample,
      supplied,
      step
    )
    assert(attached.illustrations.head.action eq supplied)
    assert(supplied.supplied.get.value.asInstanceOf[AnyRef] eq originalInput.asInstanceOf[AnyRef])
    assertEquals(attached.illustrations.head.step.facts, List(false, false))
    for (label, actionClass, suppliedStep, expected) <- List(
        ("outcome", plain(), Step(false, state, List(false)), "outcome"),
        ("fact", plain(), Step(true, state, List(true)), "fact"),
        ("foreign", foreign(), step, "not bound"),
        ("correlation", Class(correlated.decl, List(0, 1)), step, "input"),
        ("shape", Class(plain.decl, List(0)), step, "inputs")
      )
    do
      val error = intercept[IllegalArgumentException]:
        p.illustrate(
          label,
          "Invalid supplied value",
          IllustrationKind.positive,
          actionClass,
          suppliedStep
        )
      assert(error.getMessage.contains(expected), error.getMessage)
      assert(error.getMessage.contains("PropertyIllustrations.test.scala:"), error.getMessage)

  test("typed attachment refuses wrong step types before state shape and unknown kind"):
    val invalid = List(
      compileErrors(
        "import WitnessFixture.*; Subject.property.expect { check(true) }.illustrate(\"x\", \"why\", IllustrationKind.positive, start(), Step(Outcome.accepted, 1))"
      ),
      compileErrors(
        "import WitnessFixture.*; Subject.property.expect { check(true) }.illustrate(\"x\", \"why\", IllustrationKind.positive, start(), Step(false, State.open))"
      ),
      compileErrors(
        "import WitnessFixture.*; Subject.property.expect { check(true) }.illustrate(\"x\", \"why\", IllustrationKind.positive, start(), Step(Outcome.accepted, State.open, List(false)))"
      ),
      compileErrors(
        "import WitnessFixture.*; Subject.property.across.expect { check(true) }.illustrate(\"x\", \"why\", IllustrationKind.positive, start(), Step(Outcome.accepted, State.open))"
      ),
      compileErrors(
        "import WitnessFixture.*; Subject.property.expect { check(true) }.illustrate(\"x\", \"why\", IllustrationKind.positive, start(), State.open, Step(Outcome.accepted, State.open))"
      ),
      compileErrors(
        "import WitnessFixture.*; Subject.property.expect { check(true) }.illustrate(\"x\", \"why\", IllustrationKind.unspecified, start(), Step(Outcome.accepted, State.open))"
      )
    )
    assertEquals(invalid.zipWithIndex.filter((message, _) => message.isEmpty), Nil)

  test("opaque outcomes and facts use selected owner evidence instead of an ambient witness"):
    import PropertyIllustrationsFixture.*
    given Finite[Answer.Value] = new Finite[Answer.Value]:
      def values = throw AssertionError("ambient outcome/fact domain evaluated")
    val _ = () => summon[Finite[Answer.Value]]
    val p = Opaque
      .property("selectedOpaque")
      .expect:
        check(true)
    val value = p.illustrate(
      "selected",
      "Owner selected values",
      IllustrationKind.positive,
      plain(),
      Step(accepted, true, List(accepted))
    )
    assertEquals(value.illustrations.size, 1)
    for step <- List(Step(excluded, true, List(accepted)), Step(accepted, true, List(excluded))) do
      intercept[IllegalArgumentException]:
        p.illustrate("excluded", "Outside owner domain", IllustrationKind.positive, plain(), step)

  test(
    "composed owners reject attachment but generic and legacy ordinary properties keep no new duty"
  ):
    def generic[S](owner: Declares[S]): Property = owner.property.expect:
      check(true)
    assert(generic(Together).decl.illustrations.isEmpty)
    assert(generic(Subject).decl.illustrations.isEmpty)
    assert(
      Subject.property.never((_: Step[State, Outcome, Fact]) => false).decl.illustrations.isEmpty
    )
    val composed = Together.property.expect:
      check(true)
    val error = intercept[IllegalArgumentException]:
      composed.illustrate(
        "unsupported",
        "Individual only",
        IllustrationKind.positive,
        start(),
        Step[framework.dsl.State, Any, JoinedFact]("accepted", Together.init)
      )
    assert(error.getMessage.contains("composed"), error.getMessage)
    val derived = DerivedSubject.property.expect:
      check(true)
    val attached = derived.illustrate(
      "derived",
      "This owner",
      IllustrationKind.positive,
      start(),
      Step(Outcome.accepted, State.closed)
    )
    assert(attached.decl.machine eq DerivedSubject)
