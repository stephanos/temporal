package fixture.qualifiedrules

import framework.dsl.*

object WholeOutside extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    on(hand.press)(always ~> effects.keep)

object WholeInFrom extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    from(hand) {
      on(hand.press)(always ~> effects.keep)
    }

object ClassOutside extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    on(hand.turn(Dial.low))(always ~> effects.keep)

object ClassInFrom extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    from(hand) {
      on(hand.turn(Dial.low))(always ~> effects.keep)
    }

object TwoOutside extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    on(hand.press, hand.turn(Dial.low))(always ~> effects.keep)

object TwoInFrom extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    from(hand) {
      on(hand.press, hand.turn(Dial.low))(always ~> effects.keep)
    }

object ThreeOutside extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    on(hand.press, hand.turn(Dial.low), hand.stop)(always ~> effects.keep)

object ThreeInFrom extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    from(hand) {
      on(hand.press, hand.turn(Dial.low), hand.stop)(always ~> effects.keep)
    }

object FourOutside extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    on(hand.press, hand.turn(Dial.low), hand.stop, hand.reset)(always ~> effects.keep)

object FourInFrom extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object effects:
    def keep(s: State) = enter[State, Outcome, Nothing](s)
  object rules extends Rules:
    from(hand) {
      on(hand.press, hand.turn(Dial.low), hand.stop, hand.reset)(always ~> effects.keep)
    }
