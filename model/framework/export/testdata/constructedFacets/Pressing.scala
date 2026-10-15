package fixture.constructedfacets

import framework.dsl.*

trait Pressing extends Facet[State, Boolean, Fact], Phased[State, Phase]:
  object pressing:
    object effects:
      def move(s: State) = enters(Phase.done)(s)
    object rules extends Rules:
      on(hand.press):
        when(Phase.open) ~> enters(Phase.done)
      on(hand.assist):
        when(Phase.open) ~> effects.move
