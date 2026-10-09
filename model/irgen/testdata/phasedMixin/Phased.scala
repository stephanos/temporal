// fn-137.2: phasedRules/Phased.scala's lamp, its projection mixed into the machine,
// `Phased[Bulb, Light](_.light)`, on the same lines, and read by argument-less rules. Then derived
// objects, whose projection is their source's, and rules that read no projection at all.
package fixture.phased

import framework.*

enum Light derives Finite:
  case off, on, broken

final case class Bulb(light: Light) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

object hand extends Actor:
  val press = action(this)
  val drop = action(this)

object Switch extends Machine[Bulb, Outcome, Nothing], Phased[Bulb, Light](_.light):
  val init = Bulb(Light.off)
  override def end(s: State) = s.light == Light.broken
  object states:
    def dark(l: Light) = l != Light.on
  object effects:
    def light(s: Bulb) = enter[Bulb, Outcome, Nothing](s.copy(light = Light.on))
    def dark(s: Bulb) = enter[Bulb, Outcome, Nothing](s.copy(light = Light.off))
    def keep(s: Bulb) = stay[Bulb, Outcome, Nothing](s)
  object rules extends Rules:
    on(hand.press) {
      when(Light.on) ~> effects.dark
      when(states.dark).where(_.light != Light.broken) ~> effects.light
    }
    on(hand.drop)(when(Light.off, Light.on) ~> effects.keep)

// A drop that lights the lamp, and that machine unmonitored: both read Switch's projection.
object Stiff extends Derived(Switch.rebind(hand.drop ~> Switch.effects.light))
object Stiffer extends Derived(Stiff.unmonitored)

final case class Pair(left: Bulb, right: Bulb)

// Two lamps whose presses step together, phased by the left one, and the same with a stiff right
// one, which reads the twins' projection.
object Twins
    extends Composition[Pair](_.left -> Switch, _.right -> Switch),
      Phased[Pair, Light](_.left.light):
  override def end(s: Pair) = Switch.end(s.left)
  object syncs extends Syncs:
    sync(_.left -> hand.press, _.right -> hand.press)

object Lopsided extends Composition(Twins.withMember(_.right -> Stiff))

// Rules whose phase type is written out, so that `in` compiles, on a machine that is not `Phased`:
// the lifter refuses its phases, which no projection reads.
object Unprojected extends Machine[Bulb, Outcome, Nothing]:
  val init = Bulb(Light.off)
  def end(s: State) = true
  object rules extends Rules[Bulb, Outcome, Nothing, Light]:
    on(hand.press)(when(Light.off) ~> Switch.effects.light)
