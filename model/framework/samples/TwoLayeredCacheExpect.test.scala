package framework.dsl

import TwoLayeredCacheControls.WriteThroughL1
import samples.TwoLayeredCache

class TwoLayeredCacheExpectTest extends munit.FunSuite:
  test("TwoLayeredCache's three native predicates retain state and fact requirements"):
    import TwoLayeredCache.*
    import SampleClaims.judged
    val initial = Model.init
    val mismatched = initial.copy(l1 = Some(0))
    // each case judged by the claim and by the predicate it was before, on a step of the action
    // it is about, and the claim not about the steps of another action
    def cases(
        claim: Query,
        taken: Class,
        other: Class,
        steps: List[(Step[State, Outcome, Event], Boolean)]
    )(
        old: Step[State, Outcome, Event] => Boolean
    ): Unit =
      steps.foreach: (step, expected) =>
        assertEquals(old(step), expected, step)
        assertEquals(judged(claim, initial, taken, step), Some(expected), step)
        assertEquals(judged(claim, initial, other, step), None, step)
    cases(
      verify(Model.postconditions.readFillsL1),
      client.read(),
      client.write(),
      List(
        (Step(Outcome.accepted, initial, List.empty[Event]), true),
        (Step(Outcome.accepted, mismatched, List.empty[Event]), false)
      )
    )(after => agreed(Model.at(after.state)(Model.current)))
    val factCases = List(
      (Step(Outcome.accepted, initial, List.empty[Event]), false),
      (Step(Outcome.accepted, initial, List(Event.wrote)), true),
      (Step(Outcome.accepted, initial, List(Event.readL2)), false)
    )
    cases(
      verify(WriteThroughL1.postconditions.buggyWritesRecorded),
      client.write(),
      client.read(),
      factCases
    )(after => after.records(Event.wrote))
    cases(
      verify(Model.queries.servedFromL2),
      client.read(),
      client.write(),
      List(
        (Step(Outcome.accepted, initial, List.empty[Event]), false),
        (Step(Outcome.accepted, initial, List(Event.wrote)), false),
        (Step(Outcome.accepted, initial, List(Event.readL2)), true)
      )
    )(after => after.records(Event.readL2))
