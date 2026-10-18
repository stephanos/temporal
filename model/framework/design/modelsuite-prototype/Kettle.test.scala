package framework.prototype

import samples.Kettle

@scala.annotation.nowarn("msg=unused value of type .*Boolean.*")
class KettleTest extends ModelSuite(Kettle.Model):
  import model.{temperature, heatingOn, beeping}
  val room = model.init.copy(temperature = 20)
  object heater:
    def heat: Unit = step(Kettle.heater.heat())
    def failover: Unit = step(Kettle.heater.failover())
  object examples:
    test("boils from room temperature", from = room):
      step(Kettle.user.pressButton())
      repeat(80)(heater.heat)
      expect:
        temperature == 100
        heatingOn
      heater.failover
      expect:
        beeping
        !heatingOn
    test("trailing expectation remains readable", from = room, expectation = "satisfied"):
      step(Kettle.user.pressButton())
      expect:
        heatingOn
