// A fixture feature's file in the object forms (fn-126 R2, R4, R14, R17). Switch, the machine
// Dimmer that refines it, the derivation Kept and the composition Twins read as a feature file is
// read; the rest do not, at the lines the lint refuses: sections out of order; vocabulary outside
// `states`, a refinement's member outside `refinement`, a monitor in the wrong section, an effect
// and a monitor outside their sections, a Property over another machine object; a step function
// bound by hand, in a machine, in a rebind's source and as a machine's core rules; an object in a
// section, and a machine's section at the top level; and a case that reads a val its rules declare
// after it, which the rules' disjointness check calls while they initialize. Two objects' timers of
// one name, each read by a machine of its own, are named apart by their objects, and refused nothing.
package fixture.features.sectionorder

import umpire.*

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

final case class Pair(left: Lamp, right: Lamp)

val one = Limits(steps = 1, actions = 1, search = 8)

object hand extends Actor:
  val flip = action(this)
  val tap = action(this)

object clock:
  val tick = timer

// Its tick is named after its object, apart from clock's: no refusal (fn-126 decision 23).
object metronome:
  val tick = timer

// An object in an object of the signature, named after both, and a machine's section at the top.
object Holder:
  object spare:
    val bell = timer

object implements

object Switch extends Machine[Lamp, Outcome, Nothing], Phased[Lamp, Boolean](_.lit):
  val init = Lamp(false)
  override def end(s: Lamp) = true

  object states:
    def lit(s: Lamp) = s.lit

  object effects:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object monitors:
    val staysLit = sticky[Lamp, Outcome, Nothing](after => after.state.lit)

  object rules extends Rules:
    on(hand.flip)(when(false, true) ~> effects.flipped)
    on(clock.tick)(where(s => !states.lit(s)) ~> effects.kept)

  object properties:
    val turnsOn = property when hand.flip holds (after => after.state.lit)

  object queries:
    val flipped = scenario.actions(hand.flip)
    val asked = query verify properties.turnsOn in flipped limits one total 2

// A machine that refines Switch, its refinement in its own section.
object Dimmer extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object refinement extends Refinement(Switch):
    def toProduct(s: Lamp) = s
    val unobservable = List(clock.tick)

  object effects:
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object rules extends Rules:
    on(clock.tick)(always ~> effects.kept)

// A bare binding in a derivation keeps the rules' guards: no step function bound by hand.
object Kept extends Derived(Switch.rebind(hand.flip ~> Switch.effects.kept))

object Twins extends Composition[Pair](_.left -> Switch, _.right -> Kept):
  def end(s: Pair) = true

  object syncs extends Syncs:
    sync(_.left -> hand.flip, _.right -> hand.flip)

  object properties:
    val bothFlip = property holds (after => after.state.left == after.state.right)

object Backwards extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object effects:
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object states:
    def dark(s: Lamp) = !s.lit

  object properties:
    val stays = property when metronome.tick holds (after => !after.state.lit)

  object rules extends Rules:
    on(metronome.tick)(where(states.dark) ~> effects.kept)

object Misfiled extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  def stray(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))
  val watched = sticky[Lamp, Outcome, Nothing](after => !after.state.lit)
  def lit(s: Lamp) = s.lit
  def toProduct(s: Lamp) = s
  val strayBlock = effect(using machineOwner, Ok(Outcome.accepted))(reject(Outcome.accepted))

  object effects:
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object rules extends Rules:
    on(hand.tap)(always ~> effects.kept)

  object properties:
    val dim = sticky[Lamp, Outcome, Nothing](after => !after.state.lit)
    val borrowed = Switch.property when hand.flip holds (after => after.state.lit)

object HandBound extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object states:
    val bound = hand.tap ~> Switch.effects.kept

  object effects:
    object more:
      def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object rules extends Rules:
    on(hand.tap)(always ~> effects.more.kept)

object Guarded extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object effects:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))

  object rules extends Rules:
    on(hand.flip)(where(s => s.lit == armed) ~> effects.flipped)
    val armed = true

// A bare binding in the source a rebind derives from, which no rebind keeps: bound by hand.
object Escaped
    extends Derived(
      Switch.extend(hand.tap ~> Switch.effects.kept).rebind(hand.flip ~> Switch.effects.kept)
    )

// A machine whose rules are its step functions bound by hand: the core no Model writes.
object Cored extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true
  object rules extends Bindings(hand.tap ~> Switch.effects.kept)

object exports:
  val switchFile = irFile("switch")(Switch)
