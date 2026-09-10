import Temporal.Feature.Nexus.Experimental.VariationSpace

namespace Temporal.Feature.Nexus.Experimental.VariationSpaceTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Experimental.VariationSpace

private def prepared : PreparedVariationSpace :=
  preparedResult.toOption.get (by native_decide)

private def checked : CheckedExperimentSpace LawStatement := prepared.checked
private def metadata : CheckedSpaceMetadata := prepared.metadata
private def specs : List Plan := prepared.specs
private def behavior : CheckedScenario := checked.baseQuery.behavior
private def context : SpaceCheckContext LawStatement := .ofQuery checked.baseQuery

example : behaviorResult.isOk = true ∧ queryResult.isOk = true ∧
    preparedResult.isOk = true ∧ metadataResult.isOk = true ∧ batchResult.isOk = true := by
  native_decide

example : metadata.space.id = spaceId ∧
    metadata.space.baseBehavior.id = behaviorId ∧
    metadata.space.baseQuery.id = queryId ∧
    metadata.space.target.id = targetId ∧
    metadata.space.pointCount = 4 ∧
    metadata.axes.map SpaceAxisMetadataRow.id = [completionFaultAxisId, startFaultAxisId] ∧
    metadata.choices.map SpaceChoiceMetadataRow.id = [
      completionBaselineChoiceId,
      completionHandlerFailureChoiceId,
      startBaselineChoiceId,
      startDelayChoiceId
    ] ∧
    metadata.faults.map SpaceFaultMetadataRow.id = [
      completionHandlerFailureFaultId,
      startDelayFaultId
    ] ∧
    metadata.coverageGoals.map SpaceCoverageGoalMetadataRow.id = [
      completionBaselineCoverageGoalId,
      completionHandlerFailureCoverageGoalId,
      startBaselineCoverageGoalId,
      startDelayCoverageGoalId
    ] := by
  native_decide

example : behavior.requiredOccurrences = [
      { id := startOccurrenceId, action := startActionId },
      { id := successOccurrenceId, action := reportSuccessActionId }
    ] ∧
    specs.map (fun spec => spec.plan.selectedChoices) = canonicalAssignments ∧
    specs.map (fun spec => spec.plan.requestedFaults.map ModelValue.definitionId) = [
      [],
      [startDelayFaultId],
      [completionHandlerFailureFaultId],
      [completionHandlerFailureFaultId, startDelayFaultId]
    ] ∧
    specs.all (fun spec =>
      spec.plan.requestedActions == [startAction, reportSuccessAction] &&
        spec.plan.modelOutcomes == [startedOutcome, succeededOutcome] &&
        spec.plan.resultingStates == [startedState, succeededState] &&
        spec.plan.checkpoints.map ObservationCheckpoint.observations ==
          [[startedObservation], [succeededObservation]] &&
        spec.plan.selectedVariants.isEmpty &&
        spec.plan.capabilityRequirementDefinitionIds == [lifecycleCapabilityId]) = true := by
  native_decide

example : reorderedMetadataResult.toOption == metadataResult.toOption ∧
    reorderedBatchResult.toOption.map (List.map canonicalPlanBytes) =
      batchResult.toOption.map (List.map canonicalPlanBytes) := by
  native_decide

example : metadata.behaviorFingerprint.render =
      "sha256:9f1998b27b0f83f7f73e596736bbe5a76d8fc426ac8243bcaf5bc86d2f9c53de" ∧
    specs.map (fun spec =>
      (spec.plan.queryDefinitionId.value, spec.plan.behaviorDefinitionId.value,
        spec.plan.artifactChecksum.render, spec.artifactChecksum.render)) = [
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.behavior",
        "sha256:3aed9f59d131f83d81ca57b41a4757719a7886814893b48d1b0471834f6f2f5b",
        "sha256:34b04499cc9163e1a5abb39963d9ce215f82fe41e355967ceeb33b07419dd1be"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.behavior",
        "sha256:42bbe25ce5e3ec9602e37b9bbf0f1f14a0026b9b516900092beacef6459b3057",
        "sha256:79aa2d3b71be2ea76aadb3692e5e60258e1c590c6628a8c5cc0acda4f8fae555"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.behavior",
        "sha256:c514e0063a77585000e15177b9d67344c635742e6bbde0ca662e8316ed6aa871",
        "sha256:8a7e1f0f32b24b3b30024e3c6efe06274abf1726f4bf422f46dfb4112803186d"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.behavior",
        "sha256:7783e0bda139f00d09262aac5bdf1562972b9554ce1752ebcc0ab2a031158c72",
        "sha256:916a2cc537c6c35066ad8a147cf139e00c661612594e80a59d90e1fc585d6221"
      )
    ] := by
  native_decide

private def spaceErrorKindOf
    (result : Except SpaceError (CheckedExperimentSpace LawStatement)) : Option SpaceErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

private def duplicateEffectChoice : ChoiceDeclaration := {
  startDelayChoice with
  id := DefinitionId.of "temporal.nexus.basic-lifecycle.choice.start-delay-duplicate"
}

private def duplicateEffectDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [{ startFaultAxis with choices := [startDelayChoice, duplicateEffectChoice] },
    completionFaultAxis]
}

private def staleOccurrenceDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  faults := [{ startDelayFault with occurrence := (DefinitionId.of
    "temporal.nexus.basic-lifecycle.occurrence.two-action.stale") },
    completionHandlerFailureFault]
}

private def staleCapabilityDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  faults := [{ startDelayFault with capability := (DefinitionId.of
    "temporal.nexus.basic-lifecycle.capability.stale") },
    completionHandlerFailureFault]
}

private def impossibleGoalDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  coverageGoals := [{ startDelayCoverageGoal with minimum := 3 }]
}

private def incompatibleSelectionDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  faults := [
    { startDelayFault with incompatibleWith := [completionHandlerFailureFaultId] },
    { completionHandlerFailureFault with incompatibleWith := [startDelayFaultId] }
  ]
}

example : [
    duplicateEffectDeclaration,
    staleOccurrenceDeclaration,
    staleCapabilityDeclaration,
    impossibleGoalDeclaration,
    incompatibleSelectionDeclaration
  ].map (fun candidate => spaceErrorKindOf (checkExperimentSpace context candidate)) = [
    some .duplicateChoiceEffect,
    some .unknownOccurrence,
    some .unknownCapability,
    some .impossibleCoverageGoal,
    some .incompatibleFaultSelection
  ] := by
  native_decide


end Temporal.Feature.Nexus.Experimental.VariationSpaceTests
