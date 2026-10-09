// `from` blocks the lifter refuses, each at its line (fn-139.3): a `from` in a `from`, an action the
// `from`'s declarer does not declare, and an import after the first statement.
package fixture.rulerejects

import framework.*
import fixture.grouped.{clock, cook, Kettle, Mode, Outcome, Plain}

object NestedFrom extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      import cook.*
      from(clock) {
        import clock.*
        on(cool)(always ~> Plain.effects.rest)
      }
      on(fill)(always ~> Plain.effects.heat)
    }

object Foreign extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      import cook.*
      on(fill)(always ~> Plain.effects.heat)
      on(clock.cool)(always ~> Plain.effects.rest)
    }

object ImportedTwice extends Machine[Kettle, Outcome, Nothing], Phased[Kettle, Mode](_.mode):
  val init = Kettle(Mode.idle, UpTo(0))
  override def end(s: State) = true
  object rules extends Rules:
    from(cook) {
      import cook.*
      on(fill)(always ~> Plain.effects.heat)
      import clock.*
      on(cool)(always ~> Plain.effects.rest)
    }
