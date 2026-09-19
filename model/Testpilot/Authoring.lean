module

public import Testpilot.Protocol

public section

/-!
Stable, producer-neutral constructors for the generated Testpilot protocol.

Every helper returns the generated protobuf value directly. This module performs structural
assembly only: it preserves caller order and accepts the generated fixed-width numeric field types.
Every expression context shares one `Expression` type, so a reference used outside its context is
representable here and rejected by Go preparation. Go `testpilot.Prepare` owns semantic,
closure, version, and resource-limit admission. A Case carries no resource ceilings: only the
bounds that carry behavior (instruction timeouts and attempts, deadlines, correlated windows), which
preparation checks against the Profile's ceilings.
-/

namespace Testpilot.Authoring

open temporal.server.api.testpilot.v1

namespace Value

/-! Constructors for generated scalar, collection, enum, and message values. -/

def text (value : String) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.text_value value) }

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

/-- Name one enum value. Preparation resolves the name against the enum the literal's context
expects, and rejects a name that enum does not declare. -/
def enumeration (name : String) : temporal.server.api.testpilot.v1.Value :=
  { value := some (.enum_value { name }) }

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

def singular (type : SingularType) : ValueType :=
  { shape := some (.singular type) }

def repeated (element : SingularType) : ValueType :=
  { shape := some (.repeated { element := some element }) }

def map (key : ScalarKind) (value : SingularType) : ValueType :=
  { shape := some (.map { key := some { kind := key }, value := some value }) }

end Types

namespace Path

/-! Field paths, spelled in the grammar `PathExpression.path` documents: dot-separated protobuf
field names, each followed by at most one selector. `make` is the one printer, so a Producer builds a
path from segments and never spells the grammar by hand. -/

/-- A map key as a path writes it. Preparation types it by the kind of the map's keys. -/
inductive Key where
  | text (value : String)
  | integer (value : Int)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Repr

/-- What a segment selects inside the field it names. -/
inductive Selector where
  | plain
  | repeated
  | mapKey (key : Key)
  | presence
  | oneofMember (member : String)
  deriving BEq, DecidableEq, Repr

