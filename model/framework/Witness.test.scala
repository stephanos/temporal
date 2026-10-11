package framework

object WitnessFixture:
  enum State:
    case open, closed
  enum Outcome:
    case accepted
  enum Fact:
    case completed, secondary, tertiary
  enum Foreign:
    case completed

  val actor = Actor("tester")
  val start = action("start", actor)
  val finish = action("finish", actor)

  object Subject extends Machine[State, Outcome, Fact]:
    def init = State.open
    def end(s: WitnessFixture.State) = s.equals(State.closed)

    object rules
        extends Bindings(
          start ~> ((s: WitnessFixture.State) => List(Step(Outcome.accepted, s)))
        )

    object queries:
      val completed = witness(start(), finish()).records(Fact.completed)

  object DerivedSubject extends Derived(Subject.restrict(start))

  final case class Joined(left: State, right: State)

  object Other extends Machine[State, Outcome, Foreign]:
    def init = State.open
    def end(s: WitnessFixture.State) = s.equals(State.closed)

    object rules
        extends Bindings(
          start ~> ((s: WitnessFixture.State) => List(Step(Outcome.accepted, s)))
        )

  object Together extends Composition[Joined](_.left -> Subject, _.right -> Other)

class WitnessTest extends munit.FunSuite:
  import WitnessFixture.*

  test("absent primary and subsequent guards retain typed fact and member identity") {
    import realize.*
    val expected = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val space = Exploration("negative-edits", Vector.empty, runs = 1, edits = 0, dropPrefix = false)
    val q = Subject
      .witness(start())
      .excludes(Fact.completed)
      .recordsWhen(_.equals(State.closed), Fact.secondary)
      .excludes(Fact.tertiary)
      .live(expected)
      .explore(space)
      .ends(_ => true)
      .limits(Limits(steps = 1, actions = 1, search = 17))
    val holds = q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assertEquals(q.expectedRun, Some(expected))
    assertEquals(q.exploration, Some(space))
    assert(holds(Step(Outcome.accepted, State.open)))
    assert(holds(Step(Outcome.accepted, State.closed, List(Fact.secondary))))
    for facts <- List(
        List(Fact.completed),
        List(Fact.secondary, Fact.completed),
        List(Fact.secondary, Fact.tertiary)
      )
    do assert(!holds(Step(Outcome.accepted, State.closed, facts)))
    assert(!holds(Step(Outcome.accepted, State.closed)))
    val positive = Subject.witness(start()).records(Fact.completed).excludes(Fact.secondary)
    val positiveHolds =
      positive.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(positiveHolds(Step(Outcome.accepted, State.closed, List(Fact.completed))))
    assert(
      !positiveHolds(Step(Outcome.accepted, State.closed, List(Fact.completed, Fact.secondary)))
    )
    val member = Together
      .witness(Together.own(_.left, finish()))
      .excludes(_.left -> Subject, Fact.completed)
      .excludes(_.right -> Other, Foreign.completed)
    val memberHolds =
      member.property.holds.get.asInstanceOf[Step[Joined, String, String] => Boolean]
    val joined = Joined(State.closed, State.open)
    assert(memberHolds(Step("accepted", joined)))
    assert(!memberHolds(Step("accepted", joined, List("left_completed"))))
    assert(!memberHolds(Step("accepted", joined, List("right_completed"))))
    val leftOnly = Together
      .witness(Together.own(_.left, finish()))
      .excludes(_.left -> Subject, Fact.completed)
    val leftOnlyHolds = leftOnly.property.holds.get
      .asInstanceOf[Step[Joined, String, String] => Boolean]
    assert(leftOnlyHolds(Step("accepted", joined, List("right_completed"))))
    assert(!leftOnlyHolds(Step("accepted", joined, List("left_completed"))))
    val leftPresent = Together
      .witness(Together.own(_.left, finish()))
      .records(_.left -> Subject, Fact.completed)
      .excludes(_.right -> Other, Foreign.completed)
    val leftPresentHolds = leftPresent.property.holds.get
      .asInstanceOf[Step[Joined, String, String] => Boolean]
    assert(leftPresentHolds(Step("accepted", joined, List("left_completed"))))
    assert(!leftPresentHolds(Step("accepted", joined, List("left_completed", "right_completed"))))
    var mixedCalls = Vector.empty[Int]
    val mixed = Subject
      .witness(start())
      .records(Fact.completed)
      .recordsWhen(
        s =>
          mixedCalls :+= 1;
          s.equals(State.closed)
        ,
        Fact.secondary
      )
      .excludes(Fact.tertiary)
      .recordsWhen(
        s =>
          mixedCalls :+= 2;
          s.equals(State.closed)
        ,
        Fact.completed
      )
    val mixedHolds = mixed.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(
      !mixedHolds(
        Step(Outcome.accepted, State.closed, List(Fact.completed, Fact.secondary, Fact.tertiary))
      )
    )
    assertEquals(mixedCalls, Vector(1))
    mixedCalls = Vector.empty
    assert(mixedHolds(Step(Outcome.accepted, State.closed, List(Fact.completed, Fact.secondary))))
    assertEquals(mixedCalls, Vector(1, 2))
    assert(compileErrors("Subject.witness(start()).excludes(Foreign.completed)").nonEmpty)
    assert(
      compileErrors(
        "Subject.witness(start()).records(Fact.completed).excludes(Foreign.completed)"
      ).nonEmpty
    )
    assert(
      compileErrors(
        "Together.witness(Together.own(_.left, finish())).excludes(_.left -> Subject, Foreign.completed)"
      ).nonEmpty
    )
  }

  test(
    "a witness is an ordered pinned find with derived limits and a typed last-step claim"
  ) {
    val q = Subject.queries.completed.ends(_.equals(State.closed))
    assertEquals(q.form, QueryForm.find)
    assertEquals(q.scenario.actions, Vector(start(), finish()))
    assert(!q.scenario.free)
    assertEquals(q.limits, Limits(steps = 2, actions = 2, search = 512))
    assertEquals(q.property.when, Some(finish()))
    val holds =
      q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(holds(Step(Outcome.accepted, State.closed, List(Fact.completed))))
    assert(!holds(Step(Outcome.accepted, State.open, List(Fact.completed))))
    assert(!holds(Step(Outcome.accepted, State.closed)))
    val bounds = Limits("custom", steps = 3, actions = 3, search = 99)
    assertEquals(q.limits(bounds).limits, bounds)
    assertEquals(q.total, None)
  }

  test(
    "derived owners, explicit list names and Query roots retain their declaration owners"
  ) {
    val qs: Seq[Query] =
      Seq(DerivedSubject.witness("done")(start()).records(Fact.completed))
    assertEquals(qs.head.name, "done")
    assertEquals(qs.head.property.name, "done")
    assertEquals(qs.head.scenario.name, "done")
    assert(qs.head.scenario.machine eq DerivedSubject)
    assert(IrFile("witness", Seq(qs)).subjects().exists(_ eq DerivedSubject))
    assert(
      IrFile("section", Seq(Subject.queries)).subjects().exists(_ eq Subject)
    )
  }

  test("a composition's recorded fact is typed by its selected member") {
    val path = Together.own(_.left, finish())
    val q = Together
      .witness("joined")(path)
      .records(_.left -> Subject, Fact.completed)
    assertEquals(q.scenario.actions, Vector(path))
    assertEquals(q.property.when, Some(path))
    val holds =
      q.property.holds.get.asInstanceOf[Step[Joined, String, String] => Boolean]
    val state = Joined(State.closed, State.open)
    assert(holds(Step("accepted", state, List("left_completed"))))
    assert(!holds(Step("accepted", state, List("right_completed"))))
    val foreign = intercept[IllegalArgumentException](
      Together.witness(path).records(_.left -> DerivedSubject, Fact.completed)
    )
    assert(foreign.getMessage.contains("is not one unique member of together"))
    assert(
      compileErrors(
        "Subject.witness(start()).records(Foreign.completed)"
      ).nonEmpty
    )
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

  test(
    "a satisfied live witness needs no reason and keeps its typed modifiers"
  ) {
    import realize.*
    val satisfied = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val bounds = Limits(steps = 2, actions = 2, search = 32)
    val space = Exploration(
      "edits",
      Vector.empty,
      runs = 1,
      edits = 0,
      dropPrefix = false
    )
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

  test(
    "conditional facts inspect the after state and survive every immutable modifier"
  ) {
    import realize.*
    val expected = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val space = Exploration(
      "conditional",
      Vector.empty,
      runs = 1,
      edits = 0,
      dropPrefix = false
    )
    val q = Subject
      .witness(start())
      .records(Fact.completed)
      .recordsWhen(_.equals(State.closed), Fact.secondary)
      .live(expected)
      .explore(space)
      .recordsWhen(_.equals(State.closed), Fact.tertiary)
      .ends(_ => true)
      .limits(Limits(steps = 1, actions = 1, search = 17))
    assertEquals(q.expectedRun, Some(expected))
    assertEquals(q.exploration, Some(space))
    assertEquals(q.property.when, Some(start()))
    assertEquals(q.scenario.actions, Vector(start()))
    val holds =
      q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(
      holds(
        Step(
          Outcome.accepted,
          State.closed,
          List(Fact.completed, Fact.secondary, Fact.tertiary)
        )
      )
    )
    assert(holds(Step(Outcome.accepted, State.open, List(Fact.completed))))
    assert(
      !holds(
        Step(
          Outcome.accepted,
          State.closed,
          List(Fact.completed, Fact.secondary)
        )
      )
    )
    assert(
      !holds(
        Step(
          Outcome.accepted,
          State.closed,
          List(Fact.completed, Fact.tertiary)
        )
      )
    )
    assert(!holds(Step(Outcome.accepted, State.closed)))
    val joined = Together
      .witness(Together.own(_.left, finish()))
      .records(_.left -> Subject, Fact.completed)
      .recordsWhen(
        _.right.equals(State.closed),
        _.right -> Other,
        Foreign.completed
      )
      .ends(_.left.equals(State.closed))
    val joinedHolds = joined.property.holds.get
      .asInstanceOf[Step[Joined, String, String] => Boolean]
    assert(
      joinedHolds(
        Step(
          "accepted",
          Joined(State.closed, State.open),
          List("left_completed")
        )
      )
    )
    assert(
      !joinedHolds(
        Step(
          "accepted",
          Joined(State.closed, State.closed),
          List("left_completed")
        )
      )
    )
    assert(
      joinedHolds(
        Step(
          "accepted",
          Joined(State.closed, State.closed),
          List("left_completed", "right_completed")
        )
      )
    )
    assert(
      !joinedHolds(
        Step(
          "accepted",
          Joined(State.closed, State.closed),
          List("left_completed", "left_completed")
        )
      )
    )
    val seen = scala.collection.mutable.ArrayBuffer.empty[Int]
    val ordered = Subject
      .witness(start())
      .records(Fact.completed)
      .recordsWhen(
        s =>
          seen += 1;
          s.equals(State.closed)
        ,
        Fact.secondary
      )
      .recordsWhen(
        s =>
          seen += 2;
          s.equals(State.closed)
        ,
        Fact.tertiary
      )
    val orderedHolds = ordered.property.holds.get
      .asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(
      !orderedHolds(Step(Outcome.accepted, State.closed, List(Fact.completed)))
    )
    assertEquals(seen.toVector, Vector(1))
    seen.clear()
    assert(!orderedHolds(Step(Outcome.accepted, State.closed)))
    assertEquals(seen.toVector, Vector.empty)
    assert(
      orderedHolds(Step(Outcome.accepted, State.open, List(Fact.completed)))
    )
    assertEquals(seen.toVector, Vector(1, 2))
    assert(
      compileErrors(
        "Subject.witness(start()).recordsWhen((s: State) => true, Fact.completed)"
      ).nonEmpty
    )
    assert(
      compileErrors(
        "Subject.witness(start()).records(Fact.completed).recordsWhen((s: Int) => true, Fact.completed)"
      ).nonEmpty
    )
    assert(
      compileErrors(
        "Subject.witness(start()).records(Fact.completed).recordsWhen((s: State) => true, Foreign.completed)"
      ).nonEmpty
    )
    assert(
      compileErrors(
        "Together.witness(Together.own(_.left, finish())).records(_.left -> Subject, Fact.completed).recordsWhen((s: Joined) => true, _.left -> Subject, Foreign.completed)"
      ).nonEmpty
    )
  }
