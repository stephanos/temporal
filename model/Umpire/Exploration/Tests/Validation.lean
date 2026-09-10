import Umpire.Exploration
import Umpire.Variations.Tests.Fixtures

/-! Closed policy, Limit, coordinate, and pinned-Artifact validation checks. -/

namespace Umpire.ExplorationTests

open Umpire

def validLimit : Limit := { value := 4, unit := .plans }

def request
    (policy : ExplorationPolicy := .exhaustive)
    (limit : Limit := validLimit)
    (pinned : List Plan := []) :
    ExplorationRequest Umpire.Examples.Switch.LawStatement := {
  space := VariationsTests.checked
  policy
  limit
  pinned
}

def errorKindOf
    (result : Except ExplorationError (CheckedExplorationRequest LawStatement)) :
    Option ExplorationErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

def errorOf
    (result : Except ExplorationError (CheckedExplorationRequest LawStatement)) :
    Option ExplorationError :=
  match result with
  | .ok _ => none
  | .error error => some error

private def compatiblePinned : Plan :=
  Umpire.Examples.Switch.compiledArtifact

private def invalidPinned : Plan := {
  compatiblePinned with artifactChecksum := planChecksumOf "invalid"
}

private def incompatiblePinned : Plan :=
  let planDraft := {
    compatiblePinned.plan with
    artifactChecksum := drivePlanChecksumOf ""
    targetDefinitionId := DefinitionId.of "exploration.target.other"
  }
  let plan := { planDraft with artifactChecksum := planDraft.expectedArtifactChecksum }
  let specDraft := {
    compatiblePinned with
    artifactChecksum := planChecksumOf ""
    plan
  }
  { specDraft with artifactChecksum := specDraft.expectedArtifactChecksum }

private def seededSelection (seed : Nat) : Option (List ModelValue) :=
  let query : CheckedQuery Umpire.Examples.Switch.LawStatement := {
    Umpire.Examples.Switch.exploratoryQuery with
    policy := { strategy := .seeded, seed }
    behaviorFingerprint := behaviorFingerprintOf ("exploration/seeded-query/" ++ toString seed)
  }
  (search query Umpire.Examples.Switch.incrementalKernel).toOption.bind fun run =>
    run.artifact.map (fun spec => spec.plan.modelOutcomes)

/-! The renamed Query strategy and the two Exploration policies expose distinct canonical names. -/
example : [
    SearchStrategy.seeded.name,
    ExplorationPolicy.exhaustive.name,
    (ExplorationPolicy.uncoveredCoordinate .initialState).name
  ] = ["seeded", "exhaustive", "uncovered-coordinate"] := by
  native_decide

/-! Seeded traversal rotates the canonical kernel order deterministically. -/
example :
    (seededSelection 0, seededSelection 1) =
      (some [Umpire.Examples.Switch.appliedOutcome],
        some [Umpire.Examples.Switch.deferredOutcome]) := by
  native_decide

/-! A checked request retains one exact Space and canonicalizes valid pinned Artifacts. -/
example : (checkExplorationRequest <| request
    (.uncoveredCoordinate (.fact 1 1))
    { value := 2, unit := .plans }
    [compatiblePinned]).toOption.map (fun checked =>
      checked.space == VariationsTests.checked &&
        checked.policy == .uncoveredCoordinate (.fact 1 1) &&
        checked.limit == { value := 2, unit := .plans } &&
        checked.pinned.map (fun pinned => pinned.plan.artifactChecksum) ==
          [compatiblePinned.artifactChecksum]) = some true := by
  native_decide

/-! Invalid value/unit Limits and coordinates reject before any selection can begin. -/
example : [
    errorKindOf (checkExplorationRequest <| request .exhaustive
      { value := 0, unit := .plans }),
    errorKindOf (checkExplorationRequest <| request .exhaustive
      { value := 257, unit := .plans }),
    errorKindOf (checkExplorationRequest <| request .exhaustive
      { value := 1, unit := .search }),
    errorKindOf (checkExplorationRequest <| request
      (.uncoveredCoordinate (.selectedAction 0))),
    errorKindOf (checkExplorationRequest <| request
      (.uncoveredCoordinate (.selectedAction 2))),
    errorKindOf (checkExplorationRequest <| request
      (.uncoveredCoordinate (.fact 1 2)))
  ] = [
    some .invalidLimitValue,
    some .invalidLimitValue,
    some .invalidLimitUnit,
    some .unknownCoordinate,
    some .unknownCoordinate,
    some .unknownCoordinate
  ] := by
  native_decide

/-! Invalid, duplicate, and target-incompatible pinned Artifacts reject atomically. -/
example : [
    errorKindOf (checkExplorationRequest <| request .exhaustive validLimit [invalidPinned]),
    errorKindOf (checkExplorationRequest <|
      request .exhaustive validLimit [compatiblePinned, compatiblePinned]),
    errorKindOf (checkExplorationRequest <| request .exhaustive validLimit [incompatiblePinned])
  ] = [
    some .invalidPinnedArtifact,
    some .duplicatePinnedIdentity,
    some .incompatiblePinnedContract
  ] := by
  native_decide

/-! Canonical duplicate errors do not depend on authored pinned-input order. -/
example :
    let first := checkExplorationRequest <|
      request .exhaustive validLimit [compatiblePinned, invalidPinned, compatiblePinned]
    let second := checkExplorationRequest <|
      request .exhaustive validLimit [compatiblePinned, compatiblePinned, invalidPinned]
    (errorOf first).map canonicalExplorationErrorJson =
      (errorOf second).map canonicalExplorationErrorJson := by
  native_decide

end Umpire.ExplorationTests
