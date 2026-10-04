// Each claim pattern of model/umpire/Syntax.scala beside the `holds` or `holdsAcross` lambda it stands
// for, on a machine and on a composition over a member projection, with the composition's
// `records(_.member, fact)` beside its composed key and three claims written once over `Declares[S]`.
// Each pattern is named `<name>` and its lambda spelling `<name>Core`; the lifter's tests lift both
// and require one IR, but for names, positions and the names of the functions they refer to.
package fixture.patterns

import umpire.*

given Family = Family("fixture.patterns")

enum Phase derives Finite:
  case idle, running, paused, done

enum Active derives Finite:
  case none, one, two

final case class Job(phase: Phase, active: Active) derives Finite

enum Outcome derives Finite:
  case accepted

given Accepted[Outcome] = Accepted(Outcome.accepted)

enum Fact derives Finite:
  case started, paused, finished, released

val start = action(Party("user"))
val pause = action(Party("user"))
val finish = action(Party("worker"))

type JobStep = Step[Job, Outcome, Fact]

def startStep(j: Job): List[JobStep] =
  if j.phase == Phase.idle then accept(Job(Phase.running, Active.one), Fact.started) else disabled
def pauseStep(j: Job): List[JobStep] =
  if j.phase == Phase.running then accept(j.copy(phase = Phase.paused), Fact.paused) else disabled
def finishStep(j: Job): List[JobStep] =
  if j.phase.in(Phase.running, Phase.paused) then
    accept(Job(Phase.done, Active.none), Fact.finished, Fact.released)
  else disabled

val job = machine[Job, Outcome, Fact] {
  starts(Job(Phase.idle, Active.none))
  ends(j => j.phase == Phase.done)
  steps(start ~> startStep, pause ~> pauseStep, finish ~> finishStep)
}

final case class Lamp(lit: Boolean) derives Finite

enum LampFact derives Finite:
  case flipped

val flip = action(Party("user"))

def flipStep(l: Lamp): List[Step[Lamp, Outcome, LampFact]] = accept(Lamp(!l.lit), LampFact.flipped)

val lamp = machine[Lamp, Outcome, LampFact] {
  starts(Lamp(false))
  ends(_ => true)
  steps(flip ~> flipStep)
}

/** The job beside a lamp: a composition whose claims read the job through its member field. */
final case class Pair(job: Job, lamp: Lamp) derives Finite

type PairStep = Step[Pair, String, String]

val pair = compose[Pair](_.job -> job, _.lamp -> lamp).ends(p => p.job.phase == Phase.done)

// ### The predicates the claims name, over the job and over the pair's job

def isPaused(j: Job): Boolean = j.phase == Phase.paused
def isRunning(j: Job): Boolean = j.phase == Phase.running
def isDone(j: Job): Boolean = j.phase == Phase.done
def isActive(j: Job): Boolean = j.active != Active.none
def twoActive(j: Job): Boolean = j.active == Active.two
def jobPhase(j: Job): Phase = j.phase
def startedStep(after: JobStep): Boolean = isRunning(after.state)
def twoActiveStep(after: JobStep): Boolean = twoActive(after.state)
def released(after: JobStep): Boolean = after.records(Fact.released)

def pairPaused(p: Pair): Boolean = isPaused(p.job)
def pairRunning(p: Pair): Boolean = isRunning(p.job)
def pairDone(p: Pair): Boolean = isDone(p.job)
def pairActive(p: Pair): Boolean = isActive(p.job)
def pairTwo(p: Pair): Boolean = twoActive(p.job)
def pairPhase(p: Pair): Phase = p.job.phase
def pairStartedStep(after: PairStep): Boolean = isRunning(after.state.job)
def pairTwoStep(after: PairStep): Boolean = twoActive(after.state.job)
def pairReleased(after: PairStep): Boolean = after.records(_.job, Fact.released)

// ### Each pattern on the machine, beside its lambda spelling

val doneKeeps = job.property.once(isDone).keeps(_.phase)
val doneKeepsCore = job.property holdsAcross { (before, after) =>
  !isDone(before) || after.state.phase == before.phase
}

val noTwo = job.property.never(twoActiveStep)
val noTwoCore = job.property holds (after => !twoActiveStep(after))

