// `is { }` blocks the lifter refuses, each at its line (fn-135.2): a block nested in another, a block
// declared by a val of a machine's object rather than of one of its sections, a block a def's body
// returns, and a block reading a field through an accessor of any shape but the fixed one.
package fixture.blockrejects

import umpire.*
import fixture.blocks.{phase, user, Fact, Job, Outcome, Phase, given}

// Reads the phase it compares rather than one field.
def idle(using v: View[Job]): Boolean = v.get(_.phase) == Phase.idle

object NestedIs extends Machine[Job, Outcome, Fact]:
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object states:
    val nested = is(phase == Phase.done && is(phase == Phase.idle)(Job(Phase.idle, false)))
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules(_.phase):
    on(user.start)(where(states.nested) ~> effects.keep)

object HeaderIs extends Machine[Job, Outcome, Fact]:
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  val done = is(phase == Phase.done)
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules(_.phase):
    on(user.start)(where(done) ~> effects.keep)

object DefIs extends Machine[Job, Outcome, Fact]:
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object states:
    def done = is(phase == Phase.done)
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules(_.phase):
    on(user.start)(where(states.done) ~> effects.keep)

object ShapedAccessor extends Machine[Job, Outcome, Fact]:
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object states:
    val waiting = is(idle)
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules(_.phase):
    on(user.start)(where(states.waiting) ~> effects.keep)
