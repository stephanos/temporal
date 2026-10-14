package fixture.bindingdiscovery

import framework.dsl.*

object HelperMachine extends Machine[State, Outcome, Nothing]:
  val init = State.idle
  def end(s: State): Boolean = true
  object rules
      extends Bindings(
        named ~> Helpers.forward,
        second ~> ((s: State) => List(Step(Outcome.accepted, s)))
      )
