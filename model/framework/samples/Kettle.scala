package samples

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

// Modelled after https://github.com/quint-co/quint/blob/main/examples/tutorials/repl/kettle.qnt
//
// A kettle: a heating element the user switches on and off, which warms the water by one degree a
// step and fails over, beeping, once the water boils.
object Kettle:
  // the boiling point of water, in Celsius
  val boilingTemperature = 100

  // the temperature in the kettle in Celsius, whether it is heating and whether it is beeping. The
  // temperature is a host Int, as the Quint's is an unbounded int; a signed bounded reading would
  // need a domain the DSL does not have yet.
  final case class Snapshot(temperature: Int, heatingOn: Boolean, beeping: Boolean)

  object user extends Actor:
    val pressButton = action(this)
    val depressButton = action(this)

  object heater:
    val heat = internal
    val failover = internal

  object Model extends Machine[Snapshot, Outcome, Nothing]:
    // room temperature (20C), the element off and quiet
    val init = Snapshot(temperature = 20, heatingOn = false, beeping = false)
    // a run may stop in any state
    def end(state: Snapshot) = true

    object effects:
      // turn on the heating element
      def pressButton(s: Snapshot) = enter(s.copy(heatingOn = true, beeping = false))
      // turn off the kettle when the temperature is too high
      def failover(s: Snapshot) = enter(s.copy(heatingOn = false, beeping = true))
      // heat up the water by 1C; on and quiet already hold, since heat only fires while heating
      def heat(s: Snapshot) = enter(s.copy(temperature = s.temperature + 1))
      // turn off the heating element
      def depressButton(s: Snapshot) = enter(s.copy(heatingOn = false, beeping = false))

    // one step of the state machine: any enabled action
    object rules extends Rules:
      on(user.pressButton):
        where(!_.heatingOn) ~> effects.pressButton
      on(heater.failover):
        where(s => s.heatingOn && s.temperature >= boilingTemperature) ~> effects.failover
      on(heater.heat):
        where(s => s.heatingOn && s.temperature < boilingTemperature) ~> effects.heat
      on(user.depressButton):
        where(_.heatingOn) ~> effects.depressButton

  // A nondeterministic start would be anywhere in -40..40C. The coldest start reaches every one of the
  // others (press, heat, depress), so a claim verified from it holds from all of them.
  val coldest = Snapshot(temperature = -40, heatingOn = false, beeping = false)

  val beepsOnlyWhenOff = Model.property.expect:
    if after.state.beeping then check(!after.state.heatingOn)
  val warmsOnlyWhenOn = Model.property.across.expect:
    if !before.heatingOn then check(after.state.temperature <= before.temperature)
  val failsOverAtBoiling = Model.property
    .after(heater.failover)
    .expect:
      check(after.state.temperature == boilingTemperature)
      check(after.state.beeping)
  val neverBeeps = Model.property.expect:
    check(!after.state.beeping)

  val fromRoomTemperature = Model.scenario.starts(Model.init).free
  val fromAnyTemperature = Model.scenario.starts(coldest).free
  // the coldest start boils after a press and 140 heats
  val toBoiling = Limits(steps = 144, actions = 144, search = 65536)
  val queries = Vector(
    query verify beepsOnlyWhenOff in fromAnyTemperature limits toBoiling,
    query verify warmsOnlyWhenOn in fromAnyTemperature limits toBoiling,
    query verify failsOverAtBoiling in fromAnyTemperature limits toBoiling,
    query find failsOverAtBoiling in fromRoomTemperature limits toBoiling,
    query("refutesNeverBeeps") verify neverBeeps in fromRoomTemperature limits toBoiling
  )
