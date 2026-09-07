module

public import Testpilot.Protocol

public section

/-! Stable, producer-neutral constructors for the generated Testpilot protocol. -/

namespace Testpilot.Authoring

open temporal.server.api.testpilot.v1

namespace Value

def text (value : String) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.text value) }

def natural (value : Nat) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.natural value.repr) }

def boolean (value : Bool) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.bool_value value) }

def bytes (value : ByteArray) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.bytes_value value) }

def signedInteger (value : Int) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.signed_integer value.repr) }

def unsignedInteger (value : Nat) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.unsigned_integer value.repr) }

def floatingPoint (value : Float) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.floating_point value) }

def enumeration (number : Int32) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.enum_value { number }) }

def messageValue (value : google.protobuf.Any) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.message_value value) }

def list (values : Array temporal.server.api.testpilot.v1.Value) :
    temporal.server.api.testpilot.v1.Value :=
  { value := some (.list_value { values }) }

def map (entries : Array (temporal.server.api.testpilot.v1.Value ×
    temporal.server.api.testpilot.v1.Value)) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.map_value {
      entries := entries.map fun (key, value) => { key := some key, value := some value }
    }) }

end Value

namespace Types

def scalar (kind : ScalarKind) : SingularType :=
  { type := some (.scalar { kind }) }

def enumeration (protobufType : String) : SingularType :=
  { type := some (.enumeration { protobuf_type := protobufType }) }

def messageType (protobufType : String) : SingularType :=
  { type := some (.message { protobuf_type := protobufType }) }

def any : SingularType := { type := some (.any {}) }

def opaqueCapability : SingularType := { type := some (.opaque_capability {}) }

def singular (type : SingularType) : ValueType :=
  { shape := some (.singular type) }

def repeated (element : SingularType) : ValueType :=
  { shape := some (.repeated { element := some element }) }

def map (key : ScalarKind) (value : SingularType) : ValueType :=
  { shape := some (.map { key := some { kind := key }, value := some value }) }

end Types

namespace Path

def field (name : String) : FieldPathSegment := { field := name }

def repeated (name : String) : FieldPathSegment :=
  { field := name, selector := some (.repeated {}) }

def mapKey (name : String) (key : temporal.server.api.testpilot.v1.Value) : FieldPathSegment :=
  { field := name, selector := some (.map_key { key := some key }) }

def presence (name : String) : FieldPathSegment :=
  { field := name, selector := some (.presence {}) }

def oneofSelector (name selectedField : String) : FieldPathSegment :=
  { field := name, selector := some (.oneof { selected_field := selectedField }) }

def make (segments : Array FieldPathSegment) : FieldPath := { segments }

end Path

namespace Ref

def instruction (entrypointId instructionId : String) : InstructionRef :=
  { entrypoint_id := entrypointId, instruction_id := instructionId }

def slot (slotId : String) : SlotRef := { slot_id := slotId }

def outcome (instruction : InstructionRef) (field : InstructionOutcomeField) :
    InstructionOutcomeRef :=
  { instruction := some instruction, field }

def observation (observationId : String) : ObservationRef :=
  { observation_id := observationId }

def capture (captureId : String) : CaptureRef := { capture_id := captureId }

end Ref

namespace ProgramExpr

def literal (value : temporal.server.api.testpilot.v1.Value) : ProgramExpression :=
  { expression := some (.literal value) }

def slot (slotId : String) : ProgramExpression :=
  { expression := some (.slot (Ref.slot slotId)) }

def outcome (instruction : InstructionRef) (field : InstructionOutcomeField) : ProgramExpression :=
  { expression := some (.outcome (Ref.outcome instruction field)) }

def run : ProgramExpression := { expression := some (.run {}) }

def path (source : ProgramExpression) (path : FieldPath) : ProgramExpression :=
  { expression := some (.path { source := some source, path := some path }) }

def present (operand : ProgramExpression) : ProgramExpression :=
  { expression := some (.present { operand := some operand }) }

def equals (left right : ProgramExpression) : ProgramExpression :=
  { expression := some (.equals { left := some left, right := some right }) }

def compare (operator : ComparisonOperator) (left right : ProgramExpression) : ProgramExpression :=
  { expression := some (.compare { operator, left := some left, right := some right }) }

def negation (operand : ProgramExpression) : ProgramExpression :=
  { expression := some (.negation { operand := some operand }) }

def all (operands : Array ProgramExpression) : ProgramExpression :=
  { expression := some (.all { operands }) }

def any (operands : Array ProgramExpression) : ProgramExpression :=
  { expression := some (.any { operands }) }

end ProgramExpr

namespace ContractExpr

