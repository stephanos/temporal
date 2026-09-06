import Umpire.Case
import Umpire.Json

/-!
Deterministic Testpilot ProtoJSON lowering for Temporal's authored functional Cases.

The generated Lean API is a descriptor projection, so recursive protobuf fields are intentionally
opaque `MessageRef` values. This module lowers the typed Umpire producer model directly to the
refined Testpilot JSON shape without using the former Umpire protobuf representation.
-/

namespace Temporal.Testpilot.TestpilotProtoJSON

open Umpire
open Umpire.Case
open Umpire.CanonicalJson

private def string := CanonicalJson.string
private def natural := CanonicalJson.natural
private def array (items : List CanonicalJson) := CanonicalJson.array items
private def object (fields : List (String × CanonicalJson)) := CanonicalJson.object fields
private def int64 (value : Nat) := string (toString value)
private def signed64 (value : Int) := string (toString value)

private def alphabet : List Char :=
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/".toList
private def base64Char (index : Nat) : Char := alphabet.getD index 'A'
private def encodeBase64Aux : List Nat → List Char
  | [] => []
  | [a] => [base64Char (a / 4), base64Char ((a % 4) * 16), '=', '=']
  | [a, b] => [
      base64Char (a / 4),
      base64Char ((a % 4) * 16 + b / 16),
      base64Char ((b % 16) * 4),
      '='
    ]
  | a :: b :: c :: rest => [
      base64Char (a / 4),
      base64Char ((a % 4) * 16 + b / 16),
      base64Char ((b % 16) * 4 + c / 64),
      base64Char (c % 64)
    ] ++ encodeBase64Aux rest
private def bytes (value : ByteArray) : CanonicalJson :=
  string (String.ofList (encodeBase64Aux (value.data.toList.map UInt8.toNat)))

private def scalarName : ScalarKind → String
  | .text => "SCALAR_KIND_TEXT"
  | .natural => "SCALAR_KIND_NATURAL"
  | .boolean => "SCALAR_KIND_BOOLEAN"
  | .bytes => "SCALAR_KIND_BYTES"
  | .int32 => "SCALAR_KIND_INT32"
  | .int64 => "SCALAR_KIND_INT64"
  | .uint32 => "SCALAR_KIND_UINT32"
  | .uint64 => "SCALAR_KIND_UINT64"
  | .sint32 => "SCALAR_KIND_SINT32"
  | .sint64 => "SCALAR_KIND_SINT64"
  | .fixed32 => "SCALAR_KIND_FIXED32"
  | .fixed64 => "SCALAR_KIND_FIXED64"
  | .sfixed32 => "SCALAR_KIND_SFIXED32"
  | .sfixed64 => "SCALAR_KIND_SFIXED64"
  | .float => "SCALAR_KIND_FLOAT"
  | .double => "SCALAR_KIND_DOUBLE"

private partial def value : Value → CanonicalJson
  | .text item => object [("text", string item)]
  | .natural item => object [("natural", int64 item)]
  | .boolean item => object [("boolValue", .boolean item)]
  | .bytes item => object [("bytesValue", bytes item)]
  | .signedInteger item => object [("signedInteger", signed64 item)]
  | .unsignedInteger item => object [("unsignedInteger", int64 item)]
  | .floatingPoint item => object [("floatingPoint", string item.toString)]
  | .enumValue number => object [("enumValue", object [("number", .string (toString number))])]
  | .messageValue item => object [("messageValue", object [
      ("@type", string item.typeUrl), ("value", bytes item.bytes)
    ])]
  | .listValue items => object [("listValue", object [("values", array (items.map value))])]
  | .mapValue entries => object [("mapValue", object [("entries", array (entries.map fun entry =>
      object [("key", value entry.1), ("value", value entry.2)]))])]

private def singularType : SingularType → CanonicalJson
  | .scalar kind => object [("scalar", object [("kind", string (scalarName kind))])]
  | .enumeration name => object [("enumeration", object [("protobufType", string name)])]
  | .message name => object [("message", object [("protobufType", string name)])]
  | .any => object [("any", object [])]
  | .opaqueCapability => object [("opaqueCapability", object [])]
