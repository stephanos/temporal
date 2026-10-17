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

  // A nondeterministic start would be anywhere in -40..40C. The coldest start reaches every one of the
  // others (press, heat, depress), so a claim verified from it holds from all of them.
  val coldest = Model.init.copy(temperature = -40)

  object user extends Actor:
    val pressButton = action
    val depressButton = action

  object heater extends Internal:
    val heat = step
    val failover = step

  object Model extends Machine[Outcome, Nothing]:
    // the temperature in the kettle in Celsius, whether it is heating and whether it is beeping: room
    // temperature (20C), the element off and quiet. The temperature is a host Int, as the Quint's is
    // an unbounded int; a signed bounded reading would need a domain the DSL does not have yet.
    var temperature = 20
    var heatingOn = false
    var beeping = false

    // the kettle from room temperature, and from the coldest start
    val fromRoomTemperature = starts()
    val fromAnyTemperature = starts(temperature = -40)
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
      def pressButton(): Unit =
        heatingOn = true
        beeping = false
      // turn off the kettle when the temperature is too high
      def failover(): Unit =
        heatingOn = false
        beeping = true
      // heat up the water by 1C; on and quiet already hold, since heat only fires while heating
      def heat(): Unit = temperature += 1
      // turn off the heating element
      def depressButton(): Unit =
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
      // switched on, the water boils and the kettle beeps: 80 heats from room temperature and the
      // failover
      val boils = leadsTo(from = heatingOn, to = beeping, within = 81) in heatingAlone

    object queries:
      // the claims from the coldest start, which reaches every warmer one
      val beepsOnlyWhenOff = verify(invariants.beepsOnlyWhenOff) in fromAnyTemperature
      val warmsOnlyWhenOn = verify(postconditions.warmsOnlyWhenOn) in fromAnyTemperature
      val failsOverAtBoiling = verify(postconditions.failsOverAtBoiling) in fromAnyTemperature
      // the room-temperature kettle boils after a press and 80 heats
      val boilsAndBeeps = find(postconditions.failsOverAtBoiling) in fromRoomTemperature
