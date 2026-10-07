// fn-137.2: a lamp whose rules name their projection, `Rules(_.light)`. phasedMixin/Phased.scala
// is the same lamp with its projection mixed into the machine, `Phased[Bulb, Light](_.light)`, on
// the same lines, and the lifter's tests hold the two lifts equal byte for byte (expected.json).
package fixture.phased

import umpire.*

enum Light derives Finite:
  case off, on, broken

final case class Bulb(light: Light) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

object hand extends Actor:
  val press = action(this)
  val drop = action(this)

object Switch extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Light.off)
  def end(s: State) = s.light == Light.broken
  object states:
    def dark(l: Light) = l != Light.on
  object effects:
    def light(s: Bulb) = enter[Bulb, Outcome, Nothing](s.copy(light = Light.on))
    def dark(s: Bulb) = enter[Bulb, Outcome, Nothing](s.copy(light = Light.off))
    def keep(s: Bulb) = stay[Bulb, Outcome, Nothing](s)
  object rules extends Rules(_.light):
    on(hand.press) {
      in(Light.on) ~> effects.dark
      in(states.dark).where(_.light != Light.broken) ~> effects.light
    }
    on(hand.drop)(in(Light.off, Light.on) ~> effects.keep)
