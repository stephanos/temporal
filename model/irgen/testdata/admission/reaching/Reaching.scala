package fixture.admission.reaching

import framework.*
import fixture.admission.unrelated.Broken

final case class State(on: Boolean)

enum Outcome:
  case accepted

// A root that reads the refused source, which it therefore admits.
object Switch extends Machine[State, Outcome, Nothing]:
  val init = State(Broken.first > 1)
  def end(state: State) = true
  object rules extends Bindings()
