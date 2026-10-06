// `is { }` blocks beside the defs they stand for (fn-135.2): `Blocked` declares the predicates of
// its `states` section as `is { }` vals, `Defined` as defs of the state, and each uses them alike in
// a rule's condition, a capability field, `end` and a claim. The lifter's tests lift both and
// require one IR of the two, but for names and positions. Query IDs are the family's, so each
// machine's Query is named after it.
package fixture.blocks

import umpire.*

enum Phase derives Finite:
  case idle, running, paused, done

final case class Job(phase: Phase, retried: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

enum Fact derives Finite:
  case started, paused, finished

// The accessors an `is { }` block reads the fields of a `Job` by.
def phase(using v: View[Job]): Phase = v.get(_.phase)
def retried(using v: View[Job]): Boolean = v.get(_.retried)

object user extends Actor:
  val start = action(this)
  val pause = action(this)
  val finish = action(this)

// A job that pauses: `paused` says where it is.
final case class Pausable[S](paused: S => Boolean) extends CapabilityOf[S, Nothing, Nothing]

object Pausable extends CapabilityKind:
  // A step from a paused job keeps it paused.
  def pausedStays[S](m: Declares[S])(paused: S => Boolean): Property[S] = m.property.stays(paused)

val two = Limits(steps = 2, actions = 2, search = 64)

object Blocked extends Machine[Job, Outcome, Fact]:
  val init = Job(Phase.idle, false)
  def end(s: State) = states.over(s)

  object states:
    val paused = is(phase == Phase.paused)
    val busy = is(phase.in(Phase.running, Phase.paused))
    val over = is(phase == Phase.done && !retried)

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.started)
    def pause(s: Job) = enter(s.copy(phase = Phase.paused), Fact.paused)
    def finish(s: Job) = enter(s.copy(phase = Phase.done), Fact.finished)

  object rules extends Rules(_.phase):
    on(user.start)(where(states.paused) ~> effects.start)
    on(user.pause)(in(Phase.running).where(states.busy) ~> effects.pause)
    on(user.finish)(in(Phase.running) ~> effects.finish)

  object properties:
    val pausedKept = property.stays(states.paused)

  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(paused = states.paused)

  object queries:
    capabilities.bound(two)
    val starting = scenario.actions(user.start)
    val blockedKept = query verify properties.pausedKept in starting limits two

object Defined extends Machine[Job, Outcome, Fact]:
  val init = Job(Phase.idle, false)
  def end(s: State) = states.over(s)

  object states:
    def paused(s: Job) = s.phase == Phase.paused
    def busy(s: Job) = s.phase.in(Phase.running, Phase.paused)
    def over(s: Job) = s.phase == Phase.done && !s.retried

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.started)
    def pause(s: Job) = enter(s.copy(phase = Phase.paused), Fact.paused)
    def finish(s: Job) = enter(s.copy(phase = Phase.done), Fact.finished)

  object rules extends Rules(_.phase):
    on(user.start)(where(states.paused) ~> effects.start)
    on(user.pause)(in(Phase.running).where(states.busy) ~> effects.pause)
    on(user.finish)(in(Phase.running) ~> effects.finish)

  object properties:
    val pausedKept = property.stays(states.paused)

  object capabilities extends Capabilities:
    val pausable: Capability = Pausable(paused = states.paused)

  object queries:
    capabilities.bound(two)
    val starting = scenario.actions(user.start)
    val definedKept = query verify properties.pausedKept in starting limits two
