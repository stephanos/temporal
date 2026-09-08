import Umpire.Target.Tests.Parameterized
import Umpire.Planning.Tests.Fixtures

/-! Parameterized finite completeness, bounded search, and exact checked Query replay. -/
namespace Umpire.ParameterizedPlanningTests
open Umpire Operation Value TargetTests.Parameterized

private def queryDeclaration (t : QueryTarget (fun _ => True)) (budget : Nat)
    (exact : Option BehaviorTrace := none) : QueryDeclaration := {
  id := .of "example.query.call", source, target := t.id
  form := .verify PlanningTests.property
  limits := QueryLimits.bounded 1 1 budget
  policy := .exhaustive
  behavior := { PlanningTests.behavior with
    roles := [], allowedActions := [actionId],
    requiredOccurrences := [], actionsExactly := none, traceExactly := exact }
}
private def run (budget : Nat) := do
  let t ← target .samplesOnly
  let q ← checkQuery (.ofTarget t) (queryDeclaration t budget) |>.mapError (fun _ => "query")
  let k ← IncrementalPlannerKernel.ofCheckedQuery q.target.id q |>.mapError (fun _ => "planner")
  plan q k |>.mapError (fun _ => "plan")
#guard (run 1).toOption.map (fun r =>
  (r.result.outcome.name, r.result.metadata.completeness.established)) == some ("limit-reached", false)
#guard (run 100).toOption.map (fun r =>
  (r.result.outcome.name, r.result.metadata.completeness.established)) == some ("verified-within-limits", true)

#guard (do
  let ⟨_, d⟩ ← (domain .samplesOnly).toOption
  let a ← d.actions.head?
  let t ← (target .samplesOnly).toOption
  let step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue := {
    selectedAction := a.modelValue, modelOutcome := value "example.outcome.call" "accepted",
    resultingState := value "example.state.phase" "done", observations := [] }
  let trace : BehaviorTrace := {
    setup := [], trace := { initialState := value "example.state.phase" "idle", steps := [step] } }
  let bad := { trace with trace := { trace.trace with steps := [
    { step with resultingState := value "example.state.phase" "idle" }] } }
  pure ((checkQuery (.ofTarget t) (queryDeclaration t 100 (some trace))).toOption.isSome,
    (checkQuery (.ofTarget t) (queryDeclaration t 100 (some bad))).toOption.isNone)) == some (true, true)


end Umpire.ParameterizedPlanningTests
