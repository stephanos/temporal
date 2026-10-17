package samples

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

// A kettle: a heating element the user switches on and off, which warms the water by one degree a
// step and fails over, beeping, once the water boils.
object Kettle:
  // the boiling point of water, in Celsius
  inline val boilingTemperature = 100
  // the coldest water the kettle starts with, in Celsius
  inline val coldestTemperature = -40

  object user extends Actor:
    val pressButton = action
    val depressButton = action

  object heater extends Internal:
    val heat = step
    val failover = step

  object Model extends Machine[Outcome, Nothing]:
    var temperature = oneOf(coldestTemperature to 40)
    var heatingOn = false
    var beeping = false

    // one step of the state machine: any enabled action
    object transitions extends Transitions:
      on(user.pressButton):
        if !heatingOn then effects.pressButton()
      on(heater.failover):
        if heatingOn && temperature >= boilingTemperature then effects.failover()
      on(heater.heat):
        if heatingOn && temperature < boilingTemperature then effects.heat()
      on(user.depressButton):
        if heatingOn then effects.depressButton()

    object effects:
      // turn on the heating element
      def pressButton() =
        heatingOn = true
        beeping = false
      // turn off the heating element and beep
      def failover() =
        heatingOn = false
        beeping = true
      // heat up the water by 1C; `heatingOn` holds and `beeping` doesn't, since heat only fires
      // while heating
      def heat() =
        temperature += 1
      // turn off the heating element
      def depressButton() =
        heatingOn = false
        beeping = false

    object invariants:
      val beepsOnlyWhenOff =
        if beeping then
          check(!heatingOn)

    object postconditions:
      val warmsOnlyWhenOn =
        if !before.heatingOn then
          check(temperature <= before.temperature)
      val failsOverAtBoiling =
        after(heater.failover):
          check(temperature == boilingTemperature)
          check(beeping)

    // The user switches the kettle on but never off: one who switches it off keeps it from boiling,
    // whatever the heater does.
    val neverSwitchedOff =
      without(user.depressButton)

    object progress:
      // Every claim above also holds for a kettle that never heats; this one says it boils.
      // Switched on and left on, the water boils and the kettle beeps.
      val boils =
        (whenever(heatingOn):
          eventually(beeping)
        ) in neverSwitchedOff
