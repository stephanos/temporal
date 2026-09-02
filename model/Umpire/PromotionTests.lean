import Umpire.Promotion
import Umpire.Examples.Switch
import Umpire.Promotion.Tests.Fixtures.CompiledSource

/-! Focused checks for the sealed, review-only promotion source boundary. -/

namespace Umpire.PromotionTests

open Umpire
open Umpire.Examples.Switch

private def targetOwnedTrace? : Option BehaviorTrace :=
  match exactActionRun.result.outcome with
  | .found trace _ => some trace
  | _ => none

private theorem targetOwnedTrace?_isSome : targetOwnedTrace?.isSome = true := by
  native_decide

private def targetOwnedCountOneTrace : BehaviorTrace :=
  targetOwnedTrace?.get targetOwnedTrace?_isSome

private def observedCountTwoTrace : BehaviorTrace := {
  targetOwnedCountOneTrace with
  trace := {
    targetOwnedCountOneTrace.trace with
    steps := targetOwnedCountOneTrace.trace.steps.map fun step => {
      step with observations := step.observations ++ step.observations
    }
  }
}

private def baseAnchor : PromotionBaseAnchor := {
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
  expectedTrace := targetOwnedCountOneTrace
  selectionReason := .satisfyingWitness
}

private def sourceSpec : PromotionSourceSpec := {
  sourceDefinitionId := DefinitionId.of "umpire.promotion.source.switch-count-one"
  sourceLocation := {
    path := "Umpire/Promotion/Tests/Fixtures/CompiledSource.lean"
    line := 1
    column := 1
    provenance := "lean-model"
  }
  promotedBehaviorDefinitionId :=
    DefinitionId.of "umpire.behavior.regression.switch-count-one"
  promotedQueryDefinitionId := DefinitionId.of "umpire.query.regression.switch-count-one"
}

private def sourceExpectation : PromotionSourceExpectation :=
  let bytes := include_str "Promotion/Tests/Fixtures/CompiledSource.lean"
  {
    bytes
    sha256 := "sha256:66230be5a4b04ea5fd8b552212c2cf3c0eb7692bc23bacbc032274e1c3a133e9"
  }

private def errorKindOf
    (result : Except PromotionError CompiledPromotionSource) : Option PromotionErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

private def compileWith
    (anchor : PromotionBaseAnchor := baseAnchor)
    (spec : PromotionSourceSpec := sourceSpec)
    (expectation : PromotionSourceExpectation := sourceExpectation) :
    Except PromotionError CompiledPromotionSource :=
  compilePromotionSource exactActionQuery incrementalKernel anchor spec expectation

/-!
The target-owned trace contains one delivered observation. A runtime-observed duplicate may report
two, but that observation cannot replace the expected trace retained by the unchanged Query plan.
-/
example :
    (targetOwnedCountOneTrace.trace.steps.flatMap (fun step => step.observations)).length = 1 ∧
      (observedCountTwoTrace.trace.steps.flatMap (fun step => step.observations)).length = 2 ∧
      errorKindOf (compileWith
        (anchor := { baseAnchor with expectedTrace := observedCountTwoTrace })) =
        some .traceDrift := by
  native_decide

/-! The checked fixture is the exact byte sequence exposed by the sealed source. -/
example :
    compileWith.toOption.map (fun compiled =>
        compiled.sourceBytes == sourceExpectation.bytes &&
          compiled.sourceSha256 == sourceExpectation.sha256 &&
          compiled.baseExperimentSpecChecksum == compiledArtifact.artifactChecksum) = some true := by
  native_decide

private def otherId : DefinitionId :=
  DefinitionId.of "umpire.promotion.changed"

private def otherFingerprint : BehaviorFingerprint :=
  behaviorFingerprintOf "umpire/promotion/changed"

private def changedExperimentSpec : ExperimentSpec := {
  compiledArtifact with artifactChecksum := experimentSpecChecksumOf "changed"
}

private def nonFoundBehavior : CheckedBehavior := {
  exactActionBehavior with
  spaceStatus := .unsatisfiable
  canonicalMetadata := "switch-promotion-unsatisfiable/v1"
  behaviorFingerprint := behaviorFingerprintOf "switch-promotion-unsatisfiable/v1"
}

private def nonFoundQuery : CheckedQuery LawStatement := {
  exactActionQuery with
  id := DefinitionId.of "switch.query.promotion-unsatisfiable"
  behavior := nonFoundBehavior
  canonicalMetadata := "switch-promotion-unsatisfiable-query/v1"
  behaviorFingerprint := behaviorFingerprintOf "switch-promotion-unsatisfiable-query/v1"
}

private def nonFoundRun : PlannerRun :=
  plan nonFoundQuery incrementalKernel

