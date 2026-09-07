import Testpilot.Authoring
import Umpire.Case
import Umpire.Case.Provenance
import Umpire.KnownGap

/-!
Checked Case compilation consumes `ContractLowering` values produced before this boundary and
produces only the closed Program and Contract vocabulary consumed by runtime Hosts. This module has
no checked-Property-to-lowering producer and does not recognize `PropertyClause`; a future producer
must reject unsupported guarded forms before constructing `Input` rather than relying on `compile`.
-/

namespace Umpire

/-- Convert one checked planning Known Gap to the exact Case row vocabulary. -/
def KnownGap.toCaseKnownGap (gap : KnownGap) : Case.CaseKnownGap := {
  kind := match gap.kind with
    | .capabilityContract => .capabilityContract
    | .input => .input
    | .interpretation => .interpretation
    | .claim => .claim
  code := gap.code.value
  subject := gap.subject.map DefinitionId.value
  detail := gap.detail
}

/-- Convert checked planning Known Gaps to exact Case rows in one order-preserving pass. -/
def KnownGapSet.toCaseKnownGaps (gaps : KnownGapSet) : List Case.CaseKnownGap :=
  gaps.toList.map KnownGap.toCaseKnownGap

end Umpire

namespace Umpire.Case.Compiler

open Umpire
open Umpire.Case

/-- A stable failure for a checked construct outside the closed Case monitor vocabulary. -/
structure LoweringError where
  sourceDefinitionId : String
  source : SourceLocation
  construct : String
  deriving BEq, DecidableEq, Repr

private def loweringError (sourceDefinitionId construct : String) : LoweringError := {
  sourceDefinitionId
  source := { path := "" }
  construct
}

private def int32OfNat (sourceDefinitionId construct : String) (value : Nat) :
    Except LoweringError Int32 :=
  if value ≤ 2147483647 then
    pure value.toInt32
  else
    .error (loweringError sourceDefinitionId construct)

private def int32OfInt (sourceDefinitionId construct : String) (value : Int) :
    Except LoweringError Int32 :=
  if -2147483648 ≤ value ∧ value ≤ 2147483647 then
    pure value.toInt32
  else
    .error (loweringError sourceDefinitionId construct)

private def int64OfNat (sourceDefinitionId construct : String) (value : Nat) :
    Except LoweringError Int64 :=
  if value ≤ 9223372036854775807 then
    pure value.toInt64
  else
    .error (loweringError sourceDefinitionId construct)

/-- One checked property either lowered to a complete monitor or rejected explicitly. -/
inductive ContractLowering where
  | monitor (sourceDefinition : CaseDefinitionBinding) (rule : ContractRule)
  | unsupported
      (sourceDefinition : CaseDefinitionBinding)
      (source : SourceLocation)
      (construct : String)
  deriving BEq, Repr

/-- Complete checked inputs shared by all Case producers. -/
structure Input where
  version : FormatVersion
  caseId : String
  producerId : String
  producerVersion : String := ""
  definitions : List CaseDefinitionBinding
  sources : List SourceLocation
  knownGaps : List CaseKnownGap
  program : Program
  contractId : String
  properties : List ContractLowering
  contractLimits : ContractLimits
  deriving BEq, Repr

private def scalarKind : Umpire.Case.ScalarKind → temporal.server.api.testpilot.v1.ScalarKind
  | .text => .SCALAR_KIND_TEXT
  | .natural => .SCALAR_KIND_NATURAL
  | .boolean => .SCALAR_KIND_BOOLEAN
  | .bytes => .SCALAR_KIND_BYTES
  | .int32 => .SCALAR_KIND_INT32
  | .int64 => .SCALAR_KIND_INT64
  | .uint32 => .SCALAR_KIND_UINT32
  | .uint64 => .SCALAR_KIND_UINT64
  | .sint32 => .SCALAR_KIND_SINT32
  | .sint64 => .SCALAR_KIND_SINT64
  | .fixed32 => .SCALAR_KIND_FIXED32
  | .fixed64 => .SCALAR_KIND_FIXED64
  | .sfixed32 => .SCALAR_KIND_SFIXED32
  | .sfixed64 => .SCALAR_KIND_SFIXED64
  | .float => .SCALAR_KIND_FLOAT
  | .double => .SCALAR_KIND_DOUBLE

