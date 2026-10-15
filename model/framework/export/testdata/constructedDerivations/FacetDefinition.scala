package fixture.derivationfacets

import framework.dsl.*
import framework.dsl.Finite.given

val _ = () => summon[CanEqual[Phase, Phase]]
def phase(using v: View[State]): Phase = v.get(_.left)
def phase_=(p: Phase)(using d: Draft[State, ?, Fact]): Unit = d.set(p)(_.copy(left = p))

trait Pressing extends Facet[State, Boolean, Fact], Phased[State, Phase]:
  object pressing:
    object states:
      val open = is(phase == Phase.open)
    object effects:
      def move(s: State) = enters(Phase.done)(s)
      val close = effect {
        phase = Phase.done
      }
    object rules extends Rules:
      on(hand.go)(when(Phase.open) ~> effects.move)
      on(clock.tick)(when(Phase.open) ~> effects.close)
      disabled(hand.off)
