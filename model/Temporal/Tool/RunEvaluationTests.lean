import Temporal.Tool.RunEvaluation

/-! Exact protocol and semantic conformance for the closed caller-closure evaluator. -/

namespace Temporal.Tool.RunEvaluation.Tests

open Umpire
open Temporal.Tool.RunEvaluation.Protocol

private def id (value : String) : DefinitionId := DefinitionId.of value
private def json (value : String) : Lean.Json := (Lean.Json.parse value).toOption.getD .null
private def object (fields : List (String × Lean.Json)) : Lean.Json := Lean.Json.mkObj fields
private def array (values : List Lean.Json) : Lean.Json := .arr values.toArray
private def text (value : String) : Lean.Json := .str value
private def natural (value : Nat) : Lean.Json := json (toString value)
private def checksum (value : String) : ArtifactChecksum := drivePlanChecksumOf value
private def fingerprint (value : String) : BehaviorFingerprint := behaviorFingerprintOf value
private def runIdentity : DefinitionId := id "temporal.run.caller-closure.fixture"

private def binding (formatVersion value : String) : ArtifactBinding := {
  formatVersion
  artifactChecksum := checksum (value ++ ".artifact")
  behaviorFingerprint := fingerprint (value ++ ".behavior")
  provenanceChecksum := checksum (value ++ ".provenance")
}

private def source (definitionId : String) (count : Nat) : Lean.Json := object [
  ("sourceDefinitionId", text definitionId), ("status", text "closed"),
  ("factCount", natural count), ("byteCount", natural 0)
]

private def field (definitionId disposition : String) (fieldValue : Lean.Json) : Lean.Json :=
  object [("fieldDefinitionId", text definitionId), ("disposition", text disposition),
    ("value", fieldValue)]

private def fact
    (definitionId sourceId kind : String) (ordinal : Nat) (parents : List String)
    (fields : List Lean.Json) : Lean.Json := object [
  ("factDefinitionId", text definitionId), ("sourceDefinitionId", text sourceId),
  ("ordinal", natural ordinal), ("kindDefinitionId", text kind),
  ("causalFactDefinitionIds", array (parents.map text)), ("fields", array fields)
]

private def historyId (ordinal : Nat) : String :=
  "umpire.runtime.fact.history." ++ toString (ordinal + 1)

private def historyFact (ordinal : Nat) (eventType : String) : Lean.Json := fact
  (historyId ordinal) "umpire.evidence.source.history"
  "umpire.evidence.kind.workflow-history-event" ordinal
  (if ordinal == 0 then [] else [historyId (ordinal - 1)]) [
    field "umpire.evidence.field.event-id" "plain" (natural (ordinal + 1)),
    field "umpire.evidence.field.event-type" "plain" (text eventType),
    field "umpire.evidence.field.operation-correlation-id" "plain"
      (text "temporal.operation.caller-closure.fixture"),
    field "umpire.evidence.field.run-correlation-id" "plain" (text runIdentity.value),
    field "umpire.evidence.field.workflow-correlation-id" "plain"
      (text "temporal.workflow.caller-closure.fixture")
  ]

private def sources : Lean.Json := array [
  source "umpire.evidence.source.cleanup" 1,
  source "umpire.evidence.source.control-receipt" 1,
  source "umpire.evidence.source.history" 6,
  source "umpire.evidence.source.participant-output" 1
]

