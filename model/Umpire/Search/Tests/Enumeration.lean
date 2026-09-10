import Umpire.Search.Tests.Fixtures

/-! Cursor laziness and incremental enumeration instrumentation checks. -/

namespace Umpire.SearchTests

open Umpire

/-! The checked-query adapter preserves the established action, initial, and step traversal. -/
example : ((incrementalKernel? 2).map fun kernel =>
    (kernel.actionLimit,
      kernel.actionAt 0,
      kernel.actionAt 1,
      kernel.initialAt setup 0,
      kernel.stepAt initial requestValue 0)) =
    some (1, some requestValue, none, some initial, some (transition 0)) := by
  rfl

private def admissionErrorKind
    {target : QueryModel LawStatement}
    (result : Except FiniteSearchAdmissionError (SearchView target)) :
    Option FinitePlannerAdmissionErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

/-- The checked-query admission owner derives the same kernel without feature proof transport. -/
example : (SearchView.ofCheckedQuery (target 2).id (orderedQuery 2)).toOption.map
    (fun kernel =>
      (kernel.actionLimit, kernel.actionAt 0, kernel.initialAt setup 0,
        kernel.stepAt initial requestValue 0)) =
    some (1, some requestValue, some initial, some (transition 0)) := by
  native_decide

example : admissionErrorKind
    (SearchView.ofCheckedQuery (id "planner.target.other") (orderedQuery 2)) =
    some .targetMismatch := by
  native_decide

example :
    let query := { orderedQuery 2 with completeness := none }
    admissionErrorKind (SearchView.ofCheckedQuery query.target.id query) =
      some .missingFiniteCompleteness := by
  native_decide

/-!
The cursor instrumentation catches eager full-space production: a two-candidate budget over a
high-branching step pulls the root and one child, retains no pending candidates, and cannot
materialize siblings or upgrade the exhausted prefix into completeness.
-/
#guard
    let planned := run 64 (.counterexample property) .shortest 2 17 false
    planned.toOption.map (fun run =>
    (run.result.outcome.name, run.result.metadata.completeness.established,
      run.instrumentation.generatedCandidates,
      run.instrumentation.retainedPendingCandidates,
      run.instrumentation.peakActiveFrontierDepth,
      run.instrumentation.stepKernelPulls,
      run.result.metadata.explored.transitions)) ==
    some ("limit-reached", false, 2, 0, 2, 1, 1)

/-! The shared traversal preserves admitted candidate order and exhaustive completion without
exposing its cursor representation. Ordinary planning still stops at the first selected trace. -/
#guard
    let query := checkedQuery 2 (.select [property]) .exhaustive 10
    let traversed := traverseBoundedCandidates query (incrementalKernel 2) [] fun traces trace =>
      .ok (.continue (traces ++ [trace]))
    let planned := plan query (incrementalKernel 2)
    (traversed.state.map fun trace => trace.trace.steps.length,
      traversed.termination.name,
      traversed.metadata.completeness.established,
      traversed.metadata.explored.traces,
      planned.toOption.map fun run =>
        (run.result.outcome.name, run.result.metadata.explored.traces)) ==
    ([1, 1, 1], "exhaustive", true, 4, some ("found", 2))

section Replacement

variable (original : QueryModel LawStatement)
  (sameBehavior : original.machine.behaviorTable? =
    some { original.behaviorTable with terminalConditions := [] })

private def replacementWithoutPlanning : QueryModel LawStatement :=
  original.withEquivalentMachine original.machine rfl
    ⟨rfl, rfl, rfl, rfl, rfl⟩ rfl rfl sameBehavior

private theorem replacement_without_planning_has_no_completeness :
    (CheckedQueryModel.ofTarget (replacementWithoutPlanning original sameBehavior)).completeness =
      none := rfl

/-- error: Type mismatch -/
#guard_msgs (error, substring := true) in
#check (original.withEquivalentMachine original.machine rfl
  ⟨rfl, rfl, rfl, rfl, rfl⟩ rfl rfl : QueryModel LawStatement)
private theorem replacement_without_planning_rejects_planner
    (query : CheckedQuery LawStatement) :
    let replacement := replacementWithoutPlanning original sameBehavior
    let replacedQuery := { query with
      target := replacement
      completeness := (CheckedQueryModel.ofTarget replacement).completeness }
    (match SearchView.ofCheckedQuery replacement.id replacedQuery with
    | .ok _ => none
    | .error error => some error.kind) = some .missingFiniteCompleteness := by
  have sameId : (original.id != original.id) = false := by
    change (!(original.id.value == original.id.value)) = false
    simp
  simp [SearchView.ofCheckedQuery, replacementWithoutPlanning,
    CheckedModel.withEquivalentMachine, CheckedQueryModel.ofTarget, sameId]

variable (capability : FinitePlanningCapability original.machine.authoritativeStep)
/-- error: Fields missing: `actionComplete` -/
#guard_msgs (error, substring := true) in
#check ({ actions := capability.actions, actionSound := capability.actionSound } :
  FinitePlanningCapability original.kernel.authoritativeStep)

end Replacement

end Umpire.SearchTests
