import DslExperiment.Property
import Umpire.Property.Tests.Fixtures

/-!
Compare the experimental closed-trace obligation with the current checked Umpire bounded
response evaluator. The fixture bridge filters each operation into its own model trace and
encodes matching trigger/response predicates as existing fixture observations. That filtering
is an explicit experimental assumption, not a checked production evidence refinement. Traces
include arbitrary event words, so this checks property interpretation beyond legal model paths.
-/

namespace DslExperiment.Baseline

open Umpire

private def declaration (bound : Nat) : PropertyDeclaration := {
  Umpire.PropertyTests.portableProperty with
  id := Umpire.PropertyTests.id "test.property.dsl-baseline"
  clauses := [
    .eventuallyWithin (Umpire.PropertyTests.id "test.property.dsl-baseline.response")
      (Umpire.PropertyTests.pattern .observation Umpire.PropertyTests.cancelRequested)
      (Umpire.PropertyTests.pattern .observation Umpire.PropertyTests.cancelDelivered)
      (.exact { value := bound, unit := .semanticTransitions })
  ]
}

private def projectedTrace (clause : ResponseClause) (operation : Fin 2)
    (steps : List Step) : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
  initialState := Umpire.PropertyTests.value Umpire.PropertyTests.pendingCount "0"
  steps := (steps.filter fun step => step.operation == operation).map fun step => {
    selectedAction := Umpire.PropertyTests.value Umpire.PropertyTests.tick "tick"
    modelOutcome := Umpire.PropertyTests.value Umpire.PropertyTests.deliveredOutcome "delivered"
    resultingState := Umpire.PropertyTests.value Umpire.PropertyTests.pendingCount "0"
    observations :=
      (if clause.trigger.matches step then [Umpire.PropertyTests.value Umpire.PropertyTests.cancelRequested "trigger"] else []) ++
      (if clause.response.matches step then [Umpire.PropertyTests.value Umpire.PropertyTests.cancelDelivered "response"] else [])
  }
}

private def baseline (clause : ResponseClause) (steps : List Step) : Option Bool := do
  let a ← Umpire.PropertyTests.evaluationOf (declaration clause.bound) (projectedTrace clause 0 steps)
  let b ← Umpire.PropertyTests.evaluationOf (declaration clause.bound) (projectedTrace clause 1 steps)
  return a.satisfied && b.satisfied

private def words : Nat → List (List Step)
  | 0 => [[]]
  | n + 1 => [] :: alphabet.flatMap fun step => (words n).map (step :: ·)

private def clauses : List ResponseClause :=
  [0, 1, 2].flatMap fun bound => [
    { trigger := .event .requested, response := .anyOf [.canceled, .completed], bound },
    { trigger := .event .requested, response := .anyOf [.requested, .canceled, .completed], bound }
  ]

end DslExperiment.Baseline

open DslExperiment DslExperiment.Baseline in
/-- Execute bounded differential checks against the current checked Property evaluator. -/
def main : IO Unit := do
  let traces := words 3
  let mut cases := 0
  for clause in clauses do
    for trace in traces do
      let expected := evaluate clause .closedModel trace == .satisfied
      let observed := baseline clause trace
      unless observed == some expected do
        throw (IO.userError s!"baseline disagreement: {repr clause}; {repr trace}; {observed}; expected {expected}")
      cases := cases + 1
  IO.println s!"PASS {cases} closed-trace comparisons with current checked Umpire eventuallyWithin; {traces.length} words, lengths 0..3, two operations, bounds 0..2, terminal and same-step response predicates"
