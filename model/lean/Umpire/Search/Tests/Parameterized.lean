import Umpire.Model.Tests.Parameterized
import Umpire.Search.Tests.Fixtures

/-! Parameterized finite completeness, bounded search, and exact checked Query replay. -/
namespace Umpire.ParameterizedPlanningTests
open Umpire Operation Value ModelTests.Parameterized

private def authoredQuery (t : QueryModel (fun _ => True)) (budget : Nat)
    (exact : Option Scenario.Trace := none) : Query := {
  id := .of "example.query.call", source, target := t.id
  form := .verify SearchTests.property
  limits := Limits.bounded 1 1 budget
  policy := .exhaustive
  behavior := { SearchTests.behavior with
    roles := [], allowedActions := [actionId],
    requiredOccurrences := [], actionsExactly := none, traceExactly := exact }
}
private def run (budget : Nat) := do
  let t ← target .samplesOnly
  let q ← Query.check (.ofTarget t) (authoredQuery t budget) |>.mapError (fun _ => "query")
  let k ← SearchView.ofCheckedQuery q.target.id q |>.mapError (fun _ => "planner")
  search q k |>.mapError (fun _ => "plan")
#guard (run 1).toOption.map (fun r =>
  (r.result.outcome.name, r.result.metadata.completeness.established)) == some ("limit-reached", false)
#guard (run 100).toOption.map (fun r =>
  (r.result.outcome.name, r.result.metadata.completeness.established)) == some ("verified-within-limits", true)

#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  let a ← d.actions.head?
  let t ← (target .samplesOnly).toOption
  let step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue := {
    selectedAction := a.modelValue, outcome := value "example.outcome.call" "accepted",
    state := value "example.state.phase" "done", facts := [] }
  let trace : Scenario.Trace := {
    setup := [], trace := { initialState := value "example.state.phase" "idle", steps := [step] } }
  let bad := { trace with trace := { trace.trace with steps := [
    { step with state := value "example.state.phase" "idle" }] } }
  pure ((Query.check (.ofTarget t) (authoredQuery t 100 (some trace))).toOption.isSome,
    (Query.check (.ofTarget t) (authoredQuery t 100 (some bad))).toOption.isNone)) == some (true, true)


end Umpire.ParameterizedPlanningTests
