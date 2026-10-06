package fixture.features.pump
package product

import umpire.*

object PumpProduct extends Machine[Pump, Outcome, Nothing]:
  val init = Pump(on = false)
  def end(s: State) = true

  object effects:
    def start(s: State) = enter(s.copy(on = true))

  object rules extends Rules:
    on(operator.start)(where(!_.on) ~> effects.start)
