package fixture.foundations.pump
package system

import umpire.*
import product.PumpProduct

object PumpSystem extends Machine[Pump, Outcome, Nothing]:
  val init = Pump(running = false)
  def end(s: State) = true

  object refinement extends Refinement(PumpProduct):
    def toProduct(s: State) = s

  object effects:
    def started(s: State): List[PumpStep] = enter(s.copy(running = true))

  object rules extends Rules:
    on(operator.start)(where(!_.running) ~> effects.started)

// A type after the machine: a level folder's file reads as a feature file, its exports aside.
final case class Gauge(reading: Boolean) derives Finite

// An IR file outside the root feature file.
object exports:
  val pump = irFile("pump")(PumpProduct, PumpSystem)