private partial def value (sourceDefinitionId construct : String) : Umpire.Case.Value →
    Except LoweringError temporal.server.api.testpilot.v1.Value
  | .text item => pure (Testpilot.Authoring.Value.text item)
  | .natural item => pure (Testpilot.Authoring.Value.natural item)
  | .boolean item => pure (Testpilot.Authoring.Value.boolean item)
  | .bytes item => pure (Testpilot.Authoring.Value.bytes item)
  | .signedInteger item => pure (Testpilot.Authoring.Value.signedInteger item)
  | .unsignedInteger item => pure (Testpilot.Authoring.Value.unsignedInteger item)
  | .floatingPoint item => pure (Testpilot.Authoring.Value.floatingPoint item)
  | .enumValue number => do
      pure (Testpilot.Authoring.Value.enumeration
        (← int32OfInt sourceDefinitionId construct number))
  | .messageValue item => pure (Testpilot.Authoring.Value.messageValue {
      type_url := item.typeUrl, value := item.bytes })
  | .listValue items => do
      pure (Testpilot.Authoring.Value.list
        (← items.mapM (value sourceDefinitionId construct)).toArray)
  | .mapValue entries => do
      pure (Testpilot.Authoring.Value.map (← entries.mapM fun entry => do
        pure (← value sourceDefinitionId construct entry.1,
          ← value sourceDefinitionId construct entry.2)).toArray)

private def singularType : Umpire.Case.SingularType → temporal.server.api.testpilot.v1.SingularType
  | .scalar kind => Testpilot.Authoring.Types.scalar (scalarKind kind)
  | .enumeration name => Testpilot.Authoring.Types.enumeration name
  | .message name => Testpilot.Authoring.Types.messageType name
  | .any => Testpilot.Authoring.Types.any
  | .opaqueCapability => Testpilot.Authoring.Types.opaqueCapability

private def valueType : Umpire.Case.ValueType → temporal.server.api.testpilot.v1.ValueType
  | .singular item => Testpilot.Authoring.Types.singular (singularType item)
  | .repeated item => Testpilot.Authoring.Types.repeated (singularType item)
  | .map key item => Testpilot.Authoring.Types.map (scalarKind key) (singularType item)

private def fieldPath (sourceDefinitionId construct : String) (item : Umpire.Case.FieldPath) :
    Except LoweringError temporal.server.api.testpilot.v1.FieldPath := do
  pure (Testpilot.Authoring.Path.make (← item.segments.mapM fun segment =>
    match segment.selector with
    | none => pure (Testpilot.Authoring.Path.field segment.field)
    | some .repeated => pure (Testpilot.Authoring.Path.repeated segment.field)
    | some (.mapKey key) => do
        pure (Testpilot.Authoring.Path.mapKey segment.field
          (← value sourceDefinitionId construct key))
    | some .presence => pure (Testpilot.Authoring.Path.presence segment.field)
    | some (.oneof selected) =>
        pure (Testpilot.Authoring.Path.oneofSelector segment.field selected)).toArray)

private def instructionRef (item : Umpire.Case.InstructionReference) :
    temporal.server.api.testpilot.v1.InstructionRef :=
  Testpilot.Authoring.Ref.instruction item.entrypointId item.instructionId

private def outcomeField : Umpire.Case.InstructionOutcomeField →
    temporal.server.api.testpilot.v1.InstructionOutcomeField
  | .status => .INSTRUCTION_OUTCOME_FIELD_STATUS
  | .protocolCode => .INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE
  | .sdkFailureCode => .INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE
  | .detail => .INSTRUCTION_OUTCOME_FIELD_DETAIL
  | .value => .INSTRUCTION_OUTCOME_FIELD_VALUE

