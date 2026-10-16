package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Modelled after https://github.com/quint-co/quint/blob/main/examples/tutorials/repl/kettle.qnt
//
// A kettle: a heating element the user switches on and off, which warms the water by one degree a
// step and fails over, beeping, once the water boils.
object Kettle:
  // the boiling and freezing points of water, in Celsius
  val boilingTemperature = 100
  val freezingTemperature = 0

  // conversion from Celsius to Fahrenheit
  def fahrenheit(celsius: Int): Int = celsius * 9 / 5 + 32

  // the low and high range of a thermometer (fixed)
  val veryCold = fahrenheit(-40)
  val veryHot = fahrenheit(40)

  // The coldest water the kettle starts with, in Celsius.
  val floor = -40

  // the reading of the thermometer in the kettle, whether it is heating and whether it is beeping
  final case class Snapshot(reading: Int, heatingOn: Boolean, beeping: Boolean)

  // the current temperature in the kettle, in Celsius
  def temperature(state: Snapshot): Int = state.reading

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object user extends Actor:
    val pressButton = action(this)
    val depressButton = action(this)

  object heater:
    val heat = internal
    val failover = internal

  object Model extends Machine[Snapshot, Outcome, Nothing]:
    // room temperature (20C), the element off and quiet
    val init = Snapshot(reading = 20, heatingOn = false, beeping = false)
    // a run may stop in any state
    def end(state: Snapshot) = true

    object effects:
      // turn on the heating element
      def pressButton(state: Snapshot) =
        enter(state.copy(heatingOn = true, beeping = false))
      // turn off the kettle when the temperature is too high
      def failover(state: Snapshot) =
        enter(state.copy(heatingOn = false, beeping = true))
      // heat up the water by 1C
      def heat(state: Snapshot) =
        enter(
          state.copy(reading = state.reading + 1, heatingOn = true, beeping = false)
        )
      // turn off the heating element
      def depressButton(state: Snapshot) =
        enter(state.copy(heatingOn = false, beeping = false))

    // one step of the state machine: any enabled action
    object rules extends Rules:
      on(user.pressButton):
        where(!_.heatingOn) ~> effects.pressButton
      on(heater.failover):
        where(s => s.heatingOn && temperature(s) >= boilingTemperature) ~> effects.failover
      on(heater.heat):
        where(s => s.heatingOn && temperature(s) < boilingTemperature) ~> effects.heat
      on(user.depressButton):
        where(_.heatingOn) ~> effects.depressButton

  // A nondeterministic start would be anywhere in -40..40C. The coldest start reaches every one of the
  // others (press, heat, depress), so a claim verified from it holds from all of them.
  val coldest = Snapshot(reading = -40, heatingOn = false, beeping = false)

  val beepsOnlyWhenOff: Property[Snapshot] = Model.property.expect:
    (after: Expectation[Snapshot, Outcome, Nothing]) ?=>
      if after.step.state.beeping then check(!after.step.state.heatingOn)
  val warmsOnlyWhenOn: Property[Snapshot] = Model.property.across.expect:
    (before: Before[Snapshot], after: Expectation[Snapshot, Outcome, Nothing]) ?=>
      if !before.state.heatingOn then
        check(temperature(after.step.state) <= temperature(before.state))
  val failsOverAtBoiling: Property[Snapshot] = Model.property
    .after(heater.failover)
    .expect: (after: Expectation[Snapshot, Outcome, Nothing]) ?=>
      check(temperature(after.step.state) == boilingTemperature)
      check(after.step.state.beeping)
  val neverBeeps: Property[Snapshot] = Model.property.expect:
    (after: Expectation[Snapshot, Outcome, Nothing]) ?=> check(!after.step.state.beeping)

  val fromRoomTemperature: Scenario[Snapshot] = Model.scenario.starts(Model.init).free
  val fromAnyTemperature: Scenario[Snapshot] = Model.scenario.starts(coldest).free
  // the coldest start boils after a press and 140 heats
  val toBoiling = Limits(steps = 144, actions = 144, search = 65536)
  val queries = Vector(
    query("beepsOnlyWhenOffHolds") verify beepsOnlyWhenOff in fromAnyTemperature limits toBoiling,
    query("warmsOnlyWhenOnHolds") verify warmsOnlyWhenOn in fromAnyTemperature limits toBoiling,
    query("failsOverAtBoilingHolds") verify failsOverAtBoiling in fromAnyTemperature limits
      toBoiling,
    query("boilsAndBeeps") find failsOverAtBoiling in fromRoomTemperature limits toBoiling,
    query("refutesNeverBeeps") verify neverBeeps in fromRoomTemperature limits toBoiling
  )
