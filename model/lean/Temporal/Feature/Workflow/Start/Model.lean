import Temporal.Case.Syntax

/-!
# The workflow-start Model

One workflow started by a caller, and the one thing the product promises about the request that
started it: the workflow type the caller submitted is the workflow type the started event records.
That promise is a field relation -- a claim about two typed fields of one step, the request's
`workflow_type.name` and the recorded event's -- and it is what the typed unary example (fn-83)
wrote by hand as a field Property; here it is one `relates:` line, resolved against the generated
schema while the file compiles and lowered to the same Contract read.
-/

namespace Temporal.Feature.Workflow.Start

open Umpire
open Umpire.Command

/-! ### Entities and domains -/

/-- A workflow is named, across every event of its history, by the run that first executed it. -/
entity workflow
  key: firstExecutionRunId

enum Phase
  | pending
  | started

structure StartState where
  phase : Phase
  deriving BEq, DecidableEq, Repr, Finite

enum StartOutcome
  | accepted

enum StartFact
  | workflowExecutionStarted

/-! ### The action

The start command carries the generated request; its `workflow_type.name` is the field the
relation below reads. -/

action startWorkflow
  party: caller
  creates: workflow
  schema: temporal.api.workflowservice.v1.StartWorkflowExecutionRequest

/-! ### The machine -/

/-- A pending workflow starts once, and the started event records it. -/
def startStep (state : StartState) :
    List (Step StartState StartOutcome StartFact) :=
  if state.phase != .pending then [] else
  [{ outcome := .accepted, state := { phase := .started }, facts := [.workflowExecutionStarted] }]

machine workflowStart
  for: workflow
  state: StartState
  starts: [pending]
  ends: [started]
  evidence:
    workflowExecutionStarted: workflowExecutionStarted
  steps:
    startWorkflow: startStep

/-! ### What the machine promises -/

/- The start settles the workflow as started, and the started event records it. -/
property startRecorded
  machine: workflowStart
  when: startWorkflow
  holds: fun step =>
    step.state.phase == .started && step.facts.contains .workflowExecutionStarted

/- The workflow type the request submitted is the one the started event recorded: a relation over
the request's field and the recorded event's, each resolved against the generated schema. -/
property submittedTypeIsRecorded
  machine: workflowStart
  when: startWorkflow
  relates: startWorkflow.input.workflow_type.name = workflowExecutionStarted.workflow_type.name

/-! ### The path, the Query and the set -/

scenario startOnce
  model: workflowStart
  starts: pending
  actions: [startWorkflow]

limits one
  steps: 1
  actions: 1
  search: 8

query started
  find: startRecorded
  in: startOnce
  limits: one

set workflowStartTests
  purpose: functional
  bind:
    caller: driven
  queries: [started]

case workflowStartCases
  realizes workflowStartTests
  as Temporal.Case.Realization.workflowStart

end Temporal.Feature.Workflow.Start
