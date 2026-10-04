// A Model the build must refuse: its step function's match leaves a phase without a case, which
// -Werror makes a compile error. scala-cli reports it and exits 0, so its build is read by its
// diagnostics rather than its exit status, and this one is expected at the match's line.
package fixture.werror

import umpire.*

enum Phase derives Finite:
  case idle, working, done

final case class State(phase: Phase) derives Finite

enum Outcome derives Finite:
  case accepted

enum Fact derives Finite:
  case finished

val go = action("go", Party("fixture"))

def goStep(s: State): List[Step[State, Outcome, Fact]] = s.phase match
  case Phase.idle    => List(Step(Outcome.accepted, State(Phase.working)))
  case Phase.working => List(Step(Outcome.accepted, State(Phase.done), List(Fact.finished)))

val unfinished: Machine[State, Outcome, Fact] =
  machine[State, Outcome, Fact](Family("fixture.werror"), "unfinished") {
    starts(State(Phase.idle))
    ends(s => s.phase == Phase.done)
    steps(go ~> goStep)
  }
