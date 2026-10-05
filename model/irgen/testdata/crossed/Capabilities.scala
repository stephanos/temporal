// Capability declarations whose types cross, which their signatures refuse before anything is
// lifted (fn-122.2): the lifter's tests build this and expect a type error at each marked line.
package fixture.crossed

import umpire.*
import temporal.capabilities.{terminalStatesAreFinal, Pollable}

given Catalog = Catalog(Vector())

final case class Elsewhere(on: Boolean) derives Finite

def elsewhereOn(e: Elsewhere): Boolean = e.on
def hereOn(h: Here): Boolean = h.on

val flick = action("flick", Party("fixture"))
val ticks = Limits(steps = 1, actions = 1, search = 4)

// A predicate of another state type than the machine's.
val foreign = capabilities(here, ticks)(Pollable(dispatch = flick, running = elsewhereOn))

// A declaration without the Limits its generated verify Queries run under.
val noLimits = capabilities(here)(Pollable(dispatch = flick, running = hereOn))

// A waiver without a reason.
val noBecause = capabilities(here, ticks)(Pollable(dispatch = flick, running = hereOn))
  .except(terminalStatesAreFinal)

// A member read with `through` by a def of another state type.
val throughForeign =
  capabilities(here, ticks)(Pollable(dispatch = flick, running = through(_.on, elsewhereOn)))
