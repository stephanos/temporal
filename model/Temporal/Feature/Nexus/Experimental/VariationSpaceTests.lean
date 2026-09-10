import Temporal.Feature.Nexus.Experimental.VariationSpace

namespace Temporal.Feature.Nexus.Experimental.VariationSpaceTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Experimental.VariationSpace

private def prepared : PreparedVariationSpace :=
  preparedResult.toOption.get (by native_decide)

private def checked : CheckedExperimentSpace LawStatement := prepared.checked
private def metadata : CheckedSpaceMetadata := prepared.metadata
private def specs : List ExperimentSpec := prepared.specs
private def behavior : CheckedBehavior := checked.baseQuery.behavior
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
    reorderedBatchResult.toOption.map (List.map canonicalExperimentSpecBytes) =
      batchResult.toOption.map (List.map canonicalExperimentSpecBytes) := by
  native_decide

example : metadata.behaviorFingerprint.render =
      "sha256:9d7b2fd6d1a52f92321121f34a20e54cb60dbe4552f12d1fc151ea6dc5174367" ∧
    specs.map (fun spec =>
      (spec.plan.queryDefinitionId.value, spec.plan.behaviorDefinitionId.value,
        spec.plan.artifactChecksum.render, spec.artifactChecksum.render)) = [
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.behavior",
        "sha256:acbc21e0613adfebe718ba4a78c9e1698fb8e8500a1353547b75718afa1a9d4b",
        "sha256:23b6bac41bfee2637d1ba16737c50c9f3df83ba5c520b1c9e72bb6257c1e2bbc"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.behavior",
        "sha256:dc21a6e43b03971bd101cea1fb4755f4dc3038105ebaa0473e759667b7fa8518",
        "sha256:868d5978e85783eabb8af534b375be1d50c22f44456355898e8a3949d12f3607"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.behavior",
        "sha256:dc8fe19ffa609eddedeb0c9ba1edda3074c9c0054ac3b4c5bb5403fda2fd8118",
        "sha256:0beae8011517812fbf284afdbe70eb6b329b338f3efcc52a7ba20fab9c9de7b8"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.behavior",
        "sha256:b28871e488d2eb54e13fcbd1234943ba604e0b3497ce5dcebfe6c2a8b6bbcc40",
        "sha256:08ca605e2d5b556230d7593b9a7e088b00970f153e9145963f0873983b5b634b"
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
