import Temporal.Feature.Nexus.Experimental.CallerClosureFault

namespace Temporal.Feature.Nexus.Experimental.CallerClosureFaultTests

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosureFault

private def prepared : PreparedCallerClosureFaultSpace :=
  preparedResult.toOption.get (by native_decide)

private def checked : CheckedExperimentSpace LawStatement := prepared.checked
private def metadata : CheckedSpaceMetadata := prepared.metadata
private def specs : List ExperimentSpec := prepared.specs
private def baselineSpec : ExperimentSpec := specs.head?.get (by native_decide)
private def faultedSpec : ExperimentSpec := specs.tail.head?.get (by native_decide)
private def context : SpaceCheckContext LawStatement := .ofQuery exactActionQuery

private def baselineAssignment : List ModelValue := [{
  definitionId := cancellationDeliveryAxisId
  value := deliveryBaselineChoiceId.value
}]

private def faultedAssignment : List ModelValue := [{
  definitionId := cancellationDeliveryAxisId
  value := duplicateDeliveryObservationChoiceId.value
}]

private def baselinePoint := lowerSpacePoint checked baselineAssignment
private def faultedPoint := lowerSpacePoint checked faultedAssignment
private def loweredFaultedPoint := faultedPoint.toOption.get (by native_decide)

example : preparedResult.isOk = true ∧ metadataResult.isOk = true ∧ batchResult.isOk = true := by
  native_decide

example : metadata.space.id = spaceId ∧
    metadata.space.baseBehavior.id = exactActionBehaviorId ∧
    metadata.space.baseQuery.id = exactActionQueryId ∧
    metadata.space.target.id = targetId ∧
    metadata.space.pointCount = 2 ∧
    metadata.axes.map SpaceAxisMetadataRow.id = [cancellationDeliveryAxisId] ∧
    metadata.choices.map SpaceChoiceMetadataRow.id = [
      deliveryBaselineChoiceId,
      duplicateDeliveryObservationChoiceId
    ] ∧
    metadata.faults.map SpaceFaultMetadataRow.id = [duplicateDeliveryObservationFaultId] ∧
    metadata.coverageGoals.map SpaceCoverageGoalMetadataRow.id = [
      deliveryBaselineCoverageGoalId,
      duplicateDeliveryObservationCoverageGoalId
    ] := by
  native_decide

example : checked.baseQuery.id == exactActionQuery.id &&
    checked.baseQuery.behaviorFingerprint == exactActionQuery.behaviorFingerprint &&
    checked.baseQuery.behavior.requiredOccurrences == [{
      id := forceCloseOccurrenceId
      action := forceCloseActionId
    }] &&
    checked.faults.map (fun fault =>
      (fault.id, fault.occurrence.id, fault.occurrence.action, fault.capability.id)) == [(
        duplicateDeliveryObservationFaultId,
        forceCloseOccurrenceId,
        forceCloseActionId,
        cancellationCapabilityId
      )] := by
  native_decide

example : baselinePoint.toOption.map (fun point =>
      point.assignment == baselineAssignment &&
        point.intent.selectedChoices == baselineAssignment &&
        point.intent.selectedVariants.isEmpty &&
        point.intent.requestedFaults.isEmpty &&
        point.intent.additionalCapabilityRequirementDefinitionIds.isEmpty) = some true ∧
    faultedPoint.toOption.map (fun point =>
      point.assignment == faultedAssignment &&
        point.intent.selectedChoices == faultedAssignment &&
        point.intent.selectedVariants.isEmpty &&
        point.intent.requestedFaults == [{
          definitionId := duplicateDeliveryObservationFaultId
          occurrenceDefinitionId := forceCloseOccurrenceId
          actionDefinitionId := forceCloseActionId
          capabilityDefinitionId := cancellationCapabilityId
        }] &&
        point.intent.additionalCapabilityRequirementDefinitionIds ==
          [cancellationCapabilityId]) = some true := by
  native_decide

