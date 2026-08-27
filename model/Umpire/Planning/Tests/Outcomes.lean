import Umpire.Planning.Tests.Fixtures

/-! Query outcomes, invalid input, completion, exhaustion, and unsatisfiable behavior checks. -/

namespace Umpire.PlanningTests

open Umpire

def outcomeName
    (form : QueryForm)
    (strategy : SearchStrategy)
    (withCompleteness : Bool := true) : String :=
  (run 2 form strategy 10 17 withCompleteness).result.outcome.name

/-! Each Query form preserves its exact result semantics over the same deterministic kernel. -/
example : [
    outcomeName (.verify property) .exhaustive,
    outcomeName (.witness property) .shortest false,
    outcomeName (.counterexample property) .exhaustive,
    outcomeName (.select [property]) .breadthFirst false
  ] = [
    "verified-within-limits",
    "found",
    "no-such-trace-within-complete-limits",
    "found"
  ] := by
  native_decide

def invalidError : QueryError := {
  kind := .invalidLimit
  definitionId := id "planner.query.invalid"
  sourcePath := source.path
  offendingValue := "search.candidateEvaluations=0"
  relatedDefinitionIds := []
}

/-! An invalid checked-input outcome remains distinct from every search termination outcome. -/
example : (PlanningOutcome.invalid invalidError).name = "invalid" := by
  native_decide

/-! Complete absence and exhausted effort remain distinct while retaining counts and limits. -/
example :
    let complete := run 0 (.counterexample property) .exhaustive
    let exhausted := run 64 (.counterexample property) .shortest 1 17 false
    (complete.result.outcome.name, complete.result.metadata.completeness.established,
      complete.result.metadata.completeness.limits,
      exhausted.result.outcome.name, exhausted.result.metadata.completeness.established) =
      ("no-such-trace-within-complete-limits", true, limits,
        "limit-reached", false) := by
  native_decide

def targetRelativeEmptyBehavior : CheckedBehavior := {
  behavior with
  actionsExactly := some [request, request]
  behaviorFingerprint := behaviorFingerprintOf "behavior/target-relative-empty-v1"
}

/-! Exhaustive completion with no Behavior-admitted target trace is unsatisfiable, not proof. -/
example :
    let planned := run 0 (.verify property) .exhaustive 10 17 true targetRelativeEmptyBehavior
    (planned.result.outcome.name, planned.result.isVerified,
      planned.result.metadata.completeness.established) =
      ("unsatisfiable", false, false) := by
  native_decide

def staticallyUnsatisfiableBehavior : CheckedBehavior := {
  behavior with
  spaceStatus := .unsatisfiable
  behaviorFingerprint := behaviorFingerprintOf "behavior/statically-unsatisfiable-v1"
}

/-! Empty behavior is unsatisfiable, while an incomplete search is budget exhaustion; neither
can be observed as verification. -/
example :
    let empty := run 0 (.verify property) .exhaustive 10 17 true staticallyUnsatisfiableBehavior
    let exhausted := run 64 (.counterexample property) .shortest 1 17 false
    (empty.result.outcome.name, empty.result.isVerified,
      exhausted.result.outcome.name, exhausted.result.isVerified) =
      ("unsatisfiable", false, "limit-reached", false) := by
  native_decide

end Umpire.PlanningTests
