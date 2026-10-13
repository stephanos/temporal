package propertygroups.foundations.wrongowner

import framework.dsl.*
import propertygroups.*

object Home extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object safety:
      val law = shared(Home)
    object liveness:
      val progress = eventual(Home)

object WrongOwner extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object safety:
      val misplaced = shared(Home)
      val alias = Home.properties.safety.law
      val bundle = subtypes(Home)
      val laterOwner = Laws(shared(WrongOwner), Home.properties.safety.law)
    object liveness:
      val misplaced = eventual(Home)
      val alias = Home.properties.liveness.progress
      val laterOwner = ProgressPair(eventual(WrongOwner), Home.properties.liveness.progress)
