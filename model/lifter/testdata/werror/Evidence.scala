// A Model the build must refuse: its evidence match leaves a fact without a line, which -Werror makes
// a compile error (Machine.scala's `evidence`). scala-cli reports it and exits 0, so its build is
// read by its diagnostics rather than its exit status, and this one is expected at the match's line.
package fixture.werror

import umpire.*

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Outcome derives Finite:
  case accepted

enum Fact derives Finite:
  case finished, logged

val go = action("go", Party("fixture"))

def goStep(s: State): List[Step[State, Outcome, Fact]] =
  if s.phase == Phase.done then Nil
  else List(Step(Outcome.accepted, State(Phase.done), List(Fact.finished)))

val unconfirmed: Machine[State, Outcome, Fact] =
  machine[State, Outcome, Fact](Family("fixture.werror"), "unconfirmed") {
    starts(State(Phase.idle))
    ends(s => s.phase == Phase.done)
    evidence { case Fact.finished => "finished" }
    steps(go ~> goStep)
  }
