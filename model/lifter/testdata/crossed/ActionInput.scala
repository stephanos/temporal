package fixture.crossed.actioninput

import umpire.*

final case class State(on: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

val signal = action("signal", Party("fixture")).input[Boolean]("on")

def step(s: State, on: String): List[Step[State, Outcome, Nothing]] =
  List(Step(Outcome.accepted, s.copy(on = on.nonEmpty)))

val wrongInput = signal ~> step
