package fixture.derivationfacets

import framework.dsl.*
import framework.dsl.Finite.given

object A
    extends Machine[State, Boolean, Fact],
      Phased[State, Phase](_.left, (s, p) => s.copy(left = p)),
      Pressing:
  val init = State(Phase.open, Phase.done)
  object rules extends Facets(pressing.rules)

object B
    extends Machine[State, Boolean, Fact],
      Phased[State, Phase](_.right, (s, p) => s.copy(right = p)),
      Pressing:
  val init = State(Phase.done, Phase.open)
  object rules extends Facets(pressing.rules)

object ARebound extends Derived(A.rebind(hand.go ~> A.pressing.effects.move))
object BRebound extends Derived(B.rebind(hand.go ~> B.pressing.effects.move))
object AReplaced
    extends Derived(
      A.rebind(on(hand.go)(where(A.pressing.states.open) ~> A.pressing.effects.close))
    )
object BReplaced
    extends Derived(
      B.rebind(on(hand.go)(where(B.pressing.states.open) ~> B.pressing.effects.close))
    )
object AExtended extends Derived(A.extend(on(hand.more)(always ~> A.pressing.effects.move)))
object BExtended extends Derived(B.extend(on(hand.more)(always ~> B.pressing.effects.move)))
object ADisabledRebound extends Derived(A.rebind(hand.off ~> A.pressing.effects.move))
object BDisabledRebound extends Derived(B.rebind(hand.off ~> B.pressing.effects.move))
object ATimerRebound extends Derived(A.rebind(clock.tick ~> A.pressing.effects.move))
object BTimerRebound extends Derived(B.rebind(clock.tick ~> B.pressing.effects.move))
