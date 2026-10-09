// Role tests beside the case sets they stand for (fn-136.1): `Roled` tests its phases against
// roles, with `isInstanceOf` and with type patterns, and `Listed` names the same cases by hand,
// with `when(...)` and alternatives of case literals. The lifter's tests lift both and require one IR
// of the two, but for names and positions: an inherited role (`Retrying` is `Waiting` and `Live`),
// a Model's own role and its framework role, a one-case enum, and a binding type pattern.
package fixture.roles

import framework.*

// A Model's own role, through the framework role it extends.
trait Expired extends TimedOut

// A trait that extends no role, taken by a phase beside its role.
trait Audited

enum Phase derives Finite:
  case unstarted
  case queued extends Phase, Waiting
  // Waiting again beside Retrying, which implies it: no conflict.
  case backingOff extends Phase, Retrying, Waiting
  case running extends Phase, Held
  case paused extends Phase, Suspended
  case done extends Phase, Succeeded, Audited
  case failed extends Phase, Failed
  case expired extends Phase, Expired

// The phases of `Phase` with no roles, whose IR type is the same but for its name.
enum Bare derives Finite:
  case unstarted, queued, backingOff, running, paused, done, failed, expired

// A phase enum of one case, whose test against its role holds of every value.
enum Solo derives Finite:
  case over extends Solo, Closed

final case class Job(phase: Phase, solo: Solo, bare: Bare) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

enum Fact derives Finite:
  case started

object user extends Actor:
  val start = action(this)
  val stop = action(this)

object Roled extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.unstarted, Solo.over, Bare.unstarted)
  override def end(s: State) = states.closed(s.phase) && states.over(s) || states.expiring(s.phase)

  // The role test as the lifter reads it, which the Models' lint leaves to the fixtures.
  // scalafix:off DisableSyntax.isInstanceOf
  object states:
    def closed(p: Phase) = p.isInstanceOf[Closed]
    def waiting(p: Phase) = p.isInstanceOf[Waiting]
    def live(p: Phase) = p.isInstanceOf[Live]
    def expiring(p: Phase) = p.isInstanceOf[Expired] || p.isInstanceOf[TimedOut]
    def over(s: Job) = s.solo.isInstanceOf[Closed]
    def settled(p: Phase) = p match
      case _: Closed => true
      case _         => false
    def stopping(s: Job) = s.phase match
      case q: Held => q != Phase.paused
      case _       => false
  // scalafix:on DisableSyntax.isInstanceOf

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.started)
    def stop(s: Job) = enter(s.copy(phase = Phase.done))

  object rules extends Rules:
    on(user.start)(when(states.waiting) ~> effects.start)
    on(user.stop) {
      when(states.live).where(states.stopping) ~> effects.stop
      when(states.settled) ~> effects.stop
    }

object Listed extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.unstarted, Solo.over, Bare.unstarted)
  override def end(s: State) = states.closed(s.phase) && states.over(s) || states.expiring(s.phase)

  object states:
    def closed(p: Phase) = p.in(Phase.done, Phase.failed, Phase.expired)
    def waiting(p: Phase) = p.in(Phase.queued, Phase.backingOff)
    def live(p: Phase) = p.in(Phase.queued, Phase.backingOff, Phase.running, Phase.paused)
    def expiring(p: Phase) = p.in(Phase.expired) || p.in(Phase.expired)
    def over(s: Job) = s.solo.in(Solo.over)
    def settled(p: Phase) = p match
      case Phase.done | Phase.failed | Phase.expired => true
      case _                                         => false
    def stopping(s: Job) = s.phase match
      case q @ Phase.running => q != Phase.paused
      case _                 => false

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.started)
    def stop(s: Job) = enter(s.copy(phase = Phase.done))

  object rules extends Rules:
    on(user.start)(when(states.waiting) ~> effects.start)
    on(user.stop) {
      when(states.live).where(states.stopping) ~> effects.stop
      when(states.settled) ~> effects.stop
    }

// fn-136.4: the short spellings, `when[R]` in rules and `p.in[R]` elsewhere, beside the cases
// `Written` names by hand. The lifter's tests require one IR of the two, but for names and
// positions.
object Shortened extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.unstarted, Solo.over, Bare.unstarted)
  override def end(s: State) = states.closed(s.phase) && states.over(s) || states.expiring(s.phase)

  object states:
    def closed(p: Phase) = p.in[Closed]
    def expiring(p: Phase) = p.in[Expired] || p.in[TimedOut]
    def over(s: Job) = s.solo.in[Closed]
    def stopping(s: Job) = s.phase.in[Held] && s.phase != Phase.paused

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.started)
    def stop(s: Job) = enter(s.copy(phase = Phase.done))

  object rules extends Rules:
    on(user.start)(when[Waiting] ~> effects.start)
    on(user.stop) {
      when[Live].where(states.stopping) ~> effects.stop
      when[Closed] ~> effects.stop
    }

object Written extends Machine[Job, Outcome, Fact], Phased[Job, Phase](_.phase):
  val init = Job(Phase.unstarted, Solo.over, Bare.unstarted)
  override def end(s: State) = states.closed(s.phase) && states.over(s) || states.expiring(s.phase)

  object states:
    def closed(p: Phase) = p.in(Phase.done, Phase.failed, Phase.expired)
    def expiring(p: Phase) = p.in(Phase.expired) || p.in(Phase.expired)
    def over(s: Job) = s.solo.in(Solo.over)
    def stopping(s: Job) = s.phase.in(Phase.running) && s.phase != Phase.paused

  object effects:
    def start(s: Job) = enter(s.copy(phase = Phase.running), Fact.started)
    def stop(s: Job) = enter(s.copy(phase = Phase.done))

  object rules extends Rules:
    on(user.start)(when(Phase.queued, Phase.backingOff) ~> effects.start)
    on(user.stop) {
      when(Phase.queued, Phase.backingOff, Phase.running, Phase.paused)
        .where(states.stopping) ~> effects.stop
      when(Phase.done, Phase.failed, Phase.expired) ~> effects.stop
    }
