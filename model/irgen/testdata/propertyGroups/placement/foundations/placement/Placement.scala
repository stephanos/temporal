package propertygroups.foundations.placement

import framework.*
import propertygroups.*

object Owner extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object effects:
    def finish(s: State) = step(s)
  object monitors:
    val watched = seal
  object rules extends Rules:
    on(tester.advance)(always ~> effects.finish)
  object properties:
    object safety:
      val direct = property("direct").holds(after => after.state == State.closed)
      val alias = direct
      val factory = shared(Owner)
      val never = property("never").never(after => after.state == State.open)
      val neverAlias = never
      val stays = property("stays").stays(_ == State.closed)
      val staysAlias = stays
      val unless =
        property("unless").stays(_ == State.closed).unless(after => after.state == State.open)
      val bundle = subtypes(Owner)
      val bundleAlias = bundle
      val selected = bundle.never
      val watched = seal
      val monitorAlias = watched
      val monitorFactory = attached
      def progressHelper =
        Owner.leadsTo("helperProgress")(_ == State.open, _ == State.closed, 1, fair)
    object liveness:
      val direct = Owner.leadsTo("directProgress")(_ == State.open, _ == State.closed, 1, fair)
      val alias = direct
      val factory = eventual(Owner)
      def propertyHelper = property("helperProperty").holds(after => after.state == State.closed)

object Empty extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object safety
    object liveness

object Omitted extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules

object SafetyOnly extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object safety:
      val only = property("only").never(after => after.state == State.open)

object LivenessOnly extends Machine[State, Outcome, Nothing]:
  def init = State.open
  def end(s: State) = s == State.closed
  object rules extends Rules
  object properties:
    object liveness:
      val only = LivenessOnly.leadsTo("only")(_ == State.open, _ == State.closed, 3, fair)
