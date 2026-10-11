package propertygroups.foundations.hidden

import framework.*
import propertygroups.*

object Hidden extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object effects:
    def finish(s: State) = step(s)
  object rules extends Rules:
    on(tester.advance)(always ~> effects.finish)
  object properties:
    object helper:
      val ignored = Hidden.leadsTo("helperProgress")(_ == State.open, _ == State.closed, 1, fair)
