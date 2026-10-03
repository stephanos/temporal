package fixture.samestate

import umpire.*

final case class State(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val first: Machine[State, Outcome, Nothing] =
  machine[State, Outcome, Nothing](Family("fixture.samestate"), "first") {
    starts(State(false))
    ends(_ => true)
  }

val second: Machine[State, Outcome, Nothing] =
  machine[State, Outcome, Nothing](Family("fixture.samestate"), "second") {
    starts(State(false))
    ends(_ => true)
  }

val property: Property[State] = first.property("property") holds (_ => true)
val scenario: Scenario[State] = second.scenario("scenario").starts(State(false)).free
val wrongPair: Query = (query("wrongPair") find property in scenario).limits(Limits("one", 1, 1, 1))
