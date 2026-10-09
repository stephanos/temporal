// The worker entity: the worker of one task queue, as an entity of its own rather than a stutter
// row on the machines of the work it serves. A polling worker serves its queue; the worker
// stops and resumes. The machine is the one a composition synchronizes with: a workflow or an
// operation whose progress needs a worker names serve beside its own action, and a stopped worker
// has no row for it.
//
// Update this Model independently of the implementation. When conformance fails, ask a human
// rather than fitting the Model to the code.
//
// The package declares no set, Case or Query: nothing here is realized on its own, and the
// Properties about a worker are the cross-entity ones a composition states.
//
// Read top to bottom: the types; the signature (the entity, and the worker actor with its
// actions); then Polling, the worker's one machine object.
package temporal
package actors.worker

import scala.annotation.unused
import umpire.*
import umpire.outcomes.Outcome

// ### Types

enum Phase derives Finite:
  case polling, stopped

final case class State(phase: Phase) derives Finite

// A worker records nothing of its own. Its stop and resume are faults the Run records against no
// entity, and what it serves is recorded by the work it serves.
type Fact = Nothing

// ### Signature

// Named by the task queue it polls: the handler's worker and the workflow's worker are two
// instances of this entity, told apart by their queue.
val entity = Entity("worker", key = "taskQueue")

// The worker, which stops and resumes, and serves its queue. Its stop and resume name no entity, as
// the outage machine spells them. The serve action is the worker's own and takes no input, so a
// composition may synchronize it with an action of any class. A feature with actions of its own
// that this actor takes imports it under another name (`import actors.worker.{worker as process}`).
object worker extends Actor:
  val stop = action(this)
  val resume = action(this)
  val serve = action(this) on entity

// ### The machine

// The worker. A worker has no natural end: it may be left polling or stopped.
object Polling extends Machine[State, Outcome, Fact], Phased[State, Phase](_.phase):
  val init = State(Phase.polling)
  override def end(@unused state: State) = true

  object effects:
    def stop(@unused s: State) = enter(State(Phase.stopped))

    def resume(@unused s: State) = enter(State(Phase.polling))

    // A polling worker serves and keeps polling.
    def serve(s: State) = stay(s)

  // A polling worker stops and serves; a stopped one resumes, and has nothing to stop or serve.
  object rules extends Rules:
    on(worker.stop) {
      when(Phase.polling) ~> effects.stop
    }
    on(worker.resume) {
      when(Phase.stopped) ~> effects.resume
    }
    on(worker.serve) {
      when(Phase.polling) ~> effects.serve
    }
