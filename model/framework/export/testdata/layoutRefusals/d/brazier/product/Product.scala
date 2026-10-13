package fixture.features.brazier
package product

import framework.*

object BrazierProduct extends Machine[Light, Outcome, Nothing]:
  val init = Light(lit = false)
  def end(s: State) = true

  object effects:
    def flip(s: State) = enter(s.copy(lit = !s.lit))

  object rules extends Rules:
    on(user.flip)(always ~> effects.flip)

object SpareProduct extends Derived(BrazierProduct.restrict(user.flip))
