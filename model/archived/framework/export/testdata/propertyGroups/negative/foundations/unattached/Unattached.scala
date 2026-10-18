package propertygroups.foundations.unattached

import framework.dsl.*
import propertygroups.*

object Unattached extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object effects:
    def finish(s: State) = step(s)
  object rules extends Rules:
    on(tester.advance)(always ~> effects.finish)
  object properties:
    object safety:
      val notAttached = seal
