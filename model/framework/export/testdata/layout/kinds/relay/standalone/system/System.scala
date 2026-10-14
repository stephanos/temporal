package fixture.features.relay.standalone
package system

import framework.dsl.*; import framework.dsl.Finite.given
import fixture.features.relay.{given, *}
import fixture.features.relay.shared.product.RelayProduct
val _ = () => summon[CanEqual[Phase, Phase]]
enum Phase:
  case idle, done

final case class State(phase: Phase)

enum Fact:
  case completed

object RelaySystem extends Machine[system.State, Outcome, Fact]:
  val init = system.State(phase = Phase.idle)
  def end(s: State) = true

  object refinement extends Refinement(RelayProduct):
    def toProduct(s: State) =
      fixture.features.relay.shared.product.State(if s.phase == Phase.idle then
        fixture.features.relay.shared.product.Phase.idle
      else fixture.features.relay.shared.product.Phase.done)

  object effects:
    def start(s: State) = stay(s)
    def complete(s: State) = enter(s.copy(phase = Phase.done), Fact.completed)

  object rules extends Rules:
    on(formBindings.start)(always ~> effects.start)
    on(formBindings.complete)(always ~> effects.complete)