private def nonFoundAnchor : PromotionBaseAnchor := {
  queryDefinitionId := nonFoundQuery.id
  queryBehaviorFingerprint := nonFoundQuery.behaviorFingerprint
  queryCanonicalMetadata := nonFoundQuery.canonicalMetadata
  behaviorDefinitionId := nonFoundQuery.behavior.id
  behaviorFingerprint := nonFoundQuery.behavior.behaviorFingerprint
  targetDefinitionId := nonFoundQuery.target.id
  targetBehaviorFingerprint := nonFoundQuery.target.behaviorFingerprint
  kernelDefinitionId := nonFoundQuery.target.kernel.metadata.id
  kernelBehaviorFingerprint := nonFoundQuery.target.behaviorFingerprint
  plannerRun := nonFoundRun
  experimentSpec := compiledArtifact
  expectedTrace := targetOwnedCountOneTrace
  selectionReason := .satisfyingWitness
}

/-! Every meaning-bearing base or source mutation fails before a partial source is returned. -/
example :
    [
      errorKindOf (compileWith (anchor := {
        baseAnchor with queryDefinitionId := otherId
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with queryBehaviorFingerprint := otherFingerprint
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with queryCanonicalMetadata := "changed"
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with behaviorDefinitionId := otherId
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with behaviorFingerprint := otherFingerprint
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with targetDefinitionId := otherId
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with targetBehaviorFingerprint := otherFingerprint
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with kernelDefinitionId := otherId
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with kernelBehaviorFingerprint := otherFingerprint
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with plannerRun := { exactActionRun with instrumentation := {} }
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with selectionReason := .behaviorSelection
      })),
      errorKindOf (compileWith (anchor := {
        baseAnchor with experimentSpec := changedExperimentSpec
      })),
      errorKindOf (compileWith (spec := {
        sourceSpec with sourceLocation := { sourceSpec.sourceLocation with path := "" }
      })),
      errorKindOf (compileWith (spec := {
        sourceSpec with promotedBehaviorDefinitionId := exactActionQuery.behavior.id
      })),
      errorKindOf (compileWith (spec := {
        sourceSpec with promotedQueryDefinitionId := exactActionQuery.id
      })),
      errorKindOf (compileWith (expectation := {
        sourceExpectation with bytes := sourceExpectation.bytes ++ " "
      })),
      errorKindOf (compileWith (expectation := {
        sourceExpectation with sha256 := promotionSourceSha256 "changed"
      }))
    ] = [
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .baseIdentityDrift,
      some .plannerRunDrift,
      some .reasonDrift,
      some .experimentSpecDrift,
      some .invalidSourceSpec,
      some .reusedIdentity,
      some .reusedIdentity,
      some .sourceBytesDrift,
      some .sourceDigestDrift
    ] := by
  native_decide

/-! A non-found base result cannot be reclassified as a promotable source. -/
example :
    errorKindOf (compilePromotionSource nonFoundQuery incrementalKernel nonFoundAnchor
      sourceSpec sourceExpectation) = some .nonFoundResult := by
  native_decide

private def fixturePromotedQueryResult :=
  Umpire.Promotion.Tests.Fixtures.CompiledSource.promotedQueryResult exactActionQuery

private theorem fixturePromotedQueryResult_isSome :
    fixturePromotedQueryResult.toOption.isSome = true := by
  native_decide

private def fixturePromotedQuery :=
  fixturePromotedQueryResult.toOption.get fixturePromotedQueryResult_isSome

/-! The clean-elaborated source binds a typed base Query to the exact trace and fresh identities. -/
example :
    fixturePromotedQuery.id = sourceSpec.promotedQueryDefinitionId ∧
      fixturePromotedQuery.behavior.id = sourceSpec.promotedBehaviorDefinitionId ∧
      fixturePromotedQuery.behavior.traceExactly = some targetOwnedCountOneTrace := by
  native_decide

/--
error: `baseQueryReference` is not a field of structure `PromotionSourceSpec`
---
info: let __src := sourceSpec;
__src : PromotionSourceSpec
-/
#guard_msgs in
#check ({ sourceSpec with
  baseQueryReference := "Umpire.DoesNotExist.query"
} : PromotionSourceSpec)

/--
error: `imports` is not a field of structure `PromotionSourceSpec`
---
info: let __src := sourceSpec;
__src : PromotionSourceSpec
-/
#guard_msgs in
#check ({ sourceSpec with
  imports := ["Umpire.DoesNotExist"]
} : PromotionSourceSpec)

/--
error: `namespaceName` is not a field of structure `PromotionSourceSpec`
---
info: let __src := sourceSpec;
__src : PromotionSourceSpec
-/
#guard_msgs in
#check ({ sourceSpec with
  namespaceName := "namespace"
} : PromotionSourceSpec)

/-!
error: Unknown constant `Umpire.CompiledPromotionSource.mk`
-/
#guard_msgs (error, substring := true) in
#check Umpire.CompiledPromotionSource.mk

/--
error: invalid {...} notation, constructor for `CompiledPromotionSource` is marked as private
-/
#guard_msgs in
def replaceCompiledSourceBytes
    (compiled : CompiledPromotionSource) : CompiledPromotionSource := {
  compiled with sourceBytes := ""
}

end Umpire.PromotionTests
