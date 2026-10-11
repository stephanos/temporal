package framework

object WitnessFixture:
  enum State derives Finite:
    case open, closed
  enum Outcome derives Finite:
    case accepted
  enum Fact derives Finite:
    case completed
  enum Foreign derives Finite:
    case completed

  val actor = Actor("tester")
  val start = action("start", actor)
  val finish = action("finish", actor)

  object Subject extends Machine[State, Outcome, Fact]:
    def init = State.open
    def end(s: WitnessFixture.State) = s.equals(State.closed)

    object rules
        extends Bindings(start ~> ((s: WitnessFixture.State) => List(Step(Outcome.accepted, s))))

    object queries:
      val completed = witness(start(), finish()).records(Fact.completed)

  object DerivedSubject extends Derived(Subject.restrict(start))

  final case class Joined(left: State, right: State)

  object Other extends Machine[State, Outcome, Foreign]:
    def init = State.open
    def end(s: WitnessFixture.State) = s.equals(State.closed)

    object rules
        extends Bindings(start ~> ((s: WitnessFixture.State) => List(Step(Outcome.accepted, s))))

  object Together extends Composition[Joined](_.left -> Subject, _.right -> Other)

class WitnessTest extends munit.FunSuite:
  import WitnessFixture.*

  test("a witness is an ordered pinned find with derived limits and a typed last-step claim") {
    val q = Subject.queries.completed.ends(_.equals(State.closed))
    assertEquals(q.form, QueryForm.find)
    assertEquals(q.scenario.actions, Vector(start(), finish()))
    assert(!q.scenario.free)
    assertEquals(q.limits, Limits(steps = 2, actions = 2, search = 512))
    assertEquals(q.property.when, Some(finish()))
    val holds = q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(holds(Step(Outcome.accepted, State.closed, List(Fact.completed))))
    assert(!holds(Step(Outcome.accepted, State.open, List(Fact.completed))))
    assert(!holds(Step(Outcome.accepted, State.closed)))
    val bounds = Limits("custom", steps = 3, actions = 3, search = 99)
    assertEquals(q.limits(bounds).limits, bounds)
    assertEquals(q.total, None)
  }

  test("derived owners, explicit list names and Query roots retain their declaration owners") {
    val qs: Seq[Query] = Seq(DerivedSubject.witness("done")(start()).records(Fact.completed))
    assertEquals(qs.head.name, "done")
    assertEquals(qs.head.property.name, "done")
    assertEquals(qs.head.scenario.name, "done")
    assert(qs.head.scenario.machine eq DerivedSubject)
    assert(IrFile("witness", Seq(qs)).subjects().exists(_ eq DerivedSubject))
    assert(IrFile("section", Seq(Subject.queries)).subjects().exists(_ eq Subject))
  }

  test("a composition's recorded fact is typed by its selected member") {
    val path = Together.own(_.left, finish())
    val q = Together.witness("joined")(path).records(_.left -> Subject, Fact.completed)
    assertEquals(q.scenario.actions, Vector(path))
    assertEquals(q.property.when, Some(path))
    val holds = q.property.holds.get.asInstanceOf[Step[Joined, String, String] => Boolean]
    val state = Joined(State.closed, State.open)
    assert(holds(Step("accepted", state, List("left_completed"))))
    assert(!holds(Step("accepted", state, List("right_completed"))))
    val foreign = intercept[IllegalArgumentException](
      Together.witness(path).records(_.left -> DerivedSubject, Fact.completed)
    )
    assert(foreign.getMessage.contains("is not one unique member of together"))
    assert(compileErrors("Subject.witness(start()).records(Foreign.completed)").nonEmpty)
    assert(
      compileErrors(
        "Together.witness(Together.own(_.left, finish())).records(_.left -> Subject, Foreign.completed)"
      ).nonEmpty
    )
    assert(compileErrors("val q: Query = Subject.witness(start())").nonEmpty)
    assert(compileErrors("Subject.witness()").nonEmpty)
    assert(
      compileErrors(
        "Subject.witness(start()).records(Fact.completed).ends((s: Int) => true)"
      ).nonEmpty
    )
  }

  test("a satisfied live witness needs no reason and keeps its typed modifiers") {
    import realize.*
    val satisfied = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val bounds = Limits(steps = 2, actions = 2, search = 32)
    val space = Exploration("edits", Vector.empty, runs = 1, edits = 0, dropPrefix = false)
    val q = Subject
      .witness(start(), finish())
      .records(Fact.completed)
      .live(satisfied)
      .explore(space)
      .ends(_.equals(State.closed))
      .limits(bounds)
    assertEquals(q.expectedRun, Some(satisfied))
    assertEquals(q.exploration, Some(space))
    assertEquals(q.limits, bounds)
    assertEquals(q.expectedRun.get.because, "")
  }
