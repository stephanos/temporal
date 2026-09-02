import Umpire.Exploration.Selection
import Umpire.Space.Tests.Fixtures

/-! Deterministic bounded exhaustive selection over one canonical finite candidate universe. -/

namespace Umpire.ExplorationTests

open Umpire

private theorem except_eq_ok_get
    (result : Except ε α)
    (isSome : result.toOption.isSome = true) :
    result = .ok (result.toOption.get isSome) := by
  cases result with
  | error _ => cases isSome
  | ok _ => rfl

private theorem checkedSpaceResultEq :
    SpaceTests.checkedResult = .ok SpaceTests.checked :=
  except_eq_ok_get SpaceTests.checkedResult (by native_decide)

private theorem checkedSpaceTargetEq :
    SpaceTests.checked.baseQuery.target = Umpire.Examples.Switch.target := by
  exact congrArg (fun query => query.target)
    (checkExperimentSpace_baseQuery checkedSpaceResultEq)

private def kernel : IncrementalPlannerKernel SpaceTests.checked.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedSpaceTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def authoredRequest (value : Nat) :
    ExplorationRequest Umpire.Examples.Switch.LawStatement := {
  space := SpaceTests.checked
  policy := .exhaustive
  limit := { value, unit := .experimentSpecs }
}

private def checkedRequestResult := checkExplorationRequest (authoredRequest 4)

private def checkedRequest :
    CheckedExplorationRequest Umpire.Examples.Switch.LawStatement :=
  checkedRequestResult.toOption.get (by native_decide)

private theorem checkedRequestResultEq : checkedRequestResult = .ok checkedRequest :=
  except_eq_ok_get checkedRequestResult (by native_decide)

private theorem checkedRequestTargetEq :
    checkedRequest.space.baseQuery.target = Umpire.Examples.Switch.target := by
  calc
    checkedRequest.space.baseQuery.target = SpaceTests.checked.baseQuery.target :=
      congrArg (fun space => space.baseQuery.target)
        (checkExplorationRequest_space checkedRequestResultEq)
    _ = Umpire.Examples.Switch.target := checkedSpaceTargetEq

private def candidateKernel :
    IncrementalPlannerKernel checkedRequest.space.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedRequestTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def universeResult := buildCandidateUniverse checkedRequest candidateKernel

private def requestResult (value : Nat) := checkExplorationRequest (authoredRequest value)

private def checkedRequest3 := (requestResult 3).toOption.get (by native_decide)
private def checkedRequest5 := (requestResult 5).toOption.get (by native_decide)

private def selectionResult
    (request : CheckedExplorationRequest Umpire.Examples.Switch.LawStatement) :
    Option ExhaustiveSelection := do
  let candidateUniverse ← universeResult.toOption
  selectExhaustive request candidateUniverse

/-! Canonical finite candidates are returned in semantic-identity order within the Limit. -/
example : (selectionResult checkedRequest3).map (fun selection =>
    let identities := selection.identities
    identities == identities.mergeSort
      (fun left right => decide (left.render ≤ right.render)) &&
      selection.candidates.length == 3) = some true := by
  native_decide

/-!
Zero rejects before selection, N - 1 is inconclusive, and N or N + 1 exhausts the four-candidate
fixture without confusing the planner's candidate-evaluation Limit with this ExperimentSpec Limit.
-/
example :
    let zero := checkExplorationRequest (authoredRequest 0)
    let below := selectionResult checkedRequest3
    let exact := selectionResult checkedRequest
    let above := selectionResult checkedRequest5
    zero.toOption.isNone &&
      below.any (fun result =>
        result.candidates.length == 3 && result.outcome.name == "limit-reached") &&
      exact.any (fun result =>
        result.candidates.length == 4 && result.outcome.name == "exhausted") &&
      above.any (fun result =>
        result.candidates.length == 4 && result.outcome.name == "exhausted") &&
      checkedRequest.limit.unit == .experimentSpecs &&
      checkedRequest.space.baseQuery.limits.search.unit == .candidateEvaluations = true := by
  native_decide

private def firstCandidateResult : Option ExplorationCandidate :=
  universeResult.toOption.bind fun candidateUniverse => candidateUniverse.candidates.head?

private def firstCandidate := firstCandidateResult.get (by native_decide)

private def pinnedRequestResult (value : Nat) := checkExplorationRequest {
  authoredRequest value with pinned := [firstCandidate.experimentSpec]
}

private def pinnedRequest2 := (pinnedRequestResult 2).toOption.get (by native_decide)
private def pinnedRequest3 := (pinnedRequestResult 3).toOption.get (by native_decide)
private def pinnedRequest4 := (pinnedRequestResult 4).toOption.get (by native_decide)

