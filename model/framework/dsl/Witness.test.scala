package framework.dsl

import framework.dsl.Finite.given

object WitnessFixture:
  enum Phase:
    case open, closed
  enum Outcome:
    case accepted
  given Ok[Outcome] = Ok(Outcome.accepted)
  enum Fact:
    case completed, secondary, tertiary
  enum Foreign:
    case completed

  val actor = Actor("tester")
  val start = action("start", actor)
  val finish = action("finish", actor)

  object Subject extends Machine[Outcome, Fact]:
    var phase = Phase.open
    override def end = phase == Phase.closed

    object transitions extends Transitions:
      on(WitnessFixture.start):
        effects.start()

    object effects:
      def start(): Unit =
        ()

    object queries:
      val completed = witness(start(), finish()).records(Fact.completed)

  object DerivedSubject extends Derived(Subject.unmonitored)
  final case class Joined(left: Phase, right: Phase)

  object Other extends Machine[Outcome, Foreign]:
    var phase = Phase.open
    override def end = phase == Phase.closed

    object transitions extends Transitions:
      on(WitnessFixture.start):
        effects.start()

    object effects:
      def start(): Unit =
        ()

  enum JoinedFact:
    case left(value: Fact)
    case right(value: Foreign)

  object NativeSubject extends Machine[Outcome, Fact]:
    var phase = Phase.open

    object transitions extends Transitions:
      on(WitnessFixture.start):
        effects.start()
      on(WitnessFixture.finish):
        effects.finish()

    object effects:
      def start() =
        phase = Phase.closed
        records(Fact.completed)
      def finish() =
        phase = Phase.open

  object NativeOther extends Machine[Outcome, Foreign]:
    var phase = Phase.open

    object transitions extends Transitions:
      on(WitnessFixture.start):
        effects.start()

    object effects:
      def start() =
        phase = Phase.closed
        records(Foreign.completed)

  object Together extends Composition[JoinedFact]:
    val left = NativeSubject
    val right = NativeOther

  def joined(left: Phase, right: Phase): framework.dsl.State =
    Together.init.copy(
      left = NativeSubject.init.copy(phase = left),
      right = NativeOther.init.copy(phase = right)
    )

  def sameStep(name: String)(
      predicate: Step[framework.dsl.State, Outcome, Fact] => Boolean
  ): SameStepProperty[framework.dsl.State, Outcome, Fact] =
    new PropertyBuilder[framework.dsl.State, Outcome, Fact](name, Subject, None).sameStep(predicate)

  def transition(name: String)(
      predicate: (framework.dsl.State, Step[framework.dsl.State, Outcome, Fact]) => Boolean
  ): TransitionProperty[framework.dsl.State, Outcome, Fact] =
    new PropertyBuilder[framework.dsl.State, Outcome, Fact](name, Subject, None)
      .transition(predicate)

