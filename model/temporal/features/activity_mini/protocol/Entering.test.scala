package framework

import framework.outcomes.Outcome
import framework.Finite.given
import temporal.features.activity.mini.protocol.*
import temporal.features.activity.timers
import AnswerDispatch.given

object NestedAnswerFixture:
  final case class Envelope(answer: AnswerState, sentinel: UpTo[1])

  object Nested
      extends Machine[Envelope, Outcome, AnswerFact],
        Phased[Envelope, AnswerPhase](
          _.answer.phase,
          (s, p) => s.copy(answer = s.answer.copy(phase = p))
        ):
    val init = Envelope(AnswerDispatch.init, UpTo(1))
    override def end(s: Envelope) = true

    object rules extends Rules:
      on(fault.delayAnswer) {
        always ~> enters(AnswerPhase.backingOff)
      }

  object Inherited extends Derived(Nested.restrict(fault.delayAnswer))

class AnswerEnteringTest extends munit.FunSuite:
  test("plain Answering phase entries retain attempts and inbox, and add no facts"):
    given Owner[AnswerState, Outcome, AnswerFact] = Owner(AnswerDispatch)
    for
      state <- Finite[AnswerState].values;
      phase <- List(AnswerPhase.backingOff, AnswerPhase.waiting)
    do
      assertEquals(
        enters(phase)(state),
        List(Step(Outcome.accepted, state.copy(phase = phase), List.empty[AnswerFact]))
      )
      assertEquals(
        AnswerDispatch.end(state),
        state.phase.in(AnswerPhase.completed, AnswerPhase.failed)
      )

  test("the actual rules keep disabled guards and preserve every ordinary result"):
    type Effect = AnswerState => List[Step[AnswerState, Outcome, AnswerFact]]
    def effect(action: Action[EmptyTuple]) = AnswerDispatch.bindings
      .find(_.decl == action.decl)
      .get
      .function
      .asInstanceOf[Effect]
    val delay = effect(fault.delayAnswer)
    val resume = effect(timers.backoff)
    for state <- Finite[AnswerState].values do
      val delayed = if state.phase == AnswerPhase.held && state.attempts < 2 then
        List(
          Step(Outcome.accepted, state.copy(phase = AnswerPhase.backingOff), List.empty[AnswerFact])
        )
      else Nil
      val resumed = if state.phase == AnswerPhase.backingOff then
        List(
          Step(Outcome.accepted, state.copy(phase = AnswerPhase.waiting), List.empty[AnswerFact])
        )
      else Nil
      assertEquals(delay(state), delayed)
      assertEquals(resume(state), resumed)

  test("a derived subject inherits its nested projection and writer without losing either record"):
    import NestedAnswerFixture.*
    type Effect = Envelope => List[Step[Envelope, Outcome, AnswerFact]]
    val apply = Inherited.bindings.head.function.asInstanceOf[Effect]
    for state <- Finite[Envelope].values do
      assertEquals(
        apply(state),
        List(
          Step(
            Outcome.accepted,
            state.copy(answer = state.answer.copy(phase = AnswerPhase.backingOff)),
            List.empty[AnswerFact]
          )
        )
      )
