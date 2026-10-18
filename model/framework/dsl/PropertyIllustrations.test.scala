package framework.dsl

import framework.dsl.Finite.given

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
  given Ok[Boolean] = Ok(true)
  given Ok[Answer.Value] = Ok(accepted)
  def guard: Boolean = throw AssertionError("guard evaluated")

  object Native extends Machine[Boolean, Boolean](using Finite.of(true), Finite.of(false)):
    var payload = NativeState(0, Nil)
    override def end: Boolean = throw AssertionError("end evaluated")

    object transitions extends Transitions:
      on(PropertyIllustrationsFixture.plain):
        if guard then effects.fail()
      on(PropertyIllustrationsFixture.correlated): input =>
        effects.inputFail(input)

    object effects:
      def fail(): Unit = throw AssertionError("effect evaluated")
      def inputFail(input: Input): Unit =
        val _ = input
        throw AssertionError("input effect evaluated")

  object Opaque
      extends Machine[Answer.Value, Answer.Value](using Finite.of(accepted), Finite.of(accepted)):
    var ready = false
    override def end: Boolean = ready

    object transitions extends Transitions:
      on(PropertyIllustrationsFixture.plain):
        effects.noop()

    object effects:
      def noop(): Unit = ()

class PropertyIllustrationsSuite extends munit.FunSuite:
  import WitnessFixture.*

  private def opened: State = Subject.init.copy(phase = Phase.open)
  private def closed: State = Subject.init.copy(phase = Phase.closed)

  test("typed completions append ordered illustrations without changing ordinary declarations"):
    val original: SameStepProperty[State, Outcome, Fact] =
      WitnessFixture.sameStep("completion")(_.facts.contains(Fact.completed))
    val step = Step(Outcome.accepted, closed, List(Fact.completed, Fact.completed))
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
      Step(Outcome.accepted, closed)
    )
    val ordinary: Property = second
    val q = ClaimQuery.property(QueryForm.verify, ordinary)
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
    val original: TransitionProperty[State, Outcome, Fact] =
      WitnessFixture.transition("closed")((before, after) => before.equals(after.state))
    val attached = original.illustrate(
      "hypothetical",
      "The input need not be reachable",
      IllustrationKind.counterexample,
      start(),
      closed,
      Step(Outcome.accepted, opened)
    )
    assertEquals(attached.illustrations.head.before, Some(closed))
    assert(attached.decl.holds2 eq original.decl.holds2)

  test("metadata validation is located and leaves previous immutable copies intact"):
    val p = WitnessFixture.sameStep("metadata")(_ => true)
    val step = Step[State, Outcome, Fact](Outcome.accepted, opened)
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
    val p = new PropertyBuilder[State, Outcome, Fact]("unselected", Subject, Some("not-start"))
      .sameStep(_ => true)
    val step = Step[State, Outcome, Fact](Outcome.accepted, opened)
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
    val p = WitnessFixture.sameStep("independent")(_.facts.contains(Fact.completed))
    val predicate = p.decl.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(predicate(Step(Outcome.accepted, opened, List(Fact.completed))))
    assert(!predicate(Step(Outcome.accepted, opened)))
    assert(p.decl.illustrations.isEmpty)

  test(
    "admission uses actual selected domains and never reads modeled callbacks or state catalogs"
  ):
    import PropertyIllustrationsFixture.*
    def claim(read: () => Boolean) =
      new PropertyBuilder[State, Boolean, Boolean]("native", Native, Some(plain()))
        .sameStep(_ => read())
    val p = claim(() => throw AssertionError("predicate evaluated"))
    val state = Native.init.copy(payload = NativeState(Int.MaxValue, List(Int.MinValue, 1000000)))
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
      compileErrors("""
        import WitnessFixture.*
        WitnessFixture.sameStep("typed")(_ => true).illustrate(
          "x", "why", IllustrationKind.positive, start(), Step(Outcome.accepted, 1))
      """),
      compileErrors("""
        import WitnessFixture.*
        WitnessFixture.sameStep("typed")(_ => true).illustrate(
          "x", "why", IllustrationKind.positive, start(), Step(false, Subject.init))
      """),
      compileErrors("""
        import WitnessFixture.*
        WitnessFixture.sameStep("typed")(_ => true).illustrate(
          "x", "why", IllustrationKind.positive, start(),
          Step(Outcome.accepted, Subject.init, List(false)))
      """),
      compileErrors("""
        import WitnessFixture.*
        WitnessFixture.transition("typed")((_, _) => true).illustrate(
          "x", "why", IllustrationKind.positive, start(), Step(Outcome.accepted, Subject.init))
      """),
      compileErrors("""
        import WitnessFixture.*
        WitnessFixture.sameStep("typed")(_ => true).illustrate(
          "x", "why", IllustrationKind.positive, start(), Subject.init,
          Step(Outcome.accepted, Subject.init))
      """),
      compileErrors("""
        import WitnessFixture.*
        WitnessFixture.sameStep("typed")(_ => true).illustrate(
          "x", "why", IllustrationKind.unspecified, start(),
          Step(Outcome.accepted, Subject.init))
      """)
    )
    assertEquals(invalid.zipWithIndex.filter((message, _) => message.isEmpty), Nil)

  test("opaque outcomes and facts use selected owner evidence instead of an ambient witness"):
    import PropertyIllustrationsFixture.*
    given Finite[Answer.Value] = new Finite[Answer.Value]:
      def values = throw AssertionError("ambient outcome/fact domain evaluated")
    val _ = () => summon[Finite[Answer.Value]]
    val p = new PropertyBuilder[State, Answer.Value, Answer.Value]("selectedOpaque", Opaque, None)
      .sameStep(_ => true)
    val value = p.illustrate(
      "selected",
      "Owner selected values",
      IllustrationKind.positive,
      plain(),
      Step(accepted, Opaque.init.copy(ready = true), List(accepted))
    )
    assertEquals(value.illustrations.size, 1)
    for step <- List(
        Step(excluded, Opaque.init.copy(ready = true), List(accepted)),
        Step(accepted, Opaque.init.copy(ready = true), List(excluded))
      )
    do
      intercept[IllegalArgumentException]:
        p.illustrate("excluded", "Outside owner domain", IllustrationKind.positive, plain(), step)

  test(
    "composed owners reject attachment but generic ordinary properties keep no new duty"
  ):
    def generic[S](owner: Declares[S]): Property =
      new Property(new PropertyDecl("", owner, None, None, None))
    assert(generic(Together).decl.illustrations.isEmpty)
    assert(generic(Subject).decl.illustrations.isEmpty)
    val composed = new PropertyBuilder[State, Any, JoinedFact]("", Together, None)
      .sameStep(_ => true)
    val error = intercept[IllegalArgumentException]:
      composed.illustrate(
        "unsupported",
        "Individual only",
        IllustrationKind.positive,
        start(),
        Step[framework.dsl.State, Any, JoinedFact]("accepted", Together.init)
      )
    assert(error.getMessage.contains("composed"), error.getMessage)
    val derived = new PropertyBuilder[State, Outcome, Fact]("", DerivedSubject, None)
      .sameStep(_ => true)
    val attached = derived.illustrate(
      "derived",
      "This owner",
      IllustrationKind.positive,
      start(),
      Step(Outcome.accepted, closed)
    )
    assert(attached.decl.machine eq DerivedSubject)
