package fixture.facetsprojection

import framework.dsl.*

given Ok[Boolean] = Ok(true)
enum Phase:
  case open, done
final case class State(phase: Phase)
object hand extends Actor:
  val go = action(this)

trait Reading extends Facet[State, Boolean, Nothing]:
  given Phasing[State, Phase] = Projection.phased
  object reading:
    object rules extends Rules:
      on(hand.go)(when(Phase.open) ~> ((s: State) => stay[State, Boolean, Nothing](s)))

object Missing extends Machine[State, Boolean, Nothing], Reading:
  val init = State(Phase.open)
  def end(s: State) = true
  object rules extends Facets(reading.rules)

object Projection extends Phased[State, Phase](_.phase):
  val name = "projection"
