// Rules grouped by who acts (fn-139.3): `Grouped` writes its rules in `from` blocks, each opened by
// its declarer's import, with `when` cases and an `on` of two classes; `Plain` writes the same rules
// with `on` blocks and `in` cases, one block per class. The lifter's tests lift both and require one
// machine of the two, but for names and positions.
package fixture.grouped

import framework.*

enum Mode derives Finite:
  case idle, busy, halted

final case class Kettle(mode: Mode, boils: UpTo[2]) derives Finite

enum Outcome derives Finite:
  case accepted, refused

given Ok[Outcome] = Ok(Outcome.accepted)

enum Dial derives Finite:
  case low, high

object cook extends Actor:
  val fill = action(this)
  val set = action(this).input[Dial]("dial")

object clock:
  val cool = timer

// The modes both machines name a set of, so one function serves both.
object Modes:
  def working(m: Mode) = m != Mode.halted

object Grouped extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = s.mode == Mode.halted

  object effects:
    def heat(s: Kettle) = enter[Kettle, Outcome, Nothing](s.copy(mode = Mode.busy))
    def rest(s: Kettle) = enter[Kettle, Outcome, Nothing](s.copy(mode = Mode.idle))
    def halt(s: Kettle) = enter[Kettle, Outcome, Nothing](s.copy(mode = Mode.halted))
    def refuse(s: Kettle) = List(Step[Kettle, Outcome, Nothing](Outcome.refused, s))

  object rules extends Rules:
    from(cook) {
      import cook.*
      on(fill) {
        when(Mode.idle) ~> effects.heat
        when(Mode.busy).where(_.boils == 2) ~> effects.halt
      }
      on(set(Dial.low), set(Dial.high)) {
        when(Modes.working) ~> effects.refuse
      }
    }
    from(clock) {
      import clock.*
      on(cool)(when(Mode.busy, Mode.halted) ~> effects.rest)
    }

object Plain extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = s.mode == Mode.halted

  object effects:
    def heat(s: Kettle) = enter[Kettle, Outcome, Nothing](s.copy(mode = Mode.busy))
    def rest(s: Kettle) = enter[Kettle, Outcome, Nothing](s.copy(mode = Mode.idle))
    def halt(s: Kettle) = enter[Kettle, Outcome, Nothing](s.copy(mode = Mode.halted))
    def refuse(s: Kettle) = List(Step[Kettle, Outcome, Nothing](Outcome.refused, s))

  object rules extends Rules:
    on(cook.fill) {
      when(Mode.idle) ~> effects.heat
      when(Mode.busy).where(_.boils == 2) ~> effects.halt
    }
    on(cook.set(Dial.low))(when(Modes.working) ~> effects.refuse)
    on(cook.set(Dial.high))(when(Modes.working) ~> effects.refuse)
    on(clock.cool)(when(Mode.busy, Mode.halted) ~> effects.rest)