/-- One path segment: a protobuf field name (a oneof's name for a `oneofMember` selector) and its
selector. -/
structure Segment where
  field : String
  selector : Selector := .plain
  deriving BEq, DecidableEq, Repr

def field (name : String) : Segment := { field := name }

/-- Fan out over every element of a repeated field. -/
def repeated (name : String) : Segment := { field := name, selector := .repeated }

/-- Select the map entry with this key. -/
def mapKey (name : String) (key : Key) : Segment := { field := name, selector := .mapKey key }

/-- Read whether a presence-tracking field is set; only a path's last segment may. -/
def presence (name : String) : Segment := { field := name, selector := .presence }

/-- Select a oneof only when its active field has the supplied protobuf field name. -/
def oneofMember (name selectedField : String) : Segment :=
  { field := name, selector := .oneofMember selectedField }

/-- The hexadecimal digit of a nibble, lowercase. -/
private def hexDigit (nibble : Nat) : Char :=
  if nibble < 10 then Char.ofNat (48 + nibble) else Char.ofNat (87 + nibble)

/-- `text` as a JSON string escaping only the quote, the backslash and control characters, `\n` and
`\r` by name. It walks the UTF-8 bytes and decodes each character itself rather than calling
`String.toList`, which depends on `Classical.choice`, so a checked caller's axiom inventory stays as
it was. The Go path printer in `ir` and the protocol migration oracle spell keys the same way. -/
private def jsonString (text : String) : String := Id.run do
  let bytes := text.toUTF8
  let byte (index : Nat) : Nat := (bytes.get! index).toNat
  let mut out := "\""
  let mut index := 0
  for _ in [0:bytes.size] do
    if index ≥ bytes.size then break
    let lead := byte index
    if lead < 128 then
      out := if lead == 34 then out ++ "\\\""
        else if lead == 92 then out ++ "\\\\"
        else if lead == 10 then out ++ "\\n"
        else if lead == 13 then out ++ "\\r"
        else if lead < 32 then ((out ++ "\\u00").push (hexDigit (lead / 16))).push (hexDigit (lead % 16))
        else out.push (Char.ofNat lead)
      index := index + 1
    else
      -- The bytes come from a `String`, so every lead byte opens a well-formed sequence.
      let width := if lead < 224 then 2 else if lead < 240 then 3 else 4
      let mut point := lead % (if width == 2 then 32 else if width == 3 then 16 else 8)
      for offset in [1:width] do
        point := point * 64 + byte (index + offset) % 64
      out := out.push (Char.ofNat point)
      index := index + width
  return out.push '"'

/-- A key as a selector writes it: a text key as a JSON string, an integer in base 10, a boolean as
`true` or `false`. -/
def Key.render : Key → String
  | .text value => jsonString value
  | .integer value => toString value
  | .boolean value => if value then "true" else "false"

def Segment.render (segment : Segment) : String :=
  segment.field ++ match segment.selector with
    | .plain => ""
    | .repeated => "[*]"
    | .mapKey key => "[" ++ key.render ++ "]"
    | .presence => "?"
    | .oneofMember member => "<" ++ member ++ ">"

/-- Spell segments as one field path. No segments spell the empty path, the whole value. -/
def make (segments : Array Segment) : String :=
  ".".intercalate (segments.toList.map Segment.render)

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

/-- Read the value at `path`, a string `Path.make` spelled, out of `operand`. -/
def path (operand : Expression) (path : String) : Expression :=
  { expression := some (.path { operand := some operand, path }) }

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

/-- Declare one logical role and its optional symbolic namespace and resource references. The bindings
a Program uses are the ones its roles and expressions reference, derived at preparation, so a Program
declares none. -/
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

def requestAssignment (target : String) (value : Expression) : RequestAssignment :=
  { target, value := some value }

/-- Assign one symbolic environment resource directly to a singular text request field. -/
def environmentAssignment (target : String) (bindingId : String) : RequestAssignment :=
  requestAssignment target (Expr.environment bindingId)

def slotTarget (slotId : String) : ReadTarget :=
  { target := some (.slot_id slotId) }

def observationTarget (observationId : String) : ReadTarget :=
  { target := some (.observation_id observationId) }

/-- Supply one evidence field with the value `path` reads from the projected value. -/
def evidencePath (fieldId : String) (path : String) : NamedExpression :=
  { field_id := fieldId, value := some (Expr.path Expr.projectedValue path) }

/-- A Run coordinate the recorded fact does not itself carry is declared by the Case. -/
def evidenceLiteral (fieldId value : String) : NamedExpression :=
  { field_id := fieldId, value := some (Expr.literal (Testpilot.Authoring.Value.text value)) }

/-- One evidence-lift rule. `guard` is what selects it: the rule fires only where that boolean
expression over `Expr.projectedValue` is true. `kind` is therefore the literal the selected shape
denotes rather than a value read from it. -/
def correlatedEvidenceRule (guard : Expression) (evidenceSource kind : String) (operation : String)
    (scope : Array NamedExpression := #[])
    (fields : Array NamedExpression := #[]) : CorrelatedEvidenceRule :=
  { guard := some guard, scope, evidence_source := evidenceSource,
    operation, kind, fields }

/-- Lift a projected value into the declared `CorrelatedEvidence` Observation a correlated capability
reads. Rules are tried in declaration order and a value no rule claims emits nothing. -/
def correlatedEvidenceTarget (observationId : String) (rules : Array CorrelatedEvidenceRule) :
    ReadTarget :=
  { target := some (.correlated_evidence { observation_id := observationId, rules }) }

def responseRead (path : String) (cardinality : ReadCardinality)
    (targets : Array ReadTarget) : ResponseRead :=
  { path, cardinality, targets }

/-- Write the bounds that carry one instruction's behavior where they differ from the Profile's
instruction defaults: its dispatch timeout and its highest attempt. A bound left `none` takes the
Profile's default. Its resource ceilings are the Profile's, so a Case declares none of them. -/
def instructionLimits (timeoutMilliseconds : Option Int64 := none)
    (maxAttempts : Option Int64 := none) : InstructionLimits :=
  { timeout := timeoutMilliseconds.map (.timeout_milliseconds ·),
    attempts := maxAttempts.map (.max_attempts ·) }

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

/-- Name the instructions of the same entrypoint an instruction runs after, where that set is not the
instruction before it: several instructions, one earlier than its predecessor, or none for a second
root. -/
def after (instructions : Array InstructionReference) : After := { instructions }

/-- Define one instruction node. It runs after the instruction before it in its entrypoint unless
`after` names another set, and only when every instruction it runs after succeeded unless `guard`
states another condition. `limits` writes only the bounds that differ from the Profile's instruction
defaults, and a node whose limits write nothing carries none. Its outcome fields follow from its
instruction and the activations it reserves from the Profile's reservation carriers, so a node
declares neither. -/
def node (instructionId : String) (instruction : Instruction)
    (limits : InstructionLimits := {}) (after : Option After := none)
    (guard : Option Expression := none) : InstructionNode :=
  { instruction_id := instructionId, after, guard, instruction := some instruction,
    limits := if limits.timeout.isNone && limits.attempts.isNone then none else some limits }

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

/-- Assemble a generated Program while preserving every supplied declaration order. A Program
declares no resource ceilings and no environment bindings: the Profile it is admitted under supplies
the ceilings and the values of the bindings its roles and expressions reference. -/
def make (programId : String) (roles : Array Role) (slots : Array Slot)
    (observations : Array Observation) (entrypoints : Array Entrypoint)
    (cleanup : Cleanup) :
    temporal.server.api.testpilot.v1.Program :=
  { program_id := programId, roles, slots, observations, entrypoints,
    cleanup := some cleanup }

end Program

namespace Contract

/-! Constructors for generated Contract monitor machines and their deadlines. -/

/-- Declare a capture of a `Types.scalar`, `Types.enumeration` or `Types.messageType` value;
preparation rejects any other singular type. -/
def capture (captureId : String) (type : SingularType) : ContractCapture :=
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

/-- Set the bound and state entered when a bounded obligation expires. `.elapsed_milliseconds` is
host-clock dependent; `.rule_events` is not: it ticks once per Run Event the rule evaluates and
resets when the rule transitions into a new state. -/
def deadline (bound : Deadline.bound_Type) (violationStateId : String) : Deadline :=
  { violation_state_id := violationStateId, bound := some bound }

/-- Assemble one deterministic rule while preserving state and transition order. -/
def rule (ruleId : String) (kind : ContractRuleKind) (initialStateId : String)
    (states : Array ContractState) (transitions : Array ContractTransition)
    (deadline : Option Deadline := none)
    (captures : Array ContractCapture := #[]) : ContractRule :=
  { rule_id := ruleId, kind, initial_state_id := initialStateId, states, transitions,
    deadline, captures }

/-- Assemble one correlated rule: the operation-local window one checked clause lowers to. The
clock is the only one version one admits, so callers never choose it. `trigger` and `response` are
step conditions over `Expr.correlatedStep`. -/
def correlatedRule (ruleId : String) (bound : Int64) (ending : TraceEnding)
    (trigger response : Expression)
    (captures : Array CorrelatedCaptureDeclaration := #[])
    (correlation : Option Expression := none) : CorrelatedRule :=
  { rule_id := ruleId, clock := .CORRELATED_CLOCK_OPERATION_TRANSITIONS, bound, ending,
    trigger := some trigger, response := some response, captures, correlation }

/-- Assemble the correlated capability while preserving transition, projection-rule and rule order.
Its resource ceilings are the Profile's; its rules keep the windows that carry their meaning. -/
def correlated (projectionId projectionFingerprint evidenceObservationId operationField : String)
    (scopeFields sources : Array String) (initialState : ModelValue)
    (transitions : Array CorrelatedTransition) (projectionRules : Array CorrelatedProjectionRule)
    (rules : Array CorrelatedRule)
    (initialStateFields : Array ModelValue := #[]) : CorrelatedContract :=
  { projection_id := projectionId,
    projection_fingerprint := projectionFingerprint,
    evidence_observation_id := evidenceObservationId, scope_fields := scopeFields,
    operation_field := operationField, sources, initial_state := some initialState,
    transitions, projection_rules := projectionRules, rules,
    initial_state_fields := initialStateFields }

/-- Assemble a generated Contract while preserving rule order. A Contract may carry deterministic
monitor rules, one correlated capability, or both, and declares no resource ceilings. -/
def contract (contractId : String) (rules : Array ContractRule)
    (capability : Option CorrelatedContract := none) : Contract :=
  { contract_id := contractId, rules, correlated := capability }

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

/-- Record one Known Gap row; an absent subject or detail stays absent, and an empty one present. -/
def knownGap (kind : KnownGapKind) (code : String) (subject detail : Option String := none) :
    KnownGap :=
  { kind, code, subject_presence := subject.map (.subject ·),
    detail_presence := detail.map (.detail ·) }

/-- Attach producer identity and the typed provenance rows Testpilot never reads to a generated Case,
each list in the caller's order. -/
def provenance (producerId producerVersion : String)
    (definitions : Array DefinitionBinding := #[]) (sources : Array SourceLocation := #[])
    (knownGaps : Array KnownGap := #[]) (correlatedRules : Array CorrelatedRuleBinding := #[])
    (localNames : Array LocalName := #[])
    (modelValueFingerprints : Array ModelValueFingerprint := #[]) :
    CaseProvenance :=
  { producer_id := producerId, producer_version := producerVersion, definitions, sources,
    known_gaps := knownGaps, correlated_rules := correlatedRules, local_names := localNames,
    model_value_fingerprints := modelValueFingerprints }

/-- Assemble one generated Case from its version, identity, Program, Contract, and provenance. -/
def case (major : Int32) (caseId : String)
    (program : temporal.server.api.testpilot.v1.Program)
    (contract : temporal.server.api.testpilot.v1.Contract)
    (provenance : CaseProvenance) (minor : Int32 := 0) : Case :=
  { version := some { major, minor }, case_id := caseId, provenance := some provenance,
    program := some program, contract := some contract }

end Testpilot.Authoring
