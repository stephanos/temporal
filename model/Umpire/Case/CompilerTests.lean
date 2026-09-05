import Umpire.Case.Compiler

namespace Umpire.Case.CompilerTests

open Umpire
open Umpire.Case
open Umpire.Case.Compiler

private def limits : ProgramLimits := {
  maxEntrypoints := 1
  maxNodes := 1
  maxEdges := 1
  maxActivations := 1
  maxAttempts := 1
  maxRunEvents := 16
  maxExpressionDepth := 4
  maxPathFanout := 1
  maxRequestBytes := 1024
  maxResponseBytes := 1024
  maxTotalDurationMilliseconds := 1000
  maxCleanupDurationMilliseconds := 100
}

private def contractLimits : ContractLimits := {
  maxRules := 1
  maxStates := 2
  maxTransitions := 1
  maxExpressionDepth := 4
  maxWorkPerEvent := 4
  maxTotalWork := 64
  maxCaptures := 1
  maxCaptureBytes := 64
}

private def property : CaseDefinitionBinding := {
  definitionId := "example.property"
  behaviorFingerprint := "example-property/v1"
  kind := .property
}

private def source : SourceLocation := {
  path := "Example/Case.lean"
  line := 11
  column := 3
  provenance := "checked-model"
}

private def rule : ContractRule := {
  ruleId := "example.rule"
  kind := .safety
  initialState := "satisfied"
  states := [{ stateId := "satisfied", terminal := .satisfied }]
  transitions := []
}

private def input : Input := {
  version := { major := 1 }
  caseId := "example.case"
  producerId := "umpire.case.compiler"
  definitions := [property]
  sources := [source]
  knownGaps := []
  program := {
    programId := "example.program"
    roles := []
    slots := []
    observations := []
    entrypoints := []
    cleanup := { entrypointId := "cleanup", context := .controller, nodes := [] }
    limits
  }
  contractId := "example.contract"
  properties := [.monitor property rule]
  contractLimits
}

#guard match compile input with
  | .ok output =>
      output.caseId == input.caseId &&
      output.metadata.definitions == input.definitions &&
      output.program == input.program &&
      output.contract.rules == [rule] &&
      output.contract.limits == input.contractLimits
  | .error _ => false

private def unsupported := ContractLowering.unsupported
  property source "property.temporal-unbounded"

#guard match compile { input with properties := [unsupported] } with
  | .error failure =>
      failure.sourceDefinitionId == property.definitionId &&
      failure.source == source &&
      failure.construct == "property.temporal-unbounded"
  | .ok _ => false

end Umpire.Case.CompilerTests