example : specs.map (fun spec => spec.plan.selectedChoices) = [
      baselineAssignment,
      faultedAssignment
    ] ∧
    baselineSpec.plan.requestedFaults = [] ∧
    faultedSpec.plan.requestedFaults = [{
      definitionId := duplicateDeliveryObservationFaultId
      value := forceCloseOccurrenceId.value
    }] ∧
    baselineSpec.plan.capabilityRequirementDefinitionIds =
      compiledArtifact.plan.capabilityRequirementDefinitionIds ∧
    faultedSpec.plan.capabilityRequirementDefinitionIds =
      compiledArtifact.plan.capabilityRequirementDefinitionIds := by
  native_decide

example : specs.all (fun spec =>
      spec.plan.initialState == compiledArtifact.plan.initialState &&
        spec.plan.requestedActions == [forceCloseAction] &&
        spec.plan.modelOutcomes == [upgradedOutcome] &&
        spec.plan.resultingStates == [closedState] &&
        spec.plan.linearExtension == compiledArtifact.plan.linearExtension &&
        spec.plan.checkpoints == [{
          transition := 1
          observations := [
            deliveredObservation,
            cancellationCountObservation,
            ownershipObservation
          ]
        }] &&
        spec.properties == compiledArtifact.properties) = true ∧
    cancellationCountObservation.value = "1" := by
  native_decide

example : reorderedMetadataResult.toOption == metadataResult.toOption ∧
    reorderedBatchResult.toOption.map (List.map canonicalExperimentSpecBytes) =
      batchResult.toOption.map (List.map canonicalExperimentSpecBytes) := by
  native_decide

example : metadata.behaviorFingerprint.render =
      "sha256:709e415dded0028be24da720fd2239ced01f9bcb3747ba08c430396101428f50" ∧
    specs.map (fun spec =>
      (spec.plan.queryDefinitionId.value, spec.plan.behaviorDefinitionId.value,
        spec.plan.artifactChecksum.render, spec.artifactChecksum.render)) = [
      (
        "temporal.nexus.caller-closure.space.duplicate-delivery-negative-control.point.5f8b80937061161e66a35cdc18d7248ca597f7ae11bf61860602dca17c78199e.query",
        "temporal.nexus.caller-closure.space.duplicate-delivery-negative-control.point.5f8b80937061161e66a35cdc18d7248ca597f7ae11bf61860602dca17c78199e.behavior",
        "sha256:4f5934f00e93be3d8e2a94afdce0dfc299d589d03547d026c9bae1d263ec3fa5",
        "sha256:2fc8604071f9d62f0002bc08d948b82a919de372eb02bfb089710851a073d71c"
      ),
      (
        "temporal.nexus.caller-closure.space.duplicate-delivery-negative-control.point.088f85a0eb86d54cfee9512429211767702568152794d302210b0a4041327d4f.query",
        "temporal.nexus.caller-closure.space.duplicate-delivery-negative-control.point.088f85a0eb86d54cfee9512429211767702568152794d302210b0a4041327d4f.behavior",
        "sha256:cda6d0ae81e035bbfca42e4dd5b17d8b893fbd128f68f263d6693f101c44ef90",
        "sha256:09091758defd5ce50cc9acbba23a5c8499da4eef9b6e36878ac989ddea87fedf"
      )
    ] := by
  native_decide

private def spaceErrorKindOf
    (result : Except SpaceError (CheckedExperimentSpace LawStatement)) : Option SpaceErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

private def duplicateEffectChoice : ChoiceDeclaration := {
  duplicateDeliveryObservationChoice with
  id := DefinitionId.of "temporal.nexus.caller-closure.choice.duplicate-delivery-effect"
}

private def invalidLimitsDeclaration : ExperimentSpaceDeclaration := {
  declaration with axes := []
}

private def duplicateEffectDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [{ cancellationDeliveryAxis with choices := [
    deliveryBaselineChoice,
    duplicateDeliveryObservationChoice,
    duplicateEffectChoice
  ] }]
}

