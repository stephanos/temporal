package fixture.foundations.pump
package product

import framework.*

object PumpProduct extends Machine[Pump, Outcome, Nothing]:
  val init = Pump(running = false)
  def end(s: State) = true

  object effects:
    def started(s: State): List[PumpStep] = enter(s.copy(running = true))

  object rules extends Rules:
    on(operator.start)(where(!_.running) ~> effects.started)
