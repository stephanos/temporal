package fixture.retiredrulecaselifter

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
  object legacy:
    def in(first: Phase, rest: Phase*): Case[State, Outcome, Nothing] =
      rules.when(first, rest*)
  object rules extends Rules:
    // This fixture-local compatibility shim lets the retired source compile so the lifter must
    // refuse its heading too; production Rules has no semantic rule-case `in` overload.
    on(user.finish) {
      legacy.in(Phase.idle) ~> effects.finish
    }
