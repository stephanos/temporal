package fixture.features.furnace
package system

import umpire.*

object FurnaceSystem extends Machine[Light, Outcome, Nothing]:
  val init = Light(lit = false)
  def end(s: State) = true

  object refinement extends Refinement(product.FurnaceProduct):
    def toProduct(s: State) = s

  object effects:
    def flip(s: State) = enter(s.copy(lit = !s.lit))

  object rules extends Rules:
    on(user.flip)(always ~> effects.flip)

object SpareSystem extends Derived(FurnaceSystem.restrict(user.flip))
