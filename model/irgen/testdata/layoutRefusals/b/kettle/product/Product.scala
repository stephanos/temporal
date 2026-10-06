package fixture.features.kettle
package product

import umpire.*

object KettleContract extends Machine[Kettle, Outcome, Nothing]:
  val init = Kettle(hot = false)
  def end(s: State) = true

  object effects:
    def boil(s: State) = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boil)

