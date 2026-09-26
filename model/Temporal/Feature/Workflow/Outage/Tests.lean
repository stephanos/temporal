import Temporal.Feature.Workflow.Outage.Model

/-!
# What the worker-outage Model says

The pins the hand-written outage Case carried, read off the command's output: the machine, the
one path, the produced Case's Program (the stop before the start, the resume after, the wait and
the history read), the outage-order rule the Producer derived from that Program with its
event-count deadline, the one projection rule confirming the four steps at once, and the three
Known Gaps naming the steps the completed event infers rather than observes.
-/

namespace Temporal.Feature.Workflow.Outage.Tests

open Umpire
open Umpire.Case
open Umpire.Command
open Temporal.Feature.Workflow.Outage
open temporal.server.api.testpilot.v1 hiding ModelValue SourceLocation

/-! ### The machine -/

#guard workflowOutage.table.states.length == 3
#guard workflowOutage.actionKeys == #["awaitCompletion", "startWorkflow", "workerResume", "workerStop"]
#guard workflowOutage.stuck == none

/-- info: 'Temporal.Feature.Workflow.Outage.workflowOutage' depends on axioms: [propext] -/
#guard_msgs in
#print axioms workflowOutage

/-! ### The Query -/

/- The one path is found: stop, start, resume, wait. -/
#guard (match survived with
  | .ok checked => checked.run.result.outcome.name
  | .error _ => "admission failed") == "found"

#guard (match survived with
  | .ok checked => (checked.witness.map fun witness =>
      witness.trace.steps.map (·.selectedAction.value)).getD []
  | .error _ => []) == ["workerStop", "startWorkflow", "workerResume", "awaitCompletion"]

/-! ### The Case -/

#guard workerOutageCases.survived.identity.fixture == "workerOutageTests-survived"
#guard workerOutageCases.survived.identity.caseId == "temporal.case.workerOutageTests.survived"

private def produced : Option Case := workerOutageCases.survived.toOption

#guard produced.isSome

private def instructionIds (entrypointId : String) : List String :=
  ((produced.bind fun output => output.program.bind fun (program : Program) =>
    program.entrypoints.find? (·.entrypoint_id == entrypointId)).map fun entrypoint =>
      entrypoint.instructions.toList.map (·.instruction_id)).getD []

/- The controller stops the worker, starts the workflow, resumes the worker, waits for the close and
reads history; the workflow finishes. -/
#guard instructionIds "controller" ==
  ["stop-worker", "start-workflow", "resume-worker", "await-close", "history"]
#guard instructionIds "workflow" == ["finish-workflow"]

/- The two faults are injected on the Case's own task-queue role, stop then resume. -/
#guard (produced.map fun output =>
  (output.program.map Producer.injectedFaults).getD []) ==
  some [(Temporal.Case.Support.taskQueueRole, .FAULT_KIND_WORKER_STOP),
    (Temporal.Case.Support.taskQueueRole, .FAULT_KIND_WORKER_RESUME)]

/-- The rules of the Contract: id, kind, terminal states and the event-count deadline. -/
private def rules : List (String × ContractRuleKind × List (String × ContractStateStatus) ×
    Option (Option Int64 × String)) :=
  ((produced.bind (·.contract)).map fun contract => contract.rules.toList.map fun rule =>
    (rule.rule_id, rule.kind,
      rule.states.toList.map fun state => (state.state_id, state.status),
      rule.deadline.bind fun deadline => deadline.bound.map fun bound =>
        ((match bound with | .rule_events events => some events | _ => none),
          deadline.violation_state_id))).getD []

/- The outage-order rule is the one monitor rule: bounded liveness from `awaiting-stop` through
`stopped` to `resumed`, expired after sixteen evaluated Run Events and never by elapsed time. -/
#guard rules == [("worker-outage-order", .CONTRACT_RULE_KIND_BOUNDED_LIVENESS,
  [("awaiting-stop", .CONTRACT_STATE_STATUS_PENDING), ("stopped", .CONTRACT_STATE_STATUS_PENDING),
   ("resumed", .CONTRACT_STATE_STATUS_SATISFIED), ("expired", .CONTRACT_STATE_STATUS_VIOLATED)],
  some (some 16, "expired"))]

/- The completed event confirms all four steps at once: the three silent ones before it and the
wait that it records. -/
#guard ((produced.bind fun output => output.contract.bind (·.«correlated»)).map fun capability =>
  capability.projection_rules.toList.map fun rule => (rule.kind, rule.outputs.size)) ==
  some [("evidence.workflowExecutionCompleted", 4)]

/- Each silent step is a capability Known Gap naming the step the Contract infers. -/
#guard ((produced.bind (·.provenance)).map fun provenance =>
  provenance.known_gaps.toList.map fun gap => (gap.kind, gap.code)) == some [
  (.KNOWN_GAP_KIND_CAPABILITY, "temporal.workflow.outage.action.workflowOutage.startWorkflow.unobserved"),
  (.KNOWN_GAP_KIND_CAPABILITY, "temporal.workflow.outage.action.workflowOutage.workerResume.unobserved"),
  (.KNOWN_GAP_KIND_CAPABILITY, "temporal.workflow.outage.action.workflowOutage.workerStop.unobserved")]

/- The one evidence kind is the completed event, read out of history and keyed by the workflow task
that completed the workflow. -/
#guard ((produced.bind (·.program)).map fun program =>
  program.evidence.toList.map fun declaration => (declaration.evidence_id, declaration.operation)) ==
  some [("evidence.workflowExecutionCompleted",
    "attributes<workflow_execution_completed_event_attributes>.workflow_task_completed_event_id")]

end Temporal.Feature.Workflow.Outage.Tests
