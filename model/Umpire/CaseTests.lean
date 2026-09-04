import Umpire.Case

namespace Umpire.CaseTests

open Umpire.Case

private def stringType : ValueType := .singular (.scalar .text)

private def historyType : ValueType :=
  .repeated (.message "temporal.api.history.v1.HistoryEvent")

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
  observations := []
  entrypoints := []
  cleanup := { context := .controller, nodes := [] }
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
    states := [
      ({ stateId := "pending" } : ContractState),
      ({ stateId := "satisfied", terminal := .satisfied } : ContractState),
      ({ stateId := "violated", terminal := .violated } : ContractState)
    ]
    transitions := []
    horizon := some { elapsedMilliseconds := 30000, violationStateId := "violated" }
  }]
  limits := {
    maxRules := 8
    maxStates := 32
    maxTransitions := 64
    maxExpressionDepth := 8
    maxWorkPerEvent := 64
    maxTotalWork := 1024
  }
}

private def sourceShapedCase : Case := {
  version := { major := 1, minor := 0 }
  caseId := "nexus.async-success"
  metadata := {
    producerId := "lean.temporal.nexus"
    sources := [{
      path := "Temporal/Feature/Nexus/Operations.lean"
      line := 42
      column := 7
      provenance := "checked-model"
    }]
  }
  program := sourceShapedProgram
  contract := sourceShapedContract
}

#guard sourceShapedCase.version.major == 1
#guard sourceShapedCase.program.roles.isEmpty
#guard sourceShapedCase.program.slots.length == 3
#guard sourceShapedCase.contract.rules.length == 1
#guard sourceShapedCase.program.slots.map (fun slot => slot.type) ==
  [stringType, historyType, ValueType.singular SingularType.opaqueCapability]

end Umpire.CaseTests
