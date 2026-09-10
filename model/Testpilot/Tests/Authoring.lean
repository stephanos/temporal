import Testpilot.Authoring

/-! Positive construction checks for the complete neutral Testpilot authoring vocabulary. -/

open temporal.server.api.testpilot.v1
open Testpilot.Authoring

namespace Testpilot.Tests.Authoring

private def boolType := Types.singular (Types.scalar .SCALAR_KIND_BOOLEAN)
private def textType := Types.singular (Types.scalar .SCALAR_KIND_TEXT)
private def requestPath := Path.make #[Path.field "request", Path.oneofSelector "payload" "text"]
private def responsePath := Path.make #[Path.field "response", Path.presence "result"]
private def instructionRef := Ref.instruction "workflow" "start"

private def values : Array temporal.server.api.testpilot.v1.Value := #[
  Value.text "text",
  Value.natural 18446744073709551615,
  Value.boolean true,
  Value.bytes (ByteArray.mk #[0, 255]),
  Value.signedInteger (-9223372036854775808),
  Value.unsignedInteger 18446744073709551615,
  Value.floatingPoint 1.5,
  Value.enumeration 1,
  Value.messageValue ({
    type_url := "type.googleapis.com/temporal.server.api.testpilot.v1.FormatVersion",
    value := ByteArray.mk #[8, 1]
  } : google.protobuf.Any),
  Value.list #[Value.text "nested"],
  Value.map #[(Value.text "key", Value.boolean true)]
]

private def types : Array ValueType := #[
  boolType,
  Types.singular (Types.enumeration "example.Enum"),
  Types.singular (Types.messageType "example.Message"),
  Types.singular Types.any,
  Types.singular Types.opaqueCapability,
  Types.repeated (Types.scalar .SCALAR_KIND_TEXT),
  Types.map .SCALAR_KIND_TEXT (Types.scalar .SCALAR_KIND_BOOLEAN)
]

private def paths : Array FieldPathSegment := #[
  Path.field "plain",
  Path.repeated "items",
  Path.mapKey "labels" (Value.text "key"),
  Path.presence "optional",
  Path.oneofSelector "choice" "text"
]

private def programExpressions : Array ProgramExpression :=
  let literal := ProgramExpr.literal (Value.boolean true)
  let slot := ProgramExpr.slot "slot"
  let outcome := ProgramExpr.outcome instructionRef .INSTRUCTION_OUTCOME_FIELD_VALUE
  #[literal, slot, outcome, ProgramExpr.run, ProgramExpr.environment "namespace",
    ProgramExpr.path slot requestPath,
    ProgramExpr.present outcome,
    ProgramExpr.equals literal slot,
    ProgramExpr.compare .COMPARISON_OPERATOR_LESS_THAN literal outcome,
    ProgramExpr.negation literal,
    ProgramExpr.all #[literal, slot],
    ProgramExpr.any #[outcome, literal]]

private def contractExpressions : Array ContractExpression :=
  let literal := ContractExpr.literal (Value.boolean true)
  let observation := ContractExpr.observation "observed"
  let runEvent := ContractExpr.runEvent .RUN_EVENT_FIELD_SEQUENCE
  let capture := ContractExpr.capture "captured"
  #[literal, observation, runEvent, capture,
    ContractExpr.path observation responsePath,
    ContractExpr.present capture,
    ContractExpr.equals literal observation,
    ContractExpr.compare .COMPARISON_OPERATOR_GREATER_THAN runEvent literal,
    ContractExpr.negation literal,
    ContractExpr.all #[observation, capture],
    ContractExpr.any #[runEvent, literal]]

private def instructionLimits := Program.instructionLimits 1000 2 3 4096
private def assignment := Program.requestAssignment requestPath (ProgramExpr.literal (Value.text "x"))
private def environmentAssignment := Program.environmentAssignment requestPath "namespace"
private def environmentAssignmentUsesBinding : Bool :=
  match environmentAssignment.value with
  | some value =>
    match value.expression with
    | some (.environment reference) => reference.binding_id == "namespace"
    | _ => false
  | _ => false
