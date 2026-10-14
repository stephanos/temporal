package propertygroups.features.specimen.workflow

import framework.dsl.*
import propertygroups.*
import propertygroups.shared as imported

final case class Pair(value: State)

object Subject extends Machine[State, Outcome, Nothing]:
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
      val closed =
        property("closed").when(tester.advance).holds(after => after.state == State.closed)
      val stable = property("stable").holdsAcross((before, after) => before == after.state)
      val aliased = imported(Subject)
      val bundle = laws(Subject)
      val selected = bundle.closed
      val repeated = aliased
      val watched = seal
    object liveness:
      val reaches = Subject.leadsTo("eventuallyClosed")(_ == State.open, _ == State.closed, 1, fair)
      val factory = eventual(Subject)
  object queries:
    val finished =
      query("finished").verify(properties.safety.closed).in(scenario("any").free).limits(limits)

object Joined extends Composition[Pair](_.value -> Subject):
  def end(s: Pair) = s.value == State.closed
  object syncs extends Syncs
  object properties:
    object safety:
      val member = property("memberClosed").never(after => after.state.value != State.closed)
