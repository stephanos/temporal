package fixture.constructedfacets

import framework.dsl.*
import framework.dsl.Finite.given

object A
    extends Machine[State, Boolean, Fact],
      Phased[State, Phase](_.left, (s, p) => s.copy(left = p)),
      Pressing:
  val init = State(Phase.open, Phase.done)
  object rules extends Facets(pressing.rules)

object B
    extends Machine[State, Boolean, Fact],
      Phased[State, Phase](_.right, (s, p) => s.copy(right = p)),
      Pressing:
  val init = State(Phase.done, Phase.open)
  object rules extends Facets(pressing.rules)
