package samples.kettle

import plain.*

// The same Kettle as top-level definitions under the sample's package, the Machine named after the
// sample and extending plain `Machine` (no type arguments: Outcome and Fact are abstract members
// it leaves undeclared, since it never rejects or records).
val boilingTemperature = 100

object user extends Actor:
  val pressButton = action
  val depressButton = action

object heater extends Internal:
  val heat = step
  val failover = step

object Kettle extends Machine:
  var temperature = oneOf(-40 to 40)
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
    val boils =
      leadsTo(from = heatingOn, to = beeping, within = 81) in only(heater.heat, heater.failover)
