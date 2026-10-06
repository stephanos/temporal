/* The lamp's System: how the server gets there (fn-126 R20). The level's own file, named after its
 * folder, holds the feature's System machine, `<Feature>System`, whose `object refinement` refines
 * the Product. Zoom-ins on how it keeps its promise sit beside it, one file per subject, as
 * Bulb.scala does.
 */
package fixture.features.lamp
package system

import umpire.*
import product.LampProduct

/** The lamp as the circuit runs it: the switch closes and opens the circuit. */
object LampSystem extends Machine[Circuit, Outcome, Nothing]:
  val init = Circuit(closed = false)
  def end(s: State) = true

  // What a caller reads of the circuit: the lamp is lit while the circuit is closed.
  object refinement extends Refinement(LampProduct):
    def toProduct(s: State) = Lamp(lit = s.closed)

  object effects extends Section:
    def close(s: State): List[CircuitStep] = enter(s.copy(closed = true))
    def open(s: State): List[CircuitStep] = enter(s.copy(closed = false))

  object rules extends Rules:
    when(!_.closed)(user.switchOn ~> effects.close)
    when(_.closed)(user.switchOff ~> effects.open)
