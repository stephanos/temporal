import Umpire.Case.Producer

/-!
Pins for the parts of the generic Producer that are decided before any checked Model is in hand:
the identities a fixture name derives, and how a declared spelling resolves against the checked
vocabulary. The derivation itself is pinned end to end by the checked-in fixtures, which regenerate
through `produce`.
-/

namespace Umpire.Case.Tests.Producer

open Umpire.Case.Producer

/-- The fixture name is the only identity slot: everything else derives from it. -/
private def derived : Identity := Identity.ofFixture "temporal.case" "async-nexus"

#guard derived.caseId == "temporal.case.async-nexus"
#guard derived.programId == "temporal.case.async-nexus.program"
#guard derived.contractId == "temporal.case.async-nexus.contract"
#guard derived.runScope == "async-nexus"

/-- A stated Program ID overrides the derivation without disturbing the others; that is how a Case
whose bytes predate the convention keeps them. -/
private def stated : Identity := {
  caseId := "temporal.case.async-nexus-success"
  fixture := "async-nexus"
  programId := "temporal.case.async-nexus.program" }

#guard stated.contractId == "temporal.case.async-nexus-success.contract"
#guard stated.runScope == "async-nexus"

private def stateId : DefinitionId := .of "example.state"
private def actionId : DefinitionId := .of "example.action"

private def declared : Case.Producer.Vocabulary := {
  «states» := [ModelValue.named stateId "pending", ModelValue.named stateId "completed"]
  «actions» := [ModelValue.named actionId "awaitCompletion"]
  «outcomes» := []
  «facts» := [] }

#guard (Vocabulary.namedState declared "completed").value == "completed"
#guard (Vocabulary.stateAt declared 0).value == "pending"

/-! An unknown spelling resolves to the unknown Model Value rather than to a neighbour, so the
clause referencing it is rejected at admission instead of silently addressing the wrong member. -/
#guard Vocabulary.namedAction declared "awaitStart" == unknownValue
#guard Vocabulary.outcomeAt declared 0 == unknownValue


/-! ### The outage-order rule

Derived from the assembled Program alone: a role whose faults stop the worker and later resume it
gets one bounded-liveness rule; a stop without a resume, or a resume before any stop, gets none. -/

section OutageOrder

open Testpilot.Authoring
open temporal.server.api.testpilot.v1

private def faultNode (instructionId roleId : String) (kind : FaultKind) : InstructionNode :=
  Program.node instructionId (Program.injectFault roleId kind)

private def faultProgram (nodes : Array InstructionNode) : Program :=
  Program.make "example.program" #[Program.role "queue" .ROLE_KIND_TASK_QUEUE] #[] #[]
    #[Program.controller "controller" nodes] (Program.cleanup "cleanup" #[])

private def stopThenResume : Program := faultProgram #[
  faultNode "stop" "queue" .FAULT_KIND_WORKER_STOP,
  Program.node "work" (Program.finish (Expr.literal (Value.text "done"))),
  faultNode "resume" "queue" .FAULT_KIND_WORKER_RESUME]

#guard injectedFaults stopThenResume ==
  [("queue", .FAULT_KIND_WORKER_STOP), ("queue", .FAULT_KIND_WORKER_RESUME)]

/- One rule, named as the checked-in outage Case always named it: stop then resume on the one role,
expired by the event-count deadline. -/
#guard ((outageOrderRules stopThenResume 16).map fun rule =>
    (rule.rule_id, rule.kind, rule.initial_state_id,
      rule.states.toList.map fun state => (state.state_id, state.status),
      rule.transitions.toList.map fun transition =>
        (transition.transition_id, transition.source_state_id, transition.target_state_id),
      rule.deadline.bind fun deadline => deadline.bound.map fun bound =>
        ((match bound with | .rule_events events => some events | _ => none),
          deadline.violation_state_id))) ==
  [("worker-outage-order", .CONTRACT_RULE_KIND_BOUNDED_LIVENESS, "awaiting-stop",
    [("awaiting-stop", .CONTRACT_STATE_STATUS_PENDING),
     ("stopped", .CONTRACT_STATE_STATUS_PENDING),
     ("resumed", .CONTRACT_STATE_STATUS_SATISFIED),
     ("expired", .CONTRACT_STATE_STATUS_VIOLATED)],
    [("observe-stop", "awaiting-stop", "stopped"), ("observe-resume", "stopped", "resumed")],
    some (some 16, "expired"))]

/- Both transitions read the recorded fault's role and kind off the Run Event payload. -/
#guard ((outageOrderRules stopThenResume 16).flatMap fun rule =>
    rule.transitions.toList.map fun transition =>
      (transition.event_filter.map (·.kinds.toList), transition.support_kind)) ==
  [(some [.RUN_EVENT_KIND_FAULT_INJECTED], .CONTRACT_SUPPORT_KIND_MATCHING_EVENT),
   (some [.RUN_EVENT_KIND_FAULT_INJECTED], .CONTRACT_SUPPORT_KIND_MATCHING_EVENT)]

/- A stop with no resume, and a resume before any stop, derive no rule. -/
#guard (outageOrderRules (faultProgram #[faultNode "stop" "queue" .FAULT_KIND_WORKER_STOP])
  16).isEmpty
#guard (outageOrderRules (faultProgram #[
  faultNode "resume" "queue" .FAULT_KIND_WORKER_RESUME,
  faultNode "stop" "queue" .FAULT_KIND_WORKER_STOP]) 16).isEmpty

/- A second role stopped and resumed gets its own rule, carrying its ordinal; a role only stopped
gets none. -/
#guard ((outageOrderRules (faultProgram #[
    faultNode "stop-a" "queue-a" .FAULT_KIND_WORKER_STOP,
    faultNode "stop-b" "queue-b" .FAULT_KIND_WORKER_STOP,
    faultNode "stop-c" "queue-c" .FAULT_KIND_WORKER_STOP,
    faultNode "resume-b" "queue-b" .FAULT_KIND_WORKER_RESUME,
    faultNode "resume-a" "queue-a" .FAULT_KIND_WORKER_RESUME]) 8).map (·.rule_id)) ==
  ["worker-outage-order", "worker-outage-order-2"]

end OutageOrder

end Umpire.Case.Tests.Producer
