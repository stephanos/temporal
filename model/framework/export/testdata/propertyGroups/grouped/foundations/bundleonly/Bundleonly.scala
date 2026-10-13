package propertygroups.foundations.bundleonly

import framework.dsl.*
import propertygroups.*

object BundleOnly extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object effects:
    def finish(s: State) = step(s)
  object rules extends Rules:
    on(tester.advance)(always ~> effects.finish)
  object properties:
    object liveness:
      val bundle = progressLaws(BundleOnly)
