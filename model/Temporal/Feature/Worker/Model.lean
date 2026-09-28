import Temporal.Case.Conventions

/-!
# The worker entity

The worker of one task queue, as an entity of its own rather than a stutter row on the machines of
the work it serves. A polling worker serves its queue; the `worker` party stops it and resumes it.
The machine is the one a composition synchronizes with: a workflow or an operation whose progress
needs a worker names `serve` beside its own action, and a stopped worker has no row for it.

The module declares no set, Case, or Query: nothing here is realized on its own, and the Properties
about a worker are the cross-entity ones a composition states.
-/

namespace Temporal.Feature.Worker

open Umpire
open Umpire.Command

/-! ### Entities and domains -/

/-- A worker is named by the task queue it polls: the handler's worker and the workflow's worker are
two instances of this entity, told apart by their queue. -/
entity worker
  key: taskQueue

enum Phase
  | polling
  | stopped

structure WorkerState where
  phase : Phase
  deriving BEq, DecidableEq, Repr, Finite

enum WorkerOutcome
  | accepted

/-- A worker records nothing of its own: its stop and resume are faults the Run records against no
entity, and what it serves is recorded by the work it serves. -/
inductive WorkerFact
  deriving BEq, DecidableEq, Repr, Finite

/-! ### The actions

The two faults are the `worker` party's and name no entity, as the outage machine spells them. The
serve action is the worker's own and takes no input, so a composition may synchronize it with an
action of any class. -/

action workerStop
  party: worker

action workerResume
  party: worker

action serve
  party: worker
  on: worker

/-! ### The machine -/

/-- A polling worker stops; a stopped one has nothing to stop. -/
def stopStep (state : WorkerState) :
    List (Step WorkerState WorkerOutcome WorkerFact) :=
  if state.phase != .polling then [] else
  [{ outcome := .accepted, state := { phase := .stopped }, facts := [] }]

/-- A stopped worker resumes polling; a polling one has nothing to resume. -/
def resumeStep (state : WorkerState) :
    List (Step WorkerState WorkerOutcome WorkerFact) :=
  if state.phase != .stopped then [] else
  [{ outcome := .accepted, state := { phase := .polling }, facts := [] }]

/-- A polling worker serves and keeps polling; a stopped one serves nothing. -/
def serveStep (state : WorkerState) :
    List (Step WorkerState WorkerOutcome WorkerFact) :=
  if state.phase != .polling then [] else
  [{ outcome := .accepted, state, facts := [] }]

/-- A worker has no natural end: it may be left polling or stopped. -/
machine polling
  for: worker
  state: WorkerState
  starts: [polling]
  ends: [polling, stopped]
  steps:
    workerStop: stopStep
    workerResume: resumeStep
    serve: serveStep

end Temporal.Feature.Worker
