// The System machine, in a file not named after its folder: system/System.scala is missing.
package fixture.features.kettle
package system

import umpire.*
import product.KettleProduct

object KettleSystem extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object refinement extends Refinement(KettleProduct):
    def toProduct(s: State) = s

  object effects:
    def boiled(s: State): List[KettleStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boiled)
