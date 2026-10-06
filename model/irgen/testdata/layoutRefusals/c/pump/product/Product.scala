package fixture.shared.pump
package product

import umpire.*

object PumpProduct extends Machine[Pump, Outcome, Nothing]:
  val init = Pump(running = false)
  def end(s: State) = true

  object effects extends Section:
    def started(s: State): List[PumpStep] = enter(s.copy(running = true))

  object rules extends Rules:
    when(!_.running)(operator.start ~> effects.started)