private def valueType : ValueType → CanonicalJson
  | .singular item => object [("singular", singularType item)]
  | .repeated item => object [("repeated", object [("element", singularType item)])]
  | .map key item => object [("map", object [
      ("key", object [("kind", string (scalarName key))]), ("value", singularType item)
    ])]

private def selector : FieldSelector → String × CanonicalJson
  | .repeated => ("repeated", object [])
  | .mapKey key => ("mapKey", object [("key", value key)])
  | .presence => ("presence", object [])
  | .oneof selected => ("oneof", object [("selectedField", string selected)])
private def path (item : FieldPath) : CanonicalJson := object [("segments", array (item.segments.map fun segment =>
  object ([("field", string segment.field)] ++ segment.selector.toList.map selector)))]
private def instructionRef (item : InstructionReference) : CanonicalJson := object [
  ("entrypointId", string item.entrypointId), ("instructionId", string item.instructionId)
]

private def outcomeFieldName : InstructionOutcomeField → String
  | .status => "INSTRUCTION_OUTCOME_FIELD_STATUS"
  | .protocolCode => "INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE"
  | .sdkFailureCode => "INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE"
  | .detail => "INSTRUCTION_OUTCOME_FIELD_DETAIL"
  | .value => "INSTRUCTION_OUTCOME_FIELD_VALUE"
private def runEventFieldName : RunEventField → String
  | .sequence => "RUN_EVENT_FIELD_SEQUENCE"
  | .elapsedMilliseconds => "RUN_EVENT_FIELD_ELAPSED_MILLISECONDS"
  | .kind => "RUN_EVENT_FIELD_KIND"
  | .entrypointId => "RUN_EVENT_FIELD_ENTRYPOINT_ID"
  | .activationId => "RUN_EVENT_FIELD_ACTIVATION_ID"
  | .instructionId => "RUN_EVENT_FIELD_INSTRUCTION_ID"
  | .attempt => "RUN_EVENT_FIELD_ATTEMPT"
  | .sourceId => "RUN_EVENT_FIELD_SOURCE_ID"
  | .runId => "RUN_EVENT_FIELD_RUN_ID"

private partial def programExpression : ValueExpression → Except String CanonicalJson
  | .literal item => pure (object [("literal", value item)])
  | .slot item => pure (object [("slot", object [("slotId", string item.slotId)])])
  | .outcome item => pure (object [("outcome", object [
      ("instruction", instructionRef item.instruction),
      ("field", string (outcomeFieldName item.field))
    ])])
  | .runEvent .runId => pure (object [("run", object [])])
  | .path source item => do
      pure (object [("path", object [
        ("source", ← programExpression source), ("path", path item)
      ])])
  | .present operand => do
      pure (object [("present", object [("operand", ← programExpression operand)])])
  | .equals left right => do
      pure (object [("equals", object [
        ("left", ← programExpression left), ("right", ← programExpression right)
      ])])
  | .lessThan left right => programCompare "COMPARISON_OPERATOR_LESS_THAN" left right
  | .lessThanOrEqual left right =>
      programCompare "COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL" left right
  | .greaterThan left right => programCompare "COMPARISON_OPERATOR_GREATER_THAN" left right
  | .greaterThanOrEqual left right =>
      programCompare "COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL" left right
  | .negation operand => do
      pure (object [("negation", object [("operand", ← programExpression operand)])])
  | .all operands => do
      pure (object [("all", object [("operands", array (← operands.mapM programExpression))])])
  | .any operands => do
      pure (object [("any", object [("operands", array (← operands.mapM programExpression))])])
  | .observation _ | .capture _ | .runEvent _ =>
      throw "program expression contains a Contract-only reference"
where
  programCompare (operator : String) (left right : ValueExpression) : Except String CanonicalJson := do
    pure (object [("compare", object [
      ("operator", string operator),
      ("left", ← programExpression left), ("right", ← programExpression right)
    ])])

