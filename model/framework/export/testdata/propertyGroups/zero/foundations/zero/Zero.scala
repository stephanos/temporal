package propertygroups.foundations.zero

import framework.dsl.*
import propertygroups.*

object Zero extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object liveness:
      val invalid = Zero.leadsTo("invalid")(_ == State.open, _ == State.closed, 0, fair)
