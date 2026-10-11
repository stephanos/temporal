// Rules the lifter refuses, each at its line: a `from` in a `from` and an action the `from`'s
// declarer does not declare (fn-139.3); an import in a `from`, and an action written bare inside a
// `from`, outside one, in a class and in `disabled` (fn-159.1); a phase and a phase set written bare
// in `when` (fn-159.3); and `enters` through a projection that declares no writer (fn-159.7).
package fixture.rulerejects

import framework.*
import fixture.grouped.{clock, cook, Dial, Kettle, Mode, Modes, Outcome, Plain}
import fixture.grouped.given

object NestedFrom extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      from(clock) {
        on(clock.cool)(always ~> Plain.effects.rest)
      }
      on(cook.fill)(always ~> Plain.effects.heat)
    }

object Foreign extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      on(cook.fill)(always ~> Plain.effects.heat)
      on(clock.cool)(always ~> Plain.effects.rest)
    }

object ImportedInFrom extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      import cook.*
      on(fill)(always ~> Plain.effects.heat)
    }

object BareInFrom extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  import cook.fill
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      on(fill)(always ~> Plain.effects.heat)
    }

object BareImported extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  import clock.cool
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    on(cook.fill, cool)(always ~> Plain.effects.rest)

object BareClass extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  import cook.set
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    on(set(Dial.low))(always ~> Plain.effects.refuse)

object BareDisabled extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  import clock.cool
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    disabled(cool)

object BarePhase extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  import Mode.idle
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    on(cook.fill)(when(idle) ~> Plain.effects.heat)

object BareSet extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  import Modes.working
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    on(cook.fill)(when(working) ~> Plain.effects.heat)

// `enters` in the rules of a machine whose projection declares no writer (fn-159.7).
object Unwritten extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    on(cook.fill)(always ~> enters(Mode.busy))

// Two facets that both bind the fill (fn-161.1).
trait Filling extends Facet[Kettle, Outcome, Nothing], Phased[Kettle, Mode]:
  object filling:
    object rules extends Rules:
      on(cook.fill)(when(Mode.idle) ~> Plain.effects.heat)

trait Refilling extends Facet[Kettle, Outcome, Nothing], Phased[Kettle, Mode]:
  object refilling:
    object rules extends Rules:
      on(cook.fill)(when(Mode.busy) ~> Plain.effects.heat)

object TwiceFaceted
    extends Machine[Kettle, Outcome, Nothing],
      Phased[Kettle, Mode](_.mode),
      Filling,
      Refilling:
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Facets(filling.rules, refilling.rules)
