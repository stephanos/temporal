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
 * actions); then Polling, the worker's one machine object.
 */
package temporal
package shared.worker

import scala.annotation.unused
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
 * worker carries these IDs; fn-126 R18 renames them. A feature with actions of its own that this
 * party takes imports it under another name (`import shared.worker.{worker as process}`).
 */
object worker extends Actor:
  val workerStop = action(this)
  val workerResume = action(this)
  val serve = action(this) on entity

// ### The machine

/** The worker. A worker has no natural end: it may be left polling or stopped. */
object Polling extends Machine[State, Outcome, Fact]:
  val entity = shared.worker.entity
  val init = State(Phase.polling)
  def end(@unused state: State) = true

  object effects extends Section:
    def stop(@unused s: State) = enter(State(Phase.stopped))

    def resume(@unused s: State) = enter(State(Phase.polling))

    /** A polling worker serves and keeps polling. */
    def serve(s: State) = stay(s)

  // A polling worker stops and serves; a stopped one resumes, and has nothing to stop or serve.
  object rules extends Rules(_.phase):
    in(Phase.polling)(worker.workerStop ~> effects.stop)
    in(Phase.stopped)(worker.workerResume ~> effects.resume)
    in(Phase.polling)(worker.serve ~> effects.serve)
