import Umpire.Command.Authoring
import Umpire.ImplementationLink.Refinement

/-!
# A machine that refines another

`DESIGN.md` section 2.5: a detailed protocol machine `refines:` a simpler product machine through a
`map:` from its state to the product's. Every protocol row is walked through the map -- a row whose
mapped states are a product step is that step, a row whose mapped states are equal is a stutter,
and any other row rejects the refinement -- and the step mapping is derived, never written. Outcomes
and facts read as the product's value of the same name; a fact the product does not name is one the
product does not see.

This module is the data side of `Umpire.ImplementationLink.Refinement`: it builds the morphism the
`machine` command hands to the simulation, derives the step mapping and reports what it found in
the author's terms, and reads a Property declared on the product machine over the protocol machine.
The witness itself is `TableRefinement.ofChecked`, decided over the same tables.
-/

namespace Umpire.Command

open Umpire

/-- The product key a protocol key names by default: the same key, or the constructor the key
applies where the product's member is that bare constructor -- `nexusOperationTimedOut` covers
`nexusOperationTimedOut-scheduleToClose`, the way an `evidence:` line covers a constructor's
members. -/
def sameNamedKey (product : List String) (key : String) : Option String :=
  if product.contains key then some key
  else
    let constructor := ((key.splitOn "-").head?).getD key
    if product.contains constructor then some constructor else none

/-- Every protocol key paired with the product key it names; a key naming none is left out. -/
def sameNamedPairs (protocol product : List String) : List (String × String) :=
  protocol.filterMap fun key => (sameNamedKey product key).map (key, ·)

/-- The product member a protocol member's key names, through a derived pairing. Written over the
pairing and the two key functions rather than over the keys' spelling rules, so the term the
refinement is decided over does nothing but compare keys. -/
def namedMember [BEq α] (pairs : List (String × String)) (members : List α) (keyFor : α → String)
    (key : String) : Option α :=
  (pairs.lookup key).bind fun target => members.find? fun member => keyFor member == target

