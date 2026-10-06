package fixture.features.stove
package product

import umpire.*

object StoveProduct extends Machine[Light, Outcome, Nothing]:
  val init = Light(lit = false)
  def end(s: State) = true

  object effects:
    def flip(s: State) = enter(s.copy(lit = !s.lit))

  object rules extends Rules:
    on(user.flip)(always ~> effects.flip)

object Spare extends Derived(StoveProduct.restrict(user.flip))
