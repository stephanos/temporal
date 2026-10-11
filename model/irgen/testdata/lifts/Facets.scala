// Facets (fn-161.1): `Faceted` is assembled from two facets, `Pressing` and `Breaking`, which steps
// through Pressing's toggle; `Whole` is the same machine written as one object. The lifter's tests
// lift both and require one machine of the two, but for names, positions and the names of the
// effects, which each facet's object qualifies.
package fixture.facets

import framework.*

enum Lit(val status: Note) extends Recorded[Note] derives Finite:
  case off extends Lit(Note.dark)
  case on extends Lit(Note.lit)
  case broken extends Lit(Note.gone), Terminated

enum Note derives Finite:
  case dark, lit, gone

final case class Lamp(light: Lit, presses: UpTo[1]) derives Finite

enum Said derives Finite:
  case ok

given Ok[Said] = Ok(Said.ok)

object hand extends Actor:
  val press = action(this)
  val smash = action(this)

trait Pressing extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:
  object pressing:
    object effects:
      def toggle(s: Lamp) = enter[Lamp, Said, Note](
        s.copy(light = if s.light.in(Lit.on) then Lit.off else Lit.on, presses = UpTo[1](1))
      )

    object rules extends Rules:
      on(hand.press) {
        when(Lit.off, Lit.on) ~> effects.toggle
      }

trait Breaking extends Facet[Lamp, Said, Note], Phased[Lamp, Lit]:
  this: Pressing =>
  object breaking:
    object effects:
      val smashed = choice
      val toggled = choice
      def smash(s: Lamp) =
        choose(smashed -> enters(Lit.broken)(s), toggled -> pressing.effects.toggle(s))

    object rules extends Rules:
      on(hand.smash) {
        when(Lit.on) ~> effects.smash
        when(Lit.off) ~> enters(Lit.broken)
      }

object Faceted
    extends Machine[Lamp, Said, Note],
      Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)),
      Pressing,
      Breaking:
  val init = Lamp(Lit.off, UpTo(0))

  object rules extends Facets(pressing.rules, breaking.rules)

object Whole extends Machine[Lamp, Said, Note], Phased[Lamp, Lit](_.light, (s, p) => s.copy(light = p)):
  val init = Lamp(Lit.off, UpTo(0))

  object effects:
    val smashed = choice
    val toggled = choice
    def toggle(s: Lamp) = enter[Lamp, Said, Note](
      s.copy(light = if s.light.in(Lit.on) then Lit.off else Lit.on, presses = UpTo[1](1))
    )
    def smash(s: Lamp) = choose(smashed -> enters(Lit.broken)(s), toggled -> toggle(s))

  object rules extends Rules:
    on(hand.press) {
      when(Lit.off, Lit.on) ~> effects.toggle
    }
    on(hand.smash) {
      when(Lit.on) ~> effects.smash
      when(Lit.off) ~> enters(Lit.broken)
    }
