package framework.dsl

import framework.checker.Effects.*
import framework.dsl.Finite.given

object NativeRetainedContractsFixture:
  given Ok[Boolean] = Ok(true)

  object leftActor extends Actor:
    val go = action

  object rightActor extends Actor:
    val go = action
  abstract class HeadingModel extends Machine[Boolean, Nothing]:
    var ready = false

    object effects:
      def noop(): Unit =
        throw AssertionError("heading admission invoked an effect")

class NativeRetainedContractsSuite extends munit.FunSuite:
  import WitnessFixture.*

  private def opened: State = Subject.init.copy(phase = Phase.open)
  private def closed: State = Subject.init.copy(phase = Phase.closed)

  test("ordinary sticky monitors preserve independent core behavior without expectation adapters"):
    def promise(step: Step[State, Outcome, Fact]): Boolean =
      step.records(Fact.completed) && step.state.equals(closed)
    val watched = sticky[State, Outcome, Fact](promise).readAfter(_.outcome == Outcome.accepted)
    val core = monitor[State, Outcome, Fact, Boolean](false)((broken, _, step) =>
      broken || !promise(step)
    )(identity).readAfter(_.outcome == Outcome.accepted)
    def transition(before: State, after: Step[State, Outcome, Fact]): Boolean =
      !before.equals(opened) || promise(after)
    val across = stickyAcross[State, Outcome, Fact](transition)
    for
      before <- List(opened, closed); state <- List(opened, closed);
      facts <- List(Nil, List(Fact.completed)); broken <- List(false, true)
    do
      val step = Step(Outcome.accepted, state, facts)
      assertEquals(watched.next(broken, before, step), core.next(broken, before, step))
      assertEquals(watched.readAt.get(step), core.readAt.get(step))
      assertEquals(across.next(broken, before, step), broken || !transition(before, step))
    val missing = hole
    def withHole(step: Step[State, Outcome, Fact]): Boolean =
      step.state.equals(closed) && missing.reached
    val unknown = sticky[State, Outcome, Fact](withHole)
    assert(unknown.next(false, opened, Step(Outcome.accepted, opened)))
    assert(unknown.next(true, opened, Step(Outcome.accepted, closed)))
    assert(
      intercept[HoleReached](
        unknown.next(false, opened, Step(Outcome.accepted, closed))
      ).hole eq missing
    )

  test("tagged witness facts retain their member case across composition order"):
    object Reordered extends Composition[JoinedFact]:
      val right = NativeOther
      val left = NativeSubject
    val state = Reordered.init.copy(
      left = NativeSubject.init.copy(phase = Phase.open),
      right = NativeOther.init.copy(phase = Phase.closed)
    )
    val step = Step[State, Any, JoinedFact](
      "accepted",
      state,
      List(JoinedFact.left(Fact.completed))
    )
    def recorded(q: Query): Boolean =
      q.property.holds.get
        .asInstanceOf[Step[State, Any, JoinedFact] => Boolean](
          step
        ) // scalafix:ok DisableSyntax.asInstanceOf
    val target = Reordered.left(WitnessFixture.start())
    assert(recorded(Reordered.witness(target).records(JoinedFact.left(Fact.completed))))
    assert(!recorded(Reordered.witness(target).records(JoinedFact.right(Foreign.completed))))

  test("composition members refuse unrelated objects and repeated machines"):
    val computed = compileErrors("""
      import framework.dsl.*
      object NotAModel
      object Computed extends Composition[Nothing]:
        val clock = NotAModel
    """)
    assert(computed.contains("clock"), computed)
    assert(computed.contains("neither a Machine nor a Composition"), computed)
    val repeated = compileErrors("""
      import framework.dsl.*
      import framework.dsl.Finite.given
      given Ok[Boolean] = Ok(true)
      object Member extends Machine[Boolean, Nothing]:
        var ready = false
        object transitions extends Transitions
      object Repeated extends Composition[Nothing]:
        val left = Member
        val right = Member
    """)
    assert(repeated.contains("names already"), repeated)
    assert(repeated.contains("a Machine is a member once"), repeated)

  test("native headings refuse duplicate disabled foreign and nested declarations without effects"):
    import NativeRetainedContractsFixture.{*, given}
    object Repeated extends HeadingModel:
      object transitions extends Transitions:
        on(leftActor.go, leftActor.go):
          effects.noop()
    assert(
      intercept[IllegalArgumentException](Repeated.bindings).getMessage.contains("named twice")
    )
    object Disabled extends HeadingModel:
      object transitions extends Transitions:
        disabled(leftActor.go)
        on(leftActor.go):
          effects.noop()
    assert(
      intercept[IllegalArgumentException](Disabled.bindings).getMessage
        .contains("disabled and fired")
    )
    object ForeignGroup extends HeadingModel:
      object transitions extends Transitions:
        from(leftActor):
          on(rightActor.go):
            effects.noop()
    assert(
      intercept[IllegalArgumentException](ForeignGroup.bindings).getMessage.contains("declares no")
    )
    object NestedGroup extends HeadingModel:
      object transitions extends Transitions:
        from(leftActor):
          from(rightActor):
            on(rightActor.go):
              effects.noop()
    assert(
      intercept[IllegalArgumentException](NestedGroup.bindings).getMessage.contains("sits in from")
    )
    val nestedOn = compileErrors("""
      import framework.dsl.*
      import framework.dsl.NativeRetainedContractsFixture.*
      object Nested extends HeadingModel:
        object transitions extends Transitions:
          on(leftActor.go):
            on(rightActor.go):
              effects.noop()
    """)
    assert(nestedOn.contains("transition") || nestedOn.contains("leaf"), nestedOn)