/-- The morphism a refining machine reads through: the product's one setup, the authored state map,
and each outcome and fact by name. -/
def refinementMorphism [BEq Outcome'] [BEq Fact']
    (destinationSetup : Setup')
    (abstract : State → State')
    (sourceOutcomeKey : Outcome → String)
    (destinationOutcomes : List Outcome') (destinationOutcomeKey : Outcome' → String)
    (outcomePairs : List (String × String))
    (sourceFactKey : Fact → String)
    (destinationFacts : List Fact') (destinationFactKey : Fact' → String)
    (factPairs : List (String × String)) :
    RefinementMorphism Setup State Outcome Fact Setup' State' Outcome' Fact' := {
  mapSetup := fun _ => destinationSetup
  mapState := abstract
  mapOutcome := fun outcome =>
    namedMember outcomePairs destinationOutcomes destinationOutcomeKey (sourceOutcomeKey outcome)
  mapObservation := fun fact =>
    namedMember factPairs destinationFacts destinationFactKey (sourceFactKey fact)
}

/-- What the `machine` command reads back after deriving the step mapping: the rejection to report
at the `map:` line, or every row's result with the product action whose step carries it, `none`
for a stutter. Plain data, because it is evaluated during elaboration. -/
structure RefinementReport where
  rejected : Option String := none
  rows : List (String × Option String) := []
  deriving Repr, Inhabited

private def catalogKey [BEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  ((catalog.find? (·.value == value)).map (·.key)).getD "?"

/-- Derive the step mapping of one machine refining another, in the author's terms. Every outcome
must read as a product outcome, the machine must start where the product starts, and every row's
result is a product step from the mapped state -- preferring the product action of the row's own
name -- or a stutter. -/
def deriveRefinement [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [BEq Setup'] [BEq State'] [BEq Action'] [BEq Outcome'] [BEq Fact']
    (source : DeclaredModel Setup State Action Outcome Fact)
    (destination : DeclaredModel Setup' State' Action' Outcome' Fact')
    (morphism : RefinementMorphism Setup State Outcome Fact Setup' State' Outcome' Fact')
    (sourceName destinationName : String) : RefinementReport :=
  let stateKey := catalogKey source.table.states
  let stateKey' := catalogKey destination.table.states
  let outcomeKey := catalogKey source.table.outcomes
  let factKey := catalogKey source.table.facts
  let actionKey := catalogKey source.table.actions
  let actionKey' := catalogKey destination.table.actions
  let destinationActions := destination.table.actions.map (·.key)
  match source.outcomes.find? fun outcome => (morphism.mapOutcome outcome).isNone with
  | some outcome => { rejected := some (
      s!"'{outcomeKey outcome}' is an outcome of {sourceName} and no outcome of {destinationName} \
has that name; an outcome reads as the one of its name, so the refined machine declares it") }
  | none =>
  match source.initial.find? fun state => !destination.initial.contains (morphism.mapState state) with
  | some state => { rejected := some (
      s!"{sourceName} starts at '{stateKey state}', which reads as \
'{stateKey' (morphism.mapState state)}', and {destinationName} does not start there; a refining \
machine begins where the refined one does") }
  | none =>
  let carriedBy := fun (row : FiniteTransitionRow State Action Outcome Fact)
      (result : Step State Outcome Fact) =>
    let mappedFrom := morphism.mapState row.source
    let mappedTo := morphism.mapState result.state
    let mappedOutcome := morphism.mapOutcome result.outcome
    let mappedFacts := morphism.mapFacts result.facts
    let carriers := destination.table.transitions.filter fun carrier =>
      carrier.source == mappedFrom && carrier.results.any fun carried =>
        carried.state == mappedTo && some carried.outcome == mappedOutcome &&
          carried.facts.all mappedFacts.contains
    let preferred := sameNamedKey destinationActions (actionKey row.action)
    match carriers.find? fun carrier => some (actionKey' carrier.action) == preferred with
    | some carrier => some (actionKey' carrier.action)
    | none => carriers.head?.map fun carrier => actionKey' carrier.action
  let classified : Except String (List (String × Option String)) :=
    source.table.transitions.flatMapM fun row => row.results.mapM fun result =>
      match carriedBy row result with
      | some carrier => pure (row.key, some carrier)
      | none =>
          if morphism.mapState row.source == morphism.mapState result.state then
            pure (row.key, none)
          else
            let mappedFrom := stateKey' (morphism.mapState row.source)
            let mappedTo := stateKey' (morphism.mapState result.state)
            throw s!"the row '{row.key}' steps from '{stateKey row.source}' to \
'{stateKey result.state}', which read as '{mappedFrom}' and '{mappedTo}' in {destinationName}; \
{destinationName} has no step from '{mappedFrom}' reaching '{mappedTo}' with outcome \
'{outcomeKey result.outcome}' and the facts [{", ".intercalate (result.facts.map factKey)}], \
and the two are not equal, so the row is neither a step of {destinationName} nor a stutter"
  match classified with
  | .ok rows => { rows }
  | .error rejected => { rejected := some rejected }

/-- A Property declared on the refined machine, read on the refining one.

The refining machine carries the product state it reads as in a state field named after the
product machine, so a product claim about the state -- a trigger at a prior state, or the state a
clause fixes -- is a claim about that field, and the evaluator reads it apart from the state the way
it reads any field. Outcomes and facts are the refining machine's values of the same name. The
Property keeps the product Property's identity, because it is that Property and no other; only the
capability it requires is the refining machine's, which is the machine its clauses are read on. -/
def refinedProperty [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [BEq Setup'] [BEq State'] [BEq Action'] [BEq Outcome'] [BEq Fact']
    (model : DeclaredModel Setup State Action Outcome Fact)
    (refined : DeclaredModel Setup' State' Action' Outcome' Fact')
    (values : ModelVocabulary)
    (names : PropertyNames) : Property :=
  let abstractField := model.origin.ownedId "state-field" model.key refined.key
  let abstractPattern := fun (field : PropertyTraceField) (spelling : String) =>
    ({ field, reference := abstractField, constraint := .equals spelling } : PropertyPattern)
  let clauseId := refined.origin.ownedId "property" names.declaration
  {
  id := refined.origin.family.id "property" names.declaration
  source := refined.origin.source
  requires := [model.capabilityId]
  clauses := names.groups.flatMap fun group =>
    let selected := match group.trigger with
      | .action spelling => PropertyPattern.selectedAction (values.namedAction spelling)
      | .priorState spelling => abstractPattern .priorState spelling
    group.requirements.map fun requirement =>
    match requirement with
    | .stateClause label spelling =>
        .transitionContract (clauseId label) selected (abstractPattern .resultingState spelling)
    | .outcomeClause label spelling =>
        .transitionContract (clauseId label) selected (.outcome (values.namedOutcome spelling))
    | .factClause label spelling =>
        .inputOutput (clauseId label) selected (.fact (values.namedFact spelling))
  }

end Umpire.Command