val notStartedWhilePaused = job.property.never(startedStep).from(isPaused)
val notStartedWhilePausedCore = job.property holdsAcross { (before, after) =>
  !isPaused(before) || !startedStep(after)
}

val doneStays = job.property.stays(isDone)
val doneStaysCore = job.property holdsAcross { (before, after) =>
  !isDone(before) || isDone(after.state)
}

val activeStays = job.property.stays(isActive).unless(released)
val activeStaysCore = job.property holdsAcross { (before, after) =>
  !isActive(before) || isActive(after.state) || released(after)
}

// ### Each pattern on the composition, through the member projection

val pairDoneKeeps = pair.property.once(pairDone).keeps(_.job.phase)
val pairDoneKeepsCore = pair.property holdsAcross { (before, after) =>
  !pairDone(before) || after.state.job.phase == before.job.phase
}

val pairNoTwo = pair.property.never(pairTwoStep)
val pairNoTwoCore = pair.property holds (after => !pairTwoStep(after))

val pairNotStartedWhilePaused = pair.property.never(pairStartedStep).from(pairPaused)
val pairNotStartedWhilePausedCore = pair.property holdsAcross { (before, after) =>
  !pairPaused(before) || !pairStartedStep(after)
}

val pairDoneStays = pair.property.stays(pairDone)
val pairDoneStaysCore = pair.property holdsAcross { (before, after) =>
  !pairDone(before) || pairDone(after.state)
}

val pairActiveStays = pair.property.stays(pairActive).unless(pairReleased)
val pairActiveStaysCore = pair.property holdsAcross { (before, after) =>
  !pairActive(before) || pairActive(after.state) || pairReleased(after)
}

// A member's fact, by its selector and by the composed key a composition records it under.
val pairStarted = pair.property holds (after => after.records(_.job, Fact.started))
val pairStartedCore = pair.property holds (after => after.facts.contains("job_started"))

val lampFlipped = pair.property holds (after => after.records(_.lamp, LampFact.flipped))
val lampFlippedCore = pair.property holds (after => after.facts.contains("lamp_flipped"))

// ### Each pattern with a lambda of its own, lifted as `holds` lifts one

val doneKeepsInline = job.property.once(_.phase == Phase.done).keeps(_.phase)
val noTwoInline = job.property.never(_.state.active == Active.two)
val notStartedInline =
  job.property.never(_.state.phase == Phase.running).from(_.phase == Phase.paused)
val doneStaysInline = job.property.stays(_.phase == Phase.done)
val activeStaysInline =
  job.property.stays(_.active != Active.none).unless(_.records(Fact.released))

val doneKeepsInlineCore = job.property holdsAcross { (before, after) =>
  !(before.phase == Phase.done) || after.state.phase == before.phase
}
val noTwoInlineCore = job.property holds (after => !(after.state.active == Active.two))
val notStartedInlineCore = job.property holdsAcross { (before, after) =>
  !(before.phase == Phase.paused) || !(after.state.phase == Phase.running)
}
val doneStaysInlineCore = job.property holdsAcross { (before, after) =>
  !(before.phase == Phase.done) || after.state.phase == Phase.done
}
val activeStaysInlineCore = job.property holdsAcross { (before, after) =>
  !(before.active != Active.none) || after.state.active != Active.none ||
  after.records(Fact.released)
}

// ### Claims written once over `Declares[S]`, on the machine and on the composition

def notAdmittedWhilePaused[S](m: Declares[S])(
    paused: S => Boolean,
    running: S => Boolean
): Property[S] =
  m.property("notAdmittedWhilePaused").never(s => running(s.state)).from(paused)

def atMostOneActive[S](m: Declares[S])(twoActive: S => Boolean): Property[S] =
  m.property("atMostOneActive").never(s => twoActive(s.state))

def terminalStays[S, P](m: Declares[S])(terminal: S => Boolean, phase: S => P): Property[S] =
  m.property("terminalStays").once(terminal).keeps(phase)

val jobNotAdmitted = notAdmittedWhilePaused(job)(isPaused, isRunning)
val jobNotAdmittedCore = job.property("notAdmittedWhilePausedCore") holdsAcross { (before, after) =>
  !isPaused(before) || !isRunning(after.state)
}
val jobOneActive = atMostOneActive(job)(twoActive)
val jobOneActiveCore = job.property("atMostOneActiveCore") holds (after => !twoActive(after.state))
val jobTerminal = terminalStays(job)(isDone, jobPhase)
val jobTerminalCore = job.property("terminalStaysCore") holdsAcross { (before, after) =>
  !isDone(before) || after.state.phase == before.phase
}

