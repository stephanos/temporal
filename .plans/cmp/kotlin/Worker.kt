/*
 * # The worker entity
 *
 * The worker of one task queue, as an entity of its own rather than a stutter row on the machines of
 * the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes it.
 * The machine is the one a composition synchronizes with: a workflow or an operation whose progress
 * needs a worker names `serve` beside its own action, and a stopped worker has no row for it.
 *
 * The module declares no set, Case, or Query: nothing here is realized on its own, and the Properties
 * about a worker are the cross-entity ones a composition states.
 *
 * Declared as an `object` so a Model file reads `Worker.polling` and `Worker.workerStop` beside its own
 * `workerStop`, as the Lean file does with `Worker.polling`; Kotlin has no package alias.
 */
package temporal.feature.worker

import umpire.Step
import umpire.action
import umpire.entity
import umpire.machine

private typealias WorkerSteps = List<Step<Worker.WorkerState, Worker.WorkerOutcome, Worker.WorkerFact>>

object Worker {

    // ---- Entities and domains -------------------------------------------------------------------

    /**
     * A worker is named by the task queue it polls: the handler's worker and the workflow's worker are
     * two instances of this entity, told apart by their queue.
     */
    val worker = entity("worker") { key = "taskQueue" }

    enum class Phase { Polling, Stopped }

    data class WorkerState(val phase: Phase)

    enum class WorkerOutcome { Accepted }

    /**
     * A worker records nothing of its own: its stop and resume are faults the Run records against no
     * entity, and what it serves is recorded by the work it serves. A sealed interface with no
     * implementations is the empty type; `Finite.of<WorkerFact>()` lists nothing.
     */
    sealed interface WorkerFact

    // ---- The actions ------------------------------------------------------------------------------
    //
    // The two faults are the `worker` party's and name no entity, as the outage machine spells them.
    // The serve action is the worker's own and takes no input, so a composition may synchronize it
    // with an action of any class. The entity `worker` shadows the party `worker` inside this object,
    // so the party is written qualified.

    val workerStop = action("workerStop") { party = umpire.worker }

    val workerResume = action("workerResume") { party = umpire.worker }

    val serve = action("serve") {
        party = umpire.worker
        on = worker
    }

    // ---- The machine ------------------------------------------------------------------------------

    /** A polling worker stops; a stopped one has nothing to stop. */
    fun stopStep(state: WorkerState): WorkerSteps =
        if (state.phase != Phase.Polling) emptyList()
        else listOf(Step(WorkerOutcome.Accepted, WorkerState(Phase.Stopped), emptyList()))

    /** A stopped worker resumes polling; a polling one has nothing to resume. */
    fun resumeStep(state: WorkerState): WorkerSteps =
        if (state.phase != Phase.Stopped) emptyList()
        else listOf(Step(WorkerOutcome.Accepted, WorkerState(Phase.Polling), emptyList()))

    /** A polling worker serves and keeps polling; a stopped one serves nothing. */
    fun serveStep(state: WorkerState): WorkerSteps =
        if (state.phase != Phase.Polling) emptyList()
        else listOf(Step(WorkerOutcome.Accepted, state, emptyList()))

    /** A worker has no natural end: it may be left polling or stopped. */
    val polling = machine<WorkerState, WorkerOutcome, WorkerFact>("polling") {
        entity = worker
        starts(WorkerState(Phase.Polling))
        ends { true }
        steps {
            workerStop runs ::stopStep
            workerResume runs ::resumeStep
            serve runs ::serveStep
        }
    }
}
