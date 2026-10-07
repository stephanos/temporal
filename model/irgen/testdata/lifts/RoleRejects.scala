// Role tests and role declarations the lifter refuses, each at its line (fn-136.1): a role test of
// a value of no enum, a role no case has, a type test against no role, of an enum and of a
// stand-alone trait, a role a case with fields has, and the three conflicts of a case's roles.
package fixture.rolerejects

// The role tests the lifter refuses, which the Models' lint leaves to the fixtures.
// scalafix:off DisableSyntax.isInstanceOf

import umpire.*
import fixture.roles.{Audited, Phase}

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

object user extends Actor:
  val go = action(this)

final case class Job(phase: Phase, next: Option[Phase]) derives Finite

object NoEnum extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object states:
    def closed(s: Job) = s.next.isInstanceOf[Closed]
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(where(states.closed) ~> effects.keep)

object NoCase extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object states:
    def canceled(p: Phase) = p.isInstanceOf[Canceled]
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(in(states.canceled) ~> effects.keep)

object NoRole extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object states:
    def any(p: Phase) = p match
      case _: Phase => true
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(in(states.any) ~> effects.keep)

object StandAlone extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object states:
    def audited(p: Phase) = p.isInstanceOf[Audited]
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(in(states.audited) ~> effects.keep)

enum Ended derives Finite:
  case running extends Ended, Held
  case failed(retried: Boolean) extends Ended, Failed

final case class Run(ended: Ended) derives Finite

object WithFields extends Machine[Run, Outcome, Nothing]:
  val init = Run(Ended.running)
  def end(s: State) = s.ended.isInstanceOf[Failed]
  object effects:
    def keep(s: Run) = stay[Run, Outcome, Nothing](s)
  object rules extends Rules(_.ended):
    on(user.go)(in(Ended.running) ~> effects.keep)

enum LiveClosed derives Finite:
  case backingOff extends LiveClosed, Retrying
  case lost extends LiveClosed, Retrying, Failed

enum TwoLive derives Finite:
  case torn extends TwoLive, Held, Suspended

enum TwoClosed derives Finite:
  case twice extends TwoClosed, Succeeded, Canceled

final case class Mixed(phase: LiveClosed) derives Finite

object BothLiveClosed extends Machine[Mixed, Outcome, Nothing]:
  val init = Mixed(LiveClosed.backingOff)
  def end(s: State) = true
  object effects:
    def keep(s: Mixed) = stay[Mixed, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(in(LiveClosed.backingOff) ~> effects.keep)

final case class Torn(phase: TwoLive) derives Finite

object TwoLiveRoles extends Machine[Torn, Outcome, Nothing]:
  val init = Torn(TwoLive.torn)
  def end(s: State) = true
  object effects:
    def keep(s: Torn) = stay[Torn, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(in(TwoLive.torn) ~> effects.keep)

final case class Twice(phase: TwoClosed) derives Finite

object TwoClosureRoles extends Machine[Twice, Outcome, Nothing]:
  val init = Twice(TwoClosed.twice)
  def end(s: State) = true
  object effects:
    def keep(s: Twice) = stay[Twice, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(in(TwoClosed.twice) ~> effects.keep)

// fn-136.4: the short spellings refused as role tests are: `when[R]` in rules whose projection is
// not declared, against no role and of a phase of no enum; `p.in[R]` against no role and of a
// value of no enum.
object WhenNoProjection extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules[Job, Outcome, Nothing, Phase]:
    on(user.go)(when[Closed] ~> effects.keep)

object WhenNoRole extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(when[Audited] ~> effects.keep)

object WhenNoEnum extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = true
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.next):
    on(user.go)(when[Closed] ~> effects.keep)

object InNoRole extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = s.phase.in[Audited]
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(always ~> effects.keep)

object InNoEnum extends Machine[Job, Outcome, Nothing]:
  val init = Job(Phase.unstarted, None)
  def end(s: State) = s.next.in[Closed]
  object effects:
    def keep(s: Job) = stay[Job, Outcome, Nothing](s)
  object rules extends Rules(_.phase):
    on(user.go)(always ~> effects.keep)