private def runEventField : Umpire.Case.RunEventField → temporal.server.api.testpilot.v1.RunEventField
  | .sequence => .RUN_EVENT_FIELD_SEQUENCE
  | .elapsedMilliseconds => .RUN_EVENT_FIELD_ELAPSED_MILLISECONDS
  | .kind => .RUN_EVENT_FIELD_KIND
  | .entrypointId => .RUN_EVENT_FIELD_ENTRYPOINT_ID
  | .activationId => .RUN_EVENT_FIELD_ACTIVATION_ID
  | .instructionId => .RUN_EVENT_FIELD_INSTRUCTION_ID
  | .attempt => .RUN_EVENT_FIELD_ATTEMPT
  | .sourceId => .RUN_EVENT_FIELD_SOURCE_ID
  | .runId => .RUN_EVENT_FIELD_RUN_ID

private partial def programExpression (sourceDefinitionId : String) : Umpire.Case.ValueExpression →
    Except LoweringError temporal.server.api.testpilot.v1.ProgramExpression
  | .literal item => do
      pure (Testpilot.Authoring.ProgramExpr.literal
        (← value sourceDefinitionId "program.enum-range" item))
  | .slot item => pure (Testpilot.Authoring.ProgramExpr.slot item.slotId)
  | .outcome item => pure (Testpilot.Authoring.ProgramExpr.outcome
      (instructionRef item.instruction) (outcomeField item.field))
  | .runEvent .runId => pure Testpilot.Authoring.ProgramExpr.run
  | .path source item => do
      pure (Testpilot.Authoring.ProgramExpr.path (← programExpression sourceDefinitionId source)
        (← fieldPath sourceDefinitionId "program.enum-range" item))
  | .present operand => do
      pure (Testpilot.Authoring.ProgramExpr.present
        (← programExpression sourceDefinitionId operand))
  | .equals left right => do
      pure (Testpilot.Authoring.ProgramExpr.equals (← programExpression sourceDefinitionId left)
        (← programExpression sourceDefinitionId right))
  | .lessThan left right => do
      pure (Testpilot.Authoring.ProgramExpr.compare .COMPARISON_OPERATOR_LESS_THAN
        (← programExpression sourceDefinitionId left) (← programExpression sourceDefinitionId right))
  | .lessThanOrEqual left right => do
      pure (Testpilot.Authoring.ProgramExpr.compare .COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL
        (← programExpression sourceDefinitionId left) (← programExpression sourceDefinitionId right))
  | .greaterThan left right => do
      pure (Testpilot.Authoring.ProgramExpr.compare .COMPARISON_OPERATOR_GREATER_THAN
        (← programExpression sourceDefinitionId left) (← programExpression sourceDefinitionId right))
  | .greaterThanOrEqual left right => do
      pure (Testpilot.Authoring.ProgramExpr.compare .COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL
        (← programExpression sourceDefinitionId left) (← programExpression sourceDefinitionId right))
  | .negation operand => do
      pure (Testpilot.Authoring.ProgramExpr.negation
        (← programExpression sourceDefinitionId operand))
  | .all operands => do
      pure (Testpilot.Authoring.ProgramExpr.all
        (← operands.mapM (programExpression sourceDefinitionId)).toArray)
  | .any operands => do
      pure (Testpilot.Authoring.ProgramExpr.any
        (← operands.mapM (programExpression sourceDefinitionId)).toArray)
  | .observation _ | .capture _ | .runEvent _ =>
      .error (loweringError sourceDefinitionId "program.expression-context")

