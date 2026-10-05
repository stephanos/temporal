// Capability declarations, lifted through the Temporal catalog into claims named
// `<machine>.<law>`:
//   - a single capability's laws (Closable's two);
//   - the law of a pair found without being listed (Pausable with Pollable);
//   - functional laws asked from `reach` (Terminable's and Cancelable's);
//   - a composition's law read through its members' defs with `through` (the cross-entity form);
//   - a law waived with `except` and one replaced with `overriding`, each with its reason.
//   - a rogue job whose poll dispatches a paused job, which breaks the pair's law (R11);
//   - a fixture's own catalog, given explicitly, whose law keeps `status` (`terminal` a cited
//     parameter): a bound def whose body is a field path, `Jobs.phase`, is its `keeps` projection.
// Every function-valued field names a def; a `cited` binding names its server code as well.
package fixture.capabilities

import umpire.*
import temporal.capabilities.{closedIsRejectedUniformly, terminalStatesAreFinal}
import umpire.realize.{statusTable, Cleanup, Conformance, Disposition, PropertyOutcome}
import umpire.realize.{Reason, RunExpectation}
import temporal.capabilities.{given, *}
import CapabilitiesFamily.given

/** The family, in an object of its own: the machine objects read it while they initialize. */
object CapabilitiesFamily:
  given family: Family = Family("fixture.capabilities")

enum Phase derives Finite:
  case queued, running, paused, done, killed

/** A job's state, named apart from the machine object `Job`. */
final case class JobState(phase: Phase) derives Finite

enum Answer derives Finite:
  case ok, gone

given Ok[Answer] = Ok(Answer.ok)

enum Note derives Finite:
  case started, held, resumed, killedNote, cancelAsked, finished

val client: Party = Party()
val worker: Party = Party()

val poll = action(worker)
val finish = action(worker)
val pause = action(client)
val resume = action(client)
val kill = action(client)
val cancel = action(client)

/** The job's status sets and step functions; a control of a closed job is answered `gone`. */
object Jobs:
  import Phase.*

  def phase(j: JobState): Phase = j.phase
  def terminal(p: Phase): Boolean = p.in(done, killed)
  def paused(j: JobState): Boolean = j.phase == Phase.paused
  def running(j: JobState): Boolean = j.phase == Phase.running
  def over(j: JobState): Boolean = terminal(j.phase)

  private def closed(j: JobState): List[Step[JobState, Answer, Note]] = List(Step(Answer.gone, j))

  def poll(j: JobState): List[Step[JobState, Answer, Note]] =
    if j.phase == queued then enter(JobState(Phase.running), Note.started) else disabled
  def finish(j: JobState): List[Step[JobState, Answer, Note]] =
    if j.phase == Phase.running then enter(JobState(done), Note.finished) else disabled
  def pause(j: JobState): List[Step[JobState, Answer, Note]] =
    if terminal(j.phase) then closed(j)
    else if j.phase.in(queued, Phase.running) then enter(JobState(Phase.paused), Note.held)
    else disabled
  def resume(j: JobState): List[Step[JobState, Answer, Note]] =
    if terminal(j.phase) then closed(j)
    else if j.phase == Phase.paused then enter(JobState(queued), Note.resumed)
    else disabled
  def kill(j: JobState): List[Step[JobState, Answer, Note]] =
    if terminal(j.phase) then closed(j) else enter(JobState(killed), Note.killedNote)
  def cancel(j: JobState): List[Step[JobState, Answer, Note]] =
    if terminal(j.phase) then closed(j) else enter(j, Note.cancelAsked)

  /** The rogue job's poll, which dispatches a paused job as it does a queued one. */
  def rogueDispatch(j: JobState): List[Step[JobState, Answer, Note]] =
    if j.phase.in(queued, Phase.paused) then enter(JobState(Phase.running), Note.started)
    else disabled

  /** The legacy job's answer to a closed job: it keeps the state, whatever it answers. */
  def closedKeepsTheState[S, P](m: Declares[S])(
      status: S => P,
      terminal: P => Boolean,
      rejected: m.Outcome
  ): Property[S] =
    m.property holdsAcross ((before, after) => !terminal(status(before)) || after.state == before)

// 5 states; 6 action classes.
object Job extends Machine[JobState, Answer, Note]:
  val init = JobState(Phase.queued)
  def end(j: State) = Jobs.over(j)

  object rules
      extends Bindings(
        poll ~> Jobs.poll,
        finish ~> Jobs.finish,
        pause ~> Jobs.pause,
        resume ~> Jobs.resume,
        kill ~> Jobs.kill,
        cancel ~> Jobs.cancel
      )

// Its twin, which waives one law and replaces another.
object LegacyJob extends Machine[JobState, Answer, Note]:
  val init = JobState(Phase.queued)
  def end(j: State) = Jobs.over(j)

  object rules extends Bindings(poll ~> Jobs.poll, finish ~> Jobs.finish, kill ~> Jobs.kill)

val three = Limits(steps = 3, actions = 3, search = 512)

val jobStatus = statusTable(Note.started -> "RUNNING", Note.killedNote -> "TERMINATED")

/** The server code the jobs' cited bindings name: a closed job's answer, and which phases close. */
val jobsCode = "model/irgen/testdata/lifts/Capabilities.scala"

