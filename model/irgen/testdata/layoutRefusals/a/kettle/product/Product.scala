package fixture.features.kettle
package product

import umpire.*

object KettleProduct extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object effects extends Section:
    def boiled(s: State): List[KettleStep] = enter(s.copy(hot = true))

  object rules extends Rules:
    when(!_.hot)(cook.boil ~> effects.boiled)
