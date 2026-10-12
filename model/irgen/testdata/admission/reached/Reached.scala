package fixture.admission.reached

import framework.*

final case class State(on: Boolean)

enum Outcome:
  case accepted

// The root a lift names: it reaches no source of its fixture's other folders.
object Lamp extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(state: State) = true
  object rules extends Bindings()
