import Umpire.Core

/-!
Typed finite tables are inert inputs to Target authoring. Explicit ordered catalogs define the
modeled domains; unused inhabitants of the Lean carrier types need not be modeled. Setup rows and
transition rows contain all alternatives, with no competing step function or implicit fallback.

`FiniteTable.validate` returns either a typed structural error or data carrying closure and
Action-executability evidence. This is not checked Target admission: providers, capability laws,
and semantic composition still belong to `checkTarget`. Encodings are stable catalog keys, so a
collision is exactly a duplicate key; authors never supply encoder callbacks or ModelValue assembly.
-/

namespace Umpire

/-- A typed domain member and its explicit, stable wire key, independent of Lean names. -/
structure FiniteCatalogEntry (α : Type) where
  value : α
  key : String
  deriving BEq, DecidableEq, Repr

/-- Ordered modeled vocabulary; membership is never inferred from transition rows. -/
abbrev FiniteCatalog (α : Type) := List (FiniteCatalogEntry α)

namespace FiniteCatalog

/-- The declared domain in authored order, including members unused by transition rows. -/
def values (catalog : FiniteCatalog α) : List α := catalog.map (·.value)

/-- Stable keys are nonempty identifier segments, using the shared Definition ID grammar. -/
def validKey (key : String) : Bool :=
  !key.toList.isEmpty && key.toList.all (fun c => c.isAlphanum || c == '-' || c == '_')

/-- Lookup has no default encoding for an undeclared value. Validation makes it unambiguous. -/
def encode? [DecidableEq α] (catalog : FiniteCatalog α) (value : α) : Option String :=
  (catalog.find? fun entry => decide (entry.value = value)).map (·.key)

/-- Unique values and keys make the catalog a one-to-one encoding of its modeled domain. -/
structure Valid (catalog : FiniteCatalog α) : Prop where
  keys_valid : ∀ entry ∈ catalog, validKey entry.key = true
  keys_unique : (catalog.map (·.key)).Nodup
  values_unique : catalog.values.Nodup

end FiniteCatalog

/-- All initial-state alternatives for one explicit setup. -/
structure FiniteSetupRow (Setup State : Type) where
  setup : Setup
  states : List State
  deriving BEq, DecidableEq, Repr

/-- One enabled source/Action pair and all complete Target-owned result alternatives. -/
structure FiniteTransitionRow (State Action Outcome Fact : Type) where
  key : String
  source : State
  action : Action
  results : List (TransitionResult State Outcome Fact)
  deriving BEq, DecidableEq, Repr

/-- The authoritative finite data; absent source/Action pairs are disabled. -/
structure FiniteTable (Setup State Action Outcome Fact : Type) where
  setups : FiniteCatalog Setup
  states : FiniteCatalog State
  actions : FiniteCatalog Action
  outcomes : FiniteCatalog Outcome
  facts : FiniteCatalog Fact
  initial : List (FiniteSetupRow Setup State)
  transitions : List (FiniteTransitionRow State Action Outcome Fact)
  /-- Conjunctive constituent declarations; an empty constituent set prevents terminal closure. -/
  terminalConditions : List (List State) := []
  deriving BEq, DecidableEq, Repr

/-- The catalog or row field responsible for a structural admission failure. -/
inductive FiniteTableField where
  | setup | state | action | outcome | fact | initial | transition | source | resultState
  deriving BEq, DecidableEq, Repr

/-- Structural errors are field-specific; duplicate keys also reject colliding encodings. -/
inductive FiniteTableError where
  | malformedKey (field : FiniteTableField)
  | duplicateKey (field : FiniteTableField)
  | duplicateValue (field : FiniteTableField)
  | outOfDomain (field : FiniteTableField)
  | duplicateSetup
  | duplicateSourceAction
  | emptyAlternatives (field : FiniteTableField)
  | missingSetup
  | actionWithoutRow
  deriving BEq, DecidableEq, Repr

/-- Validated inert data with the exact finite closure facts needed by the kernel adapter.
Executability is existence of a row, not reachability from a setup. No providers are selected here. -/
structure ValidatedFiniteTable (Setup State Action Outcome Fact : Type) where
  private mk ::
  table : FiniteTable Setup State Action Outcome Fact
  setups_valid : table.setups.Valid
  states_valid : table.states.Valid
  actions_valid : table.actions.Valid
  outcomes_valid : table.outcomes.Valid
  facts_valid : table.facts.Valid
  row_keys_valid : ∀ row ∈ table.transitions, FiniteCatalog.validKey row.key = true
  row_keys_unique : (table.transitions.map (·.key)).Nodup
  setup_rows_unique : (table.initial.map (·.setup)).Nodup
  transition_rows_unique : (table.transitions.map fun row => (row.source, row.action)).Nodup
  setup_coverage : ∀ row ∈ table.initial, row.setup ∈ table.setups.values
  initial_state_coverage : ∀ row ∈ table.initial, ∀ state ∈ row.states, state ∈ table.states.values
  initial_nonempty : ∀ row ∈ table.initial, row.states ≠ []
  setup_defined : ∀ setup ∈ table.setups.values, ∃ row ∈ table.initial, row.setup = setup
  source_coverage : ∀ row ∈ table.transitions, row.source ∈ table.states.values
  action_coverage : ∀ row ∈ table.transitions, row.action ∈ table.actions.values
  result_state_coverage : ∀ row ∈ table.transitions, ∀ result ∈ row.results,
    result.resultingState ∈ table.states.values
  outcome_coverage : ∀ row ∈ table.transitions, ∀ result ∈ row.results,
    result.modelOutcome ∈ table.outcomes.values
  fact_coverage : ∀ row ∈ table.transitions, ∀ result ∈ row.results,
    ∀ fact ∈ result.observations, fact ∈ table.facts.values
  results_nonempty : ∀ row ∈ table.transitions, row.results ≠ []
  action_executable : ∀ action ∈ table.actions.values,
    ∃ row ∈ table.transitions, row.action = action

