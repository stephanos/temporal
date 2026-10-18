package framework.dsl

import framework.checker.*

import framework.dsl.Finite.given

object PlainCompositionFixture:
  enum Answer:
    case accepted
  enum Note:
    case completed, reopened
  enum JoinedFact:
    case left(value: Note)
    case right(value: Note)
  enum NestedFact:
    case pair(value: JoinedFact)
  given Ok[Answer] = Ok(Answer.accepted)

  object hand extends Actor:
    val start = action
    val finish = action
    val unused = action
    val set = action.input[Boolean](name = "flag")

  object Left extends Machine[Answer, Note]:
    var active = false

    object transitions extends Transitions:
      on(hand.start):
        if !active then effects.start()
      on(hand.finish):
        if active then effects.finish()

    object effects:
      def start() =
        active = true
        records(Note.completed)
      def finish() =
        active = false
        records(Note.reopened)
      def late() =
        records(Note.completed)

  object Right extends Machine[Answer, Note]:
    var active = false

    object transitions extends Transitions:
      on(hand.start):
        if !active then effects.start()

    object effects:
      def start() =
        active = true
        records(Note.completed)

  object Extra extends Machine[Answer, Nothing]:
    var active = false

    object transitions extends Transitions

  object Pair extends Composition[JoinedFact]:
    val left = Left
    val right = Right

    object syncs:
      val started = sync(left(hand.start), right(hand.start))

    object invariants:
      val ordered =
        check(!left.active || right.active)

    object postconditions:
      val started = after(syncs.started):
        check(!before.left.active && !before.right.active)
        check(left.active && right.active)
        records(JoinedFact.left(Note.completed))
        records(JoinedFact.right(Note.completed))

    object progress:
      val closes = (whenever(!left.active):
        eventually(left.active)
      ) in only(syncs.started).fair(syncs.started)

    object monitors:
      val separated = monitor(false)(seen => seen || (right.active && !left.active))(identity)

    object queries:
      val reopened = after(hand.finish):
        records(JoinedFact.left(Note.reopened))
      val found = find(reopened)
      val path = witness(syncs.started, hand.finish)
        .records(JoinedFact.left(Note.reopened))
        .ends(!left.active && right.active)

  object Nested extends Composition[NestedFact]:
    val pair = Pair
    val extra = Extra

    object postconditions:
      val started = after(Pair.syncs.started):
        check(!before.pair.left.active && !before.pair.right.active)
        check(pair.left.active && pair.right.active)
        records(NestedFact.pair(JoinedFact.left(Note.completed)))

    object queries:
      val path = witness(Pair.syncs.started, Pair.left(hand.finish))
        .records(NestedFact.pair(JoinedFact.left(Note.reopened)))

  object WrappedRight extends Composition[JoinedFact]:
    val right = Right

  object LateLeft
      extends Derived(
        Left.rebind(
          on(hand.start):
            Left.effects.late()
        )
      )

  object Late extends Composition(Pair.withMember(left = LateLeft)):

    object invariants:
      val waits = check(!Pair.left.active)

    object postconditions:
      val started = after(Pair.syncs.started):
        check(!before.left.active && !before.right.active)
        records(JoinedFact.left(Note.completed))

  object Apart extends Composition[JoinedFact]:
    val left = Left
    val right = Right

  object Chained extends Composition(Late.withMember(right = Right)):

    object postconditions:
      val started = after(Pair.syncs.started):
        check(!before.left.active && !before.right.active)
        records(JoinedFact.left(Note.completed))

  object NestedLate extends Composition(Nested.withMember(pair = Late))

  object Ambiguous extends Composition[JoinedFact]:
    val left = Left
    val right = Right

    object syncs:
      val started = sync(hand.start, hand.start)

  object AmbiguousClaim extends Composition[JoinedFact]:
    val left = Left
    val right = Right

    object postconditions:
      val started = after(hand.start):
        check(left.active || right.active)

  object Unbound extends Composition[JoinedFact]:
    val left = Left
    val right = Right

    object syncs:
      val started = sync(left(hand.start), right(hand.finish))
  val first = choice
  val second = choice

  object InputLeft extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(hand.set): flag =>
        effects.set(flag)

    object effects:
      def set(flag: Boolean) =
        choose(
          first -> { value = if flag then 1 else 0 },
          second -> { value = if flag then 2 else 0 }
        )

  object InputRight extends Machine[Answer, Nothing]:
    var value = 0

    object transitions extends Transitions:
      on(hand.set): flag =>
        effects.set(flag)

    object effects:
      def set(flag: Boolean) =
        choose(
          first -> { value = if flag then 3 else 0 },
          second -> { value = if flag then 4 else 0 }
        )

  object Inputs extends Composition[Nothing]:
    val left = InputLeft
    val right = InputRight

    object syncs:
      val set = sync(left(hand.set), right(hand.set))

    object invariants:
      val zero = check(left.value == 0 && right.value == 0)
      val leftZero = check(left.value == 0)

    object queries:
      val set = after(syncs.set):
        check(left.value > 0 && right.value > 0)

    object progress:
      val sets = (whenever(left.value == 0):
        eventually(left.value > 0)
      ) in only(syncs.set).fair(left(hand.set(true)))
      val bounded =
        (leadsTo(left.value == 0, left.value > 0, 1)) in only(syncs.set).fair(left(hand.set(true)))

  object SelectedInputs extends Composition[Nothing]:
    val left = InputLeft
    val right = InputRight

    object syncs:
      val set = sync(left(hand.set(true)), right(hand.set(false)))
  given Ok[Boolean] = Ok(true)

  object RespondLeft extends Machine[Boolean, Nothing]:
    var ready = false

    object transitions extends Transitions:
      on(hand.unused):
        effects.respond()

    object effects:
      def respond() =
        ready = true

  object RespondRight extends Machine[Boolean, Nothing]:
    var ready = false

    object transitions extends Transitions:
      on(hand.unused):
        reject(false)

  object Responses extends Composition[Nothing]:
    val left = RespondLeft
    val right = RespondRight

    object syncs:
      val responded = sync(left(hand.unused), right(hand.unused))

  object ReversedResponses extends Composition[Nothing]:
    val left = RespondLeft
    val right = RespondRight

    object syncs:
      val responded = sync(right(hand.unused), left(hand.unused))

  object StartsMember extends Machine[Boolean, Nothing]:
    var ready = oneOf(List(false, true))

    object transitions extends Transitions

  object Starts extends Composition[Nothing]:
    val member = StartsMember

    object invariants:
      val empty = check(!member.ready)