def literal (value : temporal.server.api.testpilot.v1.Value) : ContractExpression :=
  { expression := some (.literal value) }

def observation (observationId : String) : ContractExpression :=
  { expression := some (.observation (Ref.observation observationId)) }

def runEvent (field : RunEventField) : ContractExpression :=
  { expression := some (.run_event { field }) }

def capture (captureId : String) : ContractExpression :=
  { expression := some (.capture (Ref.capture captureId)) }

def path (source : ContractExpression) (path : FieldPath) : ContractExpression :=
  { expression := some (.path { source := some source, path := some path }) }

def present (operand : ContractExpression) : ContractExpression :=
  { expression := some (.present { operand := some operand }) }

def equals (left right : ContractExpression) : ContractExpression :=
  { expression := some (.equals { left := some left, right := some right }) }

def compare (operator : ComparisonOperator) (left right : ContractExpression) : ContractExpression :=
  { expression := some (.compare { operator, left := some left, right := some right }) }

def negation (operand : ContractExpression) : ContractExpression :=
  { expression := some (.negation { operand := some operand }) }

def all (operands : Array ContractExpression) : ContractExpression :=
  { expression := some (.all { operands }) }

def any (operands : Array ContractExpression) : ContractExpression :=
  { expression := some (.any { operands }) }

end ContractExpr

namespace Program

def role (roleId : String) (kind : RoleKind) : RoleDefinition :=
  { role_id := roleId, kind }

def valueSlot (slotId : String) (type : ValueType) : SlotDefinition :=
  { slot_id := slotId, content := some (.value type) }

def capabilitySlot (slotId : String) : SlotDefinition :=
  { slot_id := slotId, content := some (.opaque_capability {}) }

def observation (observationId : String) (type : ValueType) : ObservationDefinition :=
  { observation_id := observationId, type := some type }

def requestAssignment (target : FieldPath) (value : ProgramExpression) : RequestAssignment :=
  { target := some target, value := some value }

def slotTarget (slotId : String) : ProjectionTarget :=
  { target := some (.slot_id slotId) }

def observationTarget (observationId : String) : ProjectionTarget :=
  { target := some (.observation_id observationId) }

def responseProjection (source : FieldPath) (kind : ProjectionKind)
    (targets : Array ProjectionTarget) : ResponseProjection :=
  { source := some source, kind, targets }

def instructionLimits (timeoutMilliseconds maxAttempts maxEmittedEvents maxResponseBytes : Int64) :
    InstructionLimits :=
  { timeout_milliseconds := timeoutMilliseconds, max_attempts := maxAttempts,
    max_emitted_events := maxEmittedEvents, max_response_bytes := maxResponseBytes }

