import Umpire.Case
import Umpire.Json

namespace Umpire.CaseTests

open Umpire.Case

private def stringType : ValueType := .singular (.scalar .text)

private def historyType : ValueType :=
  .repeated (.message "example.api.history.v1.HistoryEvent")

private def naturalType : ValueType := .singular (.scalar .natural)

private def sourceShapedProgram : Program := {
  programId := "example.async-success.program"
  roles := []
  slots := [
    ({ slotId := "workflow-id", type := stringType } : SlotSchema),
    ({ slotId := "history-events", type := historyType } : SlotSchema),
    ({ slotId := "completion-authority",
       type := ValueType.singular SingularType.opaqueCapability,
       kind := SlotKind.opaqueCapability } : SlotSchema)
  ]
  observations := [
    { observationId := "history-event-type", type := .singular (.enumeration "example.api.enums.v1.EventType") },
    { observationId := "scheduled-event-id", type := naturalType }
  ]
  entrypoints := []
  cleanup := { entrypointId := "cleanup", context := .controller, nodes := [] }
  limits := {
    maxEntrypoints := 4
    maxNodes := 32
    maxEdges := 64
    maxActivations := 8
    maxAttempts := 32
    maxRunEvents := 256
    maxExpressionDepth := 8
    maxPathFanout := 128
    maxRequestBytes := 1048576
    maxResponseBytes := 1048576
    maxTotalDurationMilliseconds := 30000
    maxCleanupDurationMilliseconds := 5000
  }
}

private def sourceShapedContract : Contract := {
  contractId := "example.async-success.contract"
  rules := [{
    ruleId := "workflow-completes"
    kind := .boundedLiveness
    initialState := "pending"
    captures := [{ captureId := "scheduled-event-id", type := .scalar .natural }]
    states := [
      ({ stateId := "pending", terminal := .nonterminal } : ContractState),
      ({ stateId := "scheduled", terminal := .nonterminal } : ContractState),
      ({ stateId := "satisfied", terminal := .satisfied } : ContractState),
      ({ stateId := "violated", terminal := .violated } : ContractState)
    ]
    transitions := [
      ({
        transitionId := "capture-scheduled-event"
        sourceState := "pending"
        targetState := "scheduled"
        eventKinds := []
        predicate := .present (.observation { observationId := "scheduled-event-id" })
        captureAssignments := [{
          captureId := "scheduled-event-id"
          observation := { observationId := "scheduled-event-id" }
        }]
      } : ContractTransition),
      ({
        transitionId := "observe-completion"
        sourceState := "scheduled"
        targetState := "satisfied"
        eventKinds := []
        predicate := .equals
          (.capture { captureId := "scheduled-event-id" })
          (.observation { observationId := "scheduled-event-id" })
        support := .matchingEvent
      } : ContractTransition)
    ]
    horizon := some { elapsedMilliseconds := 30000, violationStateId := "violated" }
  }]
  limits := {
    maxRules := 8
    maxStates := 32
    maxTransitions := 64
    maxExpressionDepth := 8
    maxWorkPerEvent := 64
    maxTotalWork := 1024
    maxCaptures := 8
    maxCaptureBytes := 4096
  }
}

private def sourceShapedCase : Case := {
  version := { major := 1, minor := 0 }
  caseId := "example.async-success"
  metadata := {
    producerId := "lean.example"
    definitions := [
      ({ definitionId := "example.target", behaviorFingerprint := "target/v1",
         kind := .target } : CaseDefinitionBinding),
      ({ definitionId := "example.provider", behaviorFingerprint := "provider/v1",
         kind := .provider } : CaseDefinitionBinding),
      ({ definitionId := "example.law", behaviorFingerprint := "law/v1",
         kind := .law } : CaseDefinitionBinding),
      ({ definitionId := "example.connector", behaviorFingerprint := "connector/v1",
         kind := .connector } : CaseDefinitionBinding),
      ({ definitionId := "example.kernel", behaviorFingerprint := "kernel/v1",
         kind := .kernel } : CaseDefinitionBinding)
    ]
    sources := [{
      path := "Example/Feature/Operations.lean"
      line := 42
      column := 7
      provenance := "checked-model"
    }]
    knownGaps := [
      ({ kind := .interpretation, code := "example.gap" } : CaseKnownGap),
      ({ kind := .claim, code := "example.gap",
         subject := some "example.target", detail := some "claim remains local" } : CaseKnownGap)
    ]
  }
  program := sourceShapedProgram
  contract := sourceShapedContract
}

#guard sourceShapedCase.version.major == 1
#guard sourceShapedCase.program.roles.isEmpty
#guard sourceShapedCase.program.slots.length == 3
#guard sourceShapedCase.contract.rules.length == 1
#guard sourceShapedCase.contract.rules.map (fun rule => rule.captures.length) == [1]
#guard sourceShapedCase.metadata.definitions.map (fun definition => definition.kind) ==
  [.target, .provider, .law, .connector, .kernel]
#guard sourceShapedCase.metadata.knownGaps.map (fun gap => (gap.kind, gap.subject.isNone)) ==
  [(.interpretation, true), (.claim, false)]
#guard sourceShapedCase.program.slots.map (fun slot => slot.type) ==
  [stringType, historyType, ValueType.singular SingularType.opaqueCapability]

private def reservation : ActivationReservation := { entrypointId := "workflow", count := 3 }

private def encodeReservation (value : ActivationReservation) : String :=
  Umpire.CanonicalJson.compact (.object [
    ("entrypointId", .string value.entrypointId), ("count", .string (toString value.count))])

private def decodeReservation (text : String) : Except String ActivationReservation := do
  let json ← Lean.Json.parse text
  let entrypointId ← json.getObjValAs? String "entrypointId"
  let countText ← json.getObjValAs? String "count"
  let some count := countText.toNat? | throw "invalid reservation count"
  return { entrypointId, count }

#guard encodeReservation reservation == "{\"entrypointId\":\"workflow\",\"count\":\"3\"}"
#guard match decodeReservation (encodeReservation reservation) with
  | .ok actual => actual == reservation
  | .error _ => false
#guard match decodeReservation "{\"entrypointId\":\"workflow\",\"count\":\"-1\"}" with
  | .ok _ => false
  | .error _ => true

end Umpire.CaseTests
