package fixture.features.kiln
package system

import framework.*
import product.KilnProduct

object KilnSystem extends Machine[Kiln, Outcome, Nothing]:
  val init = Kiln(hot = false)
  def end(s: State) = true

  object refinement extends Refinement(KilnProduct):
    def toProduct(s: State) = s

  object effects:
    def fired(s: State): List[KilnStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    on(potter.fire)(where(!_.hot) ~> effects.fired)
