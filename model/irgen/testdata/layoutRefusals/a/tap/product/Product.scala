package fixture.features.tap
package product

import umpire.*

object Faucet extends Machine[Tap, Outcome, Nothing]:
  val init = Tap(open = false)
  def end(s: State) = true

  object effects extends Section:
    def turned(s: State): List[Step[Tap, Outcome, Nothing]] = enter(s.copy(open = !s.open))

  object rules extends Rules:
    when(_ => true)(plumber.turn ~> effects.turned)
