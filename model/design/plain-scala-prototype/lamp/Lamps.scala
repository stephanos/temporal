package proto

import plain.*

// model/framework/dsl/Facets.test.scala's fixture in the plain form: the phase is a var a facet
// declares abstractly, transitions are `match`es on it, effects are assignment blocks. A press
// wears the lamp out after three presses; a broken lamp refuses a press and can be repaired.
object Lamps:
  enum Lit derives CanEqual:
    case off, on
    case broken extends Lit, Terminated

  enum Said derives CanEqual:
    case broken

  object hand extends Actor:
    val press = action
    val smash = action
    val fix = action

  // The facet the others step through: pressing.
  trait Pressing extends Facet:
    type Outcome = Said
    var light: Lit
    var presses: UpTo[3]

    object pressing:

      object transitions:
        on(hand.press):
          light match
            case Lit.off | Lit.on if presses < 3 => effects.toggle()
            case _: Terminated                   => reject(Said.broken)
            case _                               => ()

      object effects:
        def toggle() =
          if light == Lit.on then light = Lit.off
          else light = Lit.on
          presses += 1

  // A facet that depends on Pressing's step.
  trait Breaking extends Facet:
    this: Pressing =>

    object breaking:

      object transitions:
        on(hand.smash):
          light match
            case Lit.on  => effects.smash()
            case Lit.off => effects.breakOff()
            case _       => ()
        on(hand.fix):
          light match
            case _: Terminated => effects.repair()
            case _             => ()

      object effects:
        val smashed = choice
        val toggled = choice
        def smash() =
          choose(
            smashed -> { light = Lit.broken },
            toggled -> { if presses < 3 then pressing.effects.toggle() }
          )
        def breakOff() =
          light = Lit.broken
        // sequential reads: the second statement reads the presses the first one wrote
        def repair() =
          presses = 0
          light = if presses == 0 then Lit.off else Lit.on

  object Faceted extends Machine, Pressing, Breaking:
    var light = Lit.off
    var presses: UpTo[3] = 0

  // The same Machine as one object, the facet order reversed in its extends clause.
  object Reordered extends Machine, Breaking, Pressing:
    var light = Lit.off
    var presses: UpTo[3] = 0

  object Whole extends Machine:
    type Outcome = Said
    var light = Lit.off
    var presses: UpTo[3] = 0

    object transitions:
      on(hand.press):
        light match
          case Lit.off | Lit.on if presses < 3 => effects.toggle()
          case _: Terminated                   => reject(Said.broken)
          case _                               => ()
      on(hand.smash):
        light match
          case Lit.on  => effects.smash()
          case Lit.off => effects.breakOff()
          case _       => ()
      on(hand.fix):
        light match
          case _: Terminated => effects.repair()
          case _             => ()

    object effects:
      val smashed = choice
      val toggled = choice
      def toggle() =
        if light == Lit.on then light = Lit.off
        else light = Lit.on
        presses += 1
      def smash() =
        choose(smashed -> { light = Lit.broken }, toggled -> { if presses < 3 then toggle() })
      def breakOff() =
        light = Lit.broken
      def repair() =
        presses = 0
        light = if presses == 0 then Lit.off else Lit.on

    object invariants:
      val brokenOnlyFromUse = if light == Lit.broken then check(presses <= 3)

  // Two facets that both bind the press.
  trait PressingAgain extends Facet:
    type Outcome = Said
    var light: Lit

    object pressingAgain:

      object transitions:
        on(hand.press):
          light match
            case _: Terminated => reject(Said.broken)
            case _             => ()

  object Twice extends Machine, Pressing, PressingAgain:
    var light = Lit.off
    var presses: UpTo[3] = 0

  // A bound the alias carries but Scala does not enforce: a fourth press would write 4.
  object Unbounded extends Machine:
    var presses: UpTo[3] = 3

    object transitions:
      on(hand.press):
        effects.press()

    object effects:
      def press() =
        presses += 1
