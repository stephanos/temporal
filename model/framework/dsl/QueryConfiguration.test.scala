package framework.dsl

object QueryConfigurationFixture:
  enum Phase:
    case ready, done
  enum Outcome:
    case accepted
  enum Fact:
    case recorded
  given Ok[Outcome] = Ok(Outcome.accepted)
  val actor = Actor("queryTester")
  val advance = action("advance", actor)

  object Default extends Machine[Outcome, Fact]:
    var phase = Phase.ready
    override def end: Boolean = phase.equals(Phase.done)

    object transitions extends Transitions
  val selected = PropertyBuilder[State, Outcome, Fact]("selected", Default, Some(advance()))
    .sameStep(step => Default.at(step.state)(Default.phase.equals(Phase.done)))
  val path = Default.scenario("path").starts(Default.init).actions(advance())
  def question: Query =
    new Query("question", QueryForm.find, selected.decl, path.decl, frameworkReach)

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

  test("find and verify copies retain declarations selectors predicates and realization metadata") {
    for form <- Seq(QueryForm.find, QueryForm.verify) do
      val original = new Query("copy", form, selected.decl, path.decl, frameworkReach)
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
        assertEquals(q.scenario.start, Some(Default.init))
        assertEquals(q.scenario.actions, Vector(advance()))
        assert(!q.scenario.free)
        assert(q.limits eq original.limits)
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
    object Poison extends Machine[Outcome, Fact]:
      var ready = false
      override def end: Boolean = throw AssertionError("end executed")
      def enabled: Boolean = throw AssertionError("guard executed")
      object transitions extends Transitions:
        on(QueryConfigurationFixture.advance):
          if enabled then effects.poison()
      object effects:
        def poison(): Unit =
          throw AssertionError("effect executed")
    val _ = Poison.transitions
    val p = PropertyBuilder[State, Outcome, Fact]("poison", Poison, None)
      .sameStep(_ => throw AssertionError("predicate executed"))
    val q = new Query("poison", QueryForm.verify, p.decl, Poison.scenario.free.decl, frameworkReach)
      .checkedBy(Checker.apalache)
      .confirm(Checker.tlc)
      .open(because = "initial state only")
      .live(expected)
      .explore(space)
    assert(q.property.holds.nonEmpty)
    assert(q.scenario.machine eq Poison)
    assertEquals(q.checker, Checker.apalache)
    assertEquals(q.confirmations, Vector(Checker.tlc))
    assertEquals(q.openReason, Some("initial state only"))
    val witness = Poison
      .witness("poisonWitness")(advance())
      .records(Fact.recorded)
      .recordsWhen((_: State) => throw AssertionError("witness condition executed"), Fact.recorded)
      .ends((_: State) => throw AssertionError("witness ending executed"))
      .open(because = "declaration only")
      .checkedBy(Checker.tlc)
      .confirm(Checker.apalache)
      .live(expected)
      .explore(space)
    assertEquals(witness.name, "poisonWitness")
    assert(witness.property.holds.nonEmpty)
    assertEquals(witness.expectedRun, Some(expected))
    assertEquals(witness.exploration, Some(space))
  }
