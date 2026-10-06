// Rules that name phases of no projection (fn-126 R16): `in` reads the projection the rules declare,
// `Rules(_.phase)`, and rules that declare none name no phase.
package fixture.crossed

import umpire.*

object flipper extends Actor:
  val flip = action(this)

object Unprojected extends Machine[Here, Outcome, Nothing]:
  val init = Here(false)
  def end(s: Here) = true
  object effects:
    def turn(s: Here) = List(Step[Here, Outcome, Nothing](Outcome.accepted, Here(!s.on)))
  object rules extends Rules:
    on(flipper.flip)(in(true) ~> effects.turn)
