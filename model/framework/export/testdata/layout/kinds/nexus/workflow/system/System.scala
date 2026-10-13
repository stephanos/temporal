package fixture.features.nexus.workflow
package system

import framework.dsl.*; import framework.dsl.Finite.given
import product.NexusProduct
val _ = () => summon[CanEqual[Phase, Phase]]
enum Phase:
  case idle, done

final case class State(phase: Phase)

enum Fact:
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
