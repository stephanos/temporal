import Umpire.Variations.Compiler
import Umpire.Variations.Tests.Fixtures

/-! Equivalent authoring order produces identical lowered point and Artifact bytes. -/

namespace Umpire.VariationsTests

open Umpire

private theorem except_eq_ok_get
    (result : Except ε α)
    (isSome : result.toOption.isSome = true) :
    result = .ok (result.toOption.get isSome) := by
  cases result with
  | error error => cases isSome
  | ok value => rfl

private theorem originalResultEq : checkedResult = .ok checked :=
  except_eq_ok_get checkedResult (by native_decide)

private theorem originalTargetEq :
    checked.baseQuery.target = Umpire.Examples.Switch.target :=
  congrArg (fun query => query.target) <|
    checkVariationSpace_baseQuery originalResultEq

private def originalKernel : SearchView checked.baseQuery.target :=
  Eq.mpr (congrArg SearchView originalTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def reorderedDeclaration : VariationSpace := {
  declaration with
  axes := [
    { faultAxis with choices := faultAxis.choices.reverse },
    { stateAxis with choices := stateAxis.choices.reverse }
  ]
  faults := declaration.faults.reverse
  coverageGoals := declaration.coverageGoals.reverse
}

private def reorderedResult := checkVariationSpace context reorderedDeclaration

private def reordered := reorderedResult.toOption.get (by native_decide)

private theorem reorderedResultEq : reorderedResult = .ok reordered :=
  except_eq_ok_get reorderedResult (by native_decide)

private theorem reorderedTargetEq :
    reordered.baseQuery.target = Umpire.Examples.Switch.target :=
  congrArg (fun query => query.target) <|
    checkVariationSpace_baseQuery reorderedResultEq

private def reorderedKernel : SearchView reordered.baseQuery.target :=
  Eq.mpr (congrArg SearchView reorderedTargetEq)
    Umpire.Examples.Switch.incrementalKernel

private def compiledProjection
    (result : Except SpaceCompilationError (List Plan)) :
    Option (List (DefinitionId × String)) :=
  result.toOption.map fun specs => specs.map fun spec =>
    (spec.plan.queryDefinitionId, canonicalPlanBytes spec)

/-!
Reordering axes, choices, faults, and goals preserves canonical point order and complete bytes.
-/
example :
    compiledProjection (compileBatch checked originalKernel) =
      compiledProjection (compileBatch reordered reorderedKernel) := by
  native_decide

end Umpire.VariationsTests