def invokeRPC (endpointRoleId methodName : String) (assignments : Array RequestAssignment := #[])
    (projections : Array ResponseProjection := #[]) : Instruction :=
  { instruction := some (.invoke_rpc
      (InvokeRPC.mk endpointRoleId methodName assignments projections default)) }

def awaitSlot (slotId : String) : Instruction :=
  { instruction := some (.await_slot { slot_id := slotId }) }

def completeNexusOperation (capabilitySlotId : String) (result : ProgramExpression) : Instruction :=
  { instruction := some (.complete_nexus_operation {
      capability_slot_id := capabilitySlotId, result := some result }) }

def startNexusOperation (endpointRoleId serviceName operationName : String)
    (input : ProgramExpression) :
    Instruction :=
  { instruction := some (.start_nexus_operation
      (StartNexusOperation.mk endpointRoleId serviceName operationName (some input) default)) }

def awaitOutcome (instruction : InstructionRef) : Instruction :=
  { instruction := some (.await_outcome { instruction := some instruction }) }

def finish (result : ProgramExpression) : Instruction :=
  { instruction := some (.finish { result := some result }) }

def respondNexus (kind : NexusResponseKind) (result : ProgramExpression)
    (capabilitySlotId : String := "") : Instruction :=
  { instruction := some (.respond_nexus {
      kind, result := some result, capability_slot_id := capabilitySlotId }) }

def outcomeField (field : InstructionOutcomeField) (type : ValueType) : OutcomeFieldDefinition :=
  { field, type := some type }

def outcome (fields : Array OutcomeFieldDefinition) : InstructionOutcomeDefinition := { fields }

def reservation (entrypointId : String) (count : Int64) : ActivationReservationDefinition :=
  { entrypoint_id := entrypointId, count }

def node (instructionId : String) (instruction : Instruction) (limits : InstructionLimits)
    (dependencies : Array InstructionRef := #[]) (guard : Option ProgramExpression := none)
    (outcome : Option InstructionOutcomeDefinition := none)
    (reservations : Array ActivationReservationDefinition := #[]) : InstructionDefinition :=
  { instruction_id := instructionId, dependencies, guard, instruction := some instruction,
    outcome, limits := some limits, activation_reservations := reservations }

def controller (entrypointId : String) (instructions : Array InstructionDefinition) :
    EntrypointDefinition :=
  { entrypoint_id := entrypointId, instructions, activation := some (.controller {}) }

def workflow (entrypointId workflowType workerRoleId taskQueueRoleId : String)
    (instructions : Array InstructionDefinition) : EntrypointDefinition :=
  { entrypoint_id := entrypointId, instructions, activation := some (.workflow {
      workflow_type := workflowType, worker_role_id := workerRoleId,
      task_queue_role_id := taskQueueRoleId }) }

def activity (entrypointId activityType workerRoleId taskQueueRoleId : String)
    (instructions : Array InstructionDefinition) : EntrypointDefinition :=
  { entrypoint_id := entrypointId, instructions, activation := some (.activity {
      activity_type := activityType, worker_role_id := workerRoleId,
      task_queue_role_id := taskQueueRoleId }) }

def nexusHandler (entrypointId serviceName operationName workerRoleId taskQueueRoleId : String)
    (instructions : Array InstructionDefinition) : EntrypointDefinition :=
  { entrypoint_id := entrypointId, instructions,
    activation := some (.nexus_handler
      (NexusHandlerActivation.mk serviceName operationName workerRoleId taskQueueRoleId default)) }

def cleanup (entrypointId : String) (instructions : Array InstructionDefinition) : CleanupDefinition :=
  { entrypoint_id := entrypointId, instructions }

def limits (maxEntrypoints maxNodes maxEdges maxActivations maxAttempts maxRunEvents
    maxExpressionDepth maxPathFanout maxRequestBytes maxResponseBytes
    maxTotalDurationMilliseconds maxCleanupDurationMilliseconds : Int64) : ProgramLimits :=
  { max_entrypoints := maxEntrypoints, max_nodes := maxNodes, max_edges := maxEdges,
    max_activations := maxActivations, max_attempts := maxAttempts,
    max_run_events := maxRunEvents, max_expression_depth := maxExpressionDepth,
    max_path_fanout := maxPathFanout, max_request_bytes := maxRequestBytes,
    max_response_bytes := maxResponseBytes,
    max_total_duration_milliseconds := maxTotalDurationMilliseconds,
    max_cleanup_duration_milliseconds := maxCleanupDurationMilliseconds }

def make (programId : String) (roles : Array RoleDefinition) (slots : Array SlotDefinition)
    (observations : Array ObservationDefinition) (entrypoints : Array EntrypointDefinition)
    (cleanup : CleanupDefinition) (limits : ProgramLimits) :
    temporal.server.api.testpilot.v1.Program :=
  { program_id := programId, roles, slots, observations, entrypoints,
    cleanup := some cleanup, limits := some limits }

end Program

namespace Monitor

def scalarCapture (kind : ScalarKind) : ContractCaptureType :=
  { type := some (.scalar { kind }) }

def enumCapture (protobufType : String) : ContractCaptureType :=
  { type := some (.enumeration { protobuf_type := protobufType }) }

def messageCapture (protobufType : String) : ContractCaptureType :=
  { type := some (.message { protobuf_type := protobufType }) }

def capture (captureId : String) (type : ContractCaptureType) : ContractCaptureDefinition :=
  { capture_id := captureId, type := some type }

def captureAssignment (captureId observationId : String) : ContractCaptureAssignment :=
  { capture_id := captureId, observation := some (Ref.observation observationId) }

def state (stateId : String) (status : ContractStateStatus) : ContractStateDefinition :=
  { state_id := stateId, status }

def transition (transitionId sourceStateId targetStateId : String)
    (eventKinds : Array RunEventKind) (predicate : ContractExpression)
    (support : ContractSupportKind := .CONTRACT_SUPPORT_KIND_NONE)
    (assignments : Array ContractCaptureAssignment := #[]) : ContractTransitionDefinition :=
  { transition_id := transitionId, source_state_id := sourceStateId,
    target_state_id := targetStateId, event_filter := some { kinds := eventKinds },
    predicate := some predicate, support_kind := support, capture_assignments := assignments }

def horizon (elapsedMilliseconds : Int64) (violationStateId : String) :
    ContractHorizonDefinition :=
  { elapsed_milliseconds := elapsedMilliseconds, violation_state_id := violationStateId }

def rule (ruleId : String) (kind : ContractRuleKind) (initialStateId : String)
    (states : Array ContractStateDefinition) (transitions : Array ContractTransitionDefinition)
    (horizon : Option ContractHorizonDefinition := none)
    (captures : Array ContractCaptureDefinition := #[]) : ContractRuleDefinition :=
  { rule_id := ruleId, kind, initial_state_id := initialStateId, states, transitions,
    horizon, captures }

def limits (maxRules maxStates maxTransitions maxExpressionDepth maxWorkPerEvent maxTotalWork
    maxCaptures maxCaptureBytes : Int64) : ContractLimits :=
  { max_rules := maxRules, max_states := maxStates, max_transitions := maxTransitions,
    max_expression_depth := maxExpressionDepth, max_work_per_event := maxWorkPerEvent,
    max_total_work := maxTotalWork, max_captures := maxCaptures,
    max_capture_bytes := maxCaptureBytes }

def contract (contractId : String) (rules : Array ContractRuleDefinition)
    (limits : ContractLimits) : Contract :=
  { contract_id := contractId, rules, limits := some limits }

end Monitor

namespace Run

def coordinates (entrypointId activationId instructionId : String) (attempt emittedIndex : Int64) :
    RunEventCoordinates :=
  { entrypoint_id := entrypointId, activation_id := activationId,
    instruction_id := instructionId, attempt, emitted_index := emittedIndex }

def observation (observationId : String) (value : temporal.server.api.testpilot.v1.Value) :
    ObservationResult :=
  { observation_id := observationId, value := some value }

def outcome (status : InstructionOutcomeStatus) (protocolCode sdkFailureCode detail : String := "")
    (value : Option temporal.server.api.testpilot.v1.Value := none) : InstructionOutcome :=
  { status, protocol_code := protocolCode, sdk_failure_code := sdkFailureCode, detail, value }

def event (sequence elapsedMilliseconds : Int64) (kind : RunEventKind)
    (coordinates : RunEventCoordinates) (sourceId : String)
    (causalSourceIds : Array String := #[]) (outcome : Option InstructionOutcome := none)
    (observations : Array ObservationResult := #[]) (executionIncomplete : Bool := false) : RunEvent :=
  { sequence, elapsed_milliseconds := elapsedMilliseconds, kind, coordinates := some coordinates,
    source_id := sourceId, causal_source_ids := causalSourceIds, outcome, observations,
    execution_incomplete := executionIncomplete }

def cleanup (status : CleanupStatus) (diagnosticIds : Array String := #[]) : CleanupOutcome :=
  { status, diagnostic_ids := diagnosticIds }

def diagnostic (diagnosticId : String) (kind : RunDiagnosticKind) (code detail : String)
    (supportingEventSequence : Option Int64 := none) : RunDiagnostic :=
  { diagnostic_id := diagnosticId, kind, code, detail,
    support := supportingEventSequence.map (.supporting_event_sequence ·) }

def make (runId caseId programId : String) (events : Array RunEvent) (status : RunStatus)
    (cleanup : CleanupOutcome) (verdict : Verdict) (diagnostics : Array RunDiagnostic := #[])
    (evaluationFailureSequence : Option Int64 := none) : temporal.server.api.testpilot.v1.Run :=
  { run_id := runId, case_id := caseId, program_id := programId, events, status,
    cleanup := some cleanup, verdict := some verdict, diagnostics,
    evaluation_failure := evaluationFailureSequence.map (.evaluation_failure_sequence ·) }

end Run

namespace Verdict

def rule (ruleId : String) (status : RuleVerdictStatus) (terminalStateId : String := "")
    (supportingEventSequences : Array Int64 := #[]) : RuleVerdict :=
  { rule_id := ruleId, status, terminal_state_id := terminalStateId,
    supporting_event_sequences := supportingEventSequences }

def make (status : VerdictStatus) (rules : Array RuleVerdict)
    (supportingEventSequences : Array Int64 := #[]) :
    temporal.server.api.testpilot.v1.Verdict :=
  { status, rules, supporting_event_sequences := supportingEventSequences }

end Verdict

def provenance (producerId producerVersion : String) (producerData : ByteArray := ByteArray.empty) :
    CaseProvenance :=
  { producer_id := producerId, producer_version := producerVersion, producer_data := producerData }

def case (major : Int32) (caseId : String)
    (program : temporal.server.api.testpilot.v1.Program)
    (contract : temporal.server.api.testpilot.v1.Contract)
    (provenance : CaseProvenance) (minor : Int32 := 0) : Case :=
  { version := some { major, minor }, case_id := caseId, provenance := some provenance,
    program := some program, contract := some contract }

end Testpilot.Authoring
