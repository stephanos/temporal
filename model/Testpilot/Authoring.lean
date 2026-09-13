module

public import Testpilot.Protocol

public section

/-!
Stable, producer-neutral constructors for the generated Testpilot protocol.

Every helper returns the generated protobuf value directly. This module performs structural
assembly only: it preserves caller order and accepts the generated fixed-width numeric field types.
Every expression context shares one `Expression` type, so a reference used outside its context is
representable here and rejected by Go preparation. Go `testpilot.Prepare` owns semantic,
closure, version, and resource-limit admission. Callers should use named arguments for high-arity
limit records where positional meaning would otherwise be unclear.
-/

namespace Testpilot.Authoring

open temporal.server.api.testpilot.v1

namespace Value

/-! Constructors for generated scalar, collection, enum, and message values. -/

def text (value : String) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.text_value value) }

/-- Encode a natural as the protocol's canonical decimal string representation. -/
def natural (value : Nat) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.natural_value value.repr) }

def boolean (value : Bool) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.bool_value value) }

def bytes (value : ByteArray) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.bytes_value value) }

/-- Encode a signed integer as the protocol's canonical decimal string representation. -/
def signedInteger (value : Int) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.signed_integer_value value.repr) }

/-- Encode an unsigned integer as the protocol's canonical decimal string representation. -/
def unsignedInteger (value : Nat) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.unsigned_integer_value value.repr) }

def floatingPoint (value : Float) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.floating_point_value value) }

def enumeration (number : Int32) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.enum_value { number }) }

/-- Embed an already packed protobuf message value. -/
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

/-! Constructors for generated Testpilot value schemas. -/

def scalar (kind : ScalarKind) : SingularType :=
  { type := some (.scalar { kind }) }

def enumeration (protobufType : String) : SingularType :=
  { type := some (.enumeration { protobuf_type := protobufType }) }

def messageType (protobufType : String) : SingularType :=
  { type := some (.message { protobuf_type := protobufType }) }

def any : SingularType := { type := some (.any {}) }

def opaqueHandle : SingularType := { type := some (.opaque_handle {}) }

def singular (type : SingularType) : ValueType :=
  { shape := some (.singular type) }

def repeated (element : SingularType) : ValueType :=
  { shape := some (.repeated { element := some element }) }

def map (key : ScalarKind) (value : SingularType) : ValueType :=
  { shape := some (.map { key := some { kind := key }, value := some value }) }

end Types

namespace Path

/-! Constructors for generated field paths and selectors. -/

def field (name : String) : FieldPathSegment := { field := name }

def repeated (name : String) : FieldPathSegment :=
  { field := name, selector := some (.repeated {}) }

/-- Select the map entry whose key equals the supplied generated value. -/
def mapKey (name : String) (key : temporal.server.api.testpilot.v1.Value) : FieldPathSegment :=
  { field := name, selector := some (.map_key { key := some key }) }

def presence (name : String) : FieldPathSegment :=
  { field := name, selector := some (.presence {}) }

/-- Select a oneof only when its active field has the supplied protobuf field name. -/
def oneofSelector (name selectedField : String) : FieldPathSegment :=
  { field := name, selector := some (.oneof { selected_field := selectedField }) }

def make (segments : Array FieldPathSegment) : FieldPath := { segments }

end Path

namespace Ref

/-! Constructors for generated instruction references. -/

def instruction (entrypointId instructionId : String) : InstructionReference :=
  { entrypoint_id := entrypointId, instruction_id := instructionId }

end Ref

namespace Expr

/-!
Constructors for the one expression language of instruction inputs, guards, Contract transition
predicates, correlated rule conditions, and evidence-lift guards. The type does not separate the
contexts: instruction inputs and guards may read Slots, instruction outcomes, the Run, and
environment bindings; Contract predicates may read Observations, Run Event fields, and captures;
correlated conditions may read evidence fields, correlated captures, and correlated steps; and
evidence-lift guards may read the projected value. Go preparation rejects a reference outside its
context.
-/

