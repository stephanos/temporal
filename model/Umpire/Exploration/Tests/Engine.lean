import Umpire.Exploration
import Umpire.Space.Tests.Fixtures

/-! Atomic bounded Exploration through the public pure engine. -/

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

def engineKernel : IncrementalPlannerKernel SpaceTests.checked.baseQuery.target :=
  Eq.mpr (congrArg IncrementalPlannerKernel checkedSpaceTargetEq)
    Umpire.Examples.Switch.incrementalKernel

def engineRequest
    (policy : ExplorationPolicy)
    (value : Nat)
    (pinned : List ExperimentSpec := []) :
    ExplorationRequest Umpire.Examples.Switch.LawStatement := {
  space := SpaceTests.checked
  policy
  limit := { value, unit := .experimentSpecs }
  pinned
}

def engineRun
    (policy : ExplorationPolicy)
    (value : Nat)
    (pinned : List ExperimentSpec := []) :=
  explore (engineRequest policy value pinned) engineKernel

/-!
The public engine composes both retained policies while keeping their coordinate and completion
outcomes distinct.
-/
example :
    let limited := (engineRun .exhaustive 3).toOption
    let complete := (engineRun .exhaustive 4).toOption
    let guided := (engineRun (.uncoveredCoordinate (.observation 1 1)) 1).toOption
    limited.any (fun result =>
        result.pinned.isEmpty && result.exploratory.length == 3 &&
          result.coordinateOutcome.isNone && result.completion == .limitReached) &&
      complete.any (fun result =>
        result.exploratory.length == 4 && result.completion == .exhausted) &&
      guided.any (fun result =>
        result.exploratory.length == 1 &&
          result.coordinateOutcome == some .coordinateSelected &&
          result.completion == .limitReached) = true := by
  native_decide

/-! The selected identity projection preserves the exact pinned-then-exploratory partition order. -/
example : (engineRun .exhaustive 4).toOption.map (fun result =>
    result.selectedIdentities ==
      result.pinned.map (fun pinned => pinned.experimentSpec.artifactChecksum) ++
        result.exploratory.map ExplorationCandidate.identity) = some true := by
  native_decide

/-! Input failure exposes no partial result and takes precedence over candidate compilation. -/
example :
    let rejected := engineRun .exhaustive 0
    (rejected.toOption,
      match rejected with
      | .error error => some error.kind
      | .ok _ => none) = (none, some .invalidLimitValue) := by
  native_decide

end Umpire.ExplorationTests
