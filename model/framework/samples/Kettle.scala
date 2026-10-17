package samples

import framework.dsl.*
import framework.dsl.Finite.given
import framework.dsl.outcomes.Outcome

// A kettle: a heating element the user switches on and off, which warms the water by one degree a
// step and fails over, beeping, once the water boils.
object Kettle:
  // the boiling point of water, in Celsius
  val boilingTemperature = 100

  object user extends Actor:
    val pressButton = action
    val depressButton = action

  object heater extends Internal:
    val heat = step
    val failover = step

  object Model extends Machine[Outcome, Nothing]:
    var temperature = oneOf(-40 to 40)
    var heatingOn = false
    var beeping = false

    // The kettle left to itself once the user switched it on. The user's own actions are restricted
    // rather than assumed away: a user who switches the kettle off before it boils keeps it from
    // boiling, which no fairness of the heater excludes. Pressing the button stays, since from room
    // temperature nothing heats until the user switches the element on.
    val heatingAlone = only(user.pressButton, heater.heat, heater.failover)

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
      // turn off the kettle when the temperature is too high
      def failover() =
        heatingOn = false
        beeping = true
      // heat up the water by 1C; on and quiet already hold, since heat only fires while heating
      def heat() =
        temperature += 1
      // turn off the heating element
      def depressButton() =
        heatingOn = false
        beeping = false

    object invariants:
      val beepsOnlyWhenOff = if beeping then check(!heatingOn)

    object postconditions:
      val warmsOnlyWhenOn = if !before.heatingOn then check(temperature <= before.temperature)
      val failsOverAtBoiling = after(heater.failover):
        check(temperature == boilingTemperature)
        check(beeping)

    object progress:
      // switched on, the water boils and the kettle beeps from every start it admits: 140 heats
      // from the coldest start and the failover (80 heats from room temperature)
      val boils = leadsTo(from = heatingOn, to = beeping, within = 141) in heatingAlone