def literal (value : temporal.server.api.testpilot.v1.Value) : Expression :=
  { expression := some (.literal value) }

def reference (reference : Reference) : Expression :=
  { expression := some (.reference reference) }

def slot (slotId : String) : Expression :=
  reference { reference := some (.slot_id slotId) }

def outcome (instruction : InstructionReference) (field : InstructionOutcomeField) : Expression :=
  reference { reference := some (.outcome { instruction := some instruction, field }) }

/-- Refer to one symbolic text resource supplied by the execution environment. -/
def environment (bindingId : String) : Expression :=
  reference { reference := some (.environment_binding_id bindingId) }

/-- Refer to the current Run from an instruction input. -/
def run : Expression := reference { reference := some (.run {}) }

def observation (observationId : String) : Expression :=
  reference { reference := some (.observation_id observationId) }

/-- Read one common coordinate of the Run Event currently offered to a monitor. -/
def runEvent (field : RunEventField) : Expression :=
  reference { reference := some (.run_event { selection := some (.field field) }) }

/-- Refer to the payload of the Run Event currently offered to a monitor. It is read only as the
operand of `path`, whose first segment names the payload arm, such as `fault_injected`. -/
def runEventPayload : Expression :=
  reference { reference := some (.run_event { selection := some (.payload {}) }) }

def capture (captureId : String) : Expression :=
  reference { reference := some (.capture_id captureId) }

/-- Read one declared evidence field of the correlated step being admitted. -/
def evidenceField (fieldId : String) : Expression :=
  reference { reference := some (.evidence_field_id fieldId) }

/-- Read one retained occurrence of a correlated capture by its zero-based ordinal. -/
def correlatedCapture (captureId : String) (ordinal : Int64) : Expression :=
  reference { reference := some (.correlated_capture { capture_id := captureId, ordinal }) }

/-- Read the model value of `definitionId` at one part of the correlated step being admitted. -/
def correlatedStep (field : CorrelatedStepField) (definitionId : String) : Expression :=
  reference { reference := some (.correlated_step { field, definition_id := definitionId }) }

/-- Read the value an evidence lift is projecting. -/
def projectedValue : Expression := reference { reference := some (.projected_value {}) }

def path (operand : Expression) (path : FieldPath) : Expression :=
  { expression := some (.path { operand := some operand, path := some path }) }

def present (operand : Expression) : Expression :=
  { expression := some (.present { operand := some operand }) }

def compare (operator : ComparisonOperator) (left right : Expression) : Expression :=
  { expression := some (.compare { operator, left := some left, right := some right }) }

/-- Compare two operands of one type for equality. -/
def equal (left right : Expression) : Expression :=
  compare .COMPARISON_OPERATOR_EQUAL left right

/-- Negate a boolean operand. The protocol arm is `not`; a definition of that name here would
shadow `_root_.not` inside `Testpilot.Authoring`. -/
def negate (operand : Expression) : Expression :=
  { expression := some (.not { operand := some operand }) }

def all (operands : Array Expression) : Expression :=
  { expression := some (.all { operands }) }

def any (operands : Array Expression) : Expression :=
  { expression := some (.any { operands }) }

end Expr

namespace Program

/-! Constructors for bounded generated Program declarations. Array order is preserved. -/

/-- Declare one symbolic text resource required by the Program. -/
def environment (bindingId : String) : EnvironmentDefinition := { binding_id := bindingId }

/-- Declare one logical role and its optional symbolic namespace and resource references. -/
def role (roleId : String) (kind : RoleKind) (namespaceBindingId : String := "")
    (resourceBindingId : String := "") : Role :=
  { role_id := roleId, kind, namespace_binding_id := namespaceBindingId,
    resource_binding_id := resourceBindingId }

def valueSlot (slotId : String) (type : ValueType) : Slot :=
  { slot_id := slotId, content := some (.value type) }

