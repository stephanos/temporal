package fixture.facetscontributors

import framework.dsl.*
import framework.dsl.Finite.given

given Ok[Boolean] = Ok(true)
enum Phase:
  case open extends Phase, Live
  case waiting extends Phase, Waiting
  case done extends Phase, Closed
final case class State(phase: Phase, flag: Boolean)
object first extends Actor:
  val go = action(this)
  val off = action(this)
object second extends Actor:
  val go = action("assist", this)
object clock:
  val tick = timer
object fault extends Actor:
  val stop = action(this)

trait Left extends Facet[State, Boolean, Nothing], Phased[State, Phase]:
  object left:
    object effects:
      def keep(s: State) = stay[State, Boolean, Nothing](s)
    object rules extends Rules:
      on(first.go):
        when(Phase.open, Phase.waiting).where(s => !s.flag) ~> effects.keep
        when(Phase.open, Phase.waiting).where(_.flag) ~> effects.keep
        when[Closed] ~> effects.keep
      disabled(first.off)

trait Right extends Facet[State, Boolean, Nothing], Phased[State, Phase]:
  this: Left =>
  object right:
    object states:
      def closed(p: Phase) = p == Phase.done
    object effects:
      def keep(s: State) = left.effects.keep(s)
    object rules extends Rules:
      on(second.go)(when(states.closed) ~> effects.keep)
      on(clock.tick)(always ~> effects.keep)
      on(fault.stop)(always ~> effects.keep)

trait Empty extends Facet[State, Boolean, Nothing]:
  object empty:
    object rules extends Rules

object Forward
    extends Machine[State, Boolean, Nothing],
      Phased[State, Phase](_.phase),
      Left,
      Right,
      Empty,
      FailureModel:
  val init = State(Phase.open, false)
  object rules extends Facets(left.rules, empty.rules, right.rules)

object Reverse
    extends Machine[State, Boolean, Nothing],
      Phased[State, Phase](_.phase),
      Left,
      Right,
      Empty,
      FailureModel:
  val init = State(Phase.open, false)
  object rules extends Facets(right.rules, empty.rules, left.rules)
