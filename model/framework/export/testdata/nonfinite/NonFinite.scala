package fixture.crossed.nonfinite

import framework.dsl.*
import framework.dsl.Finite.given

final case class State(value: String)

val domain = summon[Finite[State]]
