package framework.dsl

object QueryConfigurationFixture:
  enum State:
    case ready, done
  enum Outcome:
    case accepted
  enum Fact:
    case recorded

  val actor = Actor("queryTester")
  val advance = action("advance", actor)

  object Default extends Machine[State, Outcome, Fact]:
    def init = State.ready
    def end(s: QueryConfigurationFixture.State) = s.equals(State.done)

    object rules
        extends Bindings(
          advance ~> ((s: QueryConfigurationFixture.State) => List(Step(Outcome.accepted, s)))
        )

  object Bounded extends Machine[State, Outcome, Fact]:
    override val reach = Limits("ownerReach", steps = 7, actions = 13, search = 29)
    def init = State.ready
    def end(s: QueryConfigurationFixture.State) = s.equals(State.done)

    object rules
        extends Bindings(
          advance ~> ((s: QueryConfigurationFixture.State) => List(Step(Outcome.accepted, s)))
        )

  object Inherited extends Derived(Bounded.restrict(advance))

  object Overridden extends Derived(Bounded.restrict(advance)):
    override val reach = Limits("derivedReach", steps = 3, actions = 5, search = 11)

  final case class Joined(member: State)

  object DefaultComposition extends Composition[Joined](_.member -> Bounded)

  object BoundedComposition extends Composition[Joined](_.member -> Bounded):
    override val reach = Limits("compositionReach", steps = 9, actions = 17, search = 31)

  object Replaced extends Composition(BoundedComposition.withMember(_.member -> Inherited))

  object DerivedJoined
      extends DerivedComposition(BoundedComposition.withMember(_.member -> Inherited))

  val bounds = Limits("question", steps = 2, actions = 1, search = 41)
  val selected = Default
    .property("selected")
    .when(advance())
    .expect:
      check(after.state.equals(State.done))
  val path = Default.scenario("path").starts(State.ready).actions(advance())

  def question: Query = query("question").find(selected).in(path).limits(bounds)