private def facts : Lean.Json := array <| [
  fact "umpire.runtime.fact.cleanup.fixture" "umpire.evidence.source.cleanup"
    "umpire.evidence.kind.cleanup" 0 [] [
      field "umpire.evidence.field.open-handle-count" "plain" (natural 0),
      field "umpire.evidence.field.status" "plain" (text "complete")
    ],
  fact "umpire.runtime.fact.control.fixture" "umpire.evidence.source.control-receipt"
    "umpire.evidence.kind.control-receipt" 0 [] [
      field "umpire.evidence.field.action-definition-id" "plain"
        (text "workflow.action.force-close"),
      field "umpire.evidence.field.attempt" "plain" (natural 1),
      field "umpire.evidence.field.occurrence-definition-id" "plain"
        (text "workflow-nexus.occurrence.force-close"),
      field "umpire.evidence.field.status" "plain" (text "accepted")
    ]
] ++ (List.zip (List.range 6) [
  "temporal.history.WorkflowExecutionStarted", "temporal.history.NexusOperationScheduled",
  "temporal.history.NexusOperationStarted", "temporal.history.NexusOperationCancelRequested",
  "temporal.history.NexusOperationCancelRequestCompleted",
  "temporal.history.WorkflowExecutionCanceled"
]).map (fun pair => historyFact pair.1 pair.2) ++ [
  fact "umpire.runtime.fact.participant.fixture" "umpire.evidence.source.participant-output"
    "umpire.evidence.kind.participant-command" 0 [] [
      field "umpire.evidence.field.cancellation-callback-count" "plain" (natural 1),
      field "umpire.evidence.field.endpoint-identity" "sha256"
        (text (fingerprint "endpoint").render)
    ]
]

private def phaseOutcomes : Lean.Json := array <|
  ["preparation", "realization", "observation", "isolation", "cleanup"].map fun phase => object [
    ("phase", text phase), ("status", text "succeeded"),
    ("startedAtUnixMillis", natural 1), ("finishedAtUnixMillis", natural 2), ("code", .null)
  ]

private def failedPhaseOutcomes : Lean.Json := array <|
  ["preparation", "realization", "observation", "isolation", "cleanup"].map fun phase =>
    if phase == "isolation" then object [
      ("phase", text phase), ("status", text "failed"),
      ("startedAtUnixMillis", natural 1), ("finishedAtUnixMillis", natural 2),
      ("code", text "umpire.runtime.failed")
    ] else object [
      ("phase", text phase), ("status", text "succeeded"),
      ("startedAtUnixMillis", natural 1), ("finishedAtUnixMillis", natural 2), ("code", .null)
    ]

private def controlAttempts : Lean.Json := array [object [
  ("occurrenceDefinitionId", text "workflow-nexus.occurrence.force-close"),
  ("actionDefinitionId", text "workflow.action.force-close"),
  ("attempt", natural 1),
  ("receiptFactDefinitionId", text "umpire.runtime.fact.control.fixture"),
  ("status", text "accepted"), ("code", .null)
]]

private def sourceClosures : Lean.Json := array [
  object [("sourceDefinitionId", text "umpire.evidence.source.cleanup"),
    ("status", text "closed"), ("recordCount", natural 1), ("byteCount", natural 0)],
  object [("sourceDefinitionId", text "umpire.evidence.source.control-receipt"),
    ("status", text "closed"), ("recordCount", natural 1), ("byteCount", natural 0)],
  object [("sourceDefinitionId", text "umpire.evidence.source.history"),
    ("status", text "closed"), ("recordCount", natural 6), ("byteCount", natural 0)],
  object [("sourceDefinitionId", text "umpire.evidence.source.participant-output"),
    ("status", text "closed"), ("recordCount", natural 1), ("byteCount", natural 0)]
]

private def request : Request := {
  formatVersion := requestFormatVersion
  checkerIdentity
  checkerVersion
  checkerBehaviorFingerprint
  experiment := expectedExperimentBinding
  runtimeConfiguration := expectedRuntimeConfigurationBinding
  run := binding "umpire-experiment-run/v2" "run"
  rawEvidence := binding "umpire-raw-evidence/v2" "raw"
  runIdentity
  query := expectedQuery
  properties := expectedProperties
  observationProgram := expectedObservationProgram
  mapping := expectedMapping
  phaseOutcomes
  controlAttempts
  sourceClosures
  captureStatus := "closed"
  sources
  facts
  runKnownGaps := array []
  rawEvidenceKnownGaps := array []
}

