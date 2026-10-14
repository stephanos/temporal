package temporal.feature.worker

/* The worker entity
 *
 * The worker of one task queue, as an entity of its own rather than a stutter row on the machines
 * of the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes
 * it. The machine is the one a composition synchronizes with: a workflow or an operation whose
 * progress needs a worker names `serve` beside its own action, and a stopped worker has no row for
 * it.
 *
 * The module declares no set, Case, or Query: nothing here is realized on its own, and the
 * Properties about a worker are the cross-entity ones a composition states.
 */

import umpire.*

// -- authoring: worker --------------------------------------------------------------------------

/** A worker is named by the task queue it polls: the handler's worker and the workflow's worker
  * are two instances of this entity, told apart by their queue. */
val worker: Entity = entity("worker") key "taskQueue"

enum Phase derives Finite:
  case polling, stopped

final case class WorkerState(phase: Phase) derives Finite, CanEqual

enum WorkerOutcome derives Finite:
  case accepted

/** A worker records nothing of its own: its stop and resume are faults the Run records against no
  * entity, and what it serves is recorded by the work it serves. */
sealed trait WorkerFact // no implementors: an uninhabited fact type, so `Finite` is written by hand
object WorkerFact:
  given Finite[WorkerFact] with
    def values = IndexedSeq.empty

/* The actions
 *
 * The two faults are the `worker` party's and name no entity. The serve action is the worker's own
 * and takes no input, so a composition may synchronize it with an action of any class. */

val workerStop: Action[EmptyTuple] = action("workerStop") party Party.worker
val workerResume: Action[EmptyTuple] = action("workerResume") party Party.worker
val serve: Action[EmptyTuple] = action("serve") party Party.worker on worker

// -- authoring: polling -------------------------------------------------------------------------

type WorkerStep = Step[WorkerState, WorkerOutcome, WorkerFact]

/** A polling worker stops; a stopped one has nothing to stop. */
def stopStep(state: WorkerState): List[WorkerStep] = state.phase match
  case Phase.polling => List(Step(WorkerOutcome.accepted, WorkerState(Phase.stopped), Nil))
  case Phase.stopped => Nil

/** A stopped worker resumes polling; a polling one has nothing to resume. */
def resumeStep(state: WorkerState): List[WorkerStep] = state.phase match
  case Phase.stopped => List(Step(WorkerOutcome.accepted, WorkerState(Phase.polling), Nil))
  case Phase.polling => Nil

/** A polling worker serves and keeps polling; a stopped one serves nothing. */
def serveStep(state: WorkerState): List[WorkerStep] = state.phase match
  case Phase.polling => List(Step(WorkerOutcome.accepted, state, Nil))
  case Phase.stopped => Nil

/** A worker has no natural end: it may be left polling or stopped. */
val polling: Machine[WorkerState, WorkerOutcome, WorkerFact] =
  machine[WorkerState, WorkerOutcome, WorkerFact]("polling"):
    forEntity(worker)
    starts(Phase.polling)
    ends(Phase.polling, Phase.stopped)
    evidence(_ => "") // uninhabited: never called, so there is no line to write
    steps(
      workerStop ~> stopStep,
      workerResume ~> resumeStep,
      serve ~> serveStep,
    )