class PlainCompositionTest extends munit.FunSuite:
  import PlainCompositionFixture.*
  import NativeCheck.Verdict.*

  test("native syncs restore both members, tag facts, and preserve shortest paths"):
    val report = NativeCheck.verifyAll(Pair)
    assertEquals(report.result("ordered").verdict, verified)
    assertEquals(report.result("started").verdict, verified)
    assertEquals(report.result("closes").verdict, verified)
    assertEquals(report.result("separated").verdict, refuted)
    assertEquals(report.result("separated").detail.head, "counterexample: started, left.finish")
    assertEquals(NativeCheck.queries(Pair).map(_.verdict), Vector(found, found))
    assert(!Left.active && !Right.active)

  test("nested composition members step, expose nested before and retain tagged facts"):
    assertEquals(NativeCheck.verifyAll(Nested).result("started").verdict, verified)
    assertEquals(NativeCheck.queries(Nested).map(_.verdict), Vector(found))

  test("derivation-owned before reads source members while source claims are explicitly selected"):
    assertEquals(
      NativeCheck.verifyAll(NestedLate, sections = Some(Nested)).result("started").verdict,
      refuted
    )
    val report = NativeCheck.verifyAll(Late)
    assertEquals(report.verdicts, Map("waits" -> verified, "started" -> verified))
    val _ = summon[Late.Fact =:= JoinedFact]
    val _ = summon[Chained.Fact =:= JoinedFact]
    assertEquals(NativeCheck.verifyAll(Chained).result("started").verdict, verified)
    assertEquals(
      NativeCheck.verifyAll(Late, sections = Some(Pair)).result("started").verdict,
      refuted
    )
    assertEquals(
      NativeCheck.verifyAll(Apart, sections = Some(Pair)).result("started").verdict,
      verified
    )
    assert(
      NativeCheck.verifyAll(Apart, sections = Some(Pair)).warnings.exists(_.contains("vacuous"))
    )

  test("input-valued syncs enumerate every class and nondeterministic result product"):
    val _ = summon[Inputs.Fact =:= Nothing]
    val graph = NativeCheck.explore(Inputs, Inputs.initialStates)
    val transitions = graph.steps(Inputs.init).toVector
    val trueTrue = transitions
      .filter(_._1.values.equals(List(true, true)))
      .map: (_, step) =>
        Inputs.at(step.state)((InputLeft.value, InputRight.value))
    assertEquals(trueTrue, Vector((1, 3), (1, 4), (2, 3), (2, 4)))
    assertEquals(transitions.size, 16)
    assertEquals(
      NativeCheck
        .query(
          verify(Inputs.invariants.leftZero) in only(Inputs.left(hand.set(false))),
          "inputs.zero"
        )
        .verdict,
      verified
    )
    assertEquals(
      NativeCheck
        .query(find(Inputs.queries.set) in actions(Inputs.left(hand.set(true))), "inputs.set")
        .verdict,
      found
    )
    assertEquals(NativeCheck.verifyAll(Inputs).result("sets").verdict, verified)
    assertEquals(NativeCheck.verifyAll(Inputs).result("bounded").verdict, missedDeadline)

  test("selected sync classes keep whole-action pairing and first-member outcome order"):
    val selected = NativeCheck
      .explore(SelectedInputs, SelectedInputs.initialStates)
      .steps(SelectedInputs.init)
      .toVector
    assertEquals(selected.size, 4)
    assert(selected.forall(_._1.values.equals(List(true, false))))
    val forward =
      NativeCheck.explore(Responses, Responses.initialStates).steps(Responses.init).next()._2
    val reverse = NativeCheck
      .explore(ReversedResponses, ReversedResponses.initialStates)
      .steps(ReversedResponses.init)
      .next()
      ._2
    assertEquals(forward.outcome, true: Any)
    assertEquals(reverse.outcome, false: Any)
    assertEquals(Starts.initialStates.size, 2)
    assertEquals(NativeCheck.verifyAll(Starts).result("empty").verdict, refuted)

  test("composition cuts remain unknown and explicit starts retain member snapshot ownership"):
    assertEquals(NativeCheck.verifyAll(Pair, budget = 1).result("ordered").verdict, unknown)
    val scenario =
      starts(left = Left.init.copy(active = true), right = Right.init.copy(active = true))
        .only(hand.finish)
    assertEquals(
      NativeCheck.query(find(Pair.queries.reopened) in scenario, "pair.reopened").verdict,
      found
    )

  test("sync refusals retain their declaration location and member disambiguation"):
    for (model, message) <- Vector(Ambiguous -> "name the member", Unbound -> "binds no") do
      val error =
        intercept[IllegalArgumentException](NativeCheck.explore(model, model.initialStates))
      assert(error.getMessage.contains("PlainComposition.test.scala:"), error.getMessage)
      assert(error.getMessage.contains(message), error.getMessage)
    assert(
      intercept[IllegalArgumentException](Pair.withMember(rigth = LateLeft)).getMessage
        .contains("rigth")
    )
    assert(
      intercept[IllegalArgumentException](Pair.withMember(right = LateLeft)).getMessage
        .contains("does not hold")
    )
    assert(
      intercept[IllegalArgumentException](Pair.withMember(right = WrappedRight)).getMessage
        .contains("does not hold")
    )

  test("composition environments and claims refuse ambiguous and foreign targets"):
    assert(
      intercept[IllegalArgumentException](
        Apart.scenario.free.only(hand.start).bound(Apart, "ambiguous")
      ).getMessage.contains("name the member")
    )
    assert(
      intercept[IllegalArgumentException](
        Pair.scenario.free.fair(hand.unused).bound(Pair, "foreign")
      ).getMessage.contains("binds no")
    )
    val error = intercept[IllegalArgumentException](NativeCheck.verifyAll(AmbiguousClaim))
    assert(error.getMessage.contains("name the member"), error.getMessage)
    assert(error.getMessage.contains("PlainComposition.test.scala:"), error.getMessage)

  test("plain compositions declare members and claim their synchronized step"):
    assertEquals(
      compileErrors("""
      import framework.dsl.*
      given Ok[Boolean] = Ok(true)
      object hand extends Actor:
        val turn = action
      object Left extends Machine[Boolean, Nothing]:
        var active = false
        object transitions extends Transitions:
          on(hand.turn):
            effects.turn()
        object effects:
          def turn() =
            active = true
      object Right extends Machine[Boolean, Nothing]:
        var active = false
        object transitions extends Transitions:
          on(hand.turn):
            effects.turn()
        object effects:
          def turn() =
            active = true
      object Pair extends Composition[Nothing]:
        val left = Left
        val right = Right
        object syncs:
          val turn = sync(left(hand.turn), right(hand.turn))
        object postconditions:
          val together = after(syncs.turn):
            check(left.active && right.active && !before.left.active)
    """),
      ""
    )
