package fixture.derivationfacets

import framework.dsl.*

object PhaseBearing
    extends Derived(
      A.rebind(on(hand.go)(A.pressing.rules.when(Phase.open) ~> A.pressing.effects.move))
    )
