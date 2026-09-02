import Temporal.Tool.PromotionBinding

/-! Focused checks for the one static caller-closure promotion binding. -/

namespace Temporal.Tool.PromotionBindingTests

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosurePromotion
open Temporal.Tool.PromotionBinding

private def errorKindOf
    (result : Except PromotionBindingError PromotionCandidateBinding) :
    Option PromotionBindingErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

example : (resolveCandidate candidateId.value).toOption.map (fun binding =>
      binding.identity == candidateId &&
        canonicalQueryJson binding.baseQuery == canonicalQueryJson exactActionQuery &&
        binding.baseQuery.source == exactActionQuery.source &&
        binding.basePlannerRun == exactActionRun &&
        binding.baseExperimentSpec == compiledArtifact &&
        binding.faultExperimentSpec == faultExperimentSpec &&
        binding.compiledSource.baseExperimentSpecChecksum == compiledArtifact.artifactChecksum &&
        binding.compiledSource.expectedTrace == expectedTrace &&
        binding.compiledSource.promotedBehaviorDefinitionId == promotedBehaviorId &&
        binding.compiledSource.promotedQueryDefinitionId == promotedQueryId) = some true := by
  native_decide

example : [
    candidateId.value ++ ".extra",
    "Temporal.nexus.caller-closure.promotion.cancel-unique-regression",
    promotedQueryId.value,
    ""
  ].map (errorKindOf ∘ resolveCandidate) = List.replicate 4 (some .unknownCandidate) := by
  native_decide

private def changedId : DefinitionId :=
  DefinitionId.of "temporal.nexus.caller-closure.promotion.changed"

private def changedFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf "temporal/nexus/caller-closure/promotion/changed"

private def observedCountTwoTrace : BehaviorTrace := {
  expectedTrace with
  trace := {
    expectedTrace.trace with
    steps := expectedTrace.trace.steps.map fun step => {
      step with observations := step.observations ++ [cancellationCountObservation]
    }
  }
}

private def changedExperimentSpec : ExperimentSpec := {
  faultExperimentSpec with artifactChecksum := experimentSpecChecksumOf "changed"
}

example : [
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with identity := changedId
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseQuery := { exactActionQuery with id := changedId }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseAnchor := {
        baseAnchor with queryBehaviorFingerprint := changedFingerprint
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseAnchor := {
        baseAnchor with targetDefinitionId := changedId
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseAnchor := {
        baseAnchor with kernelDefinitionId := changedId
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseAnchor := {
        baseAnchor with plannerRun := { exactActionRun with instrumentation := {} }
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseAnchor := {
        baseAnchor with experimentSpec := changedExperimentSpec
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with baseAnchor := {
        baseAnchor with expectedTrace := observedCountTwoTrace
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with faultExperimentSpec := changedExperimentSpec
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with sourceSpec := {
        sourceSpec with sourceDefinitionId := changedId
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with sourceSpec := {
        sourceSpec with promotedBehaviorDefinitionId := changedId
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with sourceSpec := {
        sourceSpec with promotedQueryDefinitionId :=
          Temporal.Tool.NexusDiscovery.inventory.entries.head?.get (by native_decide) |>.query.id
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with sourceExpectation := {
        sourceExpectation with bytes := sourceExpectation.bytes ++ " "
      }
    }),
    errorKindOf (Internal.checkCandidate {
      Internal.candidate with sourceExpectation := {
        sourceExpectation with sha256 := promotionSourceSha256 "changed"
      }
    })
  ] = [
    some .candidateIdentityDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .baseLineageDrift,
    some .faultLineageDrift,
    some .sourceLineageDrift,
    some .sourceLineageDrift,
    some .promotedIdentityCollision,
    some .sourceLineageDrift,
    some .sourceLineageDrift
  ] := by
  native_decide

example : Internal.discoveryDefinitionIds.length > 12 ∧
    !Internal.discoveryDefinitionIds.contains promotedBehaviorId ∧
    !Internal.discoveryDefinitionIds.contains promotedQueryId := by
  native_decide

/--
error: `runtimeResult` is not a field of structure `PromotionCandidateInput`
---
info: let __src := Internal.candidate;
__src : PromotionCandidateInput
-/
#guard_msgs in
#check ({ Internal.candidate with
  runtimeResult := "accepted"
} : PromotionCandidateInput)

/--
error: `imports` is not a field of structure `PromotionCandidateInput`
---
info: let __src := Internal.candidate;
__src : PromotionCandidateInput
-/
#guard_msgs in
#check ({ Internal.candidate with
  imports := ["Temporal.Runtime"]
} : PromotionCandidateInput)

end Temporal.Tool.PromotionBindingTests
