import Umpire.Exploration.Guided
import Umpire.Space.Tests.Fixtures

/-! Deterministic guidance toward one uncovered Model Coordinate. -/

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

private def authoredRequest
    (coordinate : ModelCoordinate)
    (value : Nat := 1) : ExplorationRequest Umpire.Examples.Switch.LawStatement := {
  space := SpaceTests.checked
  policy := .uncoveredCoordinate coordinate
  limit := { value, unit := .experimentSpecs }
}

private def checkedRequestResult :=
  checkExplorationRequest (authoredRequest (.observation 1 1))

private def checkedRequest := checkedRequestResult.toOption.get (by native_decide)

private theorem checkedRequestResultEq : checkedRequestResult = .ok checkedRequest :=
  except_eq_ok_get checkedRequestResult (by native_decide)

private theorem checkedRequestTargetEq :
    checkedRequest.space.baseQuery.target = Umpire.Examples.Switch.target := by
  calc
    checkedRequest.space.baseQuery.target = SpaceTests.checked.baseQuery.target :=
      congrArg (fun space => space.baseQuery.target)
        (checkExplorationRequest_space checkedRequestResultEq)
    _ = Umpire.Examples.Switch.target := checkedSpaceTargetEq

private def candidateKernel : IncrementalPlannerKernel checkedRequest.space.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedRequestTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def candidateUniverse :=
  (buildCandidateUniverse checkedRequest candidateKernel).toOption.get (by native_decide)

private structure CandidateProjection where
  identity : String
  coordinates : List ModelCoordinate
  deriving BEq, DecidableEq, Repr

private def projection
    (identity : String)
    (coordinates : List ModelCoordinate) : CandidateProjection := {
  identity
  coordinates
}

private def requestedCoordinate : ModelCoordinate := .observation 1 1

private def projectedCandidates : List CandidateProjection := [
  projection "a-nonmatching" [.initialState],
  projection "c-matching" [.initialState, requestedCoordinate],
  projection "b-matching" [.initialState, requestedCoordinate]
]

/-!
The requested coordinate moves matching candidates ahead of the canonical first candidate, while
semantic identity still orders both the matching and nonmatching partitions.
-/
example :
    (GuidedSelection.Internal.prioritize requestedCoordinate
      CandidateProjection.identity CandidateProjection.coordinates projectedCandidates).map
        CandidateProjection.identity =
      ["b-matching", "c-matching", "a-nonmatching"] := by
  native_decide

/-! The explicit ExperimentSpec Limit applies after coordinate priority. -/
example :
    (GuidedSelection.Internal.prioritize requestedCoordinate
      CandidateProjection.identity CandidateProjection.coordinates projectedCandidates |>.take
        1).map
        CandidateProjection.identity = ["b-matching"] := by
  native_decide

/-! An absent match stays uncovered; the closed outcome has no reachability claim. -/
example :
    GuidedSelection.Internal.outcome requestedCoordinate CandidateProjection.coordinates [
      projection "a-nonmatching" [.initialState],
      projection "b-nonmatching" [.selectedAction 1]
    ] = .coordinateUncovered ∧
      [GuidedSelectionOutcome.coordinateSelected.name,
        GuidedSelectionOutcome.coordinateUncovered.name] =
        ["coordinate-selected", "coordinate-uncovered"] := by
  native_decide

private def selection := selectUncoveredCoordinate checkedRequest candidateUniverse

private def exhaustiveRequest :=
  (checkExplorationRequest {
    authoredRequest requestedCoordinate with policy := .exhaustive
  }).toOption.get (by native_decide)

/-! A present requested coordinate yields the selected outcome and retains that exact coordinate. -/
example : selection.map (fun result =>
    result.coordinate == requestedCoordinate &&
      result.candidates.length == 1 &&
      result.identities ==
        (candidateUniverse.candidates.take 1 |>.map ExplorationCandidate.identity) &&
      result.outcome.name == "coordinate-selected") = some true := by
  native_decide

/-! The guided selector accepts no exhaustive-policy request. -/
example : selectUncoveredCoordinate exhaustiveRequest candidateUniverse = none := by
  native_decide

/-! An out-of-vocabulary coordinate never reaches selection. -/
example :
    (checkExplorationRequest (authoredRequest (.selectedAction 2))).toOption.isNone = true := by
  native_decide

/-!
Repeated selection cannot mutate its checked request or finite universe, and the selector exposes no
runtime-observation or adaptive-state input.
-/
example :
    let requestBefore := checkedRequest
    let universeBefore := candidateUniverse
    selectUncoveredCoordinate checkedRequest candidateUniverse == selection &&
      checkedRequest.space == requestBefore.space &&
      checkedRequest.policy == requestBefore.policy &&
      checkedRequest.limit == requestBefore.limit &&
      checkedRequest.pinned == requestBefore.pinned &&
      candidateUniverse == universeBefore := by
  native_decide

private def selectorType :
    CheckedExplorationRequest Umpire.Examples.Switch.LawStatement →
      CandidateUniverse → Option GuidedSelection :=
  selectUncoveredCoordinate

end Umpire.ExplorationTests
