package fixture.features.tap
package product

import framework.*

object Faucet extends Machine[Tap, Outcome, Nothing]:
  val init = Tap(open = false)
  def end(s: State) = true

  object effects:
    def turned(s: State): List[Step[Tap, Outcome, Nothing]] = enter(s.copy(open = !s.open))

  object rules extends Rules:
    on(plumber.turn)(always ~> effects.turned)