private def evaluationResult := Temporal.Tool.RunEvaluation.evaluateSemantics request
private theorem evaluationResult_isSome : evaluationResult.toOption.isSome = true := by native_decide
private def evaluation := evaluationResult.toOption.get evaluationResult_isSome
private def responseResult := Temporal.Tool.RunEvaluation.evaluateRequest request
private theorem responseResult_isSome : responseResult.toOption.isSome = true := by native_decide
private def response := responseResult.toOption.get responseResult_isSome
private def responseOf (candidate : Request) : Response :=
  (Temporal.Tool.RunEvaluation.evaluateRequest candidate).toOption.getD response
private def requestBytes := (encodeRequest request).toOption.getD .empty
private def checkerResult := Temporal.Tool.RunEvaluation.runBytes requestBytes

/-! Exact source evidence reaches the unchanged Feature Property evaluator only after both checked
System altitudes have succeeded. -/
example : evaluation.observation.status = .accepted := by native_decide
example : evaluation.implementationLink.map ImplementationLinkResult.status = some .applied := by
  native_decide
example : evaluation.querySummary.status = .satisfied := by native_decide
example : response.observationEvaluationStatus = "accepted" &&
    response.semanticStatus = "satisfied" && response.evaluationOutcomeChecksum.isSome := by
  native_decide
example : checkerResult.status = 0 && checkerResult.stderr.isEmpty &&
    checkerResult.stdout == (encodeResponse response).toOption.getD .empty := by native_decide

private def acceptedTrace : Option EvidenceBackedTrace := match evaluation.observation with
  | .accepted trace => some trace
  | _ => none

/-! The adapter translates every source fact once; checked Observation alone decides which records
establish the one-step System trace. -/
example : acceptedTrace.any fun trace =>
    trace.evidenceIdentities.length == 9 && trace.recordSupport.length == 9 &&
      trace.trace.steps.length == 1 && trace.evidenceLinks.all fun link =>
        link.orderingSupport.length == 9 && link.closureSupport.length == 4 := by
  native_decide

private def failedOperationalResponse := responseOf
  { request with phaseOutcomes := failedPhaseOutcomes }

example : failedOperationalResponse.observationEvaluationStatus = "accepted" &&
    failedOperationalResponse.semanticStatus = "satisfied" &&
    failedOperationalResponse.evaluationOutcomeChecksum == response.evaluationOutcomeChecksum := by
  native_decide

/-! The reusable conflict checker still rejects required-capability conflicts while this exact
link admits Feature-internal provider overlap outside the required capability closure. -/
example : Temporal.System.Nexus.ImplementationLink.CallerClosure.checkedResult.isOk := by
  native_decide

/-! Request bytes are strict, deterministic two-space pretty JSON with one terminal LF. -/
private def requestRoundTrip : Bool :=
  match encodeRequest request with
  | .error _ => false
  | .ok bytes => match decodeRequest bytes with
    | .error _ => false
    | .ok decoded => decoded == request

example : requestRoundTrip := by native_decide
example : (encodeRequest request).toOption.bind String.fromUTF8? |>.any fun value =>
    value.contains "\n  \"" && value.endsWith "\n" && !(value.endsWith "\n\n") := by native_decide

private def driftResult := (encodeRequest { request with checkerIdentity := "drifted" }).bind
  decodeRequest

private def driftRejected : Bool := match driftResult with
  | .error failure => failure.kind == .closureDrift
  | .ok _ => false

example : driftRejected := by native_decide

private def exactProtocolLimit := ByteArray.mk (Array.replicate maxBytes (UInt8.ofNat 32))
private def overProtocolLimit := exactProtocolLimit.push (UInt8.ofNat 32)

private def exactProtocolLimitHandled : Bool := match decodeRequest exactProtocolLimit with
  | .error failure => failure.kind != .oversized
  | .ok _ => true

