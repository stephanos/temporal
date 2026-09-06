import Umpire.Case.Compiler
import Umpire.Artifact.Types

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

private def planningGaps : KnownGapSet :=
  (KnownGapSet.checkCanonical [
    { kind := .capabilityContract, code := DefinitionId.of "example.gap.capability" },
    {
      kind := .input
      code := DefinitionId.of "example.gap.input"
      subject := some (DefinitionId.of "example.target")
    },
    {
      kind := .interpretation
      code := DefinitionId.of "example.gap.interpretation"
      detail := some "Interpretation remains model-owned."
    },
    {
      kind := .claim
      code := DefinitionId.of "example.gap.claim"
      subject := some (DefinitionId.of "example.property")
      detail := some "Claim requires runtime evidence."
    }
  ]).toOption.get (by native_decide)

private def inputWithPlanningGaps : Input := {
  input with knownGaps := planningGaps.toCaseKnownGaps
}

/-! The single checked conversion pass retains every row field in the compiled Case. -/
#guard match compile inputWithPlanningGaps with
  | .ok output => output.metadata.knownGaps == [
      { kind := .capabilityContract, code := "example.gap.capability" },
      { kind := .input, code := "example.gap.input", subject := some "example.target" },
      {
        kind := .interpretation
        code := "example.gap.interpretation"
        detail := some "Interpretation remains model-owned."
      },
      {
        kind := .claim
        code := "example.gap.claim"
        subject := some "example.property"
        detail := some "Claim requires runtime evidence."
      }
    ]
  | .error _ => false

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

private def unsupportedGuardedTemporal := ContractLowering.unsupported
  property source "property.guarded-eventually-within"

#guard match compile { input with properties := [unsupported] } with
  | .error failure =>
      failure.sourceDefinitionId == property.definitionId &&
      failure.source == source &&
      failure.construct == "property.temporal-unbounded"
  | .ok _ => false

/- The current lowering boundary preserves a guarded temporal rejection as checked source data. -/
#guard match compile { input with properties := [unsupportedGuardedTemporal] } with
  | .error failure =>
      failure.sourceDefinitionId == property.definitionId &&
      failure.source == source &&
      failure.construct == "property.guarded-eventually-within"
  | .ok _ => false

end Umpire.Case.CompilerTests
