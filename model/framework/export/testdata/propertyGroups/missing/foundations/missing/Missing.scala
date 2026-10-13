package propertygroups.foundations.missing

import framework.dsl.*
import propertygroups.*

object Missing extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object properties:
    object liveness:
      val invalid = Missing.leadsTo("invalid")(_ == State.open, _ == State.closed)