private def overProtocolLimitRejected : Bool := match decodeRequest overProtocolLimit with
  | .error failure => failure.kind == .oversized
  | .ok _ => false

example : exactProtocolLimitHandled := by native_decide
example : overProtocolLimitRejected := by native_decide

private def replaceFact (position : Nat) (replacement : Lean.Json) : Lean.Json :=
  match facts.getArr? with
  | .error _ => facts
  | .ok values => .arr (values.set! position replacement)

private def appendValue (values value : Lean.Json) : Lean.Json :=
  match values.getArr? with
  | .error _ => values
  | .ok items => .arr (items.push value)

private def cleanupFact (extraFields : List Lean.Json := []) : Lean.Json := fact
  "umpire.runtime.fact.cleanup.fixture" "umpire.evidence.source.cleanup"
  "umpire.evidence.kind.cleanup" 0 [] <| [
    field "umpire.evidence.field.open-handle-count" "plain" (natural 0),
    field "umpire.evidence.field.status" "plain" (text "complete")
  ] ++ extraFields

private def crossedTypeCleanupFact : Lean.Json := fact
  "umpire.runtime.fact.cleanup.fixture" "umpire.evidence.source.cleanup"
  "umpire.evidence.kind.cleanup" 0 [] [
    field "umpire.evidence.field.open-handle-count" "plain" (text "0"),
    field "umpire.evidence.field.status" "plain" (text "complete")
  ]

private def malformedDigestParticipantFact : Lean.Json := fact
  "umpire.runtime.fact.participant.fixture" "umpire.evidence.source.participant-output"
  "umpire.evidence.kind.participant-command" 0 [] [
    field "umpire.evidence.field.cancellation-callback-count" "plain" (natural 1),
    field "umpire.evidence.field.endpoint-identity" "sha256" (text "sha256:malformed")
  ]

private def extraSourceRequest : Request := { request with
  sources := appendValue sources (source "umpire.evidence.source.extra" 0)
}

private def extraFactRequest : Request := { request with
  facts := appendValue facts <| fact "umpire.runtime.fact.cleanup.extra"
    "umpire.evidence.source.cleanup" "umpire.evidence.kind.cleanup" 1 [] [
      field "umpire.evidence.field.open-handle-count" "plain" (natural 0),
      field "umpire.evidence.field.status" "plain" (text "complete")
    ]
}

private def extraFieldRequest : Request := { request with
  facts := replaceFact 0 <| cleanupFact [
    field "umpire.evidence.field.extra" "plain" (text "extra")
  ]
}

private def crossedTypeRequest : Request := { request with
  facts := replaceFact 0 crossedTypeCleanupFact
}

private def malformedDigestRequest : Request := { request with
  facts := replaceFact 8 malformedDigestParticipantFact
}

private def extraSourceFactRequest : Request := { request with
  facts := appendValue facts <| fact "umpire.runtime.fact.extra"
    "umpire.evidence.source.extra" "umpire.evidence.kind.cleanup" 0 [] [
      field "umpire.evidence.field.open-handle-count" "plain" (natural 0),
      field "umpire.evidence.field.status" "plain" (text "complete")
    ]
}

private def invalidDispositionRequest : Request := { request with
  facts := replaceFact 0 <| fact "umpire.runtime.fact.cleanup.fixture"
    "umpire.evidence.source.cleanup" "umpire.evidence.kind.cleanup" 0 [] [
      field "umpire.evidence.field.open-handle-count" "masked" (natural 0),
      field "umpire.evidence.field.status" "plain" (text "complete")
    ]
}

private def sourceProjection
    (definitionId status : String)
    (count : Nat) : Lean.Json := object [
  ("sourceDefinitionId", text definitionId), ("status", text status),
  ("factCount", natural count), ("byteCount", natural 0)
]

private def closureProjection
    (definitionId status : String)
    (count : Nat) : Lean.Json := object [
  ("sourceDefinitionId", text definitionId), ("status", text status),
  ("recordCount", natural count), ("byteCount", natural 0)
]

