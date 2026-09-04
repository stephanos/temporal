import Umpire.Case

namespace Umpire.CaseTests

open Umpire.Case

private def stringType : ValueType := .singular (.scalar .text)

private def historyType : ValueType :=
  .repeated (.message "temporal.api.history.v1.HistoryEvent")

private def naturalType : ValueType := .singular (.scalar .natural)

private def sourceShapedProgram : Program := {
  programId := "nexus.async-success.program"
  roles := []
  slots := [
    ({ slotId := "workflow-id", type := stringType } : SlotSchema),
    ({ slotId := "history-events", type := historyType } : SlotSchema),
    ({ slotId := "completion-authority",
       type := ValueType.singular SingularType.opaqueCapability,
       kind := SlotKind.opaqueCapability } : SlotSchema)
  ]
  observations := [
    { observationId := "history-event-type", type := .singular (.enumeration "temporal.api.enums.v1.EventType") },
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
  contractId := "nexus.async-success.contract"
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
  caseId := "nexus.async-success"
  metadata := {
    producerId := "lean.temporal.nexus"
    definitions := [
      ({ definitionId := "temporal.nexus.target", behaviorFingerprint := "target/v1",
         kind := .target } : CaseDefinitionBinding),
      ({ definitionId := "temporal.nexus.provider", behaviorFingerprint := "provider/v1",
         kind := .provider } : CaseDefinitionBinding),
      ({ definitionId := "temporal.nexus.law", behaviorFingerprint := "law/v1",
         kind := .law } : CaseDefinitionBinding),
      ({ definitionId := "temporal.nexus.connector", behaviorFingerprint := "connector/v1",
         kind := .connector } : CaseDefinitionBinding),
      ({ definitionId := "temporal.nexus.kernel", behaviorFingerprint := "kernel/v1",
         kind := .kernel } : CaseDefinitionBinding)
    ]
    sources := [{
      path := "Temporal/Feature/Nexus/Operations.lean"
      line := 42
      column := 7
      provenance := "checked-model"
    }]
    knownGaps := [
      ({ kind := .interpretation, code := "temporal.nexus.gap" } : CaseKnownGap),
      ({ kind := .claim, code := "temporal.nexus.gap",
         subject := some "temporal.nexus.target", detail := some "claim remains local" } : CaseKnownGap)
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

end Umpire.CaseTests