private partial def contractExpression (sourceDefinitionId : String) :
    Umpire.Case.ValueExpression → Except LoweringError temporal.server.api.testpilot.v1.ContractExpression
  | .literal item => do
      pure (Testpilot.Authoring.ContractExpr.literal
        (← value sourceDefinitionId "property.enum-range" item))
  | .observation item => pure (Testpilot.Authoring.ContractExpr.observation item.observationId)
  | .capture item => pure (Testpilot.Authoring.ContractExpr.capture item.captureId)
  | .runEvent item => pure (Testpilot.Authoring.ContractExpr.runEvent (runEventField item))
  | .path source item => do
      pure (Testpilot.Authoring.ContractExpr.path (← contractExpression sourceDefinitionId source)
        (← fieldPath sourceDefinitionId "property.enum-range" item))
  | .present operand => do
      pure (Testpilot.Authoring.ContractExpr.present
        (← contractExpression sourceDefinitionId operand))
  | .equals left right => do
      pure (Testpilot.Authoring.ContractExpr.equals (← contractExpression sourceDefinitionId left)
        (← contractExpression sourceDefinitionId right))
  | .lessThan left right => do
      pure (Testpilot.Authoring.ContractExpr.compare .COMPARISON_OPERATOR_LESS_THAN
        (← contractExpression sourceDefinitionId left)
        (← contractExpression sourceDefinitionId right))
  | .lessThanOrEqual left right => do
      pure (Testpilot.Authoring.ContractExpr.compare .COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL
        (← contractExpression sourceDefinitionId left)
        (← contractExpression sourceDefinitionId right))
  | .greaterThan left right => do
      pure (Testpilot.Authoring.ContractExpr.compare .COMPARISON_OPERATOR_GREATER_THAN
        (← contractExpression sourceDefinitionId left)
        (← contractExpression sourceDefinitionId right))
  | .greaterThanOrEqual left right => do
      pure (Testpilot.Authoring.ContractExpr.compare .COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL
        (← contractExpression sourceDefinitionId left)
        (← contractExpression sourceDefinitionId right))
  | .negation operand => do
      pure (Testpilot.Authoring.ContractExpr.negation
        (← contractExpression sourceDefinitionId operand))
  | .all operands => do
      pure (Testpilot.Authoring.ContractExpr.all
        (← operands.mapM (contractExpression sourceDefinitionId)).toArray)
  | .any operands => do
      pure (Testpilot.Authoring.ContractExpr.any
        (← operands.mapM (contractExpression sourceDefinitionId)).toArray)
  | .slot _ | .outcome _ =>
      .error (loweringError sourceDefinitionId "property.expression-context")

private def roleKind : Umpire.Case.SymbolicRoleKind → temporal.server.api.testpilot.v1.RoleKind
  | .endpoint => .ROLE_KIND_ENDPOINT
  | .worker => .ROLE_KIND_WORKER
  | .taskQueue => .ROLE_KIND_TASK_QUEUE
  | .participant => .ROLE_KIND_PARTICIPANT

private def projectionKind : Umpire.Case.ProjectionCardinality →
    temporal.server.api.testpilot.v1.ProjectionKind
  | .one => .PROJECTION_KIND_ONE
  | .emitEach => .PROJECTION_KIND_EMIT_EACH

private def projectionTarget : Umpire.Case.ProjectionSink →
    temporal.server.api.testpilot.v1.ProjectionTarget
  | .slot id => Testpilot.Authoring.Program.slotTarget id
  | .observation id => Testpilot.Authoring.Program.observationTarget id

private def responseKind : Umpire.Case.NexusResponseKind →
    temporal.server.api.testpilot.v1.NexusResponseKind
  | .synchronous => .NEXUS_RESPONSE_KIND_SYNCHRONOUS
  | .asynchronous => .NEXUS_RESPONSE_KIND_ASYNCHRONOUS
  | .error => .NEXUS_RESPONSE_KIND_ERROR

