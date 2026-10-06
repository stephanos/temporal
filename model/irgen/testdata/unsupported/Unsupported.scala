// A Model the lifter must refuse: its step function loops over a mutable variable, which the IR has no
// form for. The lifter's tests lift it and expect the refusal at the loop's line.
package temporal.fixture

import umpire.*

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Outcome derives Finite:
  case accepted

val go = action(Actor("fixture"))

def goStep(s: State): List[Step[State, Outcome, Nothing]] =
  var out = List.empty[Step[State, Outcome, Nothing]]
  while out.isEmpty do out = List(Step(Outcome.accepted, State(Phase.done)))
  out

object Unsupported extends Machine[State, Outcome, Nothing]:
  val init = State(Phase.idle)
  def end(s: State) = false

  object rules extends Bindings(go ~> goStep)
