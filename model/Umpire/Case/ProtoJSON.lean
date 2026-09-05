import Umpire.Case
import Umpire.Json

namespace Umpire.Case.ProtoJSON

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
private def instructionReference (item : InstructionReference) : CanonicalJson := object [
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

private partial def expression : ValueExpression → CanonicalJson
  | .literal item => object [("literal", value item)]
  | .slot item => object [("slot", object [("slotId", string item.slotId)])]
  | .outcome item => object [("outcome", object [
      ("instruction", instructionReference item.instruction),
      ("field", string (outcomeFieldName item.field))
    ])]
  | .observation item =>
      object [("observation", object [("observationId", string item.observationId)])]
  | .capture item => object [("capture", object [("captureId", string item.captureId)])]
  | .runEvent item =>
      object [("runEvent", object [("field", string (runEventFieldName item))])]
  | .path source item =>
      object [("path", object [("source", expression source), ("path", path item)])]
  | .present operand => object [("present", object [("operand", expression operand)])]
  | .equals left right =>
      object [("equals", object [("left", expression left), ("right", expression right)])]
  | .lessThan left right => object [("compare", object [
      ("operator", string "COMPARISON_OPERATOR_LESS_THAN"),
      ("left", expression left), ("right", expression right)
    ])]
  | .lessThanOrEqual left right =>
      object [("compare", object [
        ("operator", string "COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL"),
        ("left", expression left), ("right", expression right)
      ])]
  | .greaterThan left right => object [("compare", object [
      ("operator", string "COMPARISON_OPERATOR_GREATER_THAN"),
      ("left", expression left), ("right", expression right)
    ])]
  | .greaterThanOrEqual left right =>
      object [("compare", object [
        ("operator", string "COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL"),
        ("left", expression left), ("right", expression right)
      ])]
  | .negation operand => object [("negation", object [("operand", expression operand)])]
  | .all operands => object [("all", object [("operands", array (operands.map expression))])]
  | .any operands => object [("any", object [("operands", array (operands.map expression))])]

private def contextName : EntrypointContext → String
  | .controller => "ENTRYPOINT_CONTEXT_CONTROLLER"
  | .workflow => "ENTRYPOINT_CONTEXT_WORKFLOW"
  | .activity => "ENTRYPOINT_CONTEXT_ACTIVITY"
  | .nexusHandler => "ENTRYPOINT_CONTEXT_NEXUS_HANDLER"
private def roleName : SymbolicRoleKind → String
  | .endpoint => "SYMBOLIC_ROLE_KIND_ENDPOINT"
  | .worker => "SYMBOLIC_ROLE_KIND_WORKER"
  | .taskQueue => "SYMBOLIC_ROLE_KIND_TASK_QUEUE"
  | .participant => "SYMBOLIC_ROLE_KIND_PARTICIPANT"
private def activation : ActivationBinding → CanonicalJson
  | .controller _ => object [("controller", object [])]
  | .workflow item => object [("workflow", object [
      ("workflowType", string item.workflowType), ("workerRoleId", string item.workerRoleId),
      ("taskQueueRoleId", string item.taskQueueRoleId)
    ])]
  | .activity item => object [("activity", object [
      ("activityType", string item.activityType), ("workerRoleId", string item.workerRoleId),
      ("taskQueueRoleId", string item.taskQueueRoleId)
    ])]
  | .nexusHandler item => object [("nexusHandler", object [
      ("service", string item.service), ("operation", string item.operation),
      ("workerRoleId", string item.workerRoleId), ("taskQueueRoleId", string item.taskQueueRoleId)
    ])]

private def outcomeSchema (schema : InstructionOutcomeSchema) : CanonicalJson := object [
  ("fields", array (schema.fields.map fun item => object [
    ("field", string (outcomeFieldName item.field)), ("type", valueType item.type)
  ]))
]
private def boundsJson (item : InstructionBounds) : CanonicalJson := object [
  ("timeoutMilliseconds", int64 item.timeoutMilliseconds),
  ("maxAttempts", int64 item.maxAttempts),
  ("maxEmittedEvents", int64 item.maxEmittedEvents),
  ("maxResponseBytes", int64 item.maxResponseBytes)
]
private def projectionCardinality : ProjectionCardinality → String
  | .one => "PROJECTION_CARDINALITY_ONE"
  | .emitEach => "PROJECTION_CARDINALITY_EMIT_EACH"
private def sink : ProjectionSink → CanonicalJson
  | .slot id => object [("slotId", string id)]
  | .observation id => object [("observationId", string id)]
private def projection (item : ResponseProjection) : CanonicalJson := object [
  ("source", path item.source),
  ("cardinality", string (projectionCardinality item.cardinality)),
  ("sinks", array (item.sinks.map sink))
]
private def responseKind : NexusResponseKind → String
  | .synchronous => "NEXUS_RESPONSE_KIND_SYNCHRONOUS"
  | .asynchronous => "NEXUS_RESPONSE_KIND_ASYNCHRONOUS"
  | .error => "NEXUS_RESPONSE_KIND_ERROR"
private def instruction : Instruction → CanonicalJson
  | .invokeRPC item => object [("invokeRpc", object [
      ("endpointRoleId", string item.endpointRoleId), ("method", string item.method),
      ("requestAssignments", array (item.requestAssignments.map fun assignment =>
        object [("target", path assignment.target), ("value", expression assignment.value)])),
      ("responseProjections", array (item.responseProjections.map projection))
    ])]
  | .awaitSlot item => object [("awaitSlot", object [("slotId", string item.slotId)])]
  | .completeNexusOperation item => object [("completeNexusOperation", object [
      ("capabilitySlotId", string item.capabilitySlotId), ("result", expression item.result)
    ])]
  | .startNexusOperation item => object [("startNexusOperation", object [
      ("endpointRoleId", string item.endpointRoleId), ("service", string item.service),
      ("operation", string item.operation), ("input", expression item.input)
    ])]
  | .awaitOutcome item => object [("awaitOutcome", object [
      ("instruction", instructionReference item.instruction)
    ])]
  | .finish item => object [("finish", object [("result", expression item.result)])]
  | .respondNexus item => object [("respondNexus", object [
      ("kind", string (responseKind item.kind)), ("result", expression item.result),
      ("capabilitySlotId", string item.capabilitySlotId)
    ])]

private def node (item : InstructionNode) : CanonicalJson := object ([
  ("instructionId", string item.instructionId),
  ("dependencies", array (item.dependencies.map instructionReference))
] ++ (item.guard.toList.map fun guard => ("guard", expression guard)) ++ [
  ("instruction", instruction item.instruction),
  ("outcome", outcomeSchema item.outcome),
  ("bounds", boundsJson item.bounds),
  ("activationReservations", array (item.activationReservations.map fun reservation => object [
    ("entrypointId", string reservation.entrypointId), ("count", int64 reservation.count)
  ]))
])

private def programLimitsJson (item : ProgramLimits) : CanonicalJson := object [
  ("maxEntrypoints", int64 item.maxEntrypoints), ("maxNodes", int64 item.maxNodes),
  ("maxEdges", int64 item.maxEdges), ("maxActivations", int64 item.maxActivations),
  ("maxAttempts", int64 item.maxAttempts), ("maxRunEvents", int64 item.maxRunEvents),
  ("maxExpressionDepth", int64 item.maxExpressionDepth), ("maxPathFanout", int64 item.maxPathFanout),
  ("maxRequestBytes", int64 item.maxRequestBytes), ("maxResponseBytes", int64 item.maxResponseBytes),
  ("maxTotalDurationMilliseconds", int64 item.maxTotalDurationMilliseconds),
  ("maxCleanupDurationMilliseconds", int64 item.maxCleanupDurationMilliseconds)
]
private def program (item : Program) : CanonicalJson := object [
  ("programId", string item.programId),
  ("roles", array (item.roles.map fun role =>
    object [("roleId", string role.roleId), ("kind", string (roleName role.kind))])),
  ("slots", array (item.slots.map fun slot => object [
    ("slotId", string slot.slotId), ("type", valueType slot.type),
    ("kind", string (if slot.kind == .value then "SLOT_KIND_VALUE" else "SLOT_KIND_OPAQUE_CAPABILITY"))
  ])),
  ("observations", array (item.observations.map fun schema => object [
    ("observationId", string schema.observationId), ("type", valueType schema.type)
  ])),
  ("entrypoints", array (item.entrypoints.map fun entrypoint => object [
    ("entrypointId", string entrypoint.entrypointId),
    ("context", string (contextName entrypoint.context)),
    ("activation", activation entrypoint.activation),
    ("nodes", array (entrypoint.nodes.map node))
  ])),
  ("cleanup", object [
    ("entrypointId", string item.cleanup.entrypointId),
    ("context", string (contextName item.cleanup.context)),
    ("nodes", array (item.cleanup.nodes.map node))
  ]),
  ("limits", programLimitsJson item.limits)
]

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
private def terminalName : ContractTerminalState → String
  | .nonterminal => "CONTRACT_TERMINAL_STATE_NONTERMINAL"
  | .satisfied => "CONTRACT_TERMINAL_STATE_SATISFIED"
  | .violated => "CONTRACT_TERMINAL_STATE_VIOLATED"
private def supportName : ContractSupport → String
  | .none => "CONTRACT_SUPPORT_NONE"
  | .matchingEvent => "CONTRACT_SUPPORT_MATCHING_EVENT"
private def captureType : ContractCaptureType → CanonicalJson
  | .scalar kind => object [("scalar", object [("kind", string (scalarName kind))])]
  | .enumeration name => object [("enumeration", object [("protobufType", string name)])]
  | .message name => object [("message", object [("protobufType", string name)])]
private def rule (item : ContractRule) : CanonicalJson := object ([
  ("ruleId", string item.ruleId), ("kind", string (ruleKindName item.kind)),
  ("initialState", string item.initialState),
  ("states", array (item.states.map fun state => object [
    ("stateId", string state.stateId), ("terminal", string (terminalName state.terminal))
  ])),
  ("transitions", array (item.transitions.map fun transition => object [
    ("transitionId", string transition.transitionId),
    ("sourceState", string transition.sourceState),
    ("targetState", string transition.targetState),
    ("eventKinds", object [
      ("kinds", array (transition.eventKinds.map fun kind => string (eventKindName kind)))
    ]),
    ("predicate", expression transition.predicate),
    ("support", string (supportName transition.support)),
    ("captureAssignments", array (transition.captureAssignments.map fun assignment => object [
      ("captureId", string assignment.captureId),
      ("observation", object [("observationId", string assignment.observation.observationId)])
    ]))
  ])),
  ("captures", array (item.captures.map fun capture => object [
    ("captureId", string capture.captureId), ("type", captureType capture.type)
  ]))
] ++ (item.horizon.toList.map fun horizon => ("horizon", object [
  ("elapsedMilliseconds", int64 horizon.elapsedMilliseconds),
  ("violationStateId", string horizon.violationStateId)
])))

private def contractLimitsJson (item : ContractLimits) : CanonicalJson := object [
  ("maxRules", int64 item.maxRules), ("maxStates", int64 item.maxStates),
  ("maxTransitions", int64 item.maxTransitions), ("maxExpressionDepth", int64 item.maxExpressionDepth),
  ("maxWorkPerEvent", int64 item.maxWorkPerEvent), ("maxTotalWork", int64 item.maxTotalWork),
  ("maxCaptures", int64 item.maxCaptures), ("maxCaptureBytes", int64 item.maxCaptureBytes)
]
private def contract (item : Contract) : CanonicalJson := object [
  ("contractId", string item.contractId), ("rules", array (item.rules.map rule)),
  ("limits", contractLimitsJson item.limits)
]
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
private def metadata (item : CaseMetadata) : CanonicalJson := object [
  ("producerId", string item.producerId), ("producerVersion", string item.producerVersion),
  ("definitions", array (item.definitions.map fun definition => object [
    ("definitionId", string definition.definitionId),
    ("behaviorFingerprint", string definition.behaviorFingerprint),
    ("kind", string (definitionKind definition.kind))
  ])),
  ("sources", array (item.sources.map sourceLocation)),
  ("knownGaps", array (item.knownGaps.map fun gap => object ([
    ("kind", string (gapKind gap.kind)), ("code", string gap.code)
  ] ++ gap.subject.toList.map (fun subject => ("subject", object [("value", string subject)])) ++
    gap.detail.toList.map (fun detail => ("detail", object [("value", string detail)])))))
]

/-- Render one complete Case as deterministic ProtoJSON accepted by Go's protobuf decoder. -/
def canonical (item : Case) : String := (object [
  ("version", object [("major", natural item.version.major), ("minor", natural item.version.minor)]),
  ("caseId", string item.caseId),
  ("metadata", metadata item.metadata),
  ("program", program item.program),
  ("contract", contract item.contract)
]).prettyBytes

end Umpire.Case.ProtoJSON