private def instruction (sourceDefinitionId : String) (item : Umpire.Case.Instruction) :
    Except LoweringError temporal.server.api.testpilot.v1.Instruction := match item with
  | .invokeRPC request => do
      pure (Testpilot.Authoring.Program.invokeRPC request.endpointRoleId request.method
        (← request.requestAssignments.mapM fun assignment =>
          return Testpilot.Authoring.Program.requestAssignment
            (← fieldPath sourceDefinitionId "program.enum-range" assignment.target)
            (← programExpression sourceDefinitionId assignment.value)).toArray
        (← request.responseProjections.mapM fun projection => do
          pure (Testpilot.Authoring.Program.responseProjection
            (← fieldPath sourceDefinitionId "program.enum-range" projection.source)
            (projectionKind projection.cardinality)
            (projection.sinks.map projectionTarget).toArray)).toArray)
  | .awaitSlot request => pure (Testpilot.Authoring.Program.awaitSlot request.slotId)
  | .completeNexusOperation request => do
      pure (Testpilot.Authoring.Program.completeNexusOperation request.capabilitySlotId
        (← programExpression sourceDefinitionId request.result))
  | .startNexusOperation request => do
      pure (Testpilot.Authoring.Program.startNexusOperation request.endpointRoleId request.service
        request.operation (← programExpression sourceDefinitionId request.input))
  | .awaitOutcome request =>
      pure (Testpilot.Authoring.Program.awaitOutcome (instructionRef request.instruction))
  | .finish request => do
      pure (Testpilot.Authoring.Program.finish
        (← programExpression sourceDefinitionId request.result))
  | .respondNexus request => do
      pure (Testpilot.Authoring.Program.respondNexus (responseKind request.kind)
        (← programExpression sourceDefinitionId request.result) request.capabilitySlotId)

private def instructionLimits (sourceDefinitionId : String) (item : InstructionBounds) :
    Except LoweringError temporal.server.api.testpilot.v1.InstructionLimits := do
  pure (Testpilot.Authoring.Program.instructionLimits
    (← int64OfNat sourceDefinitionId "program.instruction-limits-range" item.timeoutMilliseconds)
    (← int64OfNat sourceDefinitionId "program.instruction-limits-range" item.maxAttempts)
    (← int64OfNat sourceDefinitionId "program.instruction-limits-range" item.maxEmittedEvents)
    (← int64OfNat sourceDefinitionId "program.instruction-limits-range" item.maxResponseBytes))

private def instructionOutcome (item : InstructionOutcomeSchema) :
    temporal.server.api.testpilot.v1.InstructionOutcomeDefinition :=
  Testpilot.Authoring.Program.outcome (item.fields.map fun field =>
    Testpilot.Authoring.Program.outcomeField (outcomeField field.field)
      (valueType field.type)).toArray

private def instructionNode (sourceDefinitionId : String) (item : InstructionNode) :
    Except LoweringError temporal.server.api.testpilot.v1.InstructionDefinition := do
  let guard ← match item.guard with
    | none => pure none
    | some expression => pure (some (← programExpression sourceDefinitionId expression))
  pure (Testpilot.Authoring.Program.node item.instructionId
    (← instruction sourceDefinitionId item.instruction)
    (← instructionLimits sourceDefinitionId item.bounds)
    (dependencies := (item.dependencies.map instructionRef).toArray)
    (guard := guard)
    (outcome := some (instructionOutcome item.outcome))
    (reservations := (← item.activationReservations.mapM fun reservation => do
      Testpilot.Authoring.Program.reservation reservation.entrypointId
        <$> int64OfNat sourceDefinitionId "program.activation-reservation-range"
          reservation.count).toArray))

private def entrypoint (sourceDefinitionId : String) (item : Entrypoint) :
    Except LoweringError temporal.server.api.testpilot.v1.EntrypointDefinition := do
  let nodes ← item.nodes.mapM (instructionNode sourceDefinitionId)
  match item.activation with
  | .controller _ => pure (Testpilot.Authoring.Program.controller item.entrypointId nodes.toArray)
  | .workflow activation => pure (Testpilot.Authoring.Program.workflow item.entrypointId
      activation.workflowType activation.workerRoleId activation.taskQueueRoleId nodes.toArray)
  | .activity activation => pure (Testpilot.Authoring.Program.activity item.entrypointId
      activation.activityType activation.workerRoleId activation.taskQueueRoleId nodes.toArray)
  | .nexusHandler activation => pure (Testpilot.Authoring.Program.nexusHandler item.entrypointId
      activation.service activation.operation activation.workerRoleId activation.taskQueueRoleId
      nodes.toArray)