private partial def contractExpression : ValueExpression → Except String CanonicalJson
  | .literal item => pure (object [("literal", value item)])
  | .observation item =>
      pure (object [("observation", object [("observationId", string item.observationId)])])
  | .capture item => pure (object [("capture", object [("captureId", string item.captureId)])])
  | .runEvent item =>
      pure (object [("runEvent", object [("field", string (runEventFieldName item))])])
  | .path source item => do
      pure (object [("path", object [
        ("source", ← contractExpression source), ("path", path item)
      ])])
  | .present operand => do
      pure (object [("present", object [("operand", ← contractExpression operand)])])
  | .equals left right => do
      pure (object [("equals", object [
        ("left", ← contractExpression left), ("right", ← contractExpression right)
      ])])
  | .lessThan left right => contractCompare "COMPARISON_OPERATOR_LESS_THAN" left right
  | .lessThanOrEqual left right =>
      contractCompare "COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL" left right
  | .greaterThan left right => contractCompare "COMPARISON_OPERATOR_GREATER_THAN" left right
  | .greaterThanOrEqual left right =>
      contractCompare "COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL" left right
  | .negation operand => do
      pure (object [("negation", object [("operand", ← contractExpression operand)])])
  | .all operands => do
      pure (object [("all", object [("operands", array (← operands.mapM contractExpression))])])
  | .any operands => do
      pure (object [("any", object [("operands", array (← operands.mapM contractExpression))])])
  | .slot _ | .outcome _ => throw "contract expression contains a Program-only reference"
where
  contractCompare (operator : String) (left right : ValueExpression) : Except String CanonicalJson := do
    pure (object [("compare", object [
      ("operator", string operator),
      ("left", ← contractExpression left), ("right", ← contractExpression right)
    ])])

private def roleName : SymbolicRoleKind → String
  | .endpoint => "ROLE_KIND_ENDPOINT"
  | .worker => "ROLE_KIND_WORKER"
  | .taskQueue => "ROLE_KIND_TASK_QUEUE"
  | .participant => "ROLE_KIND_PARTICIPANT"
private def activation : ActivationBinding → String × CanonicalJson
  | .controller _ => ("controller", object [])
  | .workflow item => ("workflow", object [
      ("workflowType", string item.workflowType), ("workerRoleId", string item.workerRoleId),
      ("taskQueueRoleId", string item.taskQueueRoleId)
    ])
  | .activity item => ("activity", object [
      ("activityType", string item.activityType), ("workerRoleId", string item.workerRoleId),
      ("taskQueueRoleId", string item.taskQueueRoleId)
    ])
  | .nexusHandler item => ("nexusHandler", object [
      ("service", string item.service), ("operation", string item.operation),
      ("workerRoleId", string item.workerRoleId), ("taskQueueRoleId", string item.taskQueueRoleId)
    ])

private def outcomeDefinition (schema : InstructionOutcomeSchema) : CanonicalJson := object [
  ("fields", array (schema.fields.map fun item => object [
    ("field", string (outcomeFieldName item.field)), ("type", valueType item.type)
  ]))
]
private def instructionLimits (item : InstructionBounds) : CanonicalJson := object [
  ("timeoutMilliseconds", int64 item.timeoutMilliseconds),
  ("maxAttempts", int64 item.maxAttempts),
  ("maxEmittedEvents", int64 item.maxEmittedEvents),
  ("maxResponseBytes", int64 item.maxResponseBytes)
]
private def projectionKind : ProjectionCardinality → String
  | .one => "PROJECTION_KIND_ONE"
  | .emitEach => "PROJECTION_KIND_EMIT_EACH"
private def target : ProjectionSink → CanonicalJson
  | .slot id => object [("slotId", string id)]
  | .observation id => object [("observationId", string id)]
private def projection (item : ResponseProjection) : CanonicalJson := object [
  ("source", path item.source), ("kind", string (projectionKind item.cardinality)),
  ("targets", array (item.sinks.map target))
]
private def responseKind : NexusResponseKind → String
  | .synchronous => "NEXUS_RESPONSE_KIND_SYNCHRONOUS"
  | .asynchronous => "NEXUS_RESPONSE_KIND_ASYNCHRONOUS"
  | .error => "NEXUS_RESPONSE_KIND_ERROR"
