/* The worker entity: the worker of one task queue, as an entity of its own rather than a stutter
 * row on the machines of the work it serves. A polling worker serves its queue; the worker party
 * stops it and resumes it. The machine is the one a composition synchronizes with: a workflow or an
 * operation whose progress needs a worker names serve beside its own action, and a stopped worker
 * has no row for it.
 *
 * The package declares no set, Case or Query: nothing here is realized on its own, and the
 * Properties about a worker are the cross-entity ones a composition states.
 *
 * Read top to bottom: the types; the signature (the entity, and the worker party with its
 * actions); then Polling, the worker's one machine.
 */
package temporal
package shared.worker

import umpire.*

// The declarations were written in Worker.scala, and every Case that stops a worker carries the
// Definition IDs they had there.
given DefinitionScope = DefinitionScope("temporal.worker.Worker$package$")

given Family = Family("temporal.worker")

// ### Types

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

// ### Signature

/**
 * Named by the task queue it polls: the handler's worker and the workflow's worker are two
 * instances of this entity, told apart by their queue.
 */
val entity = Entity("worker", key = "taskQueue")

given Ok[Outcome] = Ok(Outcome.accepted)

/**
 * The worker party, which stops and resumes the worker, and serves its queue. Its two faults name
 * no entity, as the outage machine spells them. The serve action is the worker's own and takes no
 * input, so a composition may synchronize it with an action of any class.
 *
 * The actions keep the names they had as the file's top-level vals: an action's Definition ID is
 * its val's owner and name, an actor object is transparent to it, and every Case that stops a
 * worker carries these IDs. fn-126 R18 renames them. A feature that has actions of its own taken by this
 * party imports it under another name (`import shared.worker.{worker as process}`).
 */
object worker extends Actor:
  val workerStop = action(this)
  val workerResume = action(this)
  val serve = action(this) on entity

// ### The machine

object Polling:
  object effects:
    /** A polling worker stops; a stopped one has nothing to stop. */
    def stopStep(s: State) =
      if s.phase != Phase.polling then disabled else enter(State(Phase.stopped))

    /** A stopped worker resumes polling; a polling one has nothing to resume. */
    def resumeStep(s: State) =
      if s.phase != Phase.stopped then disabled else enter(State(Phase.polling))

    /** A polling worker serves and keeps polling; a stopped one serves nothing. */
    def serveStep(s: State) =
      if s.phase != Phase.polling then disabled else stay(s)

  /** The worker. A worker has no natural end: it may be left polling or stopped. */
  val polling = machine[State, Outcome, Fact] {
    forEntity(entity)
    starts(State(Phase.polling))
    ends(_ => true)
    steps(
      worker.workerStop ~> effects.stopStep,
      worker.workerResume ~> effects.resumeStep,
      worker.serve ~> effects.serveStep
    )
  }
