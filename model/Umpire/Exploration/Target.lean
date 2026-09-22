import Umpire.Command.Authoring
import Umpire.Command.Coverage

/-!
# From a coverage target to a Query

An exploratory set enumerates what it sets out to reach: rows of its machine's table, result values,
members of claimed classes. None of those is a Query. This module turns one target into the Query a
campaign runs for it, formed the way every command Query is formed, so the same admission, search
and Producer take it.

The Query's Scenario is an exact trace: a shortest path over the table from a start state to the
target's row, then the row's action and its outcome, every step carrying its outcome and resulting
state. Pinning the trace is what pins the row -- an exact action list alone leaves the state before
the final step free, and eight hundred rows of one machine share nineteen actions. The Property is
the action clause on that final step naming its outcome, so every candidate carries a clause and
the Producer accepts it.

Which row a target names is fixed here, once: a `row` target is its row; a `result` target is the
first reachable row in table order whose results contain the outcome; a `classMember` target is the
first reachable row whose action is the member. Reachable means a path within the limits' steps
along which the final action, wherever it occurs earlier, carries the row's outcome: a transition
contract binds every occurrence of its action, so a prefix that took the action to a different
outcome would make the Query unsatisfiable. A target with no such path is unreachable, and no Run
is spent on it.
-/

namespace Umpire.Exploration

open Umpire.Command

variable {Setup State Action Outcome Fact : Type}
variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]

/-- One step of a planned path over the table: the row taken and the result the path follows. -/
structure PathStep (State Action Outcome Fact : Type) where
  row : FiniteTransitionRow State Action Outcome Fact
  result : Step State Outcome Fact
  deriving BEq, Repr

/-- A state the walk reached, the start it reached it from, and the shortest path there. -/
structure Reached (State Action Outcome Fact : Type) where
  start : State
  state : State
  path : List (PathStep State Action Outcome Fact)
  deriving BEq, Repr