private def notAttemptedControlAttempts : Lean.Json := array [object [
  ("occurrenceDefinitionId", text "workflow-nexus.occurrence.force-close"),
  ("actionDefinitionId", text "workflow.action.force-close"),
  ("attempt", natural 1), ("receiptFactDefinitionId", .null),
  ("status", text "not-attempted"), ("code", .null)
]]

private def notAttemptedRequest : Request :=
  let withoutControl := match facts.getArr? with
    | .error _ => facts
    | .ok values => .arr (values.eraseIdx! 1)
  let partialSources := match sources.getArr? with
    | .error _ => sources
    | .ok values => .arr (values.set! 1 <|
        sourceProjection "umpire.evidence.source.control-receipt" "partial" 0)
  let partialClosures := match sourceClosures.getArr? with
    | .error _ => sourceClosures
    | .ok values => .arr (values.set! 1 <|
        closureProjection "umpire.evidence.source.control-receipt" "partial" 0)
  { request with
    controlAttempts := notAttemptedControlAttempts
    sourceClosures := partialClosures
    captureStatus := "partial"
    sources := partialSources
    facts := withoutControl }

private def rejectedControlFact : Lean.Json := fact
  "umpire.runtime.fact.control.fixture" "umpire.evidence.source.control-receipt"
  "umpire.evidence.kind.control-receipt" 0 [] [
    field "umpire.evidence.field.action-definition-id" "plain"
      (text "workflow.action.force-close"),
    field "umpire.evidence.field.attempt" "plain" (natural 1),
    field "umpire.evidence.field.occurrence-definition-id" "plain"
      (text "workflow-nexus.occurrence.force-close"),
    field "umpire.evidence.field.status" "plain" (text "rejected")
  ]

private def participantFact : Lean.Json := fact
  "umpire.runtime.fact.participant.fixture" "umpire.evidence.source.participant-output"
  "umpire.evidence.kind.participant-command" 0 [] [
    field "umpire.evidence.field.cancellation-callback-count" "plain" (natural 1),
    field "umpire.evidence.field.endpoint-identity" "sha256"
      (text (fingerprint "endpoint").render)
  ]

private def rejectedControlAttempts : Lean.Json := array [object [
  ("occurrenceDefinitionId", text "workflow-nexus.occurrence.force-close"),
  ("actionDefinitionId", text "workflow.action.force-close"),
  ("attempt", natural 1),
  ("receiptFactDefinitionId", text "umpire.runtime.fact.control.fixture"),
  ("status", text "rejected"), ("code", text "umpire.runtime.control.rejected")
]]

private def closedFailedRequest : Request := { request with
  phaseOutcomes := failedPhaseOutcomes
  controlAttempts := rejectedControlAttempts
  sourceClosures := array [
    closureProjection "umpire.evidence.source.cleanup" "closed" 1,
    closureProjection "umpire.evidence.source.control-receipt" "closed" 1,
    closureProjection "umpire.evidence.source.history" "closed" 1,
    closureProjection "umpire.evidence.source.participant-output" "closed" 1
  ]
  sources := array [
    sourceProjection "umpire.evidence.source.cleanup" "closed" 1,
    sourceProjection "umpire.evidence.source.control-receipt" "closed" 1,
    sourceProjection "umpire.evidence.source.history" "closed" 1,
    sourceProjection "umpire.evidence.source.participant-output" "closed" 1
  ]
  facts := array [cleanupFact, rejectedControlFact,
    historyFact 0 "temporal.history.WorkflowExecutionStarted", participantFact]
}

private def knownGapJson : Lean.Json := array [object [
  ("kind", text "input"), ("code", text "umpire.gap.fixture"),
  ("subject", .null), ("detail", .null)
]]

private def gapRequest : Request := { request with runKnownGaps := knownGapJson }

private def rejectedField (candidate : Request) : Option String :=
  match Temporal.Tool.RunEvaluation.evaluateRequest candidate with
  | .error failure => some failure.field
  | .ok _ => none

