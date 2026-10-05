// Each sugar form of model/umpire/Syntax.scala beside its core form: `sugared` binds and claims with
// the sugar, `cored` with the core spelling, action by action and Property by Property, and `watched`
// is watched by each sticky monitor and its `monitor` spelling. The lifter's tests lift both and
// require one IR of the two, but for names, positions and the names of the functions the declarations
// refer to.
package fixture.sugar

import umpire.*

given Family = Family("fixture.sugar")

enum Phase derives Finite:
  case idle, running, paused, done

final case class Job(phase: Phase, retried: Boolean) derives Finite

enum Outcome derives Finite:
  case accepted, refused

given Ok[Outcome] = Ok(Outcome.accepted)

enum Fact derives Finite:
  case started, paused, finished

val start = action(Party("user"))
val pause = action(Party("user"))
val resume = action(Party("user"))
val finish = action(Party("worker"))
val poke = action(Party("user"))
val retry = action(Party("user"))
val idle = action(Party("user"))

val retryUnknown = hole

type JobStep = Step[Job, Outcome, Fact]

// enter: one step with the ok outcome, its facts listed.
def startSugar(j: Job): List[JobStep] =
  if j.phase == Phase.idle then enter(j.copy(phase = Phase.running), Fact.started) else disabled
def startCore(j: Job): List[JobStep] =
  if j.phase == Phase.idle then
    List(Step(Outcome.accepted, j.copy(phase = Phase.running), List(Fact.started)))
  else Nil

// enter(...).because(...): the step's explanation.
def pauseSugar(j: Job): List[JobStep] = j.phase match
  case Phase.running =>
    enter(j.copy(phase = Phase.paused), Fact.paused).because("a running job pauses")
  case _ => disabled
def pauseCore(j: Job): List[JobStep] = j.phase match
  case Phase.running =>
    List(
      Step(
        Outcome.accepted,
        j.copy(phase = Phase.paused),
        List(Fact.paused),
        "a running job pauses"
      )
    )
  case _ => Nil

// in: membership in a finite set of cases, the default arm a wildcard.
def resumeSugar(j: Job): List[JobStep] = j.phase match
  case p if p.in(Phase.paused, Phase.idle) => enter(j.copy(phase = Phase.running))
  case _                                   => disabled
def resumeCore(j: Job): List[JobStep] = j.phase match
  case p if List(Phase.paused, Phase.idle).contains(p) =>
    List(Step(Outcome.accepted, j.copy(phase = Phase.running)))
  case _ => Nil

// in with one member, and enter with several facts.
def finishSugar(j: Job): List[JobStep] =
  if j.phase.in(Phase.running) then enter(j.copy(phase = Phase.done), Fact.finished, Fact.started)
  else disabled
def finishCore(j: Job): List[JobStep] =
  if List(Phase.running).contains(j.phase) then
    List(Step(Outcome.accepted, j.copy(phase = Phase.done), List(Fact.finished, Fact.started)))
  else Nil

// stay: the state kept, nothing recorded.
def pokeSugar(j: Job): List[JobStep] = stay(j)
def pokeCore(j: Job): List[JobStep] = List(Step(Outcome.accepted, j))

// implies whose right side reaches a hole only where its left side holds.
def retrySugar(j: Job): List[JobStep] =
  if j.phase == Phase.done && j.retried implies retryUnknown.reached then stay(j) else disabled
def retryCore(j: Job): List[JobStep] =
  if !(j.phase == Phase.done && j.retried) || retryUnknown.reached then
    List(Step(Outcome.accepted, j))
  else Nil

// disabled alone.
def idleSugar(j: Job): List[JobStep] = disabled
def idleCore(j: Job): List[JobStep] = Nil

val sugared = machine[Job, Outcome, Fact] {
  starts(Job(Phase.idle, false))
  ends(j => j.phase.in(Phase.done, Phase.idle))
  steps(
    start ~> startSugar,
    pause ~> pauseSugar,
    resume ~> resumeSugar,
    finish ~> finishSugar,
    poke ~> pokeSugar,
    retry ~> retrySugar,
    idle ~> idleSugar
  )
}

val cored = machine[Job, Outcome, Fact] {
  starts(Job(Phase.idle, false))
  ends(j => List(Phase.done, Phase.idle).contains(j.phase))
  steps(
    start ~> startCore,
    pause ~> pauseCore,
    resume ~> resumeCore,
    finish ~> finishCore,
    poke ~> pokeCore,
    retry ~> retryCore,
    idle ~> idleCore
  )
}

// records and implies in claims.
val recordsSugar = sugared.property when start holds (after => after.records(Fact.started))
val recordsCore = cored.property when start holds (after => after.facts.contains(Fact.started))

val impliesSugar = sugared.property holds (after =>
  after.state.phase == Phase.done implies after.records(Fact.finished)
)
val impliesCore = cored.property holds (after =>
  !(after.state.phase == Phase.done) || after.facts.contains(Fact.finished)
)

val pausedSugar = sugared.property holdsAcross { (before, after) =>
  before.phase.in(Phase.paused) implies !after.records(Fact.paused)
}
val pausedCore = cored.property holdsAcross { (before, after) =>
  !List(Phase.paused).contains(before.phase) || !after.facts.contains(Fact.paused)
}

val run = Limits(steps = 2, actions = 2, search = 64)

val sugaredStart = sugared.scenario.actions(start)
val coredStart = cored.scenario.actions(start)

val claims: Vector[Query] = Vector(
  query("recordsSugarStart") find recordsSugar in sugaredStart limits run total 8,
  query("recordsCoreStart") find recordsCore in coredStart limits run total 8,
  query("impliesSugarStart") verify impliesSugar in sugaredStart limits run total 8,
  query("impliesCoreStart") verify impliesCore in coredStart limits run total 8,
  query("pausedSugarStart") verify pausedSugar in sugaredStart limits run total 8,
  query("pausedCoreStart") verify pausedCore in coredStart limits run total 8
)

// sticky and stickyAcross: a promise that once broken stays broken, each beside the monitor it stands
// for, all four watching one machine.
def neverRefused(after: JobStep): Boolean = after.outcome != Outcome.refused
def retriedStays(before: Job, after: JobStep): Boolean = !before.retried || after.state.retried

val refusedOnce = sticky(neverRefused)
val refusedOnceSpelled = monitor[Job, Outcome, Fact, Boolean](false)((broken, before, after) =>
  broken || !neverRefused(after)
)(broken => broken)

val retriedLost = stickyAcross(retriedStays)
val retriedLostSpelled = monitor[Job, Outcome, Fact, Boolean](false)((broken, before, after) =>
  broken || !retriedStays(before, after)
)(broken => broken)

val watched = machine[Job, Outcome, Fact] {
  monitors(refusedOnce, refusedOnceSpelled, retriedLost, retriedLostSpelled)
  starts(Job(Phase.idle, false))
  ends(j => j.phase == Phase.done)
  steps(start ~> startSugar, retry ~> retrySugar)
}
