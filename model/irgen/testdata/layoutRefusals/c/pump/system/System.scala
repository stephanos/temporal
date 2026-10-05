package fixture.shared.pump
package system

import umpire.*
import product.PumpProduct

object PumpSystem extends Machine[Pump, Outcome, Nothing]:
  val init = Pump(running = false)
  def end(s: State) = true

  object refinement extends Refinement(PumpProduct):
    def toProduct(s: State) = s

  object effects extends Section:
    def started(s: State): List[PumpStep] = enter(s.copy(running = true))

  object rules extends Rules:
    when(!_.running)(operator.start ~> effects.started)

// An IR file outside the root feature file.
object exports:
  val pump = irFile("pump")(PumpProduct, PumpSystem)
