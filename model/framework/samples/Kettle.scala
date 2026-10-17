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

    // one step of the state machine: any enabled action
    object transitions extends Rules:
      on(user.pressButton):
        where(!heatingOn) ~> effects.pressButton()
      on(heater.failover):
        where(heatingOn && temperature >= boilingTemperature) ~> effects.failover()
      on(heater.heat):
        where(heatingOn && temperature < boilingTemperature) ~> effects.heat()
      on(user.depressButton):
        where(heatingOn) ~> effects.depressButton()

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
      val beepsOnlyWhenOff = property.expect:
        if beeping then check(!heatingOn)
      val neverBeeps = property.expect:
        check(!beeping)

    object postconditions:
      val warmsOnlyWhenOn = property.across.expect:
        if !before.heatingOn then check(temperature <= before.temperature)
      val failsOverAtBoiling = property
        .after(heater.failover)
        .expect:
          check(temperature == boilingTemperature)
          check(beeping)

    object queries:
      val fromRoomTemperature = scenario.starts(init).free
      val fromAnyTemperature = scenario.starts(coldest).free
      // the coldest start boils after a press and 140 heats
      val toBoiling = Limits(steps = 144, actions = 144, search = 65536)
      val beepsOnlyWhenOff =
        query verify invariants.beepsOnlyWhenOff in fromAnyTemperature limits toBoiling
      val warmsOnlyWhenOn =
        query verify postconditions.warmsOnlyWhenOn in fromAnyTemperature limits toBoiling
      val failsOverAtBoiling =
        query verify postconditions.failsOverAtBoiling in fromAnyTemperature limits toBoiling
      val boilsAndBeeps =
        query find postconditions.failsOverAtBoiling in fromRoomTemperature limits toBoiling
      val refutesNeverBeeps =
        query verify invariants.neverBeeps in fromRoomTemperature limits toBoiling

  // The kettle left to itself once the user switched it on. The user's own actions are restricted
  // rather than assumed away: a user who switches the kettle off before it boils keeps it from
  // boiling, which no fairness of the heater excludes. Pressing the button stays, since from room
  // temperature nothing heats until the user switches the element on.
  object HeatingOnly
      extends Derived(Model.restrict(user.pressButton, heater.heat, heater.failover)):
    import Model.{beeping, heatingOn}

    object progress:
      // switched on, the water boils and the kettle beeps: 80 heats from room temperature and the
      // failover
      val boils: Progress[State] = HeatingOnly.leadsTo("boils")(heatingOn, beeping, within = 81)