private def instruction (item : Instruction) : Except String CanonicalJson := do
  match item with
  | .invokeRPC request =>
      let assignments ← request.requestAssignments.mapM fun assignment => do
        pure (object [("target", path assignment.target),
          ("value", ← programExpression assignment.value)])
      pure (object [("invokeRpc", object [
        ("endpointRoleId", string request.endpointRoleId), ("method", string request.method),
        ("requestAssignments", array assignments),
        ("responseProjections", array (request.responseProjections.map projection))
      ])])
  | .awaitSlot request => pure (object [("awaitSlot", object [("slotId", string request.slotId)])])
  | .completeNexusOperation request => do
      pure (object [("completeNexusOperation", object [
        ("capabilitySlotId", string request.capabilitySlotId),
        ("result", ← programExpression request.result)
      ])])
  | .startNexusOperation request => do
      pure (object [("startNexusOperation", object [
        ("endpointRoleId", string request.endpointRoleId), ("service", string request.service),
        ("operation", string request.operation), ("input", ← programExpression request.input)
      ])])
  | .awaitOutcome request => pure (object [("awaitOutcome", object [
      ("instruction", instructionRef request.instruction)
    ])])
  | .finish request => do
      pure (object [("finish", object [("result", ← programExpression request.result)])])
  | .respondNexus request => do
      pure (object [("respondNexus", object [
        ("kind", string (responseKind request.kind)),
        ("result", ← programExpression request.result),
        ("capabilitySlotId", string request.capabilitySlotId)
      ])])

private def instructionDefinition (item : InstructionNode) : Except String CanonicalJson := do

  let guardFields ← match item.guard with
    | some guard => pure [("guard", ← programExpression guard)]
    | none => pure []
  pure (object ([
    ("instructionId", string item.instructionId),
    ("dependencies", array (item.dependencies.map instructionRef))
  ] ++ guardFields ++ [
    ("instruction", ← instruction item.instruction),
    ("outcome", outcomeDefinition item.outcome),
    ("limits", instructionLimits item.bounds),
    ("activationReservations", array (item.activationReservations.map fun reservation => object [
      ("entrypointId", string reservation.entrypointId), ("count", int64 reservation.count)
    ]))
  ]))

private def programLimits (item : ProgramLimits) : CanonicalJson := object [
  ("maxEntrypoints", int64 item.maxEntrypoints), ("maxNodes", int64 item.maxNodes),
  ("maxEdges", int64 item.maxEdges), ("maxActivations", int64 item.maxActivations),
  ("maxAttempts", int64 item.maxAttempts), ("maxRunEvents", int64 item.maxRunEvents),
  ("maxExpressionDepth", int64 item.maxExpressionDepth),
  ("maxPathFanout", int64 item.maxPathFanout), ("maxRequestBytes", int64 item.maxRequestBytes),
  ("maxResponseBytes", int64 item.maxResponseBytes),
  ("maxTotalDurationMilliseconds", int64 item.maxTotalDurationMilliseconds),
  ("maxCleanupDurationMilliseconds", int64 item.maxCleanupDurationMilliseconds)
]
private def program (item : Program) : Except String CanonicalJson := do
  let entrypoints ← item.entrypoints.mapM fun entrypoint => do
    pure (object ([
      ("entrypointId", string entrypoint.entrypointId), activation entrypoint.activation,
      ("instructions", array (← entrypoint.nodes.mapM instructionDefinition))
    ]))
  pure (object [
    ("programId", string item.programId),
    ("roles", array (item.roles.map fun role => object [
      ("roleId", string role.roleId), ("kind", string (roleName role.kind))
    ])),
    ("slots", array (item.slots.map fun slot => object [
      ("slotId", string slot.slotId),
      if slot.kind == .opaqueCapability then ("opaqueCapability", object [])
      else ("value", valueType slot.type)
    ])),
    ("observations", array (item.observations.map fun observation => object [
      ("observationId", string observation.observationId), ("type", valueType observation.type)
    ])),
    ("entrypoints", array entrypoints),
    ("cleanup", object [
      ("entrypointId", string item.cleanup.entrypointId),
      ("instructions", array (← item.cleanup.nodes.mapM instructionDefinition))
    ]),
    ("limits", programLimits item.limits)
  ])