private def projection := Program.responseProjection responsePath .PROJECTION_KIND_ONE
  #[Program.slotTarget "slot", Program.observationTarget "observed"]

private def instructions : Array Instruction := #[
  Program.invokeRPC "endpoint" "/example.Service/Call" #[assignment] #[projection],
  Program.awaitSlot "slot",
  Program.completeNexusOperation "capability" (ProgramExpr.literal (Value.text "done")),
  Program.startNexusOperation "endpoint" "service" "operation" (ProgramExpr.literal (Value.text "input")),
  Program.awaitInstruction instructionRef,
  Program.finish (ProgramExpr.literal (Value.text "result")),
  Program.respondNexus .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
    (ProgramExpr.literal (Value.text "token")) "capability",
  Program.injectFault "queue" .FAULT_KIND_WORKER_STOP
]

private def injectFaultNamesRoleAndKind : Bool :=
  match instructions[7]!.instruction with
  | some (.inject_fault fault) =>
    fault.role_id == "queue" && fault.kind == .FAULT_KIND_WORKER_STOP
  | _ => false

private def node := Program.node "start" instructions[0]!
  instructionLimits
  (dependencies := #[instructionRef])
  (guard := some programExpressions[10]!)
  (outcome := some (Program.outcome #[Program.outcomeField
    .INSTRUCTION_OUTCOME_FIELD_VALUE textType]))
  (reservations := #[Program.reservation "workflow" 1])

private def program : temporal.server.api.testpilot.v1.Program := Program.make "program"
  #[Program.role "endpoint" .ROLE_KIND_ENDPOINT (resourceBindingId := "nexus.endpoint"),
    Program.role "worker" .ROLE_KIND_WORKER (namespaceBindingId := "namespace"),
    Program.role "queue" .ROLE_KIND_TASK_QUEUE
      (namespaceBindingId := "namespace") (resourceBindingId := "task.queue")]
  #[Program.valueSlot "slot" textType, Program.capabilitySlot "capability"]
  #[Program.observation "observed" textType]
  #[Program.controller "controller" #[node],
    Program.workflow "workflow" "Workflow" "worker" "queue" #[node],
    Program.activity "activity" "Activity" "worker" "queue" #[node],
    Program.nexusHandler "handler" "service" "operation" "worker" "queue" #[node]]
  (Program.cleanup "cleanup" #[node])
  (Program.limits 4 16 16 8 4 64 16 8 4096 4096 10000 1000)
  (environment := #[Program.environment "namespace", Program.environment "task.queue",
    Program.environment "nexus.endpoint"])

private def transition := Contract.transition "take" "start" "done"
  #[.RUN_EVENT_KIND_INSTRUCTION_COMPLETED]
  contractExpressions[10]!
  .CONTRACT_SUPPORT_KIND_MATCHING_EVENT
  #[Contract.captureAssignment "captured" "observed"]

private def contract : Contract := Contract.contract "contract" #[
  Contract.rule "safety" .CONTRACT_RULE_KIND_SAFETY "start"
    #[Contract.state "start" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Contract.state "done" .CONTRACT_STATE_STATUS_SATISFIED]
    #[transition]
    (captures := #[Contract.capture "captured" (Contract.scalarCapture .SCALAR_KIND_TEXT),
      Contract.capture "enum" (Contract.enumCapture "example.Enum"),
      Contract.capture "message" (Contract.messageCapture "example.Message")]),
  Contract.rule "liveness" .CONTRACT_RULE_KIND_BOUNDED_LIVENESS "waiting"
    #[Contract.state "waiting" .CONTRACT_STATE_STATUS_NONTERMINAL,
      Contract.state "late" .CONTRACT_STATE_STATUS_VIOLATED]
    #[] (deadline := some (Contract.deadline 1000 "late"))
] (Contract.limits 2 4 2 16 64 1024 3 4096)

