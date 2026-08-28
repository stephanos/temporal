import Umpire.Space.Intent
import Umpire.Space.Tests.Fixtures

/-! Checked Artifact intent projection and fail-closed planner integration. -/

namespace Umpire.SpaceTests

open Umpire

private def occurrenceId : DefinitionId := id "switch.occurrence.flip"

private def selectedChoices : List ModelValue := [
  { definitionId := stateAxisId, value := stateOffId.value },
  { definitionId := faultAxisId, value := faultDelayId.value }
]

private def selectedVariant : RoleBinding := {
  role := Umpire.Examples.Switch.switchRoleId
  value := Umpire.Examples.Switch.offState
}

private def selectedFault : ArtifactFaultIntent := {
  definitionId := delayFaultId
  occurrenceDefinitionId := occurrenceId
  actionDefinitionId := Umpire.Examples.Switch.flipActionId
  capabilityDefinitionId := Umpire.Examples.Switch.switchCapabilityId
}

private def intentDeclaration : ArtifactIntentDeclaration := {
  selectedChoices
  selectedVariants := [selectedVariant]
  requestedFaults := [selectedFault]
  additionalCapabilityRequirementDefinitionIds := [
    Umpire.Examples.Switch.switchCapabilityId
  ]
}

private def checkedIntentResult : Except ArtifactIntentError ArtifactIntent :=
  checkArtifactIntent Umpire.Examples.Switch.exactActionQuery intentDeclaration

private theorem checkedIntentResult_isSome : checkedIntentResult.toOption.isSome = true := by
  native_decide

private def checkedIntent : ArtifactIntent :=
  checkedIntentResult.toOption.get checkedIntentResult_isSome

private def projectedRunResult : Except ArtifactIntentError PlannerRun :=
  planWithArtifactIntent Umpire.Examples.Switch.exactActionQuery
    Umpire.Examples.Switch.incrementalKernel checkedIntent

private def projectedSpec : Option ExperimentSpec :=
  projectedRunResult.toOption.bind PlannerRun.artifact

private def intentErrorKindOf
    (result : Except ArtifactIntentError α) : Option ArtifactIntentErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

/-! Distinct role selections preserve repeated semantic values in the projected variant array. -/
example :
    let repeatedValueIntent := {
      checkedIntent with selectedVariants := [
        selectedVariant,
        { role := id "switch.role.peer", value := Umpire.Examples.Switch.offState }
      ]
    }
    repeatedValueIntent.selectedVariantValues = [
      Umpire.Examples.Switch.offState,
      Umpire.Examples.Switch.offState
    ] := by
  native_decide

/-! Checked intent populates the reserved arrays and unions selected fault capabilities. -/
example : projectedSpec.map (fun spec =>
    (spec.plan.selectedChoices,
      spec.plan.selectedVariants,
      spec.plan.requestedFaults,
      spec.plan.capabilityRequirementDefinitionIds)) = some (
    [
      { definitionId := faultAxisId, value := faultDelayId.value },
      { definitionId := stateAxisId, value := stateOffId.value }
    ],
    [Umpire.Examples.Switch.offState],
    [{ definitionId := delayFaultId, value := occurrenceId.value }],
    [Umpire.Examples.Switch.switchCapabilityId]) := by
  native_decide

/-! Intent projection changes no target-owned trace semantics and recomputes both checksums. -/
example : projectedSpec.map (fun projected =>
    let ordinary := Umpire.Examples.Switch.compiledArtifact
    projected.plan.initialState = ordinary.plan.initialState &&
      projected.plan.requestedActions = ordinary.plan.requestedActions &&
      projected.plan.modelOutcomes = ordinary.plan.modelOutcomes &&
      projected.plan.resultingStates = ordinary.plan.resultingStates &&
      projected.plan.linearExtension = ordinary.plan.linearExtension &&
      projected.plan.checkpoints = ordinary.plan.checkpoints &&
      projected.plan.hasValidArtifactChecksum &&
      projected.hasValidArtifactChecksum &&
      projected.artifactChecksum != ordinary.artifactChecksum) = some true := by
  native_decide

private def changedOccurrencePositions (spec : ExperimentSpec) : List PlannedOccurrence :=
  spec.plan.linearExtension.map fun occurrence => { occurrence with position := 99 }

private def targetSemanticMutations (spec : ExperimentSpec) : List ExperimentSpec := [
  { spec with plan := { spec.plan with initialState := Umpire.Examples.Switch.onState } },
  { spec with plan := { spec.plan with requestedActions := [] } },
  { spec with plan := { spec.plan with modelOutcomes := [] } },
  { spec with plan := { spec.plan with resultingStates := [] } },
  { spec with plan := { spec.plan with linearExtension := changedOccurrencePositions spec } },
  { spec with plan := { spec.plan with checkpoints := [] } }
]

