package proto

import plain.*

// model/framework/samples/Kettle.scala in the plain-Scala form of fn-190 (spec "API Contracts").
// Modelled after https://github.com/quint-co/quint/blob/main/examples/tutorials/repl/kettle.qnt
object Kettle:
  val boilingTemperature = 100

  object user extends Actor: // outside the kettle: a test drives these
    val pressButton = action
    val depressButton = action

  object heater extends Internal: // part of the kettle: a test only observes these
    val heat = step
    val failover = step

  object Model extends Machine:
    var temperature = oneOf(-40 to 40) // any start temperature, as Quint's initNondet
    var heatingOn = false
    var beeping = false

    object transitions:
      on(user.pressButton):
        if !heatingOn then effects.pressButton()
      on(heater.failover):
        if heatingOn && temperature >= boilingTemperature then effects.failover()
      on(heater.heat):
        if heatingOn && temperature < boilingTemperature then effects.heat()
      on(user.depressButton):
        if heatingOn then effects.depressButton()

    object effects:
      def pressButton() =
        heatingOn = true
        beeping = false
      def failover() =
        heatingOn = false
        beeping = true
      def heat() =
        temperature += 1
      def depressButton() =
        heatingOn = false
        beeping = false

    object invariants:
      val beepsOnlyWhenOff = if beeping then check(!heatingOn)
      val neverBeeps = check(!beeping)

    object postconditions:
      val warmsOnlyWhenOn = if !before.heatingOn then check(temperature <= before.temperature)
      val failsOverAtBoiling = after(heater.failover):
        check(temperature == boilingTemperature)
        check(beeping)

    object progress:
      // as the spec wrote it; the probe measures it (see the design note)
      val boils =
        leadsTo(from = heatingOn, to = beeping, within = 81) in only(heater.heat, heater.failover)