private def extraFieldResponse := responseOf extraFieldRequest
private def crossedTypeResponse := responseOf crossedTypeRequest
private def gapResponse := responseOf gapRequest
private def notAttemptedResponse? :=
  (Temporal.Tool.RunEvaluation.evaluateRequest notAttemptedRequest).toOption
private def closedFailedResponse? :=
  (Temporal.Tool.RunEvaluation.evaluateRequest closedFailedRequest).toOption

private def conflictingRequest : Request := { closedFailedRequest with
  sourceClosures := array [
    closureProjection "umpire.evidence.source.cleanup" "closed" 1,
    closureProjection "umpire.evidence.source.control-receipt" "closed" 1,
    closureProjection "umpire.evidence.source.history" "closed" 2,
    closureProjection "umpire.evidence.source.participant-output" "closed" 1
  ]
  sources := array [
    sourceProjection "umpire.evidence.source.cleanup" "closed" 1,
    sourceProjection "umpire.evidence.source.control-receipt" "closed" 1,
    sourceProjection "umpire.evidence.source.history" "closed" 2,
    sourceProjection "umpire.evidence.source.participant-output" "closed" 1
  ]
  facts := array [cleanupFact, rejectedControlFact,
    historyFact 0 "temporal.history.WorkflowExecutionStarted",
    historyFact 0 "temporal.history.WorkflowExecutionStarted", participantFact]
}

private def conflictResponse := responseOf conflictingRequest

private def jsonArrayEmpty (value : Lean.Json) : Bool :=
  match value.getArr? with
  | .ok values => values.isEmpty
  | .error _ => false

private def incompleteProjection (candidate : Response) : Bool :=
  jsonArrayEmpty candidate.propertyVerdicts && candidate.semanticStatus == "incomplete" &&
    candidate.evaluationOutcomeChecksum.isNone &&
    (candidate.querySummary.getObjVal? "status").toOption.bind
      (fun value => value.getStr?.toOption) == some "incomplete" &&
    (candidate.querySummary.getObjVal? "propertyVerdicts").toOption.any jsonArrayEmpty

/-! The closed protocol rejects an extra source and malformed digest before semantics. -/
example : rejectedField extraSourceRequest = some "sources.set" &&
    rejectedField extraSourceFactRequest = some "facts.sourceDefinitionId" &&
    rejectedField extraFactRequest = some "sources.factCount" &&
    rejectedField invalidDispositionRequest = some "facts.fields.disposition" &&
    rejectedField malformedDigestRequest = some "facts.fields.value" := by native_decide

/-! Total fact/field/type closure reaches checked Observation without adapter-side semantics. -/
example : extraFieldResponse.observationEvaluationStatus = "unsupported" &&
    crossedTypeResponse.observationEvaluationStatus = "unknown" := by native_decide

/-! Every non-accepted Observation status projects a fn18-valid empty incomplete Result row. -/
example : incompleteProjection crossedTypeResponse && incompleteProjection conflictResponse &&
    incompleteProjection extraFieldResponse && conflictResponse.observationEvaluationStatus = "conflict" := by
  native_decide

/-! Upstream Known Gaps force unknown semantics without a Property verdict or outcome checksum. -/
example : gapResponse.observationEvaluationStatus = "unknown" &&
    incompleteProjection gapResponse && !jsonArrayEmpty gapResponse.resultKnownGaps := by native_decide

/-! Valid fn19 non-success control/source closures reach fn-4 instead of protocol rejection. -/
example : notAttemptedResponse?.any fun candidate =>
    candidate.observationEvaluationStatus == "unknown" && incompleteProjection candidate := by
  native_decide

example : closedFailedResponse?.any fun candidate =>
    candidate.observationEvaluationStatus == "unknown" && incompleteProjection candidate := by
  native_decide

end Temporal.Tool.RunEvaluation.Tests