/-- Shortest paths from the start states, one per reached state, in discovery order: starts first in
their order, then each frontier state's rows in table order and each row's results in row order.
A path is at most `maxSteps` long and takes only admissible steps; the walk follows terminal rows
too, as Search does. -/
def shortestPaths (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (starts : List State) (maxSteps : Nat)
    (admissible : PathStep State Action Outcome Fact → Bool := fun _ => true) :
    List (Reached State Action Outcome Fact) := Id.run do
  let mut reached : List (Reached State Action Outcome Fact) :=
    starts.foldl (init := []) fun reached start =>
      if reached.any (·.state == start) then reached
      else reached ++ [{ start, state := start, path := [] }]
  let mut frontier := reached
  for _ in [0:maxSteps] do
    let mut next : List (Reached State Action Outcome Fact) := []
    for frontierState in frontier do
      for row in rows do
        if row.source == frontierState.state then
          for result in row.results do
            unless (reached.any (·.state == result.state)) ||
                (next.any (·.state == result.state)) || !admissible { row, result } do
              next := next ++ [{
                start := frontierState.start
                state := result.state
                path := frontierState.path ++ [{ row, result }] }]
    reached := reached ++ next
    frontier := next
  return reached

/-- The shortest admissible path to one final step: within `steps` including the final step, and
never taking the final step's action to another outcome on the way. -/
def pathTo (model : DeclaredModel Setup State Action Outcome Fact) (steps : Nat)
    (final : PathStep State Action Outcome Fact) : Option (Reached State Action Outcome Fact) :=
  if steps == 0 then none else
  let admissible (step : PathStep State Action Outcome Fact) : Bool :=
    step.row.action != final.row.action || step.result.outcome == final.result.outcome
  (shortestPaths model.table.transitions model.initial (steps - 1) admissible).find?
    (·.state == final.row.source)

/-- The row and result one target names, with the shortest admissible path to the row's source, or
`none` when no such path exists within `steps`. -/
def chooseRow (model : DeclaredModel Setup State Action Outcome Fact) (steps : Nat)
    (target : CoverageTarget) :
    Option (Reached State Action Outcome Fact × PathStep State Action Outcome Fact) :=
  let actionId := catalogId model.actions model.actionIds
  let outcomeId := catalogId model.outcomes model.outcomeIds
  let rows := model.table.transitions
  let planned (row : FiniteTransitionRow State Action Outcome Fact) (result : Step State Outcome Fact) :=
    (pathTo model steps { row, result }).map fun lead => (lead, ({ row, result } : PathStep State Action Outcome Fact))
  match target with
  | .row key _ _ results =>
      -- The row's results in order, the ones the target names first: a result whose outcome the
      -- prefix must take the same action to earlier is not plannable, and the row's next one may be.
      (rows.find? (·.key == key)).bind fun row =>
        let named := row.results.filter fun result => results.contains (outcomeId result.outcome)
        let others := row.results.filter fun result => !results.contains (outcomeId result.outcome)
        (named ++ others).findSome? (planned row)
  | .result outcome =>
      rows.findSome? fun row =>
        (row.results.find? fun result => outcomeId result.outcome == outcome).bind (planned row)
  | .classMember member _ _ _ _ =>
      rows.findSome? fun row =>
        if actionId row.action == member then row.results.head?.bind (planned row) else none

/-- The spelling the table declares a member under: the key the vocabulary resolves. -/
private def keyOf [BEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  ((catalog.find? (·.value == value)).map (·.key)).getD ""

/-- The Query for one chosen row: the exact trace of the path and the final step, and the Property
that names the final step's outcome. Both authors read the vocabulary the check hands them, as the
command authors do. -/
def targetAuthors (model : DeclaredModel Setup State Action Outcome Fact) (queryKey : String)
    (lead : Reached State Action Outcome Fact) (final : PathStep State Action Outcome Fact) :
    (ModelVocabulary → Property) × (ModelVocabulary → Scenario) :=
  let table := model.table
  let steps := lead.path ++ [final]
  let actionKey (row : FiniteTransitionRow State Action Outcome Fact) := keyOf table.actions row.action
  let finalAction := actionKey final.row
  let finalOutcome := keyOf table.outcomes final.result.outcome
  let startKey := keyOf table.states lead.start
  let propertyAuthor : ModelVocabulary → Property := fun values =>
    authoredProperty model values {
      declaration := queryKey
      roleName := model.roleName
      groups := [{
        trigger := .action finalAction
        requirements := [.outcomeClause ("outcome-" ++ finalOutcome) finalOutcome] }] }
  let behaviorAuthor : ModelVocabulary → Scenario := fun values =>
    let role := model.namedRole model.roleName
    let start := values.namedState startKey
    let trace : AuthoredExactTrace := {
      setup := [{ role, value := start }]
      initialState := some start
      steps := steps.map fun step => {
        selectedAction := some (values.namedAction (actionKey step.row))
        outcome := some (values.namedOutcome (keyOf table.outcomes step.result.outcome))
        resultingState := some (values.namedState (keyOf table.states step.result.state))
        observations := some (step.result.facts.map fun fact =>
          values.namedFact (keyOf table.facts fact)) } }
    -- `Scenario.exactly` sets `actionsExactly` and every schedule field the Producer and the
    -- checker read; the exact trace is added beside it with the same actions in the same order.
    let exact := Scenario.exactly
      (family := model.origin.family)
      (key := queryKey)
      (source := model.origin.source)
      (requires := [model.roleCapability model.roleName])
      (roles := [{ id := role, valueKind := .state }])
      (setup := [SetupConstraint.roleEquals
        (model.origin.ownedId "setup" queryKey model.roleName) role start])
      (occurrences := steps.zipIdx.map fun (step, index) =>
        { key := queryKey ++ "." ++ toString (index + 1)
          action := (values.namedAction (actionKey step.row)).definitionId })
    { exact with traceExactly := some trace }
  (propertyAuthor, behaviorAuthor)

/-- The key a target is tracked under. Two targets with one key are one target. -/
def targetKey : CoverageTarget → String
  | .row key _ _ _ => "row:" ++ key
  | .result outcome => "result:" ++ outcome.value
  | .classMember member _ field className _ =>
      "class:" ++ member.value ++ ":" ++ field ++ ":" ++ className

/-- The targets a planned trace reaches: a row when a step leaves the row's state by its action
with one of its results, a result when a step ends in it, a class member when a step performs it. -/
def coveredTargets (targets : List CoverageTarget) (trace : Scenario.Trace) : List CoverageTarget :=
  let steps : List (DefinitionId × DefinitionId × DefinitionId) :=
    (trace.trace.steps.foldl (init := (trace.trace.initialState, []))
      fun (prior, acc) step =>
        (step.state, acc ++ [(prior.definitionId, step.selectedAction.definitionId,
          step.outcome.definitionId)])).2
  targets.filter fun target =>
    match target with
    | .row _ state action results =>
        steps.any fun (prior, taken, outcome) =>
          prior == state && taken == action && results.contains outcome
    | .result outcome => steps.any fun (_, _, reached) => reached == outcome
    | .classMember member _ _ _ _ => steps.any fun (_, taken, _) => taken == member

end Umpire.Exploration