class QueryConfigurationTest extends munit.FunSuite:
  import QueryConfigurationFixture.*
  import framework.realize.{
    Cleanup,
    Conformance,
    Disposition,
    Exploration,
    PropertyOutcome,
    RunExpectation
  }

  private val expected = RunExpectation(
    Conformance.conformant,
    PropertyOutcome.satisfied,
    PropertyOutcome.satisfied,
    Disposition.completed,
    Cleanup.succeeded
  )
  private val space =
    Exploration("queryEdits", Vector.empty, runs = 1, edits = 0, dropPrefix = false)

  test("ordinary declarations independently pin framework reach and checker defaults") {
    assertEquals(frameworkReach.name, "frameworkReach")
    assertEquals(frameworkReach.steps, 256)
    assertEquals(frameworkReach.actions, 0)
    assertEquals(frameworkReach.search, 65536)
    for owner <- Seq(Default, DefaultComposition) do
      assertEquals(owner.reach.steps, 256)
      assertEquals(owner.reach.actions, 0)
      assertEquals(owner.reach.search, 65536)
    val q = question
    assertEquals(q.name, "question")
    assertEquals(q.form, QueryForm.find)
    assertEquals(q.reach.steps, 256)
    assertEquals(q.reach.actions, 0)
    assertEquals(q.reach.search, 65536)
    assertEquals(q.checker, Checker.umpire)
    assertEquals(q.confirmations, Vector.empty[Checker])
    assertEquals(q.openReason, None)
    assertEquals(q.limits, Limits("question", steps = 2, actions = 1, search = 41))
  }

  test("reach follows every machine derivation and permits intentional derived overrides") {
    val owners = Seq(
      Bounded.restrict(advance),
      Bounded.rebind(advance ~> ((s: State) => List(Step(Outcome.accepted, s)))),
      Bounded.extend(),
      Bounded.refining(Default)(identity),
      Bounded.assuming(),
      Bounded.unmonitored,
      Inherited
    )
    for owner <- owners do
      assertEquals(owner.reach, Limits("ownerReach", steps = 7, actions = 13, search = 29))
    assertEquals(Overridden.reach, Limits("derivedReach", steps = 3, actions = 5, search = 11))
  }

  test("composition replacement and both derived forms retain composition reach") {
    object OverrideJoined extends Composition(BoundedComposition.withMember(_.member -> Inherited)):
      override val reach = Limits("replacementReach", steps = 2, actions = 19, search = 37)
    for owner <- Seq(BoundedComposition.withMember(_.member -> Inherited), Replaced, DerivedJoined)
    do assertEquals(owner.reach, Limits("compositionReach", steps = 9, actions = 17, search = 31))
    assertEquals(
      OverrideJoined.reach,
      Limits("replacementReach", steps = 2, actions = 19, search = 37)
    )
  }

  test("a refined Property takes reach from its Scenario owner") {
    object Refining extends Derived(Bounded.refining(Default)(identity))
    val q = query("refined").verify(selected).in(Refining.scenario("refining").free).limits(bounds)
    assert(q.property.machine eq Default)
    assert(q.scenario.machine eq Refining)
    assertEquals(q.reach, Limits("ownerReach", steps = 7, actions = 13, search = 29))
  }

  test("reach is not read before the owner constructor finishes") {
    var initialized = false
    var reads = 0
    class ConstructorBound extends Machine[State, Outcome, Fact]:
      override def reach: Limits =
        assert(initialized, "reach read during superclass construction")
        reads += 1
        Limits("constructorReach", steps = 4, actions = 6, search = 53)
      def init = State.ready
      def end(s: QueryConfigurationFixture.State) = s.equals(State.done)
      object rules
          extends Bindings(
            QueryConfigurationFixture.advance ~>
              ((s: QueryConfigurationFixture.State) => List(Step(Outcome.accepted, s)))
          )
      initialized = true
    val owner = new ConstructorBound
    assert(initialized)
    assertEquals(reads, 0)
    val q = query("constructed").find(selected).in(owner.scenario.free).limits(bounds)
    assertEquals(q.reach, Limits("constructorReach", steps = 4, actions = 6, search = 53))
    assert(reads > 0)
  }

  test("reach validation rejects malformed bounds when consumed natively") {
    for (limits, field) <- Seq(
        Limits(steps = -1, actions = 0, search = 1) -> "steps",
        Limits(steps = 0, actions = -1, search = 1) -> "actions",
        Limits(steps = 0, actions = 0, search = 0) -> "search",
        Limits(steps = 0, actions = 0, search = -1) -> "search"
      )
    do
      object Invalid extends Derived(Default.restrict(advance)):
        override val reach = limits
      val error = intercept[IllegalArgumentException](
        query("invalid").find(selected).in(Invalid.scenario.free).limits(bounds).reach
      )
      assert(error.getMessage.contains("reach"), error.getMessage)
      assert(error.getMessage.contains(field), error.getMessage)
  }

  test("zero depth reach and ordinary zero Query controls remain valid independently") {
    object InitialOnly extends Derived(Default.restrict(advance)):
      override val reach = Limits(steps = 0, actions = 0, search = 1)
    val q = query("initialOnly")
      .find(selected)
      .in(InitialOnly.scenario.free)
      .limits(Limits(steps = 0, actions = 0, search = 0))
    assertEquals(q.reach, Limits(steps = 0, actions = 0, search = 1))
    assertEquals(q.limits, Limits(steps = 0, actions = 0, search = 0))
  }

  test("find and verify copies retain declarations selectors predicates and realization metadata") {
    for form <- Seq(QueryForm.find, QueryForm.verify) do
      val root = if form.equals(QueryForm.find) then query("copy").find(selected)
      else query("copy").verify(selected)
      val original = root.in(path).limits(bounds)
      val configured = original
        .open(because = "  bounded intentionally\n")
        .checkedBy(Checker.tlc)
        .confirm(Checker.apalache, Checker.umpire)
      val copies = Seq(
        configured.explore(space).live(expected),
        original
          .live(expected)
          .explore(space)
          .open(because = "  bounded intentionally\n")
          .checkedBy(Checker.tlc)
          .confirm(Checker.apalache, Checker.umpire)
      )
      for q <- copies do
        assertEquals(q.name, "copy")
        assertEquals(q.form, form)
        assert(q.property eq selected.decl)
        assert(q.scenario eq path.decl)
        assertEquals(q.property.when, Some(advance()))
        assertEquals(q.property.holds, original.property.holds)
        assertEquals(q.property.holds2, original.property.holds2)
        assertEquals(q.scenario.start, Some(State.ready))
        assertEquals(q.scenario.actions, Vector(advance()))
        assert(!q.scenario.free)
        assertEquals(q.limits, Limits("question", steps = 2, actions = 1, search = 41))
        assertEquals(q.expectedRun, Some(expected))
        assertEquals(q.exploration, Some(space))
        assertEquals(q.openReason, Some("  bounded intentionally\n"))
        assertEquals(q.checker, Checker.tlc)
        assertEquals(q.confirmations, Vector(Checker.apalache, Checker.umpire))
  }

  test("open refuses blank reasons") {
    for reason <- Seq("", " ", "\t\n") do
      val error = intercept[IllegalArgumentException](question.open(because = reason))
      assert(error.getMessage.contains("open"), error.getMessage)
      assert(error.getMessage.contains("reason"), error.getMessage)
  }

  test("open is assigned once even for the same reason") {
    val error = intercept[IllegalArgumentException](
      question.open(because = "bounded").live(expected).open(because = "bounded")
    )
    assert(error.getMessage.contains("open"), error.getMessage)
    assert(error.getMessage.contains("once"), error.getMessage)
  }

  test("checkedBy is assigned once including the default checker") {
    val error = intercept[IllegalArgumentException](
      question.checkedBy(Checker.umpire).explore(space).checkedBy(Checker.umpire)
    )
    assert(error.getMessage.contains("checkedBy"), error.getMessage)
    assert(error.getMessage.contains("once"), error.getMessage)
  }

  test("confirm requires at least one checker") {
    val error = intercept[IllegalArgumentException](question.confirm())
    assert(error.getMessage.contains("confirm"), error.getMessage)
    assert(error.getMessage.contains("nonempty"), error.getMessage)
  }

  test("confirm refuses duplicate checkers") {
    val error = intercept[IllegalArgumentException](question.confirm(Checker.tlc, Checker.tlc))
    assert(error.getMessage.contains("duplicate"), error.getMessage)
  }

  test("confirm is assigned once rather than appended") {
    val error = intercept[IllegalArgumentException](
      question.confirm(Checker.tlc).live(expected).confirm(Checker.apalache)
    )
    assert(error.getMessage.contains("confirm"), error.getMessage)
    assert(error.getMessage.contains("once"), error.getMessage)
  }

  test("confirm excludes both the default and explicitly selected primary") {
    for q <- Seq(question, question.checkedBy(Checker.apalache)) do
      val error = intercept[IllegalArgumentException](q.confirm(q.checker))
      assert(error.getMessage.contains("primary"), error.getMessage)
  }

  test("confirm fixes the primary and checkedBy reports the required ordering") {
    val q = question.confirm(Checker.tlc, Checker.apalache)
    assertEquals(q.checker, Checker.umpire)
    assertEquals(q.confirmations, Vector(Checker.tlc, Checker.apalache))
    val error = intercept[IllegalArgumentException](q.checkedBy(Checker.tlc))
    assert(error.getMessage.contains("checkedBy"), error.getMessage)
    assert(error.getMessage.contains("before"), error.getMessage)
    assert(error.getMessage.contains("confirm"), error.getMessage)
  }

  test("metadata construction copying and inspection do not execute model callbacks") {
    object Poison extends Machine[State, Outcome, Fact]:
      override val reach = Limits("poisonReach", steps = 0, actions = 23, search = 43)
      def init: State = throw AssertionError("init executed")
      override def end(s: State): Boolean = throw AssertionError("end executed")
      object rules extends Rules:
        on(QueryConfigurationFixture.advance):
          where((_: State) => throw AssertionError("guard executed")) ~>
            ((_: State) => throw AssertionError("effect executed"))
    val _ = Poison.rules
    val p = Poison.property("poison").sameStep(_ => throw AssertionError("predicate executed"))
    val q = query("poison")
      .verify(p)
      .in(Poison.scenario.free)
      .limits(bounds)
      .checkedBy(Checker.apalache)
      .confirm(Checker.tlc)
      .open(because = "initial state only")
      .live(expected)
      .explore(space)
    assertEquals(q.reach, Limits("poisonReach", steps = 0, actions = 23, search = 43))
    assert(q.property.holds.nonEmpty)
    assert(q.scenario.machine eq Poison)
    assertEquals(q.checker, Checker.apalache)
    assertEquals(q.confirmations, Vector(Checker.tlc))
    assertEquals(q.openReason, Some("initial state only"))
    val witness = Poison
      .witness("poisonWitness")(advance())
      .records(Fact.recorded)
      .recordsWhen(_ => throw AssertionError("witness condition executed"), Fact.recorded)
      .ends(_ => throw AssertionError("witness ending executed"))
      .open(because = "declaration only")
      .checkedBy(Checker.tlc)
      .confirm(Checker.apalache)
      .limits(bounds)
      .live(expected)
      .explore(space)
    assertEquals(witness.name, "poisonWitness")
    assertEquals(witness.reach.steps, 0)
    assert(witness.property.holds.nonEmpty)
    assertEquals(witness.expectedRun, Some(expected))
    assertEquals(witness.exploration, Some(space))
  }