/** The Run a server is expected to give the job's functional laws, a hole of the Model in reach. */
val settles = RunExpectation(
  Conformance.inconclusive,
  PropertyOutcome.satisfied,
  PropertyOutcome.satisfied,
  Disposition.completed,
  Cleanup.succeeded,
  conformanceReason = Some(Reason.hole)
)

// Free verify Queries: 5 states x 6 classes x 3 steps = 90; finds: 5 states x min(3, 2) = 10.
val jobCapabilities = capabilities(Job, limits = three)(
  Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = cited(Answer.gone, jobsCode)),
  Pausable(pause = pause, unpause = resume, paused = Jobs.paused),
  Pollable(dispatch = poll, running = Jobs.running),
  Terminable(terminate = kill, settled = Note.killedNote, reach = Seq(poll), expect = settles),
  Cancelable(
    requestCancel = cancel,
    requested = Note.cancelAsked,
    reach = Seq(poll),
    expect = settles
  ),
  Describable(status = jobStatus)
)

// 5 states x 3 classes x 3 steps = 45.
val legacyCapabilities = capabilities(LegacyJob, limits = three)(
  Closable(status = Jobs.phase, terminal = Jobs.terminal, rejected = Answer.gone)
)
  .except(terminalStatesAreFinal, because = "a fixture's waiver: the legacy job keeps no status")
  .overriding(
    closedIsRejectedUniformly -> Jobs.closedKeepsTheState,
    because = "a fixture's override: the legacy job answers a closed job as it likes"
  )

// ### The cross-entity form: a composition's capabilities, read through a member's projection

/** Two jobs' state, named apart from the composition object `Pair`. */
final case class PairState(left: JobState, right: JobState) derives Finite

object Pairs:
  def over(p: PairState): Boolean = Jobs.over(p.left) && Jobs.over(p.right)

def runs(p: Phase): Boolean = p == Phase.running

object Pair extends Composition[PairState](_.left -> Job, _.right -> LegacyJob):
  def end(p: State) = Pairs.over(p)
  object syncs extends Syncs

// 25 states x (6 + 3) classes x 3 steps = 675.
val pairCapabilities = capabilities(Pair, limits = three)(
  Pausable(
    pause = Pair.own(_.left, pause),
    unpause = Pair.own(_.left, resume),
    paused = through(_.left, Jobs.paused)
  ),
  Pollable(dispatch = Pair.own(_.left, poll), running = through(_.left.phase, runs))
)

// ### A catalog of the fixture's own, whose law reads a bound field-path def as its `keeps`
/** Once closed, the status stays: `once(...).keeps(status)` over the bound `status`. */
object statusStaysClosed
    extends Law(
      cites = Seq("model/irgen/testdata/lifts/Capabilities.scala"),
      promises = "a closed job keeps its status",
      doesNotPromise = "anything a fixture does not need",
      parameters = Seq("terminal")
    ):
  def apply[S, P](m: Declares[S])(status: S => P, terminal: P => Boolean): Property[S] =
    m.property.once(s => terminal(status(s))).keeps(status)

val keptCatalog: Catalog = Catalog.single(Closable)(statusStaysClosed)

// Another twin, which declares Closable under the fixture's catalog.
object KeptJob extends Machine[JobState, Answer, Note]:
  val init = JobState(Phase.queued)
  def end(j: State) = Jobs.over(j)

  object rules extends Bindings(poll ~> Jobs.poll, finish ~> Jobs.finish, kill ~> Jobs.kill)

// 5 states x 3 classes x 3 steps = 45.
val keptCapabilities = capabilities(KeptJob, limits = three)(
  Closable(status = Jobs.phase, terminal = cited(Jobs.terminal, jobsCode), rejected = Answer.gone)
)(using keptCatalog)

// ### A Query of the job's own reading a generated Property by its law

// Pinned: 5 states x min(3 steps, 1 scheduled kill) = 5.
val queuedThenKilled = Job.scenario.actions(kill)
val killedWhileQueued =
  query find jobCapabilities.claim(terminateSettles) in queuedThenKilled limits three total 5

// ### A Model that breaks a law: the rogue job's poll dispatches a paused job

object RogueJob extends Machine[JobState, Answer, Note]:
  val init = JobState(Phase.queued)
  def end(j: State) = Jobs.over(j)

  object rules
      extends Bindings(poll ~> Jobs.rogueDispatch, pause ~> Jobs.pause, resume ~> Jobs.resume)

// 5 states x 3 classes x 3 steps = 45; violated after pause, poll.
val rogueCapabilities = capabilities(RogueJob, limits = three)(
  Pausable(pause = pause, unpause = resume, paused = Jobs.paused),
  Pollable(dispatch = poll, running = Jobs.running)
)

// ### A declaring function's function-valued argument read with `through`

/** The member a selector reads is never held: a shared claim, as `atMostOneActive` is. */
def neverHeld[S](m: Declares[S])(held: S => Boolean): Property[S] =
  m.property("neverHeld").never(s => held(s.state))

// 25 states x (6 + 3) classes x 3 steps = 675; the right job binds no pause.
val pairAny = Pair.scenario("pairAny").free
val rightNeverHeld =
  query verify neverHeld(Pair)(through(_.right, Jobs.paused)) in pairAny limits three total 675
