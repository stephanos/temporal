import Temporal.Feature.Nexus.Experimental.CallerClosurePromotion
import Temporal.Tool.NexusDiscovery

/-!
# Static caller-closure promotion binding

This module resolves one fixed candidate and checks its base, fault, identity, discovery, and source
lineages before exposing a sealed `CompiledPromotionSource`. The public resolver accepts only the
candidate identity; mutation inputs live under `Internal` for focused fail-closed tests. Runtime
results, replay status, reduction evidence, and eligibility remain downstream concerns.
-/

namespace Temporal.Tool.PromotionBinding

open Umpire
open Temporal.Feature.Nexus.Experimental.CallerClosure
open Temporal.Feature.Nexus.Experimental.CallerClosurePromotion

/-- Stable failure classes for the exact static promotion binding. -/
inductive PromotionBindingErrorKind where
  | unknownCandidate
  | candidateIdentityDrift
  | baseLineageDrift
  | faultLineageDrift
  | promotedIdentityCollision
  | sourceLineageDrift
  | sourceCompilation
  deriving BEq, DecidableEq, Ord, Repr

/-- Structured failure returned without source bytes or runtime claims. -/
structure PromotionBindingError where
  kind : PromotionBindingErrorKind
  subject : DefinitionId
  detail : String
  deriving BEq, DecidableEq, Repr

/-- Untrusted static input used only by the closed binding checker and its mutation tests. -/
structure PromotionCandidateInput where
  identity : DefinitionId
  baseAnchor : PromotionBaseAnchor
  faultExperimentSpec : ExperimentSpec
  sourceSpec : PromotionSourceSpec
  sourceExpectation : PromotionSourceExpectation

/-- Checked base/fault pair and sealed source for the one retained candidate. -/
structure PromotionCandidateBinding where
  private mk ::
  identity : DefinitionId
  baseQuery : CheckedQuery LawStatement
  basePlannerRun : PlannerRun
  baseExperimentSpec : ExperimentSpec
  faultExperimentSpec : ExperimentSpec
  compiledSource : CompiledPromotionSource

private def bindingError
    (kind : PromotionBindingErrorKind)
    (subject : DefinitionId)
    (detail : String) : PromotionBindingError := {
  kind
  subject
  detail
}

namespace Internal

/-- Every Definition ID exposed by the checked closed Nexus discovery inventory. -/
def discoveryDefinitionIds : List DefinitionId :=
  NexusDiscovery.inventory.entries.flatMap fun entry =>
    [
      entry.property.id,
      entry.behavior.id,
      entry.query.id,
      entry.plan.queryDefinitionId,
      entry.plan.behaviorDefinitionId,
      entry.plan.targetDefinitionId,
      entry.plan.kernelDefinitionId
    ] ++ entry.plan.properties.map (·.definitionId) ++ entry.plan.provenanceDefinitionIds

/-- The only complete candidate input accepted by the binding checker. -/
def candidate : PromotionCandidateInput := {
  identity := candidateId
  baseAnchor
  faultExperimentSpec
  sourceSpec
  sourceExpectation
}

private def promotedIdentitiesCollide
    (candidate : PromotionCandidateInput) : Bool :=
  discoveryDefinitionIds.contains candidate.sourceSpec.promotedBehaviorDefinitionId ||
    discoveryDefinitionIds.contains candidate.sourceSpec.promotedQueryDefinitionId

/-- Validate the fixed candidate atomically and expose no source on any drift. -/
def checkCandidate
    (candidate : PromotionCandidateInput) :
    Except PromotionBindingError PromotionCandidateBinding := do
  if candidate.identity != candidateId then
    throw (bindingError .candidateIdentityDrift candidate.identity
      "candidate identity does not match the closed binding")
  if candidate.baseAnchor != baseAnchor then
    throw (bindingError .baseLineageDrift candidate.identity
      "base Query, Target, kernel, PlannerRun, ExperimentSpec, or expected trace drift")
  if candidate.faultExperimentSpec != faultExperimentSpec ||
      candidate.faultExperimentSpec == candidate.baseAnchor.experimentSpec then
    throw (bindingError .faultLineageDrift candidate.identity
      "fault-bearing ExperimentSpec is crossed or does not match the selected Space point")
  if promotedIdentitiesCollide candidate then
    throw (bindingError .promotedIdentityCollision candidate.identity
      "promoted identity collides with the closed Nexus discovery inventory")
  if candidate.sourceSpec != sourceSpec || candidate.sourceExpectation != sourceExpectation then
    throw (bindingError .sourceLineageDrift candidate.identity
      "source identity, location, promoted identity, bytes, or digest drift")
  let compiled ← match compilePromotionSource exactActionQuery incrementalKernel
      candidate.baseAnchor candidate.sourceSpec candidate.sourceExpectation with
    | .ok compiled => pure compiled
    | .error error =>
        throw (bindingError .sourceCompilation error.subject
          "sealed promotion source compilation failed")
  pure (PromotionCandidateBinding.mk candidateId exactActionQuery exactActionRun compiledArtifact
    faultExperimentSpec compiled)

end Internal

/-- Resolve the sole fixed candidate by its exact canonical identity. -/
def resolveCandidate
    (requestedIdentity : String) : Except PromotionBindingError PromotionCandidateBinding :=
  if requestedIdentity == candidateId.value then
    Internal.checkCandidate Internal.candidate
  else
    .error (bindingError .unknownCandidate (DefinitionId.of requestedIdentity)
      "unknown promotion candidate")

end Temporal.Tool.PromotionBinding
