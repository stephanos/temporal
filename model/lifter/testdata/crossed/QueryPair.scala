package fixture.crossed.querypair

import umpire.*

final case class First(on: Boolean) derives Finite

final case class Second(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val first: Machine[First, Outcome, Nothing] =
  machine[First, Outcome, Nothing](Family("fixture.crossed"), "first") {
    starts(First(false))
    ends(_ => true)
  }

val second: Machine[Second, Outcome, Nothing] =
  machine[Second, Outcome, Nothing](Family("fixture.crossed"), "second") {
    starts(Second(false))
    ends(_ => true)
  }

val property: Property[First] = first.property("property") holds (_ => true)
val scenario: Scenario[Second] = second.scenario("scenario").starts(Second(false)).free
val wrongPair: Query = (query("wrongPair") find property in scenario).limits(Limits("one", 1, 1, 1))
