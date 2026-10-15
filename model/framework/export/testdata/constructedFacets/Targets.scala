package fixture.facetstargets

import framework.dsl.*

trait WholeOutsideFacet extends Facet[State, Outcome, Nothing]:
  object wholeOutsideFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      on(hand.press)(always ~> effects.keep)

object WholeOutside extends Machine[State, Outcome, Nothing], WholeOutsideFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(wholeOutsideFacet.rules)

trait WholeInFromFacet extends Facet[State, Outcome, Nothing]:
  object wholeInFromFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      from(hand) {
        on(hand.press)(always ~> effects.keep)
      }

object WholeInFrom extends Machine[State, Outcome, Nothing], WholeInFromFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(wholeInFromFacet.rules)

trait ClassOutsideFacet extends Facet[State, Outcome, Nothing]:
  object classOutsideFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      on(hand.turn(Dial.low))(always ~> effects.keep)

object ClassOutside extends Machine[State, Outcome, Nothing], ClassOutsideFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(classOutsideFacet.rules)

trait ClassInFromFacet extends Facet[State, Outcome, Nothing]:
  object classInFromFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      from(hand) {
        on(hand.turn(Dial.low))(always ~> effects.keep)
      }

object ClassInFrom extends Machine[State, Outcome, Nothing], ClassInFromFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(classInFromFacet.rules)

trait TwoOutsideFacet extends Facet[State, Outcome, Nothing]:
  object twoOutsideFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      on(hand.press, hand.turn(Dial.low))(always ~> effects.keep)

object TwoOutside extends Machine[State, Outcome, Nothing], TwoOutsideFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(twoOutsideFacet.rules)

trait TwoInFromFacet extends Facet[State, Outcome, Nothing]:
  object twoInFromFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      from(hand) {
        on(hand.press, hand.turn(Dial.low))(always ~> effects.keep)
      }

object TwoInFrom extends Machine[State, Outcome, Nothing], TwoInFromFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(twoInFromFacet.rules)

trait ThreeOutsideFacet extends Facet[State, Outcome, Nothing]:
  object threeOutsideFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      on(hand.press, hand.turn(Dial.low), hand.stop)(always ~> effects.keep)

object ThreeOutside extends Machine[State, Outcome, Nothing], ThreeOutsideFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(threeOutsideFacet.rules)

trait ThreeInFromFacet extends Facet[State, Outcome, Nothing]:
  object threeInFromFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      from(hand) {
        on(hand.press, hand.turn(Dial.low), hand.stop)(always ~> effects.keep)
      }

object ThreeInFrom extends Machine[State, Outcome, Nothing], ThreeInFromFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(threeInFromFacet.rules)

trait FourOutsideFacet extends Facet[State, Outcome, Nothing]:
  object fourOutsideFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      on(hand.press, hand.turn(Dial.low), hand.stop, hand.reset)(always ~> effects.keep)

object FourOutside extends Machine[State, Outcome, Nothing], FourOutsideFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(fourOutsideFacet.rules)

trait FourInFromFacet extends Facet[State, Outcome, Nothing]:
  object fourInFromFacet:
    object effects:
      def keep(s: State) = enter[State, Outcome, Nothing](s)
    object rules extends Rules:
      from(hand) {
        on(hand.press, hand.turn(Dial.low), hand.stop, hand.reset)(always ~> effects.keep)
      }

object FourInFrom extends Machine[State, Outcome, Nothing], FourInFromFacet:
  val init = State(false)
  def end(s: State) = true
  object rules extends Facets(fourInFromFacet.rules)
