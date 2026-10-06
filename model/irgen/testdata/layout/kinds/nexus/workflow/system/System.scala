package fixture.features.nexus.workflow
package system

import umpire.*
import product.NexusProduct

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case completed

object NexusSystem extends Machine[system.State, Outcome, Fact]:
  val init = system.State(phase = Phase.idle)
  def end(s: State) = true

  object refinement extends Refinement(NexusProduct):
    def toProduct(s: State) =
      product.State(if s.phase == Phase.idle then product.Phase.idle else product.Phase.done)

  object effects:
    def complete(s: State) = enter(s.copy(phase = Phase.done), Fact.completed)

  object rules extends Rules:
    on(user.complete)(always ~> effects.complete)
