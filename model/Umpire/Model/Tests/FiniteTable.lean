import Umpire.Model

/-! Typed finite table admission through the public Target facade. -/

namespace Umpire.ModelTests.FiniteTable

#check Umpire.FiniteCatalog
#check Umpire.FiniteTable.validate
#check Umpire.CheckedTable


open Umpire

private def entry (value : Nat) (key : String) : FiniteCatalogEntry Nat := ⟨value, key⟩

private def first : Step Nat Nat Nat := ⟨0, 1, [0, 1]⟩

private def alternate : Step Nat Nat Nat := ⟨1, 0, [1]⟩

private def row : FiniteTransitionRow Nat Nat Nat Nat :=
  ⟨"advance", 0, 1, [first, alternate]⟩

private def otherRow : FiniteTransitionRow Nat Nat Nat Nat :=
  ⟨"return", 1, 0, [alternate]⟩

private def table : FiniteTable Nat Nat Nat Nat Nat := {
  setups := [entry 1 "ready"]
  states := [entry 2 "isolated", entry 0 "idle", entry 1 "active"]
  actions := [entry 1 "advance", entry 0 "return"]
  outcomes := [entry 0 "advanced", entry 1 "returned"]
  facts := [entry 0 "changed", entry 1 "observed"]
  initial := [⟨1, [0, 1]⟩]
  transitions := [row, otherRow]
}

private def error? (input : FiniteTable Nat Nat Nat Nat Nat) : Option FiniteTableError :=
  match input.validate with
  | .ok _ => none
  | .error error => some error

/-- The isolated state remains declared, and carrier values beyond the catalog are not inferred. -/
example : error? table = none := by decide

/-- One mutation per error case; duplicate stable keys are exactly collisions in the derived encoding. -/
private def negatives : List (FiniteTable Nat Nat Nat Nat Nat × FiniteTableError) := [
  ({ table with setups := [entry 1 ""] }, .malformedKey .setup),
  ({ table with states := [entry 0 "bad.key"] }, .malformedKey .state),
  ({ table with actions := [entry 1 "bad key"] }, .malformedKey .action),
  ({ table with outcomes := [entry 0 "bad/key"] }, .malformedKey .outcome),
  ({ table with facts := [entry 0 "bad:key"] }, .malformedKey .fact),
  ({ table with transitions := [{ row with key := "" }] }, .malformedKey .transition),
  ({ table with states := [entry 0 "same", entry 1 "same"] }, .duplicateKey .state),
  ({ table with states := [entry 0 "first", entry 0 "second"] }, .duplicateValue .state),
  ({ table with transitions := [row, { otherRow with key := row.key }] },
    .duplicateKey .transition),
  ({ table with initial := table.initial ++ table.initial }, .duplicateSetup),
  ({ table with transitions := [row, { row with key := "another" }] }, .duplicateSourceAction),
  ({ table with initial := [⟨9, [0]⟩] }, .outOfDomain .setup),
  ({ table with initial := [⟨1, [9]⟩] }, .outOfDomain .initial),
  ({ table with initial := [⟨1, []⟩] }, .emptyAlternatives .initial),
  ({ table with initial := [] }, .missingSetup),
  ({ table with transitions := [{ row with source := 9 }] }, .outOfDomain .source),
  ({ table with transitions := [{ row with action := 9 }] }, .outOfDomain .action),
  ({ table with transitions := [{ row with results := [{ first with state := 9 }] }] },
    .outOfDomain .resultState),
  ({ table with transitions := [{ row with results := [{ first with outcome := 9 }] }] },
    .outOfDomain .outcome),
  ({ table with transitions := [{ row with results := [{ first with facts := [9] }] }] },
    .outOfDomain .fact),
  ({ table with transitions := [{ row with results := [] }] }, .emptyAlternatives .transition),
  ({ table with transitions := [row] }, .actionWithoutRow),
  ({ table with actions := [entry 0 "return"] }, .outOfDomain .action),
  ({ table with outcomes := [entry 1 "returned"] }, .outOfDomain .outcome),
  ({ table with facts := [entry 1 "observed"] }, .outOfDomain .fact)
]

example : negatives.all (fun (input, expected) => error? input == some expected) = true := by
  decide

/-- Successful admission preserves every catalog, setup, row, fact, and alternative position. -/
example : (table.validate.toOption.map (·.table)) = some table := by decide

private def reordered : FiniteTable Nat Nat Nat Nat Nat := {
  table with
  states := table.states.reverse
  actions := table.actions.reverse
  transitions := [{ otherRow with results := otherRow.results.reverse },
    { row with results := row.results.reverse }]
}

example : (reordered.validate.toOption.map (·.table)) = some reordered := by decide

/-- Reversing competing rows never makes either definition admissible. -/
example : error? { table with transitions := [{ row with key := "another" }, row] } =
    some .duplicateSourceAction := by decide

/-- An Action may be executable only from an isolated state; validation makes no reachability claim. -/
example : error? { table with transitions := [{ row with source := 2 }, otherRow] } = none := by
  decide

/-- Catalog lookup retains explicit keys and fails for an undeclared carrier value. -/
example : table.states.values = [2, 0, 1] ∧
    table.states.encode? 0 = some "idle" ∧ table.states.encode? 9 = none := by decide

/-- No fact is required on a result; an entirely empty modeled domain is also valid. -/
example : error? { table with facts := [], transitions := [
    { row with results := [{ first with facts := [] }] },
    { otherRow with results := [{ alternate with facts := [] }] }] } = none ∧
    error? ⟨[], [], [], [], [], [], [], []⟩ = none := by decide

/-- Proof projections are usable without unfolding validation or constructing a semantic kernel. -/
example (checked : CheckedTable Nat Nat Nat Nat Nat)
    (transition : FiniteTransitionRow Nat Nat Nat Nat)
    (member : transition ∈ checked.table.transitions)
    (result : Step Nat Nat Nat) (emitted : result ∈ transition.results) :
    transition.source ∈ checked.table.states.values ∧
      result.state ∈ checked.table.states.values :=
  ⟨checked.source_coverage transition member,
    checked.result_state_coverage transition member result emitted⟩

example : error? { table with terminalConditions := [[1], [0, 1]] } = none ∧
    error? { table with terminalConditions := [[9]] } = some (.outOfDomain .state) := by decide

#print axioms Umpire.FiniteTable.validate
#print axioms Umpire.FiniteCatalog.encode?

end Umpire.ModelTests.FiniteTable
