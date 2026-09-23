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


/-! ### The results a witnessed row could have taken

A lamp whose `toggle` from `warm` can end `bright` (recording `high`) or `stuck` (recording `low`):
the witness takes `bright`, and the alternative's kind is declared and projected to its own row,
inheriting the silent step before it. A kind another row would record rejects by name. -/

private def lampAction (name : String) : ModelValue :=
  { definitionId := DefinitionId.of s!"lamp.action.{name}", value := name }
private def lampState (name : String) : ModelValue :=
  { definitionId := DefinitionId.of s!"lamp.state.{name}", value := name }
private def lampOutcome (name : String) : ModelValue :=
  { definitionId := DefinitionId.of s!"lamp.outcome.{name}", value := name }
private def lampFact (name : String) : ModelValue :=
  { definitionId := DefinitionId.of s!"lamp.fact.{name}", value := name }

private def lampSource (kind : String) : EvidenceSource :=
  { eventKind := kind
    recorded := .runEvent .RUN_EVENT_KIND_INSTRUCTION_COMPLETED
    operationKeyPath := "operation"
    kindId := DefinitionId.of s!"lamp.evidence.{kind}"
    sourceId := DefinitionId.of "lamp.source" }

private def lampSources : List EvidenceSource :=
  [lampSource "completed", lampSource "failed", lampSource "warmed"]

private def lampCatalog : List (String × String) :=
  [("high", "completed"), ("low", "failed"), ("warm", "warmed")]

private def lampStep (stateName outcomeName : String) (facts : List String) :
    Step ModelValue ModelValue ModelValue :=
  { «state» := lampState stateName, «outcome» := lampOutcome outcomeName,
    «facts» := facts.map lampFact }

private def bright := lampStep "bright" "changed" ["high"]
private def stuck := lampStep "stuck" "held" ["low"]
private def warmed := lampStep "warm" "warmed" []
private def dimmed := lampStep "dim" "changed" ["low"]

/-- `warm` from `dim` is silent; `toggle` from `warm` ends `bright` or `stuck`; a `toggle` from
`bright` records `low` too, which is what makes the kind ambiguous when that row is witnessed. -/
private def lampResults (prior action : ModelValue) : List (Step ModelValue ModelValue ModelValue) :=
  if action == lampAction "warm" && prior == lampState "dim" then [warmed]
  else if action == lampAction "toggle" && prior == lampState "warm" then [bright, stuck]
  else if action == lampAction "toggle" && prior == lampState "bright" then [dimmed]
  else []

private def witnessStep (actionName : String) (taken : Step ModelValue ModelValue ModelValue) :
    ModelTraceStep ModelValue ModelValue ModelValue ModelValue :=
  { selectedAction := lampAction actionName, «outcome» := taken.outcome, «state» := taken.state,
    «facts» := taken.facts }

private def witnessSteps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue) :=
  [witnessStep "warm" warmed, witnessStep "toggle" bright]

/-- The witness's own rule: `completed` confirms the silent `warm` and the `toggle` that ended bright. -/
private def witnessRule : ResolvedRule :=
  ({ action := lampAction "toggle", source := lampSource "completed" },
    [(lampAction "warm", warmed), (lampAction "toggle", bright)])

private def lampSourceLocation : SourceLocation := { path := "lamp.lean" }

private def alternatives (sources : List EvidenceSource)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (resolved : List ResolvedRule) : Except Umpire.Case.Compiler.Error (List ResolvedRule) :=
  alternativeRules lampSourceLocation sources lampResults lampCatalog (lampState "dim") steps resolved

private def rendered (rules : List ResolvedRule) : List (String × String × List (String × String)) :=
  rules.map fun rule => (rule.1.source.eventKind, rule.1.action.value,
    rule.2.map fun (action, result) => (action.value, result.state.value))

/-! The alternative's kind is declared once more, projected to its own row, after the witness's own
rule, and it confirms the same silent step before the row. -/
#guard (alternatives lampSources witnessSteps [witnessRule]).toOption.map rendered ==
  some [("completed", "toggle", [("warm", "warm"), ("toggle", "bright")]),
        ("failed", "toggle", [("warm", "warm"), ("toggle", "stuck")])]

/-! A row with one result adds nothing: the rules are the witness's, byte for byte. -/
#guard (alternatives lampSources (witnessSteps.take 1) []).toOption.map rendered == some []

private def constructOf : Except Umpire.Case.Compiler.Error (List ResolvedRule) → Option String
  | .error error => some error.construct
  | .ok _ => none

/-! A kind two rows would record rejects by name: when the witness also takes the toggle from
bright, whose one result records `low`, the `failed` kind would confirm both rows. -/
#guard constructOf (alternatives lampSources (witnessSteps ++ [witnessStep "toggle" dimmed])
    [witnessRule, ({ action := lampAction "toggle", source := lampSource "failed" },
      [(lampAction "toggle", dimmed)])]) == some "evidence.kind-ambiguous"

/-! A kind the realization does not admit rejects by name. -/
#guard constructOf (alternatives [lampSource "completed"] witnessSteps [witnessRule]) ==
  some "evidence.kind-unknown"

end Umpire.Case.Tests.Producer
