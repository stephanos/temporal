package samples

import framework.dsl.*
import framework.dsl.Finite.given

// Two constructs outside the liftable subset, each refused while the Model compiles.
object Broken:
  final case class Counter(n: UpTo[3], on: Boolean)

  enum Outcome:
    case accepted

  given Ok[Outcome] = Ok(Outcome.accepted)

  object hand extends Actor:
    val press = action(this)
    val spin = action(this)

  object Model extends Machine[Counter, Outcome, Nothing]:
    val init = Counter(UpTo(0), false)
    def end(state: Counter) = true

    object effects:
      def press(state: Counter) =
        var next: Int = state.n
        next = next + 1
        enter(state.copy(n = UpTo(next)))
      def spin(state: Counter) =
        while state.on do ()
        enter(state)

    object rules extends Rules:
      on(hand.press)(where(_.on) ~> effects.press)
      on(hand.spin)(where(!_.on) ~> effects.spin)
