// Machine objects the build refuses (fn-126 R15, R16), each at its line: a machine object is its
// init, its end and its rules, which it must declare, and a derived machine adds no rules of its
// own. Rules that name phases of no projection are refused beside the crossed types
// (crossed/Rules.scala), as the build reports them in an earlier phase.
package fixture.objectforms

import framework.*

enum Glow derives Finite:
  case dim, bright

final case class Bulb(glow: Glow) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

object hand extends Actor:
  val push = action(this)

object Lit extends Machine[Bulb, Outcome, Nothing], Phased[Bulb, Glow](_.glow):
  val init = Bulb(Glow.dim)
  override def end(s: Bulb) = true
  object effects:
    def brighten(s: Bulb) = enter[Bulb, Outcome, Nothing](Bulb(Glow.bright))
  object rules extends Rules:
    on(hand.push)(when(Glow.dim) ~> effects.brighten)

// No init.
object Unstarted extends Machine[Bulb, Outcome, Nothing]:
  def end(s: Bulb) = true
  object rules extends Rules:
    on(hand.push)(always ~> Lit.effects.brighten)

// No end.
object Endless extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  object rules extends Rules:
    on(hand.push)(always ~> Lit.effects.brighten)

// No rules.
object Ruleless extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Glow.dim)
  def end(s: Bulb) = true

// A derived machine with rules of its own: its rules are its derivation's.
object Overruled extends Derived(Lit.restrict(hand.push)):
  object rules extends Rules:
    on(hand.push)(always ~> Lit.effects.brighten)