private def eventsDeadline : ContractDeadline := Contract.deadlineEvents 3 "late"

private def correlatedPredicate (field : CorrelatedPredicateField) (definitionId value : String) :
    CorrelatedPredicate :=
  { field, definition_id := definitionId, constraint := some (.equals_text value) }

private def correlatedValue (definitionId value : String) : CorrelatedValue :=
  { definition_id := definitionId, value }

private def correlatedCapability : CorrelatedContract :=
  Contract.correlated "projection" "sha256:projection" "evidence" "operation"
    #["run"] #["source"] (correlatedValue "state" "open")
    #[{ prior_state := some (correlatedValue "state" "open"),
        action := some (correlatedValue "action" "request"),
        state := some (correlatedValue "state" "open"),
        outcome := some (correlatedValue "outcome" "accepted") }]
    #[] #[
      Contract.correlatedRule "response" 1 .TRACE_ENDING_PARTIAL
        (correlatedPredicate .CORRELATED_PREDICATE_FIELD_ACTION "action" "request")
        (correlatedPredicate .CORRELATED_PREDICATE_FIELD_OUTCOME "outcome" "accepted")
    ]
    { max_events := 16, max_buffered := 8, max_keys := 8, max_support := 256,
      max_projection_work := 1000000, max_event_bytes := 512, max_semantic_transitions := 32,
      max_obligations := 16, max_obligation_work := 1000000 }

private def correlatedContract : Contract :=
  Contract.contract "correlated" #[] (Contract.limits 0 0 0 16 64 1024 0 0) (some correlatedCapability)

private def verdict := Verdict.make .VERDICT_STATUS_SATISFIED #[
  Verdict.rule "safety" .RULE_VERDICT_STATUS_SATISFIED "done" #[1]
] #[1]

private def run : temporal.server.api.testpilot.v1.Run := Run.make "run" "case" "program" #[
  Run.event 1 10 .RUN_EVENT_KIND_INSTRUCTION_COMPLETED
    (Run.coordinates "workflow" "activation" "start" 1 0) "source"
    (outcome := some (Run.outcome .INSTRUCTION_OUTCOME_STATUS_SUCCEEDED
      (value := some (Value.text "result"))))
    (observations := #[Run.observation "observed" (Value.text "result")])
] .RUN_STATUS_COMPLETED (Run.cleanup .CLEANUP_STATUS_SUCCEEDED) verdict
  #[Run.diagnostic "diagnostic" .RUN_DIAGNOSTIC_KIND_EXECUTION "code" "detail" (some 1)]
  (some 1)

#guard values.size == 11
#guard types.size == 7
#guard paths.size == 5
#guard programExpressions.size == 12
#guard environmentAssignmentUsesBinding
#guard contractExpressions.size == 11
#guard instructions.size == 8
#guard program.entrypoints.size == 4
#guard program.environment.size == 3
#guard program.roles[1]!.namespace_binding_id == "namespace"
#guard program.roles[2]!.resource_binding_id == "task.queue"
#guard contract.rules.size == 2
#guard run.events.size == 1
#guard injectFaultNamesRoleAndKind
#guard eventsDeadline.rule_events == 3
#guard eventsDeadline.elapsed_milliseconds == 0
#guard eventsDeadline.violation_state_id == "late"
#guard contract.correlated.isNone
#guard correlatedContract.correlated.any (fun capability =>
  capability.version == 1 && capability.projection_id == "projection" &&
    capability.evidence_observation_id == "evidence" && capability.clauses.size == 1)
#guard correlatedCapability.clauses[0]!.clock == .CORRELATED_CLOCK_OPERATION_TRANSITIONS
#guard correlatedCapability.clauses[0]!.ending == .TRACE_ENDING_PARTIAL
#guard correlatedCapability.clauses[0]!.captures.isEmpty
#guard correlatedCapability.clauses[0]!.correlation.isNone

end Testpilot.Tests.Authoring
