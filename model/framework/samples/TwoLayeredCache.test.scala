package framework.dsl

import TwoLayeredCacheControls.WriteThroughL1
import samples.TwoLayeredCache

class TwoLayeredCacheTest extends munit.FunSuite:
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

object TwoLayeredCacheControls:
  import samples.TwoLayeredCache.*

  object effects:
    // A buggy write that stores its value in layer 1 only.
    def writeL1Only() =
      Model.l1 = Some(Model.num)
      Model.num = Model.num + 1
      Model.stateVars.get.add(Seq(Event.wrote))

  // A negative control: writes that store their value in layer 1 only. The readAfterWrite monitor
  // it keeps from Model is refuted, in the tests.
  object WriteThroughL1
      extends Derived(
        Model.rebind(
          on(client.write):
            if Model.num < 3 then effects.writeL1Only()
        )
      ):

    object postconditions:
      // Every write records itself, in the buggy cache too: what refutes it is the readAfterWrite
      // monitor, which a read of a value older than the last write breaks.
      val buggyWritesRecorded = after(client.write):
        records(Event.wrote)
