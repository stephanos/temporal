// The rule-case `in`, retired in favor of `when`, with membership `phase.in(...)` still live.
package fixture.retiredrulecase

import framework.*

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

object user extends Actor:
  val finish = action(this)

object Retired extends Machine[State, Outcome, Nothing], Phased[State, Phase](_.phase):
  val init = State(Phase.idle)
  override def end(s: State) = s.phase.in(Phase.done)
  object effects:
    def finish(s: State) = enter(s.copy(phase = Phase.done))
  object rules extends Rules:
    on(user.finish) {
      in(Phase.idle) ~> effects.finish
    }
