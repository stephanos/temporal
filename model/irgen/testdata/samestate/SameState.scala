package fixture.samestate

import framework.*

final case class State(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

object First extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(state: State) = true
  object rules extends Bindings()

object Second extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(state: State) = true
  object rules extends Bindings()

val property: Property[State] = First.property holds (_ => true)
val scenario: Scenario[State] = Second.scenario.starts(State(false)).free
val wrongPair: Query = (query find property in scenario).limits(Limits("one", 1, 1, 1))