private def eventKindName : RunEventKind → String
  | .runOpened => "RUN_EVENT_KIND_RUN_OPENED"
  | .activationOpened => "RUN_EVENT_KIND_ACTIVATION_OPENED"
  | .instructionStarted => "RUN_EVENT_KIND_INSTRUCTION_STARTED"
  | .instructionCompleted => "RUN_EVENT_KIND_INSTRUCTION_COMPLETED"
  | .instructionTimedOut => "RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT"
  | .activationClosed => "RUN_EVENT_KIND_ACTIVATION_CLOSED"
  | .cleanupStarted => "RUN_EVENT_KIND_CLEANUP_STARTED"
  | .cleanupCompleted => "RUN_EVENT_KIND_CLEANUP_COMPLETED"
  | .runClosed => "RUN_EVENT_KIND_RUN_CLOSED"
  | .diagnostic => "RUN_EVENT_KIND_DIAGNOSTIC"
private def ruleKindName : ContractRuleKind → String
  | .safety => "CONTRACT_RULE_KIND_SAFETY"
  | .boundedLiveness => "CONTRACT_RULE_KIND_BOUNDED_LIVENESS"
private def stateStatusName : ContractTerminalState → String
  | .nonterminal => "CONTRACT_STATE_STATUS_NONTERMINAL"
  | .satisfied => "CONTRACT_STATE_STATUS_SATISFIED"
  | .violated => "CONTRACT_STATE_STATUS_VIOLATED"
private def supportKindName : ContractSupport → String
  | .none => "CONTRACT_SUPPORT_KIND_NONE"
  | .matchingEvent => "CONTRACT_SUPPORT_KIND_MATCHING_EVENT"
private def captureType : ContractCaptureType → CanonicalJson
  | .scalar kind => object [("scalar", object [("kind", string (scalarName kind))])]
  | .enumeration name => object [("enumeration", object [("protobufType", string name)])]
  | .message name => object [("message", object [("protobufType", string name)])]
private def rule (item : ContractRule) : Except String CanonicalJson := do
  let transitions ← item.transitions.mapM fun transition => do
    pure (object [
      ("transitionId", string transition.transitionId),
      ("sourceStateId", string transition.sourceState),
      ("targetStateId", string transition.targetState),
      ("eventFilter", object [
        ("kinds", array (transition.eventKinds.map fun kind => string (eventKindName kind)))
      ]),
      ("predicate", ← contractExpression transition.predicate),
      ("supportKind", string (supportKindName transition.support)),
      ("captureAssignments", array (transition.captureAssignments.map fun assignment => object [
        ("captureId", string assignment.captureId),
        ("observation", object [("observationId", string assignment.observation.observationId)])
      ]))
    ])
  pure (object ([
    ("ruleId", string item.ruleId), ("kind", string (ruleKindName item.kind)),
    ("initialStateId", string item.initialState),
    ("states", array (item.states.map fun state => object [
      ("stateId", string state.stateId), ("status", string (stateStatusName state.terminal))
    ])),
    ("transitions", array transitions),
    ("captures", array (item.captures.map fun capture => object [
      ("captureId", string capture.captureId), ("type", captureType capture.type)
    ]))
  ] ++ (item.horizon.toList.map fun horizon => ("horizon", object [
    ("elapsedMilliseconds", int64 horizon.elapsedMilliseconds),
    ("violationStateId", string horizon.violationStateId)
  ]))))

