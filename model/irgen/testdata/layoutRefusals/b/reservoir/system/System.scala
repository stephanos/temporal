package fixture.features.reservoir
package system

import framework.*
import product.ReservoirProduct

object ReservoirSystem extends Machine[SystemState, Outcome, SystemFact]:
  val init = SystemState(Phase.empty)
  def end(s: State) = s.phase == Phase.full

  object refinement extends Refinement(ReservoirProduct):
    def toProduct(s: State) = ProductState(ProductPhase.valueOf(s.phase.toString))

  object effects:
    def fill(s: State) = enter(s.copy(phase = Phase.full), SystemFact.filled)

  object rules extends Rules:
    on(user.fill)(where(_.phase == Phase.empty) ~> effects.fill)
