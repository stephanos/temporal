// Status assignments the lifter refuses, each at its line (fn-135.5): a block that records the status
// its assignment already records, an assignment of a phase that names no case, so its status is
// not known, and an assignment by a setter of the plain shape to a field whose values declare their
// status, which would hand the draft no status to record.
package fixture.statusfactrejects

import umpire.*
import fixture.statusfacts.{phase, phase_=, user, Fact, Job, Outcome, Phase, given}

// Assigns the phase without handing the draft the phase it assigns.
def stage(using v: View[Job]): Phase = v.get(_.phase)
def stage_=(p: Phase)(using d: Draft[Job, ?, ?]): Unit = d.set(_.copy(phase = p))

object ExplicitStatus extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, UpTo(0))
  override def end(s: State) = true
  object effects:
    val pause = effect {
      phase = Phase.paused
      record(Fact.attempted, Fact.statusPaused)
    }
  object rules extends Rules:
    on(user.pause)(always ~> effects.pause)

object ComputedStatus extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, UpTo(0))
  override def end(s: State) = true
  object effects:
    val pause = effect {
      phase = if phase == Phase.running then Phase.paused else Phase.running
    }
  object rules extends Rules:
    on(user.pause)(always ~> effects.pause)

object PlainStatusSetter extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, UpTo(0))
  override def end(s: State) = true
  object effects:
    val pause = effect { stage = Phase.paused }
  object rules extends Rules:
    on(user.pause)(always ~> effects.pause)
