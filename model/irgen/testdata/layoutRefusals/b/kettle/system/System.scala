package fixture.features.kettle
package system

import umpire.*
import product.KettleContract

object KettleSystem extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object refinement extends Refinement(KettleContract):
    def toProduct(s: State) = s

  object effects:
    def boil(s: State) = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boil)

