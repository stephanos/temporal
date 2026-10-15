package fixture.constructedfacets

import framework.dsl.*
import framework.dsl.Finite.given

object PhaseAccess:
  @lifted def left(s: State): Phase = s.left
  @lifted def right(s: State): Phase = s.right
  @lifted def write(left: Boolean)(s: State, p: Phase): State =
    if left then s.copy(left = p) else s.copy(right = p)

object MultiA
    extends Machine[State, Boolean, Fact],
      Phased[State, Phase](
        s => PhaseAccess.left(s),
        (s, p) => PhaseAccess.write(true)(s, p)
      ),
      Pressing:
  val init = State(Phase.open, Phase.done)
  object rules extends Facets(pressing.rules)

object MultiB
    extends Machine[State, Boolean, Fact],
      Phased[State, Phase](
        s => PhaseAccess.right(s),
        (s, p) => PhaseAccess.write(false)(s, p)
      ),
      Pressing:
  val init = State(Phase.done, Phase.open)
  object rules extends Facets(pressing.rules)
