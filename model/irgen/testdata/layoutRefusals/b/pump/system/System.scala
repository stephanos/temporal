package fixture.features.pump
package system

import umpire.*
import product.PumpProduct

object PumpProtocol extends Machine[Pump, Outcome, Nothing]:
  val init = Pump(on = false)
  def end(s: State) = true

  object refinement extends Refinement(PumpProduct):
    def toProduct(s: State) = s

  object effects:
    def start(s: State) = enter(s.copy(on = true))

  object rules extends Rules:
    on(operator.start)(where(!_.on) ~> effects.start)