private def programLimits (sourceDefinitionId : String) (item : Umpire.Case.ProgramLimits) :
    Except LoweringError temporal.server.api.testpilot.v1.ProgramLimits := do
  let lower := int64OfNat sourceDefinitionId "program.limits-range"
  pure (Testpilot.Authoring.Program.limits (← lower item.maxEntrypoints) (← lower item.maxNodes)
    (← lower item.maxEdges) (← lower item.maxActivations) (← lower item.maxAttempts)
    (← lower item.maxRunEvents) (← lower item.maxExpressionDepth) (← lower item.maxPathFanout)
    (← lower item.maxRequestBytes) (← lower item.maxResponseBytes)
    (← lower item.maxTotalDurationMilliseconds) (← lower item.maxCleanupDurationMilliseconds))

private def program (sourceDefinitionId : String) (item : Umpire.Case.Program) :
    Except LoweringError temporal.server.api.testpilot.v1.Program := do
  let entrypoints ← item.entrypoints.mapM (entrypoint sourceDefinitionId)
  let cleanupNodes ← item.cleanup.nodes.mapM (instructionNode sourceDefinitionId)
  pure (Testpilot.Authoring.Program.make item.programId
    (item.roles.map fun role => Testpilot.Authoring.Program.role role.roleId
      (roleKind role.kind)).toArray
    (item.slots.map fun slot => if slot.kind == .opaqueCapability then
      Testpilot.Authoring.Program.capabilitySlot slot.slotId else
        Testpilot.Authoring.Program.valueSlot slot.slotId (valueType slot.type)).toArray
    (item.observations.map fun observation =>
      Testpilot.Authoring.Program.observation observation.observationId
        (valueType observation.type)).toArray
    entrypoints.toArray
    (Testpilot.Authoring.Program.cleanup item.cleanup.entrypointId cleanupNodes.toArray)
    (← programLimits sourceDefinitionId item.limits))

private def eventKind : Umpire.Case.RunEventKind → temporal.server.api.testpilot.v1.RunEventKind
  | .runOpened => .RUN_EVENT_KIND_RUN_OPENED
  | .activationOpened => .RUN_EVENT_KIND_ACTIVATION_OPENED
  | .instructionStarted => .RUN_EVENT_KIND_INSTRUCTION_STARTED
  | .instructionCompleted => .RUN_EVENT_KIND_INSTRUCTION_COMPLETED
  | .instructionTimedOut => .RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT
  | .activationClosed => .RUN_EVENT_KIND_ACTIVATION_CLOSED
  | .cleanupStarted => .RUN_EVENT_KIND_CLEANUP_STARTED
  | .cleanupCompleted => .RUN_EVENT_KIND_CLEANUP_COMPLETED
  | .runClosed => .RUN_EVENT_KIND_RUN_CLOSED
  | .diagnostic => .RUN_EVENT_KIND_DIAGNOSTIC

private def contractState : Umpire.Case.ContractTerminalState →
    temporal.server.api.testpilot.v1.ContractStateStatus
  | .nonterminal => .CONTRACT_STATE_STATUS_NONTERMINAL
  | .satisfied => .CONTRACT_STATE_STATUS_SATISFIED
  | .violated => .CONTRACT_STATE_STATUS_VIOLATED

private def contractSupport : Umpire.Case.ContractSupport →
    temporal.server.api.testpilot.v1.ContractSupportKind
  | .none => .CONTRACT_SUPPORT_KIND_NONE
  | .matchingEvent => .CONTRACT_SUPPORT_KIND_MATCHING_EVENT

