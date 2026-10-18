package framework.fixture.expect

import framework.dsl.*
import framework.dsl.Finite.given

// Expectation blocks beside independently written core twins (fn-164.3). Each `<name>` is an
// expectation block and each `<name>Core` the ordinary predicate it stands for, written with the
// core finishers directly, never through the block. `skewed` is a twin with another association:
// the comparison must tell it apart.

enum Phase derives CanEqual:
  case idle, running, done
enum Said derives CanEqual:
  case accepted
enum Note derives CanEqual:
  case beat, finished

final case class Job(phase: Phase, beating: Boolean) derives CanEqual

object worker extends Actor:
  val start = action(this)
  val beat = action(this)
  val finish = action(this)
  val reset = action(this)

object Subject extends Machine[Job, Said, Note]:
  val init = Job(Phase.idle, false)
  def end(s: State) = s.phase == Phase.done
  object rules
      extends Bindings(
        worker.start ~> starting,
        worker.beat ~> beating,
        worker.finish ~> finishing,
        worker.reset ~> resetting
      )

def starting(s: Job): List[Step[Job, Said, Note]] =
  List(Step(Said.accepted, s.copy(phase = Phase.running)))
def beating(s: Job): List[Step[Job, Said, Note]] =
  List(Step(Said.accepted, s.copy(beating = true), List(Note.beat)))
def finishing(s: Job): List[Step[Job, Said, Note]] =
  List(Step(Said.accepted, s.copy(phase = Phase.done), List(Note.finished)))
def resetting(s: Job): List[Step[Job, Said, Note]] =
  List(Step(Said.accepted, s.copy(phase = Phase.idle, beating = false)))

// Heartbeat-shaped: a fact and an implication, restricted to one action class.
val heartbeat = Subject
  .property("heartbeat")
  .after(worker.beat)
  .expect:
    records(Note.beat)
    if after.state.phase == Phase.running then check(after.state.beating)
val heartbeatCore = Subject
  .property("heartbeatCore")
  .when(worker.beat)
  .sameStep(after =>
    after.facts
      .contains(Note.beat) && (!(after.state.phase == Phase.running) || after.state.beating)
  )
val skewed = Subject
  .property("skewed")
  .when(worker.beat)
  .sameStep(after =>
    (after.facts
      .contains(Note.beat) && !(after.state.phase == Phase.running)) || after.state.beating
  )

// Unrestricted, three requirements in written order.
val finished = Subject
  .property("finished")
  .expect:
    check(after.state.phase != Phase.idle)
    records(Note.finished)
    check(after.state.phase == Phase.done)
val finishedCore = Subject
  .property("finishedCore")
  .sameStep(after =>
    after.state.phase != Phase.idle && after.facts.contains(Note.finished) &&
      after.state.phase == Phase.done
  )

// Reset-shaped: an eager helper val and explicit alternatives over a transition.
val reset = Subject
  .property("reset")
  .afterAction("reset")
  .across
  .expect:
    val restarted = before.phase == Phase.done
    oneOf:
      branch:
        check(restarted)
        check(after.state.phase == Phase.idle)
      branch:
        check(!restarted)
        check(after.state == before)
val resetCore = Subject
  .property("resetCore")
  .whenAction("reset")
  .transition((before, after) =>
    val restarted = before.phase == Phase.done
    restarted && after.state.phase == Phase.idle || !restarted && after.state == before
  )

// A conditional with an else is the conditional of both requirement blocks.
val settled = Subject
  .property("settled")
  .expect:
    if after.state.phase == Phase.done then check(!after.state.beating)
    else check(after.state.phase != Phase.idle || !after.state.beating)
val settledCore = Subject
  .property("settledCore")
  .sameStep(after =>
    if after.state.phase == Phase.done then !after.state.beating
    else after.state.phase != Phase.idle || !after.state.beating
  )

// Explicit context names are the parameters' names.
val explicit = Subject
  .property("explicit")
  .across
  .expect: (b: Before[Job], a: Expectation[Job, Said, Note]) ?=>
    check(b.state.phase != Phase.done || a.step.state.phase == Phase.done)
val explicitCore = Subject
  .property("explicitCore")
  .transition((b, a) => b.phase != Phase.done || a.state.phase == Phase.done)

// A standalone expectation is an ordinary predicate a block calls by name.
val beatingNow: Step[Job, Said, Note] => Boolean = expect:
  check(after.state.beating)
val forwarded = Subject
  .property("forwarded")
  .expect:
    check(beatingNow(after))
def beatingNowCore(after: Step[Job, Said, Note]): Boolean = after.state.beating
val forwardedCore = Subject.property("forwardedCore").sameStep(after => beatingNowCore(after))

val one = Limits("one", steps = 2, actions = 2, search = 64)
val any = Subject.scenario("any").free
// Each side is lifted apart: a lift refuses a second Property that states the same promise.
val queries: Vector[Query] = Vector(
  query("heartbeat") verify heartbeat in any limits one,
  query("finished") verify finished in any limits one,
  query("reset") verify reset in any limits one,
  query("settled") verify settled in any limits one,
  query("explicit") verify explicit in any limits one,
  query("forwarded") verify forwarded in any limits one
)
val cores: Vector[Query] = Vector(
  query("heartbeatCore") verify heartbeatCore in any limits one,
  query("finishedCore") verify finishedCore in any limits one,
  query("resetCore") verify resetCore in any limits one,
  query("settledCore") verify settledCore in any limits one,
  query("explicitCore") verify explicitCore in any limits one,
  query("forwardedCore") verify forwardedCore in any limits one
)
val skewedQueries: Vector[Query] = Vector(query("skewed") verify skewed in any limits one)
