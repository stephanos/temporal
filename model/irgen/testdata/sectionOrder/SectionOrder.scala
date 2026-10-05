// A fixture feature's file in the object forms (fn-126 R2, R4, R14, R17). Switch, the machine
// Dimmer that refines it, the derivation Kept and the composition Twins read as a feature file is
// read; the rest do not, at the lines the lint refuses: sections out of order; vocabulary outside
// `states`, a refinement's member outside `refinement`, a monitor in the wrong section, an effect
// and a monitor outside their sections, a Property over another machine object; a step function
// bound by hand, in a machine and in a rebind's source; a section in a section, one in an object of
// the signature and a machine's section at the top level; two sections' timers that would share an
// ID, each read by a machine of its own; and a guard that reads a val its rules declare after it,
// which the rules' disjointness check calls while they initialize.
package fixture.features.sectionorder

import umpire.*

given Family = Family("fixture.sectionorder")

final case class Lamp(lit: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

final case class Pair(left: Lamp, right: Lamp)

val one = Limits(steps = 1, actions = 1, search = 8)

object hand extends Actor:
  val flip = action(this)
  val tap = action(this)

object clock extends Section:
  val tick = timer

// Its tick takes the ID clock's takes, the file's: a section is transparent to Definition IDs.
object metronome extends Section:
  val tick = timer

// A section in an object of the signature, and a machine's section at the top level.
object Holder:
  object spare extends Section:
    val bell = timer

object implements extends Section

object Switch extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object states extends Section:
    def lit(s: Lamp) = s.lit

  object effects extends Section:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object monitors extends Section:
    val staysLit = sticky[Lamp, Outcome, Nothing](after => after.state.lit)

  object rules extends Rules(_.lit):
    in(false, true)(hand.flip ~> effects.flipped)
    when(s => !states.lit(s))(clock.tick ~> effects.kept)

  object properties extends Section:
    val turnsOn = property when hand.flip holds (after => after.state.lit)

  object queries extends Section:
    val flipped = scenario.actions(hand.flip)
    val asked = query verify properties.turnsOn in flipped limits one total 2

/** A machine that refines Switch, its refinement in its own section. */
object Dimmer extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object refinement extends Refinement(Switch):
    def toProduct(s: Lamp) = s
    val unobservable = List(clock.tick)

  object effects extends Section:
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object rules extends Rules:
    when(_ => true)(clock.tick ~> effects.kept)

/** A bare binding in a derivation keeps the rules' guards: no step function bound by hand. */
object Kept extends Derived(Switch.rebind(hand.flip ~> Switch.effects.kept))

object Twins extends Composition[Pair](_.left -> Switch, _.right -> Kept):
  def end(s: Pair) = true

  object syncs extends Syncs:
    sync(_.left -> hand.flip, _.right -> hand.flip)

  object properties extends Section:
    val bothFlip = property holds (after => after.state.left == after.state.right)

object Backwards extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object effects extends Section:
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object states extends Section:
    def dark(s: Lamp) = !s.lit

  object properties extends Section:
    val stays = property when metronome.tick holds (after => !after.state.lit)

  object rules extends Rules:
    when(states.dark)(metronome.tick ~> effects.kept)

object Misfiled extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  def stray(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))
  val watched = sticky[Lamp, Outcome, Nothing](after => !after.state.lit)
  def lit(s: Lamp) = s.lit
  def toProduct(s: Lamp) = s

  object effects extends Section:
    def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object rules extends Rules:
    when(_ => true)(hand.tap ~> effects.kept)

  object properties extends Section:
    val dim = sticky[Lamp, Outcome, Nothing](after => !after.state.lit)
    val borrowed = Switch.property when hand.flip holds (after => after.state.lit)

object HandBound extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object states extends Section:
    val bound = hand.tap ~> Switch.effects.kept

  object effects extends Section:
    object more extends Section:
      def kept(s: Lamp): List[Step[Lamp, Outcome, Nothing]] = List(Step(Outcome.accepted, s))

  object rules extends Rules:
    when(_ => true)(hand.tap ~> effects.more.kept)

object Guarded extends Machine[Lamp, Outcome, Nothing]:
  val init = Lamp(false)
  def end(s: Lamp) = true

  object effects extends Section:
    def flipped(s: Lamp): List[Step[Lamp, Outcome, Nothing]] =
      List(Step(Outcome.accepted, s.copy(lit = !s.lit)))

  object rules extends Rules:
    when(s => s.lit == armed)(hand.flip ~> effects.flipped)
    val armed = true

/** A bare binding in the source a rebind derives from, which no rebind keeps: bound by hand. */
object Escaped
    extends Derived(
      Switch.extend(hand.tap ~> Switch.effects.kept).rebind(hand.flip ~> Switch.effects.kept)
    )

object exports:
  val switchFile = irFile("switch")(Switch)
