package fixture.features.brazier
package system

import umpire.*

object BrazierSystem extends Machine[Light, Outcome, Nothing]:
  val init = Light(lit = false)
  def end(s: State) = true

  object refinement extends Refinement(product.BrazierProduct):
    def toProduct(s: State) = s

  object effects:
    def flip(s: State) = enter(s.copy(lit = !s.lit))

  object rules extends Rules:
    on(user.flip)(always ~> effects.flip)