namespace FiniteTable

private def requireProof (p : Prop) [Decidable p] (error : FiniteTableError) :
    Except FiniteTableError (PLift p) :=
  if proof : p then .ok ⟨proof⟩ else .error error

private def validateCatalog [DecidableEq α] (field : FiniteTableField)
    (catalog : FiniteCatalog α) : Except FiniteTableError (PLift catalog.Valid) := do
  let keysValid ← requireProof (∀ entry ∈ catalog, FiniteCatalog.validKey entry.key = true)
    (.malformedKey field)
  let keysUnique ← requireProof (catalog.map (·.key)).Nodup (.duplicateKey field)
  let valuesUnique ← requireProof catalog.values.Nodup (.duplicateValue field)
  pure ⟨⟨keysValid.down, keysUnique.down, valuesUnique.down⟩⟩

/-- Validate catalogs, then rows, retaining input order and returning the first structural error.
Empty tables are allowed; declared setups and enabled pairs must have nonempty alternatives. -/
def validate [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact) :
    Except FiniteTableError (ValidatedFiniteTable Setup State Action Outcome Fact) := do
  let setups ← validateCatalog .setup table.setups
  let states ← validateCatalog .state table.states
  let actions ← validateCatalog .action table.actions
  let outcomes ← validateCatalog .outcome table.outcomes
  let facts ← validateCatalog .fact table.facts
  let rowKeys ← requireProof
    (∀ row ∈ table.transitions, FiniteCatalog.validKey row.key = true) (.malformedKey .transition)
  let uniqueKeys ← requireProof (table.transitions.map (·.key)).Nodup (.duplicateKey .transition)
  let uniqueSetups ← requireProof (table.initial.map (·.setup)).Nodup .duplicateSetup
  let uniquePairs ← requireProof
    (table.transitions.map fun row => (row.source, row.action)).Nodup .duplicateSourceAction
  let setupCoverage ← requireProof
    (∀ row ∈ table.initial, row.setup ∈ table.setups.values) (.outOfDomain .setup)
  let initialCoverage ← requireProof
    (∀ row ∈ table.initial, ∀ state ∈ row.states, state ∈ table.states.values) (.outOfDomain .initial)
  let initialNonempty ← requireProof
    (∀ row ∈ table.initial, row.states ≠ []) (.emptyAlternatives .initial)
  let setupDefined ← requireProof
    (∀ setup ∈ table.setups.values, ∃ row ∈ table.initial, row.setup = setup) .missingSetup
  let sourceCoverage ← requireProof
    (∀ row ∈ table.transitions, row.source ∈ table.states.values) (.outOfDomain .source)
  let actionCoverage ← requireProof
    (∀ row ∈ table.transitions, row.action ∈ table.actions.values) (.outOfDomain .action)
  let resultCoverage ← requireProof
    (∀ row ∈ table.transitions, ∀ result ∈ row.results,
      result.resultingState ∈ table.states.values) (.outOfDomain .resultState)
  let outcomeCoverage ← requireProof
    (∀ row ∈ table.transitions, ∀ result ∈ row.results,
      result.modelOutcome ∈ table.outcomes.values) (.outOfDomain .outcome)
  let factCoverage ← requireProof
    (∀ row ∈ table.transitions, ∀ result ∈ row.results,
      ∀ fact ∈ result.observations, fact ∈ table.facts.values) (.outOfDomain .fact)
  let resultsNonempty ← requireProof
    (∀ row ∈ table.transitions, row.results ≠ []) (.emptyAlternatives .transition)
  let _ ← requireProof
    (∀ states ∈ table.terminalConditions, ∀ state ∈ states, state ∈ table.states.values)
    (.outOfDomain .state)
  let executable ← requireProof
    (∀ action ∈ table.actions.values, ∃ row ∈ table.transitions, row.action = action) .actionWithoutRow
  pure ⟨table, setups.down, states.down, actions.down, outcomes.down, facts.down,
    rowKeys.down, uniqueKeys.down, uniqueSetups.down, uniquePairs.down, setupCoverage.down,
    initialCoverage.down, initialNonempty.down, setupDefined.down, sourceCoverage.down,
    actionCoverage.down, resultCoverage.down, outcomeCoverage.down, factCoverage.down,
    resultsNonempty.down, executable.down⟩

end FiniteTable

end Umpire
