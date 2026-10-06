// Each claim pattern of model/umpire/Syntax.scala beside the `holds` or `holdsAcross` lambda it stands
// for, on a machine and on a composition over a member projection, with the composition's
// `records(_.member, fact)` beside its composed key and three claims written once over `Declares[S]`.
// Each pattern is named `<name>` and its lambda spelling `<name>Core`; the lifter's tests lift both
// and require one IR, but for names, positions and the names of the functions they refer to.
package fixture.patterns

import umpire.*

enum Phase derives Finite:
  case idle, running, paused, done

enum Active derives Finite:
  case none, one, two

/** The job's state, named apart from the machine object `Job`. */
final case class JobState(phase: Phase, active: Active) derives Finite

enum Outcome derives Finite:
  case accepted

given Ok[Outcome] = Ok(Outcome.accepted)

enum Fact derives Finite:
  case started, paused, finished, released

val start = action(Actor("user"))
val pause = action(Actor("user"))
val finish = action(Actor("worker"))

type JobStep = Step[JobState, Outcome, Fact]

def startStep(j: JobState): List[JobStep] =
  if j.phase == Phase.idle then enter(JobState(Phase.running, Active.one), Fact.started)
  else disabled
def pauseStep(j: JobState): List[JobStep] =
  if j.phase == Phase.running then enter(j.copy(phase = Phase.paused), Fact.paused) else disabled
def finishStep(j: JobState): List[JobStep] =
  if j.phase.in(Phase.running, Phase.paused) then
    enter(JobState(Phase.done, Active.none), Fact.finished, Fact.released)
  else disabled

object Job extends Machine[JobState, Outcome, Fact]:
  val init = JobState(Phase.idle, Active.none)
  def end(j: State) = j.phase == Phase.done

  object rules extends Bindings(start ~> startStep, pause ~> pauseStep, finish ~> finishStep)

/** The lamp's state, named apart from the machine object `Lamp`. */
final case class LampState(lit: Boolean) derives Finite

enum LampFact derives Finite:
  case flipped

val flip = action(Actor("user"))

def flipStep(l: LampState): List[Step[LampState, Outcome, LampFact]] =
  enter(LampState(!l.lit), LampFact.flipped)

object Lamp extends Machine[LampState, Outcome, LampFact]:
  val init = LampState(false)
  def end(lamp: State) = true

  object rules extends Bindings(flip ~> flipStep)

/**
 * The job beside a lamp: a composition whose claims read the job through its member field, its state
 * named apart from the composition object `Pair`.
 */
final case class PairState(job: JobState, lamp: LampState) derives Finite

type PairStep = Step[PairState, String, String]

object Pair extends Composition[PairState](_.job -> Job, _.lamp -> Lamp):
  def end(p: State) = p.job.phase == Phase.done
  object syncs extends Syncs

// ### The predicates the claims name, over the job and over the pair's job

def isPaused(j: JobState): Boolean = j.phase == Phase.paused
def isRunning(j: JobState): Boolean = j.phase == Phase.running
def isDone(j: JobState): Boolean = j.phase == Phase.done
def isActive(j: JobState): Boolean = j.active != Active.none
def twoActive(j: JobState): Boolean = j.active == Active.two
def jobPhase(j: JobState): Phase = j.phase
def startedStep(after: JobStep): Boolean = isRunning(after.state)
def twoActiveStep(after: JobStep): Boolean = twoActive(after.state)
def released(after: JobStep): Boolean = after.records(Fact.released)

def pairPaused(p: PairState): Boolean = isPaused(p.job)
def pairRunning(p: PairState): Boolean = isRunning(p.job)
def pairDone(p: PairState): Boolean = isDone(p.job)
def pairActive(p: PairState): Boolean = isActive(p.job)
def pairTwo(p: PairState): Boolean = twoActive(p.job)
def pairPhase(p: PairState): Phase = p.job.phase
def pairStartedStep(after: PairStep): Boolean = isRunning(after.state.job)
def pairTwoStep(after: PairStep): Boolean = twoActive(after.state.job)
def pairReleased(after: PairStep): Boolean = after.records(_.job, Fact.released)

// ### Each pattern on the machine, beside its lambda spelling

val doneKeeps = Job.property.once(isDone).keeps(_.phase)
val doneKeepsCore = Job.property holdsAcross { (before, after) =>
  !isDone(before) || after.state.phase == before.phase
}

val noTwo = Job.property.never(twoActiveStep)
val noTwoCore = Job.property holds (after => !twoActiveStep(after))

val notStartedWhilePaused = Job.property.never(startedStep).from(isPaused)
val notStartedWhilePausedCore = Job.property holdsAcross { (before, after) =>
  !isPaused(before) || !startedStep(after)
}

