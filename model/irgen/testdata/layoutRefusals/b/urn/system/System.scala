package fixture.features.urn
package system

import umpire.*

object UrnSystem extends Machine[Urn, Outcome, Nothing]:
  val init = Urn(hot = false)
  def end(s: State) = true

  object effects:
    def boil(s: State) = enter(s.copy(hot = true))

  object rules extends Rules:
    on(cook.boil)(where(!_.hot) ~> effects.boil)
