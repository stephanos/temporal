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
      "sha256:d49e750c2205de6b20faf347bda1d1207a2e9ca9ba185c670a2f4c3f361ac3aa" ∧
    specs.map (fun spec =>
      (spec.plan.queryDefinitionId.value, spec.plan.behaviorDefinitionId.value,
        spec.plan.artifactChecksum.render, spec.artifactChecksum.render)) = [
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.2a58049440a727cf7c6d4fc6ee6170ad93e4e760e8335e898b56334e36e8b49f.behavior",
        "sha256:3a8aad1483f564d7fcb0ad52fde0c6919025fd17f042c88a8605ea7332249301",
        "sha256:29b441158e81bc6ed56ef4d9644c4b17dbc6887c5137f2f0ee7bb0f8c3635e35"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.eeb5f0ebe497093667fd32438f2fdbb86bcf280365384d6052c55f974928bc57.behavior",
        "sha256:6961096176f6d14bc7b4e83af78c824280ca2519ff21e05a077e80ddb9fd8771",
        "sha256:0c7c1802a9a680e5418de51a0bad8b546cf4186c85e72468086bcc2114f867be"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.e0236c7b60bb7385d889ca90eb37214572f2944773155cfcb22beeac62531d5c.behavior",
        "sha256:8865b6538b3998f77e9f620cee526fce3381f79136fd94a522da65dd30dab323",
        "sha256:f96937ff66ce229d722bd1c5de5b723785058f930ebde6495779d5805f7defc6"
      ),
      (
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.query",
        "temporal.nexus.basic-lifecycle.space.fault-matrix.point.75801c294e9ad01c61860ac4febbac4498c47a19d8355ab6204a32ed0247afef.behavior",
        "sha256:44829b64d547ab876f498c239ed4ed17b5a9185b901062e587c7349214d41330",
        "sha256:e4c43fb50dfc533e683e900472c9241a638707f1b1e91862e2f592e4ed13d3e1"
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
