package outside

import framework.dsl.*
import framework.dsl.Finite.given

// A Machine written outside the framework's packages, as a sample is, in every word of the plain
// form: the claim words expand into its package, where the framework's own members are out of
// reach.
object Lamp:
  enum Said:
    case ok, refused

  enum Note:
    case lit, darkened

  given Ok[Said] = Ok(Said.ok)

  object hand extends Actor:
    val press = action
    val set = action.input[Boolean](name = "on")

  object Dark extends Machine[Said, Note]:
    var dark = true

    object transitions extends Transitions:
      on(hand.press):
        effects.press()

    object effects:
      def press() =
        dark = !dark

  object Model extends Machine[Said, Note]:
    var lit = false
    var presses = 0

    val pressing = only(hand.press).starts(presses = 0)

    object refinement extends Refinement(Dark):
      def toProduct(state: State) = Dark.init.copy(dark = !lit)

    object transitions extends Transitions:
      on(hand.press):
        if presses < 3 then effects.press() else reject(Said.refused)
      on(hand.set): to =>
        if to != lit && presses < 3 then effects.set(to)

    object effects:
      def press() =
        lit = !lit
        presses += 1
        records(if lit then Note.lit else Note.darkened)
      def set(to: Boolean) =
        lit = to
        presses += 1
        records(if lit then Note.lit else Note.darkened)

    object invariants:
      val bounded = check(presses <= 3)

    object postconditions:
      val counts = if outcome == Said.ok then check(presses == before.presses + 1)
      val lights = after(hand.press):
        oneOf:
          branch:
            records(Note.lit)
            check(lit)
          branch:
            records(Note.darkened)
            check(!lit)
          branch:
            check(outcome == Said.refused)

    object monitors:
      val litCount = monitor(0)(n => if recorded(Note.lit) then n + 1 else n)(_ > 3)
      val quiet = sticky(facts.size <= 1)

    object progress:
      val wearsOut = leadsTo(from = presses == 0, to = presses == 3, within = 3) in pressing

    object queries:
      val lightsUp = after(hand.press):
        check(lit)
      val lighting = find(lightsUp) in pressing
      val bounding = verify(invariants.bounded) in starts(presses = 1)

  object Stuck
      extends Derived(
        Model.rebind(
          on(hand.press):
            if Model.presses < 1 then Model.effects.press()
          ,
          on(hand.set): to =>
            if to && Model.presses < 3 then Model.effects.set(to)
        )
      )
