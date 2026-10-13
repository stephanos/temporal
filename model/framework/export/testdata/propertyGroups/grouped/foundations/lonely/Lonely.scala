package propertygroups.foundations.lonely

import framework.dsl.*
import propertygroups.*

object Lonely extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object effects:
    def finish(s: State) = step(s)
  object rules extends Rules:
    on(tester.advance)(always ~> effects.finish)
  object properties:
    object liveness:
      private val progress =
        Lonely.leadsTo("onlyProgress")(_ == State.open, _ == State.closed, 3, fair)
      def selectedProgress = progress