val pairNotAdmitted = notAdmittedWhilePaused(pair)(pairPaused, pairRunning)
val pairNotAdmittedCore = pair.property("notAdmittedWhilePausedCore") holdsAcross {
  (before, after) => !pairPaused(before) || !pairRunning(after.state)
}
val pairOneActive = atMostOneActive(pair)(pairTwo)
val pairOneActiveCore =
  pair.property("atMostOneActiveCore") holds (after => !pairTwo(after.state))
val pairTerminal = terminalStays(pair)(pairDone, pairPhase)
val pairTerminalCore = pair.property("terminalStaysCore") holdsAcross { (before, after) =>
  !pairDone(before) || after.state.job.phase == before.job.phase
}

// ### One Query per Property, so a lift of `claims` reaches every one

val run = Limits(steps = 2, actions = 2, search = 64)
val jobAny = job.scenario.free
val pairAny = pair.scenario.free

def onJob(name: String, p: Property[Job]): Query =
  query(name) verify p in jobAny limits run total 72
def onPair(name: String, p: Property[Pair]): Query =
  query(name) verify p in pairAny limits run total 192

val claims: Vector[Query] = Vector(
  onJob("doneKeeps", doneKeeps),
  onJob("doneKeepsCore", doneKeepsCore),
  onJob("noTwo", noTwo),
  onJob("noTwoCore", noTwoCore),
  onJob("notStartedWhilePaused", notStartedWhilePaused),
  onJob("notStartedWhilePausedCore", notStartedWhilePausedCore),
  onJob("doneStays", doneStays),
  onJob("doneStaysCore", doneStaysCore),
  onJob("activeStays", activeStays),
  onJob("activeStaysCore", activeStaysCore),
  onJob("doneKeepsInline", doneKeepsInline),
  onJob("doneKeepsInlineCore", doneKeepsInlineCore),
  onJob("noTwoInline", noTwoInline),
  onJob("noTwoInlineCore", noTwoInlineCore),
  onJob("notStartedInline", notStartedInline),
  onJob("notStartedInlineCore", notStartedInlineCore),
  onJob("doneStaysInline", doneStaysInline),
  onJob("doneStaysInlineCore", doneStaysInlineCore),
  onJob("activeStaysInline", activeStaysInline),
  onJob("activeStaysInlineCore", activeStaysInlineCore),
  onJob("jobNotAdmitted", jobNotAdmitted),
  onJob("jobNotAdmittedCore", jobNotAdmittedCore),
  onJob("jobOneActive", jobOneActive),
  onJob("jobOneActiveCore", jobOneActiveCore),
  onJob("jobTerminal", jobTerminal),
  onJob("jobTerminalCore", jobTerminalCore),
  onPair("pairDoneKeeps", pairDoneKeeps),
  onPair("pairDoneKeepsCore", pairDoneKeepsCore),
  onPair("pairNoTwo", pairNoTwo),
  onPair("pairNoTwoCore", pairNoTwoCore),
  onPair("pairNotStartedWhilePaused", pairNotStartedWhilePaused),
  onPair("pairNotStartedWhilePausedCore", pairNotStartedWhilePausedCore),
  onPair("pairDoneStays", pairDoneStays),
  onPair("pairDoneStaysCore", pairDoneStaysCore),
  onPair("pairActiveStays", pairActiveStays),
  onPair("pairActiveStaysCore", pairActiveStaysCore),
  onPair("pairStarted", pairStarted),
  onPair("pairStartedCore", pairStartedCore),
  onPair("lampFlipped", lampFlipped),
  onPair("lampFlippedCore", lampFlippedCore),
  onPair("pairNotAdmitted", pairNotAdmitted),
  onPair("pairNotAdmittedCore", pairNotAdmittedCore),
  onPair("pairOneActive", pairOneActive),
  onPair("pairOneActiveCore", pairOneActiveCore),
  onPair("pairTerminal", pairTerminal),
  onPair("pairTerminalCore", pairTerminalCore)
)
