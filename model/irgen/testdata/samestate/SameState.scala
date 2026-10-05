package fixture.samestate

import umpire.*

given Family = Family("fixture.samestate")

final case class State(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val first = machine[State, Outcome, Nothing] {
  starts(State(false))
  ends(_ => true)
}

val second = machine[State, Outcome, Nothing] {
  starts(State(false))
  ends(_ => true)
}

val property: Property[State] = first.property holds (_ => true)
val scenario: Scenario[State] = second.scenario.starts(State(false)).free
val wrongPair: Query = (query find property in scenario).limits(Limits("one", 1, 1, 1))
