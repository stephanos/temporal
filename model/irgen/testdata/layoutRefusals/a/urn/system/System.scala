package fixture.features.urn
package system

import umpire.*
import product.{brewer, UrnProduct, given}

object UrnSystem extends Machine[Urn, Outcome, Nothing]:
  val init = Urn(full = false)
  def end(s: State) = true

  object refinement extends Refinement(UrnProduct):
    def toProduct(s: State) = s

  object effects:
    def filled(s: State): List[UrnStep] = enter(s.copy(full = true))

  object rules extends Rules:
    on(brewer.fill)(where(!_.full) ~> effects.filled)
