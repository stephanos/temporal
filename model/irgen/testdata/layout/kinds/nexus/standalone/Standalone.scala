package fixture.features.nexus
package standalone

import umpire.*

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case completed

enum Outcome derives Finite:
  case accepted

object user extends Actor:
  val complete = action(this)

given Ok[Outcome] = Ok(Outcome.accepted)

object NexusOperation extends Machine[standalone.State, Outcome, Fact]:
  val init = standalone.State(phase = Phase.idle)
  def end(s: State) = true

  object effects:
    def complete(s: State) = enter(s.copy(phase = Phase.done), Fact.completed)

  object rules extends Rules:
    on(user.complete)(always ~> effects.complete)

object exports:
  val nexusStandalone = irFile("fixture-nexus-standalone")(NexusOperation)
