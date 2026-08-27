import Umpire.ImplementationLink.Tests.Fixtures

/-! Canonical checked identity and exhaustive declaration/witness rejection fixtures. -/

namespace Umpire.ImplementationLinkTests

open Umpire
open Umpire.TargetTests

/-- Complete declarations with exact witnesses check to one canonical identity. -/
example : checkedIdentityOf baseDeclaration baseWitness =
    checkedIdentityOf reorderedDeclaration reorderedWitness := by
  native_decide

/-- Proof implementation and documentation are excluded from behavior identity bytes. -/
example : checkedIdentityOf baseDeclaration baseWitness =
    checkedIdentityOf baseDeclaration alternateProofWitness := by
  native_decide

def staleSourceDeclaration := {
  baseDeclaration with sourceTarget := {
    baseDeclaration.sourceTarget with id := id "test.target.stale-source"
  }
}

def staleDestinationDeclaration := {
  baseDeclaration with destinationTarget := {
    baseDeclaration.destinationTarget with id := id "test.target.stale-destination"
  }
}

def wrongTargetKindDeclaration := {
  baseDeclaration with sourceTarget := {
    baseDeclaration.sourceTarget with kind := .state
  }
}

def targetFingerprintDriftDeclaration := {
  baseDeclaration with sourceTarget := {
    baseDeclaration.sourceTarget with behaviorFingerprint := behaviorFingerprintOf "mutated-target"
  }
}

def duplicateMappingDeclaration := {
  baseDeclaration with setupMappings := baseDeclaration.setupMappings ++ baseDeclaration.setupMappings
}

def ambiguousStateMapping : ImplementationValueMapping Bool Bool := {
  source := false
  destination := true
}

def ambiguousMappingDeclaration := {
  baseDeclaration with stateMappings := baseDeclaration.stateMappings ++ [ambiguousStateMapping]
}

def wrongSemanticKindDeclaration := {
  baseDeclaration with relationMappings := [{
    relationMapping with source := { relationMapping.source with kind := .state }
  }]
}

def semanticFingerprintDriftDeclaration := {
  baseDeclaration with relationMappings := [{
    relationMapping with source := {
      relationMapping.source with behaviorFingerprint := behaviorFingerprintOf "mutated-relation"
    }
  }]
}

def incompleteSupportDeclaration := {
  baseDeclaration with stateMappings := baseDeclaration.stateMappings.filter fun mapping => mapping.source
}

def contradictorySupportDeclaration := {
  baseDeclaration with stateKnownGaps := [{
    source := false
    code := id "test.known-gap.false-state"
    reason := "Deliberately overlaps supported state."
  }]
}

def supportedTrueStateMappings : List (ImplementationValueMapping Bool Bool) :=
  baseDeclaration.stateMappings.filter fun mapping => mapping.source

def invalidKnownGapDeclaration := {
  baseDeclaration with
  stateMappings := supportedTrueStateMappings
  stateKnownGaps := [{
    source := false
    code := id "gap"
    reason := ""
  }]
}

def zeroLimitDeclaration := {
  baseDeclaration with applicationLimit := { value := 0, unit := .semanticTransitions }
}

def wrongLimitUnitDeclaration := {
  baseDeclaration with applicationLimit := { value := 10, unit := .selectedActions }
}

def declarationFailures : List (Option ImplementationLinkErrorKind) := [
  errorKindOf (checkImplementationLink staleSourceDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness staleSourceDeclaration)),
  errorKindOf (checkImplementationLink staleDestinationDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness staleDestinationDeclaration)),
  errorKindOf (checkImplementationLink wrongTargetKindDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness wrongTargetKindDeclaration)),
  errorKindOf (checkImplementationLink targetFingerprintDriftDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness targetFingerprintDriftDeclaration)),
  errorKindOf (checkImplementationLink duplicateMappingDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness duplicateMappingDeclaration)),
  errorKindOf (checkImplementationLink ambiguousMappingDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness ambiguousMappingDeclaration)),
  errorKindOf (checkImplementationLink wrongSemanticKindDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness wrongSemanticKindDeclaration)),
  errorKindOf (checkImplementationLink semanticFingerprintDriftDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness semanticFingerprintDriftDeclaration)),
  errorKindOf (checkImplementationLink incompleteSupportDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness incompleteSupportDeclaration)),
  errorKindOf (checkImplementationLink contradictorySupportDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness contradictorySupportDeclaration)),
  errorKindOf (checkImplementationLink invalidKnownGapDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness invalidKnownGapDeclaration)),
  errorKindOf (checkImplementationLink zeroLimitDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness zeroLimitDeclaration)),
  errorKindOf (checkImplementationLink wrongLimitUnitDeclaration checkedSourceTarget
    checkedDestinationTarget (incompleteWitness wrongLimitUnitDeclaration))
]

