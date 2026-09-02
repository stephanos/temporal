import Temporal.Feature.Nexus.Experimental.CallerClosurePromotion

/-! Focused checks for the caller-closure promotion source inputs. -/

namespace Temporal.Feature.Nexus.Experimental.CallerClosurePromotionTests

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosureFault
open Temporal.Feature.Nexus.Experimental.CallerClosurePromotion

example : candidateId.value =
      "temporal.nexus.caller-closure.promotion.cancel-unique-regression" ∧
    promotedBehaviorId.value = "workflow-nexus.behavior.regression.cancel-is-unique" ∧
    promotedQueryId.value = "workflow-nexus.query.regression.cancel-is-unique" := by
  native_decide

example : baseAnchor.queryDefinitionId = exactActionQuery.id ∧
    baseAnchor.plannerRun = exactActionRun ∧
    baseAnchor.experimentSpec = compiledArtifact ∧
    baseAnchor.expectedTrace = expectedTrace ∧
    baseAnchor.selectionReason = .satisfyingWitness ∧
    expectedTrace.trace.steps.flatMap (fun step => step.observations) = [
      deliveredObservation,
      cancellationCountObservation,
      ownershipObservation
    ] ∧
    cancellationCountObservation.value = "1" := by
  native_decide

example : faultExperimentSpec.plan.requestedFaults = [{
      definitionId := duplicateDeliveryObservationFaultId
      value := forceCloseOccurrenceId.value
    }] ∧
    faultExperimentSpec.plan.queryDefinitionId != baseAnchor.queryDefinitionId ∧
    faultExperimentSpec.plan.artifactChecksum != baseAnchor.experimentSpec.plan.artifactChecksum ∧
    faultExperimentSpec.artifactChecksum != baseAnchor.experimentSpec.artifactChecksum ∧
    faultExperimentSpec.plan.targetDefinitionId = baseAnchor.targetDefinitionId ∧
    faultExperimentSpec.plan.kernelDefinitionId = baseAnchor.kernelDefinitionId ∧
    faultExperimentSpec.plan.checkpoints = baseAnchor.experimentSpec.plan.checkpoints := by
  native_decide

example : sourceSpec.promotedBehaviorDefinitionId = promotedBehaviorId ∧
    sourceSpec.promotedQueryDefinitionId = promotedQueryId ∧
    sourceExpectation.bytes = renderPromotionSource sourceSpec expectedTrace ∧
    sourceExpectation.sha256 = promotionSourceSha256 sourceExpectation.bytes ∧
    sourceExpectation.bytes.startsWith "import Umpire.Promotion\n\n" ∧
    !sourceExpectation.bytes.contains "import Temporal" := by
  native_decide

end Temporal.Feature.Nexus.Experimental.CallerClosurePromotionTests
