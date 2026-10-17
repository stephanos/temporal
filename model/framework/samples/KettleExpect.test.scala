package framework.dsl

import KettleControls.{coldest, room}
import samples.Kettle

class KettleExpectTest extends munit.FunSuite:
  test("Kettle's four native predicates retain implication and conjunction truth tables"):
    import Kettle.*
    import Kettle.Model.{beeping, heatingOn, temperature}
    import framework.dsl.outcomes.Outcome
    import NativeControls.invariant
    import SampleClaims.{holds, judged}
    val quiet = room
    val hot = quiet.copy(temperature = 100, heatingOn = false, beeping = true)
    val heating = quiet.copy(heatingOn = true)
    // each case judged by the claim and by the predicate it was before, on the same state or step
    def invariantCases(claim: State => Boolean, cases: List[(State, Boolean)])(
        old: State => Boolean
    ): Unit =
      cases.foreach: (state, expected) =>
        assertEquals(old(state), expected, state)
        assertEquals(claim(state), expected, state)
    def stepCases(claim: Query, taken: Class, cases: List[(State, State, Boolean)])(
        old: (State, State) => Boolean
    ): Unit =
      cases.foreach: (before, after, expected) =>
        assertEquals(old(before, after), expected, (before, after))
        val step = Step[State, Outcome, Nothing](Outcome.accepted, after)
        assertEquals(judged(claim, before, taken, step), Some(expected), (before, after))
    invariantCases(
      holds(verify(Model.invariants.beepsOnlyWhenOff), _),
      List(
        (quiet, true),
        (heating, true),
        (hot, true),
        (hot.copy(heatingOn = true), false)
      )
    )(state => Model.at(state)(!beeping || !heatingOn))
    stepCases(
      verify(Model.postconditions.warmsOnlyWhenOn),
      heater.heat(),
      List(
        (heating, hot, true),
        (quiet, quiet, true),
        (quiet, coldest, true),
        (quiet, hot, false)
      )
    )((before, after) =>
      Model.at(before)(heatingOn) || Model.at(after)(temperature) <= Model.at(before)(temperature)
    )
    // failsOverAtBoiling is about failover's steps alone
    val failsOver = verify(Model.postconditions.failsOverAtBoiling)
    stepCases(
      failsOver,
      heater.failover(),
      List(
        (heating, hot, true),
        (heating, hot.copy(beeping = false), false),
        (heating, quiet.copy(beeping = true), false)
      )
    )((_, after) => Model.at(after)(temperature == boilingTemperature && beeping))
    assertEquals(
      judged(
        failsOver,
        heating,
        heater.heat(),
        Step[State, Outcome, Nothing](Outcome.accepted, quiet)
      ),
      None
    )
    // neverBeeps, the negative control, written in its test
    invariantCases(
      holds(Model, invariant(check(!Model.beeping)), _),
      List(
        (quiet, true),
        (hot, false)
      )
    )(state => Model.at(state)(!beeping))