/-!
Pinned overlap is removed before the exhaustive Limit: N - 1 remains inconclusive, while N and
N + 1 exhaust the three non-pinned candidates without selecting the pinned identity again.
-/
example :
    let below := selectionResult pinnedRequest2
    let exact := selectionResult pinnedRequest3
    let above := selectionResult pinnedRequest4
    below.any (fun result =>
        result.candidates.length == 2 && result.outcome == .limitReached) &&
      exact.any (fun result =>
        result.candidates.length == 3 &&
          !result.identities.contains firstCandidate.identity && result.outcome == .exhausted) &&
      above.any (fun result =>
        result.candidates.length == 3 && result.outcome == .exhausted) = true := by
  native_decide

private def differentDeclaration : ExperimentSpaceDeclaration := {
  SpaceTests.declaration with
  coverageGoals := [
    { SpaceTests.stateGoal with minimum := 1 },
    SpaceTests.delayGoal,
    SpaceTests.semanticGoal,
    SpaceTests.propertyGoal
  ]
}

private def differentSpaceResult :=
  checkExperimentSpace SpaceTests.context differentDeclaration

private def differentSpace := differentSpaceResult.toOption.get (by native_decide)

private def differentRequestResult := checkExplorationRequest {
  space := differentSpace
  policy := ExplorationPolicy.exhaustive
  limit := { value := 3, unit := .experimentSpecs }
}

private def differentRequest := differentRequestResult.toOption.get (by native_decide)

/-! A same-ID but semantically different checked Space cannot select from this universe. -/
example :
    SpaceTests.checked.id == differentSpace.id &&
      SpaceTests.checked.behaviorFingerprint != differentSpace.behaviorFingerprint &&
      (selectionResult differentRequest).isNone = true := by
  native_decide

private def reorderedDeclaration : ExperimentSpaceDeclaration := {
  SpaceTests.declaration with
  axes := (SpaceTests.declaration.axes.map fun axis => {
    axis with choices := axis.choices.reverse
  }).reverse
  faults := SpaceTests.declaration.faults.reverse
  coverageGoals := SpaceTests.declaration.coverageGoals.reverse
}

private def reorderedSpaceResult :=
  checkExperimentSpace SpaceTests.context reorderedDeclaration

private def reorderedSpace := reorderedSpaceResult.toOption.get (by native_decide)

private theorem reorderedSpaceResultEq : reorderedSpaceResult = .ok reorderedSpace :=
  except_eq_ok_get reorderedSpaceResult (by native_decide)

private theorem reorderedSpaceTargetEq :
    reorderedSpace.baseQuery.target = Umpire.Examples.Switch.target := by
  exact congrArg (fun query => query.target)
    (checkExperimentSpace_baseQuery reorderedSpaceResultEq)

private def reorderedRequest : ExplorationRequest Umpire.Examples.Switch.LawStatement := {
  space := reorderedSpace
  policy := .exhaustive
  limit := { value := 3, unit := .experimentSpecs }
}

private def checkedReorderedResult := checkExplorationRequest reorderedRequest

private def checkedReordered := checkedReorderedResult.toOption.get (by native_decide)

private theorem checkedReorderedResultEq : checkedReorderedResult = .ok checkedReordered :=
  except_eq_ok_get checkedReorderedResult (by native_decide)

private theorem checkedReorderedTargetEq :
    checkedReordered.space.baseQuery.target = Umpire.Examples.Switch.target := by
  calc
    checkedReordered.space.baseQuery.target = reorderedSpace.baseQuery.target :=
      congrArg (fun space => space.baseQuery.target)
        (checkExplorationRequest_space checkedReorderedResultEq)
    _ = Umpire.Examples.Switch.target := reorderedSpaceTargetEq

private def reorderedKernel :
    IncrementalPlannerKernel checkedReordered.space.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedReorderedTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def reorderedSelection : Option ExhaustiveSelection := do
  let candidateUniverse ← (buildCandidateUniverse checkedReordered reorderedKernel).toOption
  selectExhaustive checkedReordered candidateUniverse

private def canonicalSelectionProjection (selection : ExhaustiveSelection) :=
  (selection.candidates.map fun candidate =>
      (candidate.identity.render, candidate.canonicalBytes),
    selection.outcome.name)

/-! Reordering every authored list preserves selected identities, bytes, and the Limit outcome. -/
example :
    (selectionResult checkedRequest3).map canonicalSelectionProjection =
      reorderedSelection.map canonicalSelectionProjection := by
  native_decide

end Umpire.ExplorationTests
