/* The worker entity: the worker of one task queue, as an entity of its own rather than a stutter
 * row on the machines of the work it serves. A polling worker serves its queue; the worker party
 * stops it and resumes it. The machine is the one a composition synchronizes with: a workflow or an
 * operation whose progress needs a worker names serve beside its own action, and a stopped worker
 * has no row for it.
 *
 * The package declares no set, Case or Query: nothing here is realized on its own, and the
 * Properties about a worker are the cross-entity ones a composition states.
 *
 *
 */
package temporal
package worker

import umpire.*

val Family: umpire.Family = umpire.Family("temporal.worker")

/** The worker party, which stops and resumes the worker. */
val party: Party = Party("worker")

// ### Entities and domains

/**
 * Named by the task queue it polls: the handler's worker and the workflow's worker are two
 * instances of this entity, told apart by their queue.
 */
val entity: Entity = Entity("worker", key = "taskQueue")

enum Phase derives Finite:
  case polling, stopped

final case class State(phase: Phase) derives Finite

enum Outcome derives Finite:
  case accepted

/**
 * A worker records nothing of its own. Its stop and resume are faults the Run records against no
 * entity, and what it serves is recorded by the work it serves.
 */
type Fact = Nothing

type WorkerStep = Step[State, Outcome, Fact]

// ### The actions
//
// The two faults are the worker party's and name no entity, as the outage machine spells them. The
// serve action is the worker's own and takes no input, so a composition may synchronize it with an
// action of any class.
//
// The names keep their `worker` prefix although a feature could read them as `worker.stop`: an
// action's Definition ID is its val's owner and name (umpire.DefinitionScope pins the owner, never
// the name), and every Case that stops a worker carries these IDs. A feature imports them by name
// (`import worker.{serve, workerStop}`) rather than writing `worker.workerStop` or an alias.

val workerStop = action("workerStop", party)
val workerResume = action("workerResume", party)
val serve = action("serve", party) on entity

// ### The machine

/** A polling worker stops; a stopped one has nothing to stop. */
def stopStep(s: State): List[WorkerStep] =
  if s.phase != Phase.polling then Nil else List(Step(Outcome.accepted, State(Phase.stopped)))

/** A stopped worker resumes polling; a polling one has nothing to resume. */
def resumeStep(s: State): List[WorkerStep] =
  if s.phase != Phase.stopped then Nil else List(Step(Outcome.accepted, State(Phase.polling)))

/** A polling worker serves and keeps polling; a stopped one serves nothing. */
def serveStep(s: State): List[WorkerStep] =
  if s.phase != Phase.polling then Nil else List(Step(Outcome.accepted, s))

/** The worker. A worker has no natural end: it may be left polling or stopped. */
val polling: Machine[State, Outcome, Fact] = machine[State, Outcome, Fact](Family, "polling") {
  forEntity(entity)
  starts(State(Phase.polling))
  ends(_ => true)
  steps(workerStop ~> stopStep, workerResume ~> resumeStep, serve ~> serveStep)
}
