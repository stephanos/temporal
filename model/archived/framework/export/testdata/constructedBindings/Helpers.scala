package fixture.bindingdiscovery

import framework.dsl.*

object Helpers extends Lifted:
  def leaf(s: State): List[Step[State, Outcome, Nothing]] =
    List(Step(Outcome.accepted, s))
  def forward(s: State): List[Step[State, Outcome, Nothing]] = leaf(s)
