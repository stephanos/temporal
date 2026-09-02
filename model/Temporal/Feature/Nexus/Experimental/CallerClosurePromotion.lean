import Temporal.Feature.Nexus.Experimental.CallerClosureFault
import Umpire.Promotion

/-!
# Caller-closure promotion inputs

This module fixes the one caller-closure promotion candidate to two separate checked lineages. The
unchanged base Query and PlannerRun supply the target-owned expected trace, while the selected
duplicate-delivery Space point supplies only the fault-bearing `ExperimentSpec` later consumed by
runtime replay. The source inputs remain inert and contain no runtime eligibility claim.
-/

namespace Temporal.Feature.Nexus.Experimental.CallerClosurePromotion

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosureFault

/-- The sole caller-closure failure that may be considered for reviewed promotion. -/
def candidateId : DefinitionId :=
  DefinitionId.of "temporal.nexus.caller-closure.promotion.cancel-unique-regression"

/-- Fixed fresh Behavior identity carried by the review-only source. -/
def promotedBehaviorId : DefinitionId :=
  DefinitionId.of "workflow-nexus.behavior.regression.cancel-is-unique"

/-- Fixed fresh Query identity carried by the review-only source. -/
def promotedQueryId : DefinitionId :=
  DefinitionId.of "workflow-nexus.query.regression.cancel-is-unique"

/-- Stable identity of the sealed review-only source. -/
def sourceDefinitionId : DefinitionId :=
  DefinitionId.of "workflow-nexus.promotion-source.regression.cancel-is-unique"

/-- Source location retained by the proposed checked declarations. -/
def sourceLocation : SourceLocation := {
  path := "Temporal/Feature/Nexus/Experimental/CallerClosurePromotion.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

private def expectedTrace? : Option BehaviorTrace :=
  match exactActionRun.result.outcome with
  | .found trace _ => some trace
  | _ => none

private theorem expectedTrace?_isSome : expectedTrace?.isSome = true := by
  native_decide

/-- Target-owned count-one trace selected by the unchanged exact-action Query. -/
def expectedTrace : BehaviorTrace :=
  expectedTrace?.get expectedTrace?_isSome

/-- Complete unchanged base lineage against which source compilation replans. -/
def baseAnchor : PromotionBaseAnchor := {
  queryDefinitionId := exactActionQuery.id
  queryBehaviorFingerprint := exactActionQuery.behaviorFingerprint
  queryCanonicalMetadata := exactActionQuery.canonicalMetadata
  behaviorDefinitionId := exactActionQuery.behavior.id
  behaviorFingerprint := exactActionQuery.behavior.behaviorFingerprint
  targetDefinitionId := exactActionQuery.target.id
  targetBehaviorFingerprint := exactActionQuery.target.behaviorFingerprint
  kernelDefinitionId := exactActionQuery.target.kernel.metadata.id
  kernelBehaviorFingerprint := exactActionQuery.target.behaviorFingerprint
  plannerRun := exactActionRun
  experimentSpec := compiledArtifact
  expectedTrace
  selectionReason := .satisfyingWitness
}

private def faultExperimentSpec? : Option ExperimentSpec :=
  batchResult.toOption.bind fun specs => specs.tail.head?

private theorem faultExperimentSpec?_isSome : faultExperimentSpec?.isSome = true := by
  native_decide

/-- Selected duplicate-delivery Space-point Artifact, distinct from the base Artifact. -/
def faultExperimentSpec : ExperimentSpec :=
  faultExperimentSpec?.get faultExperimentSpec?_isSome

/-- Fixed quoted data accepted by the closed source renderer. -/
def sourceSpec : PromotionSourceSpec := {
  sourceDefinitionId
  sourceLocation
  promotedBehaviorDefinitionId := promotedBehaviorId
  promotedQueryDefinitionId := promotedQueryId
}

private def sourceBytes : String :=
  renderPromotionSource sourceSpec expectedTrace

/-- Exact source bytes and pinned digest for the one review-only declaration. -/
def sourceExpectation : PromotionSourceExpectation := {
  bytes := sourceBytes
  sha256 := "sha256:e96b36e27a71279e67e04f6dbbb9310daba587a1bfe9a75c20482ffe596ce2fe"
}

end Temporal.Feature.Nexus.Experimental.CallerClosurePromotion
