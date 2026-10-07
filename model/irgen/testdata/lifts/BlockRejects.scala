// `is { }` blocks the lifter refuses, each at its line (fn-135.2): a block nested in another, a block
// declared by a val of a machine's object rather than of one of its sections, a block a def's body
// returns, and a block reading a field through an accessor of any shape but the fixed one.
package fixture.blockrejects

import umpire.*
import fixture.blocks.{phase, phase_=, retried, retried_=, user, Fact, Job, Outcome, Phase, given}

// Reads the phase it compares rather than one field.
def idle(using v: View[Job]): Boolean = v.get(_.phase) == Phase.idle

object NestedIs extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object states:
    val nested = is(phase == Phase.done && is(phase == Phase.idle)(Job(Phase.idle, false)))
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules:
    on(user.start)(where(states.nested) ~> effects.keep)

object HeaderIs extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  val done = is(phase == Phase.done)
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules:
    on(user.start)(where(done) ~> effects.keep)

object DefIs extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object states:
    def done = is(phase == Phase.done)
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules:
    on(user.start)(where(states.done) ~> effects.keep)

object ShapedAccessor extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object states:
    val waiting = is(idle)
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Fact](s)
  object rules extends Rules:
    on(user.start)(where(states.waiting) ~> effects.keep)

// `effect { }` blocks the lifter refuses, each at its line (fn-135.3): a statement in a branch, a
// rejecting block that also assigns, a field assigned twice, a field read after its assignment, a
// local val, a bare expression, another call, a nested block, an assignment by a setter of any shape
// but the fixed one, and a block written in a rule rather than declared by a section's val.

// Reads the phase, but assigns `done` whatever it is given.
def stage(using v: View[Job]): Phase = v.get(_.phase)
def stage_=(p: Phase)(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(phase = Phase.done))

object BranchedEffect extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      if retried then phase = Phase.done
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object RejectingEffect extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val refuse = effect {
      phase = Phase.done
      reject(Outcome.refused)
    }
  object rules extends Rules:
    on(user.refuse)(always ~> effects.refuse)

object TwiceAssigned extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      phase = Phase.running
      phase = Phase.done
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object ReadAfterAssigned extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      phase = Phase.done
      retried = phase == Phase.running
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object LocalVal extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      val next = Phase.done
      phase = next
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object BareExpression extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      phase == Phase.running
      phase = Phase.done
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object OtherCall extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      require(retried)
      phase = Phase.done
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object NestedEffect extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect {
      retried = effect(record(Fact.finished))(Job(Phase.idle, false)).isEmpty
    }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object ShapedSetter extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object effects:
    val finish = effect { stage = Phase.running }
  object rules extends Rules:
    on(user.finish)(always ~> effects.finish)

object RuleEffect extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, false)
  def end(s: State) = true
  object rules extends Rules:
    on(user.finish)(always ~> effect { phase = Phase.done })
