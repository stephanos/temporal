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
      "sha256:f5af0c8a23a8cdcd1ef4aaddf1da5ca5564a8b49bd063dab5bd0fd736d5ef6ac" ∧
    specs.map (fun spec =>
      (spec.plan.queryDefinitionId.value, spec.plan.behaviorDefinitionId.value,
        spec.plan.artifactChecksum.render, spec.artifactChecksum.render)) = [
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.behavior",
        "sha256:fad2b87bbf67a53eafdc654be67d07e25d15eb6f0342bf1a5582398a9a7e57e6",
        "sha256:873abde394db383b2a9a750c6ecddebe1dedcf20faaea9ed75c9e623f3bca4cb"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.behavior",
        "sha256:14166e6a70911df0d72f444085d36a997ef01e2fc17112ed26cc0bacf8fb54f9",
        "sha256:ac24d6bac3eb7ac03b1bee26374c8a84bc06b211163d5c3f0169f7e2a6aa9ed4"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.behavior",
        "sha256:4f59bd1c683dfe41194c9446b1f24bf0cd77579d251c95b71855a05fb72a079f",
        "sha256:1272a5861a0a491cca53cd228dc8d72c1bcc970fd0c77b8eeee4eadcb3f2641e"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.behavior",
        "sha256:4997158e72edf2c8fc151d1c91a3d8e0407842eed2402b9f9ac1d349cc57aa5b",
        "sha256:b38b568baaaea2bf7e1d9aa6ca1b24e26f161c65c3fa3234e4f52ab741efc720"
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
