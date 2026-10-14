package fixture.bindingdiscovery

import framework.dsl.*

object Helpers:
  @lifted def leaf(s: State): List[Step[State, Outcome, Nothing]] =
    List(Step(Outcome.accepted, s))
  @lifted def forward(s: State): List[Step[State, Outcome, Nothing]] = leaf(s)