/-- Stale, ambiguous, wrong-kind, incomplete, and invalid-Limit declarations return no checked value. -/
example : declarationFailures = [
  some .staleSourceTarget,
  some .staleDestinationTarget,
  some .wrongKind,
  some .behaviorFingerprintDrift,
  some .duplicateMapping,
  some .ambiguousMapping,
  some .wrongKind,
  some .behaviorFingerprintDrift,
  some .incompleteSupportPartition,
  some .contradictorySupportPartition,
  some .invalidKnownGap,
  some .invalidLimitValue,
  some .invalidLimitUnit
] := by
  native_decide

def missingObligationFailures : List (Option ImplementationLinkErrorKind) := [
  errorKindOf (checkImplementationLink baseDeclaration checkedSourceTarget checkedDestinationTarget
    (incompleteWitness baseDeclaration [.initialForward])),
  errorKindOf (checkImplementationLink baseDeclaration checkedSourceTarget checkedDestinationTarget
    (incompleteWitness baseDeclaration [.stepForward])),
  errorKindOf (checkImplementationLink baseDeclaration checkedSourceTarget checkedDestinationTarget
    (incompleteWitness baseDeclaration [.requiredCoverage]))
]

/-- Every missing forward or coverage obligation has one exact typed failure. -/
example : missingObligationFailures = [
  some .missingInitialForward,
  some .missingStepForward,
  some .missingRequiredCoverage
] := by
  native_decide

def outcomeInventionDeclaration :
    ImplementationLinkDeclaration Unit Bool Bool Bool Bool Unit Bool Bool SparseOutcome Bool := {
  id := id "test.implementation-link.outcome-invention"
  source := source "Test/ImplementationLink.lean"
  sourceTarget := .ofTarget checkedSourceTarget
  destinationTarget := .ofTarget checkedSparseOutcomeTarget
  setupMappings := [{ source := (), destination := () }]
  stateMappings := baseDeclaration.stateMappings
  actionMappings := baseDeclaration.actionMappings
  outcomeMappings := [
    { source := false, destination := .off },
    { source := true, destination := .invented }
  ]
  observationMappings := baseDeclaration.observationMappings
  relationMappings := baseDeclaration.relationMappings
  capabilityMappings := baseDeclaration.capabilityMappings
  applicationLimit := baseDeclaration.applicationLimit
}

/-- A destination outcome absent from the checked destination domain is never invented by checking. -/
example : errorKindOf (checkImplementationLink outcomeInventionDeclaration checkedSourceTarget
    checkedSparseOutcomeTarget (.incomplete
      (implementationLinkWitnessIndex outcomeInventionDeclaration checkedSourceTarget
        checkedSparseOutcomeTarget) [])) = some .unknownDestinationValue := by
  native_decide

def mismatchedDeclarationIndex : ImplementationLinkWitnessIndex := {
  implementationLinkWitnessIndex baseDeclaration checkedSourceTarget checkedDestinationTarget with
  declarationVersion := 2
}

def mismatchedSourceIndex : ImplementationLinkWitnessIndex := {
  implementationLinkWitnessIndex baseDeclaration checkedSourceTarget checkedDestinationTarget with
  sourceTarget := {
    baseDeclaration.sourceTarget with behaviorFingerprint := behaviorFingerprintOf "stale-source"
  }
}

def mismatchedDestinationIndex : ImplementationLinkWitnessIndex := {
  implementationLinkWitnessIndex baseDeclaration checkedSourceTarget checkedDestinationTarget with
  destinationTarget := {
    baseDeclaration.destinationTarget with behaviorFingerprint := behaviorFingerprintOf "stale-destination"
  }
}

def witnessIndexFailures : List (Option ImplementationLinkErrorKind) := [
  errorKindOf (checkImplementationLink baseDeclaration checkedSourceTarget checkedDestinationTarget
    (.incomplete mismatchedDeclarationIndex [.requiredCoverage])),
  errorKindOf (checkImplementationLink baseDeclaration checkedSourceTarget checkedDestinationTarget
    (.incomplete mismatchedSourceIndex [.requiredCoverage])),
  errorKindOf (checkImplementationLink baseDeclaration checkedSourceTarget checkedDestinationTarget
    (.incomplete mismatchedDestinationIndex [.requiredCoverage]))
]

/-- Runtime witness labels reject stale declaration, source, and destination indices deterministically. -/
example : witnessIndexFailures = [
  some .witnessDeclarationMismatch,
  some .witnessSourceMismatch,
  some .witnessDestinationMismatch
] := by
  native_decide

def oneStepTrace : ModelTrace Bool Bool Bool Bool := {
  initialState := false
  steps := [{
    selectedAction := true
    modelOutcome := true
    resultingState := true
    observations := [false]
  }]
}

/-- Forward trace authority is derived from the exact initial and step witnesses. -/
example (trace : ModelTrace Bool Bool Bool Bool)
    (admitted : AuthoritativeModelTrace checkedSourceTarget.kernel () trace) :
    AuthoritativeModelTrace checkedDestinationTarget.kernel ()
      (baseWitness.translateTrace trace) :=
  baseWitness.traceForward () trace admitted

end Umpire.ImplementationLinkTests