private def staleOccurrenceDeclaration : ExperimentSpaceDeclaration := {
  declaration with faults := [{ duplicateDeliveryObservationFault with
    occurrence := DefinitionId.of "workflow-nexus.occurrence.stale-force-close" }]
}

private def staleActionDeclaration : ExperimentSpaceDeclaration := {
  declaration with faults := [{ duplicateDeliveryObservationFault with
    action := DefinitionId.of "workflow.action.stale-force-close" }]
}

private def staleCapabilityDeclaration : ExperimentSpaceDeclaration := {
  declaration with faults := [{ duplicateDeliveryObservationFault with
    capability := DefinitionId.of "nexus.capability.stale-cancellation" }]
}

private def duplicateSelectionDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [{ cancellationDeliveryAxis with choices := [
    deliveryBaselineChoice,
    { duplicateDeliveryObservationChoice with faults := [
      duplicateDeliveryObservationFaultId,
      duplicateDeliveryObservationFaultId
    ] }
  ] }]
}

private def incompatibleFaultId : DefinitionId :=
  DefinitionId.of "temporal.nexus.caller-closure.fault.incompatible-duplicate-delivery"

private def incompatibleSelectionDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [{ cancellationDeliveryAxis with choices := [
    deliveryBaselineChoice,
    { duplicateDeliveryObservationChoice with faults := [
      duplicateDeliveryObservationFaultId,
      incompatibleFaultId
    ] }
  ] }]
  faults := [
    { duplicateDeliveryObservationFault with incompatibleWith := [incompatibleFaultId] },
    { duplicateDeliveryObservationFault with
      id := incompatibleFaultId
      incompatibleWith := [duplicateDeliveryObservationFaultId] }
  ]
}

private def invalidGoalDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  coverageGoals := [{ deliveryBaselineCoverageGoal with minimum := 2 }]
}

private def outcomeAuthoringDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [{ cancellationDeliveryAxis with
    role := some operationRoleId
    choices := [
      deliveryBaselineChoice,
      { duplicateDeliveryObservationChoice with binding := some upgradedOutcome }
    ] }]
}

private def evidenceAuthoringDeclaration : ExperimentSpaceDeclaration := {
  declaration with
  axes := [{ cancellationDeliveryAxis with
    role := some operationRoleId
    choices := [
      deliveryBaselineChoice,
      { duplicateDeliveryObservationChoice with binding := some {
        definitionId := DefinitionId.of "temporal.nexus.caller-closure.evidence.synthetic"
        value := "synthetic"
      } }
    ] }]
}

example : [
    invalidLimitsDeclaration,
    duplicateEffectDeclaration,
    staleOccurrenceDeclaration,
    staleActionDeclaration,
    staleCapabilityDeclaration,
    duplicateSelectionDeclaration,
    incompatibleSelectionDeclaration,
    invalidGoalDeclaration,
    outcomeAuthoringDeclaration,
    evidenceAuthoringDeclaration
  ].map (fun candidate => spaceErrorKindOf (checkExperimentSpace context candidate)) = [
    some .axisCountOutOfRange,
    some .duplicateChoiceEffect,
    some .unknownOccurrence,
    some .occurrenceActionMismatch,
    some .unknownCapability,
    some .duplicateFaultSelection,
    some .incompatibleFaultSelection,
    some .impossibleCoverageGoal,
    some .wrongValueKind,
    some .unknownValue
  ] := by
  native_decide

example : (match lowerSpacePoint checked [] with
    | .error error => error.kind == .missingChoice
    | .ok _ => false) = true := by
  native_decide

private def planningFailureResult :=
  SpaceCompiler.Internal.appendPlannerRun checked loweredFaultedPoint.id [] verifyRun

example : (match planningFailureResult with
    | .error error => error.kind == .verifiedWithoutArtifact
    | .ok _ => false) = true := by
  native_decide

end Temporal.Feature.Nexus.Experimental.CallerClosureFaultTests