def handleSlot (slotId : String) : Slot :=
  { slot_id := slotId, content := some (.opaque_handle {}) }

def observation (observationId : String) (type : ValueType) : Observation :=
  { observation_id := observationId, type := some type }

def requestAssignment (target : FieldPath) (value : Expression) : RequestAssignment :=
  { target := some target, value := some value }

/-- Assign one symbolic environment resource directly to a singular text request field. -/
def environmentAssignment (target : FieldPath) (bindingId : String) : RequestAssignment :=
  requestAssignment target (Expr.environment bindingId)

def slotTarget (slotId : String) : ReadTarget :=
  { target := some (.slot_id slotId) }

def observationTarget (observationId : String) : ReadTarget :=
  { target := some (.observation_id observationId) }

def correlatedEvidenceBinding (fieldId : String) (path : FieldPath) : CorrelatedEvidenceBinding :=
  { field_id := fieldId, value := some (.path path) }

/-- A Run coordinate the recorded fact does not itself carry is declared by the Case. -/
def correlatedEvidenceLiteral (fieldId value : String) : CorrelatedEvidenceBinding :=
  { field_id := fieldId, value := some (.literal value) }

/-- One evidence-lift rule. `guard` is what selects it: the rule fires only where that boolean
expression over `Expr.projectedValue` is true. `kind` is therefore the literal the selected shape
denotes rather than a value read from it. -/
def correlatedEvidenceRule (guard : Expression) (evidenceSource kind : String) (operation : FieldPath)
    (scope : Array CorrelatedEvidenceBinding := #[])
    (fields : Array CorrelatedEvidenceBinding := #[]) : CorrelatedEvidenceRule :=
  { guard := some guard, scope, evidence_source := evidenceSource,
    operation := some operation, kind, fields }

/-- Lift a projected value into the declared `CorrelatedEvidence` Observation a correlated capability
reads. Rules are tried in declaration order and a value no rule claims emits nothing. -/
def correlatedEvidenceTarget (observationId : String) (rules : Array CorrelatedEvidenceRule) :
    ReadTarget :=
  { target := some (.correlated_evidence { observation_id := observationId, rules }) }

def responseRead (path : FieldPath) (cardinality : ReadCardinality)
    (targets : Array ReadTarget) : ResponseRead :=
  { path := some path, cardinality, targets }

/-- Attach the four fixed-width resource bounds enforced for one instruction. -/
def instructionLimits (timeoutMilliseconds maxAttempts maxEmittedEvents maxResponseBytes : Int64) :
    InstructionLimits :=
  { timeout_milliseconds := timeoutMilliseconds, max_attempts := maxAttempts,
    max_emitted_events := maxEmittedEvents, max_response_bytes := maxResponseBytes }

def invokeRpc (endpointRoleId methodName : String) (assignments : Array RequestAssignment := #[])
    (reads : Array ResponseRead := #[]) : Instruction :=
  { instruction := some (.invoke_rpc
      (InvokeRpc.mk endpointRoleId methodName assignments reads default)) }

def awaitSlot (slotId : String) : Instruction :=
  { instruction := some (.await_slot { slot_id := slotId }) }

def completeNexusOperation (handleSlotId : String) (result : Expression) : Instruction :=
  { instruction := some (.complete_nexus_operation {
      handle_slot_id := handleSlotId, result := some result }) }

def startNexusOperation (endpointRoleId serviceName operationName : String)
    (input : Expression) :
    Instruction :=
  { instruction := some (.start_nexus_operation
      (StartNexusOperation.mk endpointRoleId serviceName operationName (some input) default)) }

def awaitInstruction (instruction : InstructionReference) : Instruction :=
  { instruction := some (.await_instruction { instruction := some instruction }) }

def finish (result : Expression) : Instruction :=
  { instruction := some (.finish { result := some result }) }

def respondNexus (kind : NexusResponseKind) (result : Expression)
    (handleSlotId : String := "") : Instruction :=
  { instruction := some (.respond_nexus {
      kind, result := some result, handle_slot_id := handleSlotId }) }

/-- Request one deliberate outage. `roleId` names the task-queue role whose worker the Driver
stops or resumes; the role's own resource binding identifies the queue. -/
def injectFault (roleId : String) (kind : FaultKind) : Instruction :=
  { instruction := some (.inject_fault { role_id := roleId, kind }) }

def outcomeField (field : InstructionOutcomeField) (type : ValueType) : OutcomeFieldDefinition :=
  { field, type := some type }

def outcome (fields : Array OutcomeFieldDefinition) : InstructionOutcomeDefinition := { fields }

def reservation (entrypointId : String) (count : Int64) : ActivationReservationDefinition :=
  { entrypoint_id := entrypointId, count }

/-- Define one instruction node with its explicit dependencies, guard, outcome, and reservations. -/
def node (instructionId : String) (instruction : Instruction) (limits : InstructionLimits)
    (dependencies : Array InstructionReference := #[]) (guard : Option Expression := none)
    (outcome : Option InstructionOutcomeDefinition := none)
    (reservations : Array ActivationReservationDefinition := #[]) : InstructionNode :=
  { instruction_id := instructionId, dependencies, guard, instruction := some instruction,
    outcome, limits := some limits, activation_reservations := reservations }

def controller (entrypointId : String) (instructions : Array InstructionNode) :
    Entrypoint :=
  { entrypoint_id := entrypointId, instructions, activation := some (.controller {}) }

def workflow (entrypointId workflowType workerRoleId taskQueueRoleId : String)
    (instructions : Array InstructionNode) : Entrypoint :=
  { entrypoint_id := entrypointId, instructions, activation := some (.workflow {
      workflow_type := workflowType, worker_role_id := workerRoleId,
      task_queue_role_id := taskQueueRoleId }) }

def activity (entrypointId activityType workerRoleId taskQueueRoleId : String)
    (instructions : Array InstructionNode) : Entrypoint :=
  { entrypoint_id := entrypointId, instructions, activation := some (.activity {
      activity_type := activityType, worker_role_id := workerRoleId,
      task_queue_role_id := taskQueueRoleId }) }

def nexusHandler (entrypointId serviceName operationName workerRoleId taskQueueRoleId : String)
    (instructions : Array InstructionNode) : Entrypoint :=
  { entrypoint_id := entrypointId, instructions,
    activation := some (.nexus_handler
      (NexusHandlerActivation.mk serviceName operationName workerRoleId taskQueueRoleId default)) }

def cleanup (entrypointId : String) (instructions : Array InstructionNode) : Cleanup :=
  { entrypoint_id := entrypointId, instructions }

/-- Construct Program-wide bounds; callers should name arguments where the positions are unclear. -/
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

/-- Assemble a generated Program while preserving every supplied declaration order. -/
def make (programId : String) (roles : Array Role) (slots : Array Slot)
    (observations : Array Observation) (entrypoints : Array Entrypoint)
    (cleanup : Cleanup) (limits : ProgramLimits)
    (environment : Array EnvironmentDefinition := #[]) :
    temporal.server.api.testpilot.v1.Program :=
  { program_id := programId, roles, slots, observations, entrypoints,
    cleanup := some cleanup, limits := some limits, environment }

end Program

namespace Contract

/-! Constructors for generated Contract monitor machines and their bounds. -/

def scalarCapture (kind : ScalarKind) : ContractCaptureType :=
  { type := some (.scalar { kind }) }

def enumCapture (protobufType : String) : ContractCaptureType :=
  { type := some (.enumeration { protobuf_type := protobufType }) }

def messageCapture (protobufType : String) : ContractCaptureType :=
  { type := some (.message { protobuf_type := protobufType }) }

def capture (captureId : String) (type : ContractCaptureType) : ContractCapture :=
  { capture_id := captureId, type := some type }

def captureAssignment (captureId observationId : String) : ContractCaptureAssignment :=
  { capture_id := captureId, observation_id := observationId }

def state (stateId : String) (status : ContractStateStatus) : ContractState :=
  { state_id := stateId, status }

/-- Define one transition, including its event filter, predicate, support, and capture updates. -/
def transition (transitionId sourceStateId targetStateId : String)
    (eventKinds : Array RunEventKind) (predicate : Expression)
    (support : ContractSupportKind := .CONTRACT_SUPPORT_KIND_NONE)
    (assignments : Array ContractCaptureAssignment := #[]) : ContractTransition :=
  { transition_id := transitionId, source_state_id := sourceStateId,
    target_state_id := targetStateId, event_filter := some { kinds := eventKinds },
    predicate := some predicate, support_kind := support, capture_assignments := assignments }

/-- Set the elapsed-time deadline and state entered when a bounded obligation expires. -/
def deadline (elapsedMilliseconds : Int64) (violationStateId : String) :
    ContractDeadline :=
  { elapsed_milliseconds := elapsedMilliseconds, violation_state_id := violationStateId }

/-- Set the evaluated-event deadline and state entered when a bounded obligation expires. The
count is host-clock independent: it ticks once per Run Event the rule evaluates and resets when
the rule transitions into a new state. -/
def deadlineEvents (ruleEvents : Int64) (violationStateId : String) :
    ContractDeadline :=
  { rule_events := ruleEvents, violation_state_id := violationStateId }

/-- Assemble one deterministic rule while preserving state and transition order. -/
def rule (ruleId : String) (kind : ContractRuleKind) (initialStateId : String)
    (states : Array ContractState) (transitions : Array ContractTransition)
    (deadline : Option ContractDeadline := none)
    (captures : Array ContractCapture := #[]) : ContractRule :=
  { rule_id := ruleId, kind, initial_state_id := initialStateId, states, transitions,
    deadline, captures }

/-- Construct Contract-wide bounds; callers should name arguments where the positions are unclear. -/
def limits (maxRules maxStates maxTransitions maxExpressionDepth maxWorkPerEvent maxTotalWork
    maxCaptures maxCaptureBytes : Int64) : ContractLimits :=
  { max_rules := maxRules, max_states := maxStates, max_transitions := maxTransitions,
    max_expression_depth := maxExpressionDepth, max_work_per_event := maxWorkPerEvent,
    max_total_work := maxTotalWork, max_captures := maxCaptures,
    max_capture_bytes := maxCaptureBytes }

/-- Assemble one correlated rule: the operation-local window one checked clause lowers to. The
clock is the only one version one admits, so callers never choose it. `trigger` and `response` are
step conditions over `Expr.correlatedStep`. -/
def correlatedRule (ruleId : String) (bound : Int64) (ending : TraceEnding)
    (trigger response : Expression)
    (captures : Array CorrelatedCaptureDeclaration := #[])
    (correlation : Option Expression := none) : CorrelatedRule :=
  { rule_id := ruleId, clock := .CORRELATED_CLOCK_OPERATION_TRANSITIONS, bound, ending,
    trigger := some trigger, response := some response, captures, correlation }

/-- Assemble the version-one correlated capability while preserving transition, projection-rule and
rule order. Version one is the only admitted version, so callers never choose it. -/
def correlated (projectionId projectionFingerprint evidenceObservationId operationField : String)
    (scopeFields sources : Array String) (initialState : ModelValue)
    (transitions : Array CorrelatedTransition) (projectionRules : Array CorrelatedProjectionRule)
    (rules : Array CorrelatedRule) (limits : CorrelatedLimits) : CorrelatedContract :=
  { version := 1, projection_id := projectionId,
    projection_fingerprint := projectionFingerprint,
    evidence_observation_id := evidenceObservationId, scope_fields := scopeFields,
    operation_field := operationField, sources, initial_state := some initialState,
    transitions, projection_rules := projectionRules, rules,
    limits := some limits }

/-- Assemble a generated Contract while preserving rule order. A Contract may carry deterministic
monitor rules, one correlated capability, or both. -/
def contract (contractId : String) (rules : Array ContractRule)
    (limits : ContractLimits) (capability : Option CorrelatedContract := none) : Contract :=
  { contract_id := contractId, rules, limits := some limits, correlated := capability }

end Contract

namespace Run

/-! Constructors for immutable generated runtime evidence. -/

/-- Identify the instruction attempt and emission position responsible for one Run Event. -/
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

/-- Assemble one sequenced Run Event with an optional payload, observations, and causal sources. -/
def event (sequence elapsedMilliseconds : Int64) (kind : RunEventKind)
    (coordinates : RunEventCoordinates) (sourceId : String)
    (causalSourceIds : Array String := #[]) (payload : Option RunEvent.payload_Type := none)
    (observations : Array ObservationResult := #[]) (executionIncomplete : Bool := false) : RunEvent :=
  { sequence, elapsed_milliseconds := elapsedMilliseconds, kind, coordinates := some coordinates,
    source_id := sourceId, causal_source_ids := causalSourceIds, observations,
    execution_incomplete := executionIncomplete, payload }

def cleanup (status : CleanupStatus) (diagnosticIds : Array String := #[]) : CleanupOutcome :=
  { status, diagnostic_ids := diagnosticIds }

def diagnostic (diagnosticId : String) (kind : RunDiagnosticKind) (code detail : String)
    (supportingEventSequence : Option Int64 := none) : RunDiagnostic :=
  { diagnostic_id := diagnosticId, kind, code, detail,
    support := supportingEventSequence.map (.supporting_event_sequence ·) }

/-- Assemble a closed generated Run and embedded Verdict from already collected evidence. -/
def make (runId caseId programId : String) (events : Array RunEvent) (disposition : RunDisposition)
    (cleanup : CleanupOutcome) (verdict : Verdict) (diagnostics : Array RunDiagnostic := #[])
    (evaluationFailureSequence : Option Int64 := none) : temporal.server.api.testpilot.v1.Run :=
  { run_id := runId, case_id := caseId, program_id := programId, events, disposition,
    cleanup := some cleanup, verdict := some verdict, diagnostics,
    evaluation_failure := evaluationFailureSequence.map (.evaluation_failure_sequence ·) }

end Run

namespace Verdict

/-! Constructors for generated per-rule and aggregate Verdict data. -/

/-- Record the terminal status and exact supporting events for one Contract rule. -/
def rule (ruleId : String) (status : RuleVerdictStatus) (terminalStateId : String := "")
    (supportingEventSequences : Array Int64 := #[]) : RuleVerdict :=
  { rule_id := ruleId, status, terminal_state_id := terminalStateId,
    supporting_event_sequences := supportingEventSequences }

/-- Assemble an aggregate Verdict while preserving per-rule and supporting-event order. -/
def make (status : VerdictStatus) (rules : Array RuleVerdict)
    (supportingEventSequences : Array Int64 := #[]) :
    temporal.server.api.testpilot.v1.Verdict :=
  { status, rules, supporting_event_sequences := supportingEventSequences }

end Verdict

/-- Attach producer identity and opaque producer-owned bytes to a generated Case. -/
def provenance (producerId producerVersion : String) (producerData : ByteArray := ByteArray.empty) :
    CaseProvenance :=
  { producer_id := producerId, producer_version := producerVersion, producer_data := producerData }

/-- Assemble one generated Case from its version, identity, Program, Contract, and provenance. -/
def case (major : Int32) (caseId : String)
    (program : temporal.server.api.testpilot.v1.Program)
    (contract : temporal.server.api.testpilot.v1.Contract)
    (provenance : CaseProvenance) (minor : Int32 := 0) : Case :=
  { version := some { major, minor }, case_id := caseId, provenance := some provenance,
    program := some program, contract := some contract }

end Testpilot.Authoring
