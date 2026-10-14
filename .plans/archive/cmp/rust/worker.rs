//! # The worker entity
//!
//! The worker of one task queue, as an entity of its own rather than a stutter row on the machines
//! of the work it serves. A polling worker serves its queue; the `worker` party stops it and
//! resumes it. The machine is the one a composition synchronizes with: a workflow or an operation
//! whose progress needs a worker names `serve` beside its own action, and a stopped worker has no
//! row for it.
//!
//! The module declares no set, Case, or Query: nothing here is realized on its own, and the
//! Properties about a worker are the cross-entity ones a composition states.

use umpire::{action, entity, machine, Finite, Step};

// authoring: worker

// ### Entities and domains

/// A worker is named by the task queue it polls: the handler's worker and the workflow's worker are
/// two instances of this entity, told apart by their queue.
entity! { worker key: taskQueue }

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum Phase {
    Polling,
    Stopped,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub struct WorkerState {
    pub phase: Phase,
}

#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum WorkerOutcome {
    Accepted,
}

/// A worker records nothing of its own: its stop and resume are faults the Run records against no
/// entity, and what it serves is recorded by the work it serves. An enum with no variants derives
/// `Finite` with `CARDINALITY = 0`.
#[derive(Clone, Copy, PartialEq, Eq, Hash, Debug, Finite)]
pub enum WorkerFact {}

pub type WorkerStep = Step<WorkerState, WorkerOutcome, WorkerFact>;

// ### The actions
//
// The two faults are the `worker` party's and name no entity, as the outage machine spells them.
// The serve action is the worker's own and takes no input, so a composition may synchronize it
// with an action of any class.

action! { workerStop
    party: worker
}

action! { workerResume
    party: worker
}

action! { serve
    party: worker
    on: worker
}

// authoring: polling

// ### The machine

/// A polling worker stops; a stopped one has nothing to stop.
pub fn stop_step(state: &WorkerState) -> Vec<WorkerStep> {
    match state.phase {
        Phase::Polling => vec![Step { outcome: WorkerOutcome::Accepted, state: WorkerState { phase: Phase::Stopped }, facts: vec![] }],
        Phase::Stopped => vec![],
    }
}

/// A stopped worker resumes polling; a polling one has nothing to resume.
pub fn resume_step(state: &WorkerState) -> Vec<WorkerStep> {
    match state.phase {
        Phase::Stopped => vec![Step { outcome: WorkerOutcome::Accepted, state: WorkerState { phase: Phase::Polling }, facts: vec![] }],
        Phase::Polling => vec![],
    }
}

/// A polling worker serves and keeps polling; a stopped one serves nothing.
pub fn serve_step(state: &WorkerState) -> Vec<WorkerStep> {
    match state.phase {
        Phase::Polling => vec![Step { outcome: WorkerOutcome::Accepted, state: *state, facts: vec![] }],
        Phase::Stopped => vec![],
    }
}

/// A worker has no natural end: it may be left polling or stopped.
machine! { polling
    for: worker
    state: WorkerState
    outcome: WorkerOutcome
    facts: WorkerFact
    starts: [Polling]
    ends: [Polling, Stopped]
    steps: {
        workerStop: stop_step,
        workerResume: resume_step,
        serve: serve_step,
    }
}

// authoring: end
