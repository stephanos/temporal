package fixture.features.kiln
package system

import umpire.*
import product.KilnProduct

object KilnSystem extends Machine[Kiln, Outcome, Nothing]:
  val init = Kiln(hot = false)
  def end(s: State) = true

  object refinement extends Refinement(KilnProduct):
    def toProduct(s: State) = s

  object effects extends Section:
    def fired(s: State): List[KilnStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    when(!_.hot)(potter.fire ~> effects.fired)
