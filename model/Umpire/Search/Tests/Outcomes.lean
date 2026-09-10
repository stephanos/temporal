import Umpire.Search.Tests.Fixtures

/-! Query outcomes, invalid input, completion, exhaustion, and unsatisfiable behavior checks. -/

namespace Umpire.SearchTests

open Umpire

def outcomeName
    (form : Query.Form)
    (strategy : SearchStrategy)
    (withCompleteness : Bool := true) : Option String :=
  (run 2 form strategy 10 17 withCompleteness).toOption.map fun run =>
    run.result.outcome.name

/-! Each Query form preserves its exact result semantics over the same deterministic kernel. -/
example : [
    outcomeName (.verify property) .exhaustive,
    outcomeName (.find property) .shortest false,
    outcomeName (.findViolation property) .exhaustive,
    outcomeName (.pick [property]) .breadthFirst false
  ] = [
    some "verified-within-limits",
    some "found",
    some "none-found",
    some "found"
  ] := by
  native_decide

def invalidError : QueryError := {
  kind := .invalidLimit
  definitionId := id "planner.query.invalid"
  sourcePath := source.path
  offendingValue := "search=0"
  relatedDefinitionIds := []
}

/-! An invalid checked-input outcome remains distinct from every search termination outcome. -/
example : (PlanningOutcome.invalid invalidError).name = "invalid" := by
  native_decide

/-! Complete absence and exhausted effort remain distinct while retaining counts and limits. -/
example :
    let complete := run 0 (.findViolation property) .exhaustive
    let exhausted := run 64 (.findViolation property) .shortest 1 17 false
    (complete.toOption.map fun run =>
      (run.result.outcome.name, run.result.metadata.completeness.established,
        run.result.metadata.completeness.limits),
      exhausted.toOption.map fun run =>
        (run.result.outcome.name, run.result.metadata.completeness.established)) =
      (some ("none-found", true, limits),
        some ("limit-reached", false)) := by
  native_decide

def targetRelativeEmptyBehavior : CheckedScenario := {
  behavior with
  actionsExactly := some [request, request]
  behaviorFingerprint := behaviorFingerprintOf "behavior/target-relative-empty-v1"
}

/-! Exhaustive completion with no Behavior-admitted target trace is unsatisfiable, not proof. -/
example :
    let planned := run 0 (.verify property) .exhaustive 10 17 true targetRelativeEmptyBehavior
    planned.toOption.map (fun run =>
      (run.result.outcome.name, run.result.isVerified,
        run.result.metadata.completeness.established)) =
      some ("unsatisfiable", false, false) := by
  native_decide

def staticallyUnsatisfiableBehavior : CheckedScenario := {
  behavior with
  spaceStatus := .unsatisfiable
  behaviorFingerprint := behaviorFingerprintOf "behavior/statically-unsatisfiable-v1"
}

/-! Empty behavior is unsatisfiable, while an incomplete search is budget exhaustion; neither
can be observed as verification. -/
example :
    let empty := run 0 (.verify property) .exhaustive 10 17 true staticallyUnsatisfiableBehavior
    let exhausted := run 64 (.findViolation property) .shortest 1 17 false
    (empty.toOption.map fun run => (run.result.outcome.name, run.result.isVerified),
      exhausted.toOption.map fun run => (run.result.outcome.name, run.result.isVerified)) =
      (some ("unsatisfiable", false), some ("limit-reached", false)) := by
  native_decide

end Umpire.SearchTests