class WitnessTest extends munit.FunSuite:
  import WitnessFixture.*

  private def opened: State = Subject.init.copy(phase = Phase.open)
  private def closed: State = Subject.init.copy(phase = Phase.closed)

  extension [S, O, F](value: Witness[S, O, F])
    private def withBounds(bounds: QueryBounds): Witness[S, O, F] =
      new Witness(
        value.owner,
        value.name,
        value.path,
        value.recorded,
        value.conditional,
        value.ending,
        bounds,
        value.configuration
      )

  test("query configuration preserves specialized witness refinements in both permitted orders") {
    import framework.realize.{
      Cleanup,
      Conformance,
      Disposition,
      Exploration,
      PropertyOutcome,
      RunExpectation
    }
    val expected = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val space = Exploration("witnessEdits", Vector.empty, runs = 1, edits = 0, dropPrefix = false)
    val bounds = QueryBounds("witnessBounds", steps = 3, actions = 2, search = 47)
    val before: Witness[State, Outcome, Fact] = Subject
      .witness("configured")(start(), finish())
      .records(Fact.completed)
      .open(because = "  frontier accepted\n")
      .checkedBy(Checker.apalache)
      .confirm(Checker.tlc, Checker.umpire)
      .recordsWhen(_.equals(closed), Fact.secondary)
      .excludes(Fact.tertiary)
      .ends(_.equals(closed))
      .withBounds(bounds)
      .live(expected)
      .explore(space)
    val after: Witness[State, Outcome, Fact] = Subject
      .witness("configured")(start(), finish())
      .records(Fact.completed)
      .recordsWhen(_.equals(closed), Fact.secondary)
      .excludes(Fact.tertiary)
      .ends(_.equals(closed))
      .withBounds(bounds)
      .live(expected)
      .explore(space)
      .checkedBy(Checker.apalache)
      .confirm(Checker.tlc, Checker.umpire)
      .open(because = "  frontier accepted\n")
    for q <- Seq(before, after) do
      assertEquals(q.name, "configured")
      assertEquals(q.property.name, "configured")
      assertEquals(q.scenario.name, "configured")
      assertEquals(q.form, QueryForm.find)
      assert(q.property.machine eq Subject)
      assert(q.scenario.machine eq Subject)
      assertEquals(q.property.when, Some(finish()))
      assertEquals(q.property.holds2, None)
      assertEquals(q.scenario.start, None)
      assertEquals(q.scenario.actions, Vector(start(), finish()))
      assert(!q.scenario.free)
      assertEquals(q.limits, QueryBounds("witnessBounds", steps = 3, actions = 2, search = 47))
      assertEquals(q.expectedRun, Some(expected))
      assertEquals(q.exploration, Some(space))
      assertEquals(q.openReason, Some("  frontier accepted\n"))
      assertEquals(q.checker, Checker.apalache)
      assertEquals(q.confirmations, Vector(Checker.tlc, Checker.umpire))
      val holds = q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
      assert(holds(Step(Outcome.accepted, closed, List(Fact.completed, Fact.secondary))))
      for (state, facts) <- Seq(
          opened -> List(Fact.completed, Fact.secondary),
          closed -> List(Fact.secondary),
          closed -> List(Fact.completed),
          closed -> List(Fact.completed, Fact.secondary, Fact.tertiary)
        )
      do assert(!holds(Step(Outcome.accepted, state, facts)))
  }

  test("absent primary and subsequent guards retain typed fact and member identity") {
    import framework.realize.*
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
      .recordsWhen(_.equals(closed), Fact.secondary)
      .excludes(Fact.tertiary)
      .live(expected)
      .explore(space)
      .ends(_ => true)
      .withBounds(QueryBounds(steps = 1, actions = 1, search = 17))
    val holds = q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assertEquals(q.expectedRun, Some(expected))
    assertEquals(q.exploration, Some(space))
    assert(holds(Step(Outcome.accepted, opened)))
    assert(holds(Step(Outcome.accepted, closed, List(Fact.secondary))))
    for facts <- List(
        List(Fact.completed),
        List(Fact.secondary, Fact.completed),
        List(Fact.secondary, Fact.tertiary)
      )
    do assert(!holds(Step(Outcome.accepted, closed, facts)))
    assert(!holds(Step(Outcome.accepted, closed)))
    val positive = Subject.witness(start()).records(Fact.completed).excludes(Fact.secondary)
    val positiveHolds =
      positive.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(positiveHolds(Step(Outcome.accepted, closed, List(Fact.completed))))
    assert(
      !positiveHolds(Step(Outcome.accepted, closed, List(Fact.completed, Fact.secondary)))
    )
    val member = Together
      .witness(Together.left(finish()))
      .excludes(JoinedFact.left(Fact.completed))
      .excludes(JoinedFact.right(Foreign.completed))
    val memberHolds =
      member.property.holds.get
        .asInstanceOf[Step[framework.dsl.State, String, JoinedFact] => Boolean]
    val joined = WitnessFixture.joined(Phase.closed, Phase.open)
    assert(memberHolds(Step("accepted", joined)))
    assert(!memberHolds(Step("accepted", joined, List(JoinedFact.left(Fact.completed)))))
    assert(!memberHolds(Step("accepted", joined, List(JoinedFact.right(Foreign.completed)))))
    val leftOnly = Together
      .witness(Together.left(finish()))
      .excludes(JoinedFact.left(Fact.completed))
    val leftOnlyHolds = leftOnly.property.holds.get
      .asInstanceOf[Step[framework.dsl.State, String, JoinedFact] => Boolean]
    assert(leftOnlyHolds(Step("accepted", joined, List(JoinedFact.right(Foreign.completed)))))
    assert(!leftOnlyHolds(Step("accepted", joined, List(JoinedFact.left(Fact.completed)))))
    val leftPresent = Together
      .witness(Together.left(finish()))
      .records(JoinedFact.left(Fact.completed))
      .excludes(JoinedFact.right(Foreign.completed))
    val leftPresentHolds = leftPresent.property.holds.get
      .asInstanceOf[Step[framework.dsl.State, String, JoinedFact] => Boolean]
    assert(leftPresentHolds(Step("accepted", joined, List(JoinedFact.left(Fact.completed)))))
    assert(
      !leftPresentHolds(
        Step(
          "accepted",
          joined,
          List(JoinedFact.left(Fact.completed), JoinedFact.right(Foreign.completed))
        )
      )
    )
    var mixedCalls = Vector.empty[Int]
    val mixed = Subject
      .witness(start())
      .records(Fact.completed)
      .recordsWhen(
        s =>
          mixedCalls :+= 1;
          s.equals(closed)
        ,
        Fact.secondary
      )
      .excludes(Fact.tertiary)
      .recordsWhen(
        s =>
          mixedCalls :+= 2;
          s.equals(closed)
        ,
        Fact.completed
      )
    val mixedHolds = mixed.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(
      !mixedHolds(
        Step(Outcome.accepted, closed, List(Fact.completed, Fact.secondary, Fact.tertiary))
      )
    )
    assertEquals(mixedCalls, Vector(1))
    mixedCalls = Vector.empty
    assert(mixedHolds(Step(Outcome.accepted, closed, List(Fact.completed, Fact.secondary))))
    assertEquals(mixedCalls, Vector(1, 2))
    assert(compileErrors("Subject.witness(start()).excludes(Foreign.completed)").nonEmpty)
    assert(
      compileErrors(
        "Subject.witness(start()).records(Fact.completed).excludes(Foreign.completed)"
      ).nonEmpty
    )
    assert(
      compileErrors(
        "Together.witness(Together.left(finish())).excludes(Fact.completed)"
      ).nonEmpty
    )
  }

  test(
    "a witness is an ordered pinned find with derived limits and a typed last-step claim"
  ) {
    val q = Subject.queries.completed.ends(_.equals(closed))
    assertEquals(q.form, QueryForm.find)
    assertEquals(q.scenario.actions, Vector(start(), finish()))
    assert(!q.scenario.free)
    assertEquals(q.limits, QueryBounds(steps = 2, actions = 2, search = 512))
    assertEquals(q.property.when, Some(finish()))
    val holds =
      q.property.holds.get.asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(holds(Step(Outcome.accepted, closed, List(Fact.completed))))
    assert(!holds(Step(Outcome.accepted, opened, List(Fact.completed))))
    assert(!holds(Step(Outcome.accepted, closed)))
    val bounds = QueryBounds("custom", steps = 3, actions = 3, search = 99)
    assertEquals(q.withBounds(bounds).limits, bounds)
    assertEquals(q.name, "")
    assertEquals(q.property.name, "")
    assertEquals(q.scenario.name, "")
    assert(q.property.machine eq Subject)
    assert(q.scenario.machine eq Subject)
    assertEquals(q.scenario.start, None)
    assertEquals(q.property.holds2, None)
    assertEquals(q.expectedRun, None)
    assertEquals(q.exploration, None)
    assertEquals(q.reach.steps, 256)
    assertEquals(q.reach.actions, 0)
    assertEquals(q.reach.search, 65536)
    assertEquals(q.checker, Checker.umpire)
    assertEquals(q.confirmations, Vector.empty[Checker])
    assertEquals(q.openReason, None)
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

  test("a composition's recorded fact requires its member case") {
    val path = Together.left(finish())
    val q = Together
      .witness("joined")(path)
      .records(JoinedFact.left(Fact.completed))
    assertEquals(q.scenario.actions, Vector(path))
    assertEquals(q.property.when, Some(path))
    val holds =
      q.property.holds.get.asInstanceOf[Step[framework.dsl.State, String, JoinedFact] => Boolean]
    val state = WitnessFixture.joined(Phase.closed, Phase.open)
    assert(holds(Step("accepted", state, List(JoinedFact.left(Fact.completed)))))
    assert(!holds(Step("accepted", state, List(JoinedFact.right(Foreign.completed)))))
    assert(
      compileErrors(
        "Subject.witness(start()).records(Foreign.completed)"
      ).nonEmpty
    )
    assert(
      compileErrors(
        "Together.witness(Together.left(finish())).records(Fact.completed)"
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
    import framework.realize.*
    val satisfied = RunExpectation(
      Conformance.conformant,
      PropertyOutcome.satisfied,
      PropertyOutcome.satisfied,
      Disposition.completed,
      Cleanup.succeeded
    )
    val bounds = QueryBounds(steps = 2, actions = 2, search = 32)
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
      .ends(_.equals(closed))
      .withBounds(bounds)
    assertEquals(q.expectedRun, Some(satisfied))
    assertEquals(q.exploration, Some(space))
    assertEquals(q.limits, bounds)
    assertEquals(q.expectedRun.get.because, "")
  }

  test(
    "conditional facts inspect the after state and survive every immutable modifier"
  ) {
    import framework.realize.*
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
      .recordsWhen(_.equals(closed), Fact.secondary)
      .live(expected)
      .explore(space)
      .recordsWhen(_.equals(closed), Fact.tertiary)
      .ends(_ => true)
      .withBounds(QueryBounds(steps = 1, actions = 1, search = 17))
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
          closed,
          List(Fact.completed, Fact.secondary, Fact.tertiary)
        )
      )
    )
    assert(holds(Step(Outcome.accepted, opened, List(Fact.completed))))
    assert(
      !holds(
        Step(
          Outcome.accepted,
          closed,
          List(Fact.completed, Fact.secondary)
        )
      )
    )
    assert(
      !holds(
        Step(
          Outcome.accepted,
          closed,
          List(Fact.completed, Fact.tertiary)
        )
      )
    )
    assert(!holds(Step(Outcome.accepted, closed)))
    val joined = Together
      .witness(Together.left(finish()))
      .records(JoinedFact.left(Fact.completed))
      .recordsWhen(
        Together.right.phase.equals(Phase.closed),
        JoinedFact.right(Foreign.completed)
      )
      .ends(Together.left.phase.equals(Phase.closed))
    val joinedHolds = joined.property.holds.get
      .asInstanceOf[Step[framework.dsl.State, String, JoinedFact] => Boolean]
    assert(
      joinedHolds(
        Step(
          "accepted",
          WitnessFixture.joined(Phase.closed, Phase.open),
          List(JoinedFact.left(Fact.completed))
        )
      )
    )
    assert(
      !joinedHolds(
        Step(
          "accepted",
          WitnessFixture.joined(Phase.closed, Phase.closed),
          List(JoinedFact.left(Fact.completed))
        )
      )
    )
    assert(
      joinedHolds(
        Step(
          "accepted",
          WitnessFixture.joined(Phase.closed, Phase.closed),
          List(JoinedFact.left(Fact.completed), JoinedFact.right(Foreign.completed))
        )
      )
    )
    assert(
      !joinedHolds(
        Step(
          "accepted",
          WitnessFixture.joined(Phase.closed, Phase.closed),
          List(JoinedFact.left(Fact.completed), JoinedFact.left(Fact.completed))
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
          s.equals(closed)
        ,
        Fact.secondary
      )
      .recordsWhen(
        s =>
          seen += 2;
          s.equals(closed)
        ,
        Fact.tertiary
      )
    val orderedHolds = ordered.property.holds.get
      .asInstanceOf[Step[State, Outcome, Fact] => Boolean]
    assert(
      !orderedHolds(Step(Outcome.accepted, closed, List(Fact.completed)))
    )
    assertEquals(seen.toVector, Vector(1))
    seen.clear()
    assert(!orderedHolds(Step(Outcome.accepted, closed)))
    assertEquals(seen.toVector, Vector.empty)
    assert(
      orderedHolds(Step(Outcome.accepted, opened, List(Fact.completed)))
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
        "Together.witness(Together.left(finish())).records(JoinedFact.left(Fact.completed)).recordsWhen((s: Joined) => true, Fact.completed)"
      ).nonEmpty
    )
  }
