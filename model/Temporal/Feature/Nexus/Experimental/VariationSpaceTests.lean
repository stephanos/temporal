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
      "sha256:f52c963f12c8be77047648461516754301c8091e03cbf020e2cf5fd29fd1c4bf" ∧
    specs.map (fun spec =>
      (spec.plan.queryDefinitionId.value, spec.plan.behaviorDefinitionId.value,
        spec.plan.artifactChecksum.render, spec.artifactChecksum.render)) = [
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.behavior",
        "sha256:6149c181b5cd6cc77b107fc2264fa4b024f094ce9214456cfee3150d9f841d1d",
        "sha256:825d01d0a03e84b4234eb590ffab2afefcf92bce659a6efd1320bd128c3d01cf"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.behavior",
        "sha256:a24387de4c9bf793a5955fb5e3393f214dbe4e7bb4e98eba1cf5b8222f35a894",
        "sha256:d036efe10679c29dce068d75933ba5c12ebe574dd71bcc40790406ef64339c3b"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.behavior",
        "sha256:10f93d7664ae421adccaf7ea8f829b2b24aa832569f08f063de59666df26c06f",
        "sha256:3746b90f78163b803a68575d8352bce26d0f66cdbd72b65a5b80f3469bd31ca6"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.behavior",
        "sha256:5a6070eebe17b150917b5a8d944fda90ae26c2b9ef11e33871a1858b46bd7772",
        "sha256:add1bd4201541142516de626e3f218b5709b6b4466bee7c84f6fe8b26c4f77f8"
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
