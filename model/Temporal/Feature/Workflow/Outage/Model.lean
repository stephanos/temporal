import Temporal.Case.Syntax

/-!
# The worker-outage Model

One workflow started while the worker of its own task queue is stopped, and the one thing the
product promises about it: the work survives the outage. The controller stops the worker, starts
the workflow while nothing is polling its queue, resumes the worker, and waits for the workflow the
resumed worker then completes. The two faults are actions of the `worker` party; the Run records
them, but nothing recorded names the workflow, so the machine keeps its state and records nothing
at either, and the Case says so in a Known Gap. Whether the faults happen in that order, and the
resume is not long in coming, is the outage-order rule the Producer derives from the Program: a
bounded-liveness rule over the recorded faults whose deadline is an event count, never elapsed
time, so a slow runner cannot turn a healthy outage into a violated one.
-/

namespace Temporal.Feature.Workflow.Outage

open Umpire
open Umpire.Command

/-! ### Entities and domains -/

/-- A workflow the outage is survived by is named by the workflow task that completed it: the one
task that can only have been dispatched once the worker was back. -/
entity workflow
  key: workflowTaskCompletedEventId

enum Phase
  | pending
  | started
  | completed

structure OutageState where
  phase : Phase
  deriving BEq, DecidableEq, Repr, Finite

enum OutageOutcome
  | accepted

enum OutageFact
  | workflowExecutionCompleted

/-! ### The actions

The start command carries the generated request. The worker's two faults name no entity: the Run
records them and nothing recorded names the workflow. The caller's wait for completion is what the
completed event confirms. -/

action startWorkflow
  party: caller
  creates: workflow
  schema: temporal.api.workflowservice.v1.StartWorkflowExecutionRequest

action workerStop
  party: worker

action workerResume
  party: worker

action awaitCompletion
  party: caller
  on: workflow

/-! ### The machine -/

/-- A pending workflow starts once. The started event is recorded, but it names the workflow by a
run id no later event carries, so the start is confirmed by the completion it leads to rather than
observed on its own. -/
def startStep (state : OutageState) :
    List (Step OutageState OutageOutcome OutageFact) :=
  if state.phase != .pending then [] else
  [{ outcome := .accepted, state := { phase := .started }, facts := [] }]

/-- A fault the Run records and the workflow does not feel: the step keeps the state and records
nothing, and on a path it is confirmed by the evidence of the step after it. -/
def faultStep (state : OutageState) :
    List (Step OutageState OutageOutcome OutageFact) :=
  [{ outcome := .accepted, state, facts := [] }]

/-- A started workflow completes once, and the completed event records it. -/
def completionStep (state : OutageState) :
    List (Step OutageState OutageOutcome OutageFact) :=
  if state.phase != .started then [] else
  [{ outcome := .accepted, state := { phase := .completed }, facts := [.workflowExecutionCompleted] }]

machine workflowOutage
  for: workflow
  state: OutageState
  starts: [pending]
  ends: [completed]
  evidence:
    workflowExecutionCompleted: workflowExecutionCompleted
  steps:
    startWorkflow: startStep
    workerStop: faultStep
    workerResume: faultStep
    awaitCompletion: completionStep

/-! ### What the machine promises -/

/- The wait settles the workflow as completed, and the completed event records it: the queued
workflow task survived the outage rather than being lost with the worker. -/
property completes
  machine: workflowOutage
  when: awaitCompletion
  holds: fun step =>
    step.state.phase == .completed && step.facts.contains .workflowExecutionCompleted

/-! ### The path, the Query and the set

The stop precedes the start, so no workflow task is in flight when the worker stops; the resume
follows it, so the task the start queued is dispatched only once the worker is back. -/

scenario outage
  model: workflowOutage
  starts: pending
  actions: [workerStop, startWorkflow, workerResume, awaitCompletion]

limits four
  steps: 4
  actions: 4
  search: 64

query survived
  find: completes
  in: outage
  limits: four

set workerOutageTests
  purpose: functional
  bind:
    caller: driven
    worker: driven
  queries: [survived]

case workerOutageCases
  realizes workerOutageTests
  as Temporal.Case.Realization.workflowOutage

end Temporal.Feature.Workflow.Outage
