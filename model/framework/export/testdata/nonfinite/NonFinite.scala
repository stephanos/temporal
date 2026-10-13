package fixture.crossed.nonfinite

import framework.*
import framework.Finite.given

final case class State(value: String)

val domain = summon[Finite[State]]