/-! Projection never legitimizes stale-checksum mutations of target-owned trace semantics. -/
example : (targetSemanticMutations Umpire.Examples.Switch.compiledArtifact).all fun mutated =>
    intentErrorKindOf (mutated.withArtifactIntent Umpire.Examples.Switch.exactActionQuery
      checkedIntent) == some .identityDrift := by
  native_decide

private def withValidChecksums (spec : ExperimentSpec) : ExperimentSpec :=
  let plan := { spec.plan with artifactChecksum := spec.plan.expectedArtifactChecksum }
  let spec := { spec with plan }
  { spec with artifactChecksum := spec.expectedArtifactChecksum }

private def duplicateDeclarations : List ArtifactIntentDeclaration := [
  { intentDeclaration with selectedChoices := selectedChoices ++ [{
      definitionId := stateAxisId
      value := stateOffId.value
    }] },
  { intentDeclaration with selectedVariants := [selectedVariant, selectedVariant] },
  { intentDeclaration with requestedFaults := [selectedFault, selectedFault] }
]

/-! Duplicate axis, role, and fault intent entries fail before planning. -/
example : duplicateDeclarations.map (fun declaration =>
    intentErrorKindOf (checkArtifactIntent Umpire.Examples.Switch.exactActionQuery declaration)) =
    [some .duplicateEntry, some .duplicateEntry, some .duplicateEntry] := by
  native_decide

/-! Missing and action-mismatched authored occurrences fail closed. -/
example : [
    { selectedFault with occurrenceDefinitionId := id "switch.occurrence.stale" },
    { selectedFault with actionDefinitionId := id "switch.action.stale" }
  ].map (fun fault => intentErrorKindOf <| checkArtifactIntent
    Umpire.Examples.Switch.exactActionQuery { intentDeclaration with requestedFaults := [fault] }) =
    [some .missingOccurrence, some .occurrenceMismatch] := by
  native_decide

/-! Unknown and wrong-kind fault capability references cannot enter checked intent. -/
example : [
    { selectedFault with capabilityDefinitionId := id "switch.capability.stale" },
    { selectedFault with capabilityDefinitionId := Umpire.Examples.Switch.flipActionId }
  ].map (fun fault => intentErrorKindOf <| checkArtifactIntent
    Umpire.Examples.Switch.exactActionQuery {
      intentDeclaration with requestedFaults := [fault]
    }) = [some .invalidCapability, some .invalidCapability] := by
  native_decide

/-! A selected role variant must agree with the kernel-produced setup. -/
example :
    let mismatched : ArtifactIntentDeclaration := {
      intentDeclaration with selectedVariants := [{
        selectedVariant with value := Umpire.Examples.Switch.onState
      }]
    }
    intentErrorKindOf (checkArtifactIntent Umpire.Examples.Switch.exactActionQuery mismatched) =
      some .variantMismatch := by
  native_decide

/-!
Projection resolves faults against the selected linear extension, never only the Behavior declaration.
-/
example :
    let ordinary := Umpire.Examples.Switch.compiledArtifact
    let missing := withValidChecksums {
      ordinary with plan := { ordinary.plan with linearExtension := [] }
    }
    let mismatchedOccurrence := ordinary.plan.linearExtension.map fun occurrence => {
      occurrence with actionDefinitionId := id "switch.action.stale"
    }
    let mismatched := withValidChecksums {
      ordinary with plan := { ordinary.plan with linearExtension := mismatchedOccurrence }
    }
    [missing, mismatched].map (fun spec =>
      intentErrorKindOf (spec.withArtifactIntent Umpire.Examples.Switch.exactActionQuery
        checkedIntent)) = [some .missingOccurrence, some .occurrenceMismatch] := by
  native_decide

/-! Checked intent cannot be reused after Query semantic identity drift. -/
example :
    let drifted := {
      Umpire.Examples.Switch.exactActionQuery with
      behaviorFingerprint := behaviorFingerprintOf "switch.query.drifted"
    }
    let ordinary := Umpire.Examples.Switch.compiledArtifact
    let driftedArtifact := {
      ordinary with plan := {
        ordinary.plan with kernelDefinitionId := id "switch.kernel.stale"
      }
    }
    intentErrorKindOf (planWithArtifactIntent drifted Umpire.Examples.Switch.incrementalKernel
      checkedIntent) = some .identityDrift &&
      intentErrorKindOf (driftedArtifact.withArtifactIntent
        Umpire.Examples.Switch.exactActionQuery checkedIntent) = some .identityDrift := by
  native_decide

end Umpire.SpaceTests
