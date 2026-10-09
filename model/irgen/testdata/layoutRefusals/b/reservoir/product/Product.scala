package fixture.features.reservoir
package product

import framework.*

object ReservoirProduct extends Machine[ProductState, Outcome, ProductFact]:
  val init = ProductState(ProductPhase.empty)
  def end(s: State) = s.phase == ProductPhase.full

  object effects:
    def fill(s: State) = enter(s.copy(phase = ProductPhase.full), ProductFact.filled)

  object rules extends Rules:
    on(user.fill)(where(_.phase == ProductPhase.empty) ~> effects.fill)
