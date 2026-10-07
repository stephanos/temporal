// Status facts an `effect { }` block derives from the phase it assigns, beside the facts the method
// form writes (fn-135.5): `Derived` declares its effects as blocks, `Written` as defs that record
// each status by hand. The cases are an assignment that changes the phase, one that assigns the
// phase the step starts in (it still records), one that assigns the other field only (no status
// fact), one with an explicit fact the phase does not determine (that fact first, then the status)
// and a rejection (no fact). The lifter's tests lift both and require one IR of the two, but for
// names and positions, and the phase enum to lift as an enum with no fields.
package fixture.statusfacts

import umpire.*

enum Fact derives Finite:
  case statusIdle, statusRunning, statusPaused, attempted

// Each case declares the status it is recorded as.
enum Phase(val status: Fact) extends Recorded[Fact] derives Finite:
  case idle extends Phase(Fact.statusIdle)
  case running extends Phase(Fact.statusRunning)
  case paused extends Phase(Fact.statusPaused)

final case class Job(phase: Phase, attempts: UpTo[2]) derives Finite

enum Outcome derives Finite:
  case accepted, refused

given Ok[Outcome] = Ok(Outcome.accepted)

// The accessors the blocks read and assign the fields of a `Job` by: the phase's setter hands the
// draft the phase it assigns, whose status the step records.
def phase(using v: View[Job]): Phase = v.get(_.phase)
def phase_=(p: Phase)(using d: Draft[Job, ?, Fact]): Unit = d.set(p)(_.copy(phase = p))
def attempts(using v: View[Job]): UpTo[2] = v.get(_.attempts)
def attempts_=(a: UpTo[2])(using d: Draft[Job, ?, Fact]): Unit = d.set(_.copy(attempts = a))

object user extends Actor:
  val start = action(this)
  val resume = action(this)
  val retry = action(this)
  val pause = action(this)
  val refuse = action(this)

object Derived extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, UpTo(0))
  def end(s: State) = s.phase == Phase.paused

  object effects:
    val start = effect { phase = Phase.running }
    val resume = effect { phase = Phase.running }
    val retry = effect { attempts = UpTo(1) }
    val pause = effect {
      phase = Phase.paused
      record(Fact.attempted)
    }
    val refuse = effect(reject(Outcome.refused))

  object rules extends Rules:
    on(user.start)(in(Phase.idle) ~> effects.start)
    on(user.resume)(in(Phase.running) ~> effects.resume)
    on(user.retry)(in(Phase.running) ~> effects.retry)
    on(user.pause)(in(Phase.running) ~> effects.pause)
    on(user.refuse)(in(Phase.paused) ~> effects.refuse)

object Written extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.idle, UpTo(0))
  def end(s: State) = s.phase == Phase.paused

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.statusRunning)
    def resume(s: Job) = enter(s.copy(phase = Phase.running), Fact.statusRunning)
    def retry(s: Job) = enter(s.copy(attempts = UpTo(1)))
    def pause(s: Job) = enter(s.copy(phase = Phase.paused), Fact.attempted, Fact.statusPaused)
    def refuse(s: Job) = reject(Outcome.refused, s)

  object rules extends Rules:
    on(user.start)(in(Phase.idle) ~> effects.start)
    on(user.resume)(in(Phase.running) ~> effects.resume)
    on(user.retry)(in(Phase.running) ~> effects.retry)
    on(user.pause)(in(Phase.running) ~> effects.pause)
    on(user.refuse)(in(Phase.paused) ~> effects.refuse)
