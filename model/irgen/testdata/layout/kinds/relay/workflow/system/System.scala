package fixture.features.relay.workflow
package system

import umpire.*
import fixture.features.relay.{given, *}
import fixture.features.relay.product.RelayProduct

enum Phase derives Finite:
  case idle, done

final case class State(phase: Phase) derives Finite

enum Fact derives Finite:
  case completed

object RelaySystem extends Machine[system.State, Outcome, Fact]:
  val init = system.State(phase = Phase.idle)
  def end(s: State) = true

  object refinement extends Refinement(RelayProduct):
    def toProduct(s: State) =
      fixture.features.relay.product.State(if s.phase == Phase.idle then
        fixture.features.relay.product.Phase.idle
      else fixture.features.relay.product.Phase.done)

  object effects:
    def start(s: State) = stay(s)
    def complete(s: State) = enter(s.copy(phase = Phase.done), Fact.completed)

  object rules extends Rules:
    on(formBindings.start)(always ~> effects.start)
    on(formBindings.complete)(always ~> effects.complete)
