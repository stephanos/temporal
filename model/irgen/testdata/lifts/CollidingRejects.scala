// A machine object of another package with the name of fixture.rejects' `Lookalike`: both are named
// `lookalike`, and claims name a machine by its name, so the lifter refuses the pair
// `LookalikePair` composes (fn-126 R15).
package fixture.rejects.elsewhere

import umpire.*
import fixture.rejects.{bulbHand, Bulb, Glow, Outcome, Ruled}

object Lookalike extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: State) = true
  object rules extends Rules:
    on(bulbHand.squeeze)(always ~> Ruled.effects.brighten)
