package fixture.features.boiler
package system

import umpire.*
import product.BoilerContract

object BoilerImplementation extends Machine[Boiler, Outcome, Nothing]:
  val init = Boiler(hot = false)
  def end(s: State) = true

  object refinement extends Refinement(BoilerContract):
    def toProduct(s: State) = s

  object effects:
    def boil(s: State) = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boil)
