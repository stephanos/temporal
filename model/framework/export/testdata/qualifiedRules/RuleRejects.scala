package fixture.qualifiedrules

import framework.dsl.*

object BareWholeOutside extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    on(press)(always ~> WholeOutside.effects.keep)

object BareWholeInFrom extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      on(press)(always ~> WholeOutside.effects.keep)
    }

object BareClassOutside extends Machine[State, Outcome, Nothing]:
  import hand.turn
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    on(turn(Dial.low))(always ~> WholeOutside.effects.keep)

object BareClassInFrom extends Machine[State, Outcome, Nothing]:
  import hand.turn
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      on(turn(Dial.low))(always ~> WholeOutside.effects.keep)
    }

object BareTwoOutside extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    on(press, hand.turn(Dial.low))(always ~> WholeOutside.effects.keep)

object BareTwoInFrom extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      on(press, hand.turn(Dial.low))(always ~> WholeOutside.effects.keep)
    }

object BareThreeOutside extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    on(press, hand.turn(Dial.low), hand.stop)(always ~> WholeOutside.effects.keep)

object BareThreeInFrom extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      on(press, hand.turn(Dial.low), hand.stop)(always ~> WholeOutside.effects.keep)
    }

object BareFourOutside extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    on(press, hand.turn(Dial.low), hand.stop, hand.reset)(always ~> WholeOutside.effects.keep)

object BareFourInFrom extends Machine[State, Outcome, Nothing]:
  import hand.press
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      on(press, hand.turn(Dial.low), hand.stop, hand.reset)(always ~> WholeOutside.effects.keep)
    }

object ImportedWildcard extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      import hand.*
      on(press)(always ~> WholeOutside.effects.keep)
    }

object ImportedSelective extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      import hand.press
      on(press)(always ~> WholeOutside.effects.keep)
    }

object Foreign extends Machine[State, Outcome, Nothing]:
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(hand) {
      on(clock.tick)(always ~> WholeOutside.effects.keep)
    }

object BareDisabled extends Machine[State, Outcome, Nothing]:
  import clock.tick
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    disabled(tick)

object BareDisabledInFrom extends Machine[State, Outcome, Nothing]:
  import clock.tick
  val init = State(false)
  def end(s: State) = true
  object rules extends Rules:
    from(clock) {
      disabled(tick)
    }