private def captureType : Umpire.Case.ContractCaptureType →
    temporal.server.api.testpilot.v1.ContractCaptureType
  | .scalar kind => Testpilot.Authoring.Monitor.scalarCapture (scalarKind kind)
  | .enumeration name => Testpilot.Authoring.Monitor.enumCapture name
  | .message name => Testpilot.Authoring.Monitor.messageCapture name

private def contractRule (sourceDefinitionId : String) (item : Umpire.Case.ContractRule) :
    Except LoweringError temporal.server.api.testpilot.v1.ContractRuleDefinition := do
  let transitions ← item.transitions.mapM fun transition => do
    pure (Testpilot.Authoring.Monitor.transition transition.transitionId transition.sourceState
      transition.targetState
      (transition.eventKinds.map eventKind).toArray
      (← contractExpression sourceDefinitionId transition.predicate)
      (contractSupport transition.support)
      (transition.captureAssignments.map fun assignment =>
        Testpilot.Authoring.Monitor.captureAssignment assignment.captureId
          assignment.observation.observationId).toArray)
  pure (Testpilot.Authoring.Monitor.rule item.ruleId
    (match item.kind with
      | .safety => .CONTRACT_RULE_KIND_SAFETY
      | .boundedLiveness => .CONTRACT_RULE_KIND_BOUNDED_LIVENESS)
    item.initialState
    (item.states.map fun state =>
      Testpilot.Authoring.Monitor.state state.stateId (contractState state.terminal)).toArray
    transitions.toArray
    (horizon := ← item.horizon.mapM fun horizon => do
      pure (Testpilot.Authoring.Monitor.horizon
        (← int64OfNat sourceDefinitionId "property.horizon-range" horizon.elapsedMilliseconds)
        horizon.violationStateId))
    (captures := (item.captures.map fun capture =>
      Testpilot.Authoring.Monitor.capture capture.captureId (captureType capture.type)).toArray))

private def contractLimits (sourceDefinitionId : String) (item : Umpire.Case.ContractLimits) :
    Except LoweringError temporal.server.api.testpilot.v1.ContractLimits := do
  let lower := int64OfNat sourceDefinitionId "contract.limits-range"
  pure (Testpilot.Authoring.Monitor.limits (← lower item.maxRules) (← lower item.maxStates)
    (← lower item.maxTransitions) (← lower item.maxExpressionDepth) (← lower item.maxWorkPerEvent)
    (← lower item.maxTotalWork) (← lower item.maxCaptures) (← lower item.maxCaptureBytes))

private def lowerProperty : ContractLowering → Except LoweringError
    temporal.server.api.testpilot.v1.ContractRuleDefinition
  | .monitor sourceDefinition rule =>
      if sourceDefinition.kind != .property then
        .error {
          sourceDefinitionId := sourceDefinition.definitionId
          source := { path := "" }
          construct := "property.definition-kind"
        }
      else
        contractRule sourceDefinition.definitionId rule
  | .unsupported sourceDefinition source construct =>
      .error { sourceDefinitionId := sourceDefinition.definitionId, source, construct }

/-- Compile declaration-ordered checked properties without weakening unsupported constructs. -/
def compile (input : Input) : Except LoweringError temporal.server.api.testpilot.v1.Case := do
  let rules ← input.properties.mapM lowerProperty
  let major ← int32OfNat input.caseId "case.version-range" input.version.major
  let minor ← int32OfNat input.caseId "case.version-range" input.version.minor
  let loweredProgram ← program input.caseId input.program
  let loweredContractLimits ← contractLimits input.contractId input.contractLimits
  let metadata : CaseMetadata := {
    producerId := input.producerId
    producerVersion := input.producerVersion
    definitions := input.definitions
    sources := input.sources
    knownGaps := input.knownGaps
  }
  pure (Testpilot.Authoring.case major input.caseId loweredProgram
    (Testpilot.Authoring.Monitor.contract input.contractId rules.toArray
      loweredContractLimits)
    (Provenance.make metadata) minor)

end Umpire.Case.Compiler