val doneStays = Job.property.stays(isDone)
val doneStaysCore = Job.property holdsAcross { (before, after) =>
  !isDone(before) || isDone(after.state)
}

val activeStays = Job.property.stays(isActive).unless(released)
val activeStaysCore = Job.property holdsAcross { (before, after) =>
  !isActive(before) || isActive(after.state) || released(after)
}

// ### Each pattern on the composition, through the member projection

val pairDoneKeeps = Pair.property.once(pairDone).keeps(_.job.phase)
val pairDoneKeepsCore = Pair.property holdsAcross { (before, after) =>
  !pairDone(before) || after.state.job.phase == before.job.phase
}

val pairNoTwo = Pair.property.never(pairTwoStep)
val pairNoTwoCore = Pair.property holds (after => !pairTwoStep(after))

val pairNotStartedWhilePaused = Pair.property.never(pairStartedStep).from(pairPaused)
val pairNotStartedWhilePausedCore = Pair.property holdsAcross { (before, after) =>
  !pairPaused(before) || !pairStartedStep(after)
}

val pairDoneStays = Pair.property.stays(pairDone)
val pairDoneStaysCore = Pair.property holdsAcross { (before, after) =>
  !pairDone(before) || pairDone(after.state)
}

val pairActiveStays = Pair.property.stays(pairActive).unless(pairReleased)
val pairActiveStaysCore = Pair.property holdsAcross { (before, after) =>
  !pairActive(before) || pairActive(after.state) || pairReleased(after)
}

// A member's fact, by its selector and by the composed key a composition records it under.
val pairStarted = Pair.property holds (after => after.records(_.job, Fact.started))
val pairStartedCore = Pair.property holds (after => after.facts.contains("job_started"))

val lampFlipped = Pair.property holds (after => after.records(_.lamp, LampFact.flipped))
val lampFlippedCore = Pair.property holds (after => after.facts.contains("lamp_flipped"))

// ### Each pattern with a lambda of its own, lifted as `holds` lifts one

val doneKeepsInline = Job.property.once(_.phase == Phase.done).keeps(_.phase)
val noTwoInline = Job.property.never(_.state.active == Active.two)
val notStartedInline =
  Job.property.never(_.state.phase == Phase.running).from(_.phase == Phase.paused)
val doneStaysInline = Job.property.stays(_.phase == Phase.done)
val activeStaysInline =
  Job.property.stays(_.active != Active.none).unless(_.records(Fact.released))

val doneKeepsInlineCore = Job.property holdsAcross { (before, after) =>
  !(before.phase == Phase.done) || after.state.phase == before.phase
}
val noTwoInlineCore = Job.property holds (after => !(after.state.active == Active.two))
val notStartedInlineCore = Job.property holdsAcross { (before, after) =>
  !(before.phase == Phase.paused) || !(after.state.phase == Phase.running)
}
val doneStaysInlineCore = Job.property holdsAcross { (before, after) =>
  !(before.phase == Phase.done) || after.state.phase == Phase.done
}
val activeStaysInlineCore = Job.property holdsAcross { (before, after) =>
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

val jobNotAdmitted = notAdmittedWhilePaused(Job)(isPaused, isRunning)
val jobNotAdmittedCore = Job.property("notAdmittedWhilePausedCore") holdsAcross { (before, after) =>
  !isPaused(before) || !isRunning(after.state)
}
val jobOneActive = atMostOneActive(Job)(twoActive)
val jobOneActiveCore = Job.property("atMostOneActiveCore") holds (after => !twoActive(after.state))
val jobTerminal = terminalStays(Job)(isDone, jobPhase)
val jobTerminalCore = Job.property("terminalStaysCore") holdsAcross { (before, after) =>
  !isDone(before) || after.state.phase == before.phase
}

val pairNotAdmitted = notAdmittedWhilePaused(Pair)(pairPaused, pairRunning)
val pairNotAdmittedCore = Pair.property("notAdmittedWhilePausedCore") holdsAcross {
  (before, after) => !pairPaused(before) || !pairRunning(after.state)
}
val pairOneActive = atMostOneActive(Pair)(pairTwo)
val pairOneActiveCore =
  Pair.property("atMostOneActiveCore") holds (after => !pairTwo(after.state))
val pairTerminal = terminalStays(Pair)(pairDone, pairPhase)
val pairTerminalCore = Pair.property("terminalStaysCore") holdsAcross { (before, after) =>
  !pairDone(before) || after.state.job.phase == before.job.phase
}

// ### One Query per Property, so a lift of `claims` reaches every one

val run = Limits(steps = 2, actions = 2, search = 64)
val jobAny = Job.scenario.free
val pairAny = Pair.scenario.free

def onJob(name: String, p: Property[JobState]): Query =
  query(name) verify p in jobAny limits run total 72
def onPair(name: String, p: Property[PairState]): Query =
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
