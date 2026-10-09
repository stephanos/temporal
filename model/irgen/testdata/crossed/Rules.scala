// Rules on a machine that is not Phased (fn-137 R5): `in` needs the machine's projection,
// `Phased[State, Phase](_.phase)`, and rules on a plain machine name no phase.
package fixture.crossed

import framework.*

object flipper extends Actor:
  val flip = action(this)

object Unprojected extends Machine[Here, Outcome, Nothing]:
  val init = Here(false)
  def end(s: Here) = true
  object effects:
    def turn(s: Here) = List(Step[Here, Outcome, Nothing](Outcome.accepted, Here(!s.on)))
  object rules extends Rules:
    on(flipper.flip)(when(true) ~> effects.turn)
