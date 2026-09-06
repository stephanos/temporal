import Umpire.Planning.Tests.Fixtures

/-! Cursor laziness and incremental enumeration instrumentation checks. -/

namespace Umpire.PlanningTests

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
    {target : QueryTarget LawStatement}
    (result : Except FinitePlannerAdmissionError (IncrementalPlannerKernel target)) :
    Option FinitePlannerAdmissionErrorKind :=
  match result with
  | .ok _ => none
  | .error error => some error.kind

/-- The checked-query admission owner derives the same kernel without feature proof transport. -/
example : (IncrementalPlannerKernel.ofCheckedQuery (target 2).id (orderedQuery 2)).toOption.map
    (fun kernel =>
      (kernel.actionLimit, kernel.actionAt 0, kernel.initialAt setup 0,
        kernel.stepAt initial requestValue 0)) =
    some (1, some requestValue, some initial, some (transition 0)) := by
  native_decide

example : admissionErrorKind
    (IncrementalPlannerKernel.ofCheckedQuery (id "planner.target.other") (orderedQuery 2)) =
    some .targetMismatch := by
  native_decide

example :
    let query := { orderedQuery 2 with completeness := none }
    admissionErrorKind (IncrementalPlannerKernel.ofCheckedQuery query.target.id query) =
      some .missingFiniteCompleteness := by
  native_decide

/-!
The cursor instrumentation catches eager full-space production: a two-candidate budget over a
high-branching step pulls the root and one child, retains no pending candidates, and cannot
materialize siblings or upgrade the exhausted prefix into completeness.
-/
example :
    let planned := run 64 (.counterexample property) .shortest 2 17 false
    (planned.result.outcome.name, planned.result.metadata.completeness.established,
      planned.instrumentation.generatedCandidates,
      planned.instrumentation.retainedPendingCandidates,
      planned.instrumentation.peakActiveFrontierDepth,
      planned.instrumentation.stepKernelPulls,
      planned.result.metadata.explored.transitions) =
    ("limit-reached", false, 2, 0, 2, 1, 1) := by
  native_decide

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
      planned.result.outcome.name,
      planned.result.metadata.explored.traces) ==
    ([1, 1, 1], "exhaustive", true, 4, "found", 2)

end Umpire.PlanningTests
