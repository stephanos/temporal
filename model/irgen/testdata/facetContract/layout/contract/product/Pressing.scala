package fixture.features.contract
package product

import framework.*

trait Pressing extends Facet[QueueView, Outcome, Fact], Phased[QueueView, Phase]:
  object pressing:
    object effects:
      def settle(s: State) = enter(s.copy(phase = Phase.on), Fact.changed)
    object rules extends Rules:
      on(hand.press) {
        when(Phase.off) ~> effects.settle
      }