private def contractLimits (item : ContractLimits) : CanonicalJson := object [
  ("maxRules", int64 item.maxRules), ("maxStates", int64 item.maxStates),
  ("maxTransitions", int64 item.maxTransitions),
  ("maxExpressionDepth", int64 item.maxExpressionDepth),
  ("maxWorkPerEvent", int64 item.maxWorkPerEvent), ("maxTotalWork", int64 item.maxTotalWork),
  ("maxCaptures", int64 item.maxCaptures), ("maxCaptureBytes", int64 item.maxCaptureBytes)
]
private def contract (item : Contract) : Except String CanonicalJson := do
  pure (object [
    ("contractId", string item.contractId), ("rules", array (← item.rules.mapM rule)),
    ("limits", contractLimits item.limits)
  ])

private def definitionKind : CaseDefinitionKind → String
  | .setup => "CASE_DEFINITION_KIND_SETUP"
  | .state => "CASE_DEFINITION_KIND_STATE"
  | .action => "CASE_DEFINITION_KIND_ACTION"
  | .outcome => "CASE_DEFINITION_KIND_OUTCOME"
  | .observation => "CASE_DEFINITION_KIND_OBSERVATION"
  | .relation => "CASE_DEFINITION_KIND_RELATION"
  | .capability => "CASE_DEFINITION_KIND_CAPABILITY"
  | .property => "CASE_DEFINITION_KIND_PROPERTY"
  | .query => "CASE_DEFINITION_KIND_QUERY"
  | .behavior => "CASE_DEFINITION_KIND_BEHAVIOR"
  | .target => "CASE_DEFINITION_KIND_TARGET"
  | .compiler => "CASE_DEFINITION_KIND_COMPILER"
  | .provider => "CASE_DEFINITION_KIND_PROVIDER"
  | .law => "CASE_DEFINITION_KIND_LAW"
  | .connector => "CASE_DEFINITION_KIND_CONNECTOR"
  | .kernel => "CASE_DEFINITION_KIND_KERNEL"
  | .experimentSpace => "CASE_DEFINITION_KIND_EXPERIMENT_SPACE"
  | .variationAxis => "CASE_DEFINITION_KIND_VARIATION_AXIS"
  | .choice => "CASE_DEFINITION_KIND_CHOICE"
  | .fault => "CASE_DEFINITION_KIND_FAULT"
  | .coverageGoal => "CASE_DEFINITION_KIND_COVERAGE_GOAL"
private def gapKind : CaseKnownGapKind → String
  | .capabilityContract => "CASE_KNOWN_GAP_KIND_CAPABILITY_CONTRACT"
  | .input => "CASE_KNOWN_GAP_KIND_INPUT"
  | .interpretation => "CASE_KNOWN_GAP_KIND_INTERPRETATION"
  | .claim => "CASE_KNOWN_GAP_KIND_CLAIM"
private def sourceLocation (item : SourceLocation) : CanonicalJson := object [
  ("path", string item.path), ("line", int64 item.line), ("column", int64 item.column),
  ("provenance", string item.provenance)
]
private def producerData (item : CaseMetadata) : ByteArray := (object [
  ("definitions", array (item.definitions.map fun definition => object [
    ("definitionId", string definition.definitionId),
    ("behaviorFingerprint", string definition.behaviorFingerprint),
    ("kind", string (definitionKind definition.kind))
  ])),
  ("sources", array (item.sources.map sourceLocation)),
  ("knownGaps", array (item.knownGaps.map fun gap => object ([
    ("kind", string (gapKind gap.kind)), ("code", string gap.code)
  ] ++ gap.subject.toList.map (fun subject => ("subject", string subject)) ++
    gap.detail.toList.map (fun detail => ("detail", string detail)))))
]).prettyBytes.toUTF8

/-- Render one authored Case directly as deterministic ProtoJSON for the refined Testpilot schema. -/
def canonical (item : Case) : Except String String := do
  pure (object [
    ("version", object [("major", natural item.version.major), ("minor", natural item.version.minor)]),
    ("caseId", string item.caseId),
    ("provenance", object [
      ("producerId", string item.metadata.producerId),
      ("producerVersion", string item.metadata.producerVersion),
      ("producerData", bytes (producerData item.metadata))
    ]),
    ("program", ← program item.program),
    ("contract", ← contract item.contract)
  ]).prettyBytes

end Temporal.Testpilot.TestpilotProtoJSON
