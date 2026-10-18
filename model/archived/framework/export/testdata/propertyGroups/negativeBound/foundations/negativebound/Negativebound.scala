package propertygroups.foundations.negativebound

import framework.dsl.*
import propertygroups.*

object NegativeBound extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object liveness:
      val invalid = NegativeBound.leadsTo("invalid")(_ == State.open, _ == State.closed, -1, fair)
