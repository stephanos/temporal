package fixture.features.contract
package protocol

import framework.dsl.*

trait Pressing extends Facet[QueueDetail, Outcome, Fact], Phased[QueueDetail, Phase]:
  object pressing:
    object effects:
      def settle(s: State) = enter(s.copy(phase = Phase.on), Fact.changed)
    object rules extends Rules:
      on(hand.press) {
        when(Phase.off) ~> effects.settle
      }
