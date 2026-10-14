"""
# The worker entity

The worker of one task queue, as an entity of its own rather than a stutter row on the machines of
the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes it.
The machine is the one a composition synchronizes with: a workflow or an operation whose progress
needs a worker names `serve` beside its own action, and a stopped worker has no row for it.

The module declares no set, Case, or Query: nothing here is realized on its own, and the Properties
about a worker are the cross-entity ones a composition states.
"""
module Worker

using Umpire

# authoring: worker

# ### Entities and domains

# A worker is named by the task queue it polls: the handler's worker and the workflow's worker are
# two instances of this entity, told apart by their queue.
@entity worker begin
    key = taskQueue
end

@enumx Phase polling stopped

Base.@kwdef struct WorkerState
    phase::Phase.T
end

@enumx WorkerOutcome accepted

# A worker records nothing of its own: its stop and resume are faults the Run records against no
# entity, and what it serves is recorded by the work it serves. An empty EnumX has no members, so
# `finite(WorkerFact.T)` is `[]` and `WStep(...; facts = [])` is the only step there is.
@enumx WorkerFact

const WStep = Step{WorkerState,WorkerOutcome.T,WorkerFact.T}

# ### The actions
#
# The two faults are the `worker` party's and name no entity, as the outage machine spells them. The
# serve action is the worker's own and takes no input, so a composition may synchronize it with an
# action of any class.

@action workerStop begin
    party = worker
end

@action workerResume begin
    party = worker
end

@action serve begin
    party = worker
    on = worker
end

# authoring: polling

# ### The machine

"""A polling worker stops; a stopped one has nothing to stop."""
function stopStep(state::WorkerState)::Vector{WStep}
    state.phase == Phase.polling || return WStep[]
    return [WStep(outcome = WorkerOutcome.accepted, state = WorkerState(phase = Phase.stopped))]
end

"""A stopped worker resumes polling; a polling one has nothing to resume."""
function resumeStep(state::WorkerState)::Vector{WStep}
    state.phase == Phase.stopped || return WStep[]
    return [WStep(outcome = WorkerOutcome.accepted, state = WorkerState(phase = Phase.polling))]
end

"""A polling worker serves and keeps polling; a stopped one serves nothing."""
function serveStep(state::WorkerState)::Vector{WStep}
    state.phase == Phase.polling || return WStep[]
    return [WStep(outcome = WorkerOutcome.accepted, state = state)]
end

# A worker has no natural end: it may be left polling or stopped.
@machine polling begin
    var"for" = worker
    state = WorkerState
    outcome = WorkerOutcome.T
    fact = WorkerFact.T
    starts = [polling]
    ends = [polling, stopped]
    steps = (
        workerStop = stopStep,
        workerResume = resumeStep,
        serve = serveStep,
    )
end

# authoring: end

end # module Worker
