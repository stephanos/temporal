package fixture.features.boiler
package product

import umpire.*

object BoilerContract extends Machine[Boiler, Outcome, Nothing]:
  val init = Boiler(hot = false)
  def end(s: State) = true

  object effects:
    def boil(s: State) = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boil)
