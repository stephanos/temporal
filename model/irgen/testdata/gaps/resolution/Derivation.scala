// fn-167.3: machines derived from `Plain` by each derivation operation and a chain, with gaps the lift
// keeps where an operation changes or may change their findings, and refuses where the finding is
// provably the source's.
package fixture.gapresolution

import framework.*

val calm = assume
val other = assume

def sawFlag(seen: Boolean, before: Job, after: Step[Job, Outcome, Fact]): Boolean =
  seen || after.state.flag

val flaggedOnce = monitor[Job, Outcome, Fact, Boolean](false)(sawFlag)(seen => !seen)

// The product a refinement may be replaced by: Worker's types, its own rules.
object Shadow
    extends Machine[Job, Outcome, Fact],
      Phased[Job, Stage](_.stage, (s, p) => s.copy(stage = p)):
  val init = Job(Stage.idle, false)
  object effects:
    def move(s: Job, m: Mode) = stay[Job, Outcome, Fact](s)
  object rules extends Bindings(caller.move ~> effects.move)

def shadowed(s: Job): Job = s

def reboundMove(s: Job, m: Mode) =
  if s.stage == Stage.idle then List(Step(Outcome.refused, s)) else disabled

def plainFinish(s: Job) =
  if s.stage == Stage.busy then enter[Job, Outcome, Fact](s.copy(stage = Stage.done)) else disabled

// The source of a chain: a finding of it is Plain's (see `refused.Assumed`).
object PlainAssumed extends Derived(Plain.assuming(other))

@gap(unconstrained(caller.move(Mode.slow)))(because = "unmonitored: monitors dropped")
@gap(stuck(Job(Stage.busy, true)))(because = "unmonitored: refinement dropped")
object PlainUnmonitored extends Derived(Plain.unmonitored)

@gap(neverEnabled(caller.move(Mode.fast)))(because = "rebound: outcome changed")
object PlainRebound extends Derived(Plain.rebind(caller.move ~> reboundMove))

@gap(neverEnabled(caller.move(Mode.fast)))(because = "guarded: guard changed")
object PlainGuarded
    extends Derived(Plain.rebind(on(caller.move)(where(_.flag) ~> Shadow.effects.move)))

@gap(neverEnabled(caller.move(Mode.fast)))(because = "restricted")
object PlainRestricted extends Derived(Plain.restrict(caller.move))

@gap(neverEnabled(caller.finish), neverEnabled(caller.move(Mode.slow)))(because = "extended")
object PlainExtended extends Derived(Plain.extend(caller.finish ~> plainFinish))

@gap(unread)(because = "refinement replaced")
object PlainRefined extends Derived(Plain.refining(Shadow)(shadowed))

// Refused: each finding is the source's, unchanged.
object refused:
  @gap(neverEnabled(caller.move(Mode.slow)))(because = "assumed again: inherited")
  object Assumed extends Derived(Plain.assuming(other))

  @gap(neverEnabled(caller.move(Mode.fast)))(because = "unmonitored: rows kept")
  object Unmonitored extends Derived(Plain.unmonitored)

  @gap(unproduced(Outcome.refused))(because = "chained: inherited through two")
  object Chained extends Derived(PlainAssumed.unmonitored)
