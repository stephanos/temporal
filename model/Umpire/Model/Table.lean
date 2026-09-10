import Umpire.Model.Check

/-!
Typed finite tables are inert inputs to Target authoring. Explicit ordered catalogs define the
modeled domains; unused inhabitants of the Lean carrier types need not be modeled. Setup rows and
transition rows contain all alternatives, with no competing step function or implicit fallback.

`FiniteTable.validate` returns either a typed structural error or data carrying closure and
Action-executability evidence. This is not checked Target admission: providers, capability laws,
and semantic composition still belong to `checkModel`. Encodings are stable catalog keys, so a
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
  results : List (Step State Outcome Fact)
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
structure CheckedTable (Setup State Action Outcome Fact : Type) where
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
    result.state ∈ table.states.values
  outcome_coverage : ∀ row ∈ table.transitions, ∀ result ∈ row.results,
    result.outcome ∈ table.outcomes.values
  fact_coverage : ∀ row ∈ table.transitions, ∀ result ∈ row.results,
    ∀ fact ∈ result.facts, fact ∈ table.facts.values
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
    Except FiniteTableError (CheckedTable Setup State Action Outcome Fact) := do
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
      result.state ∈ table.states.values) (.outOfDomain .resultState)
  let outcomeCoverage ← requireProof
    (∀ row ∈ table.transitions, ∀ result ∈ row.results,
      result.outcome ∈ table.outcomes.values) (.outOfDomain .outcome)
  let factCoverage ← requireProof
    (∀ row ∈ table.transitions, ∀ result ∈ row.results,
      ∀ fact ∈ result.facts, fact ∈ table.facts.values) (.outOfDomain .fact)
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


/--
An enumerator-authoritative finite Target. Authors provide the semantic enumerators and the
evidence that their emitted values stay in the declared domains; Target derives the routine
membership relations, exhaustive-domain plumbing, kernel, and finite planning capability.
-/
structure FiniteMachine (Setup State Action Outcome Observation : Type) where
  metadata : MachineMetadata
  setups : List Setup
  states : List State
  actions : List Action
  outcomes : List Outcome
  observations : List Observation
  encodeSetup : Setup → String
  encodeState : State → String
  encodeAction : Action → String
  encodeOutcome : Outcome → String
  encodeObservation : Observation → String
  initialStates : Setup → List State
  steps : State → Action → List (Step State Outcome Observation)
  setupCoverage : ∀ setup state, state ∈ initialStates setup → setup ∈ setups
  initialStateCoverage : ∀ setup state, state ∈ initialStates setup → state ∈ states
  transitionSourceCoverage : ∀ state action result,
    result ∈ steps state action → state ∈ states
  actionCoverage : ∀ state action result, result ∈ steps state action → action ∈ actions
  resultingStateCoverage : ∀ state action result,
    result ∈ steps state action → result.state ∈ states
  outcomeCoverage : ∀ state action result,
    result ∈ steps state action → result.outcome ∈ outcomes
  observationCoverage : ∀ state action result value,
    result ∈ steps state action → value ∈ result.facts → value ∈ observations
  actionExecutable : ∀ action, action ∈ actions →
    ∃ state result, result ∈ steps state action

/-- Target metadata authored alongside a finite table; behavioral domains and setups come from
the table's successful validation branch. -/
structure TableModelSpec where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  metadata : MachineMetadata

/-- Finite admission keeps structural table failures separate from semantic Target failures. -/
inductive TableAdmissionError where
  | invalidTable (error : FiniteTableError)
  | invalidTarget (diagnostic : LocatedError)

/-- One typed state binding used to encode an authored setup for ModelValue-owned Query APIs. -/
structure FiniteModelSetupBinding (State : Type) where
  roleId : DefinitionId
  state : State

/-- Explicit field identities for lowering typed finite vocabulary through stable catalog keys. -/
structure FiniteModelIdentity (Setup State Action Outcome Fact : Type) where
  setupBindings : Setup → List (FiniteModelSetupBinding State)
  stateId : State → DefinitionId
  actionId : Action → DefinitionId
  outcomeId : Outcome → DefinitionId
  factId : Fact → DefinitionId

namespace FiniteMachine

/-- Derive the complete membership-based transition kernel represented by the descriptor. -/
def kernel
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    Machine Setup State Action Outcome Observation := {
  metadata := machine.metadata
  setupDomain := fun setup => setup ∈ machine.setups
  stateDomain := fun state => state ∈ machine.states
  actionDomain := fun action => action ∈ machine.actions
  outcomeDomain := fun outcome => outcome ∈ machine.outcomes
  observationDomain := fun observation => observation ∈ machine.observations
  initialStates := machine.initialStates
  authoritativeInitial := fun setup state => state ∈ machine.initialStates setup
  initialSound := by intro _ _ member; exact member
  initialComplete := by intro _ _ member; exact member
  steps := machine.steps
  authoritativeStep := fun state action result => result ∈ machine.steps state action
  stepSound := by intro _ _ _ member; exact member
  stepComplete := by intro _ _ _ member; exact member
  vocabulary := .complete {
    setups := machine.setups
    states := machine.states
    actions := machine.actions
    outcomes := machine.outcomes
    observations := machine.observations
    encodeSetup := machine.encodeSetup
    encodeState := machine.encodeState
    encodeAction := machine.encodeAction
    encodeOutcome := machine.encodeOutcome
    encodeObservation := machine.encodeObservation
    setupSound := by intro _ member; exact member
    setupComplete := by intro _ member; exact member
    stateSound := by intro _ member; exact member
    stateComplete := by intro _ member; exact member
    actionSound := by intro _ member; exact member
    actionComplete := by intro _ member; exact member
    outcomeSound := by intro _ member; exact member
    outcomeComplete := by intro _ member; exact member
    observationSound := by intro _ member; exact member
    observationComplete := by intro _ member; exact member
    setupCoverage := machine.setupCoverage
    initialStateCoverage := machine.initialStateCoverage
    transitionSourceCoverage := machine.transitionSourceCoverage
    actionCoverage := machine.actionCoverage
    resultingStateCoverage := machine.resultingStateCoverage
    outcomeCoverage := machine.outcomeCoverage
    observationCoverage := machine.observationCoverage
  }
}

/-- The checked-kernel input consumed by ordinary Target definitions. -/
def machineAvailability
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    MachineAvailability Setup State Action Outcome Observation :=
  .checked machine.kernel

/-- Derive finite planning from the same ordered action list and exact kernel relation. -/
def planning
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    FinitePlanningCapability machine.kernel.authoritativeStep := {
  actions := machine.actions
  actionSound := machine.actionExecutable
  actionComplete := machine.actionCoverage
}

/-- The dependent planning input consumed by `DraftModel.make`. -/
def authoredPlanning
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    AuthoredPlanningCapability machine.machineAvailability :=
  .available machine.kernel rfl machine.planning

/-- Assemble ordinary Target metadata around the finite machine's exact setup list and checked
kernel. The machine remains the evidence boundary: authors still provide every domain, encoder,
enumerator, closure proof, and executable-action proof when constructing it.

This constructor calls only `machineAvailability` (which calls `kernel`) and projects `setups`; both
functions assemble records without traversal, normalization, validation, or scanning. One
invocation therefore adds one record assembly regardless of domain size, and 1×/10× independent
declaration sets add exactly 1×/10× assemblies. -/
def modelSpec
    (machine : FiniteMachine Setup State Action Outcome Observation)
    (id : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (requiredCapabilities : List DefinitionId) :
    ModelSpec LawStatement Setup State Action Outcome Observation := {
  id
  source
  definitions
  requiredCapabilities
  resolvedSetups := machine.setups
  machine := machine.machineAvailability
}

/-- Assemble an authored Target with explicit composition and occurrence inputs, while deriving the
dependent planning witness from the same finite machine. This calls `modelSpec`,
`DraftModel.make`, and `authoredPlanning`; transitively, `authoredPlanning` calls
`machineAvailability`, `kernel`, and `planning`. Every call only assembles records or projects the
machine's existing values. The constructor adds no list traversal, normalization, validation, or
nested scan, and does not run `checkModel`. One call adds one authored assembly per declaration, so
1×/10× independent declaration sets add exactly 1×/10× assemblies before unchanged checker work. -/
def draftModel
    (machine : FiniteMachine Setup State Action Outcome Observation)
    (id : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (requiredCapabilities : List DefinitionId)
    (composition : Providers LawStatement)
    (occurrences : List SourceRef) :
    DraftModel LawStatement Setup State Action Outcome Observation :=
  DraftModel.make
    (machine.modelSpec id source definitions requiredCapabilities)
    composition
    machine.authoredPlanning
    occurrences

@[simp] theorem kernel_setupDomain_iff
    (machine : FiniteMachine Setup State Action Outcome Observation) (setup : Setup) :
    machine.kernel.setupDomain setup ↔ setup ∈ machine.setups :=
  Iff.rfl

@[simp] theorem kernel_stateDomain_iff
    (machine : FiniteMachine Setup State Action Outcome Observation) (state : State) :
    machine.kernel.stateDomain state ↔ state ∈ machine.states :=
  Iff.rfl

@[simp] theorem kernel_actionDomain_iff
    (machine : FiniteMachine Setup State Action Outcome Observation) (action : Action) :
    machine.kernel.actionDomain action ↔ action ∈ machine.actions :=
  Iff.rfl

@[simp] theorem kernel_outcomeDomain_iff
    (machine : FiniteMachine Setup State Action Outcome Observation) (outcome : Outcome) :
    machine.kernel.outcomeDomain outcome ↔ outcome ∈ machine.outcomes :=
  Iff.rfl

@[simp] theorem kernel_observationDomain_iff
    (machine : FiniteMachine Setup State Action Outcome Observation) (observation : Observation) :
    machine.kernel.observationDomain observation ↔ observation ∈ machine.observations :=
  Iff.rfl

@[simp] theorem kernel_authoritativeInitial_iff
    (machine : FiniteMachine Setup State Action Outcome Observation)
    (setup : Setup) (state : State) :
    machine.kernel.authoritativeInitial setup state ↔ state ∈ machine.initialStates setup :=
  Iff.rfl

@[simp] theorem kernel_authoritativeStep_iff
    (machine : FiniteMachine Setup State Action Outcome Observation)
    (state : State) (action : Action) (result : Step State Outcome Observation) :
    machine.kernel.authoritativeStep state action result ↔
      result ∈ machine.steps state action :=
  Iff.rfl

end FiniteMachine

namespace CheckedTable

private def encode [DecidableEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  (catalog.encode? value).getD ""

private def initialStates [DecidableEq Setup]
    (table : FiniteTable Setup State Action Outcome Observation) (setup : Setup) : List State :=
  table.initial.flatMap fun row => if row.setup = setup then row.states else []

private def steps [DecidableEq State] [DecidableEq Action]
    (table : FiniteTable Setup State Action Outcome Observation)
    (state : State) (action : Action) : List (Step State Outcome Observation) :=
  table.transitions.flatMap fun row =>
    if row.source = state ∧ row.action = action then row.results else []

/-- Derive every finite enumerator and mechanical witness from the successfully validated rows. -/
def machine [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Observation]
    (validated : CheckedTable Setup State Action Outcome Observation)
    (metadata : MachineMetadata) : FiniteMachine Setup State Action Outcome Observation := {
  metadata
  setups := validated.table.setups.values
  states := validated.table.states.values
  actions := validated.table.actions.values
  outcomes := validated.table.outcomes.values
  observations := validated.table.facts.values
  encodeSetup := encode validated.table.setups
  encodeState := encode validated.table.states
  encodeAction := encode validated.table.actions
  encodeOutcome := encode validated.table.outcomes
  encodeObservation := encode validated.table.facts
  initialStates := initialStates validated.table
  steps := steps validated.table
  setupCoverage := by
    intro setup state member
    simp only [initialStates, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, stateMember⟩
    by_cases setupEq : row.setup = setup
    · rw [← setupEq]
      exact validated.setup_coverage row rowMember
    · simp [setupEq] at stateMember
  initialStateCoverage := by
    intro setup state member
    simp only [initialStates, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, stateMember⟩
    by_cases setupEq : row.setup = setup
    · simp [setupEq] at stateMember
      exact validated.initial_state_coverage row rowMember state stateMember
    · simp [setupEq] at stateMember
  transitionSourceCoverage := by
    intro state action result member
    simp only [steps, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, resultMember⟩
    by_cases pairEq : row.source = state ∧ row.action = action
    · simp [pairEq] at resultMember
      rw [← pairEq.1]
      exact validated.source_coverage row rowMember
    · simp [pairEq] at resultMember
  actionCoverage := by
    intro state action result member
    simp only [steps, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, resultMember⟩
    by_cases pairEq : row.source = state ∧ row.action = action
    · simp [pairEq] at resultMember
      rw [← pairEq.2]
      exact validated.action_coverage row rowMember
    · simp [pairEq] at resultMember
  resultingStateCoverage := by
    intro state action result member
    simp only [steps, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, resultMember⟩
    by_cases pairEq : row.source = state ∧ row.action = action
    · simp [pairEq] at resultMember
      exact validated.result_state_coverage row rowMember result resultMember
    · simp [pairEq] at resultMember
  outcomeCoverage := by
    intro state action result member
    simp only [steps, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, resultMember⟩
    by_cases pairEq : row.source = state ∧ row.action = action
    · simp [pairEq] at resultMember
      exact validated.outcome_coverage row rowMember result resultMember
    · simp [pairEq] at resultMember
  observationCoverage := by
    intro state action result observation member observationMember
    simp only [steps, List.mem_flatMap] at member
    rcases member with ⟨row, rowMember, resultMember⟩
    by_cases pairEq : row.source = state ∧ row.action = action
    · simp [pairEq] at resultMember
      exact validated.fact_coverage row rowMember result resultMember observation observationMember
    · simp [pairEq] at resultMember
  actionExecutable := by
    intro action actionMember
    rcases validated.action_executable action actionMember with ⟨row, rowMember, actionEq⟩
    have resultsNonempty := validated.results_nonempty row rowMember
    rcases resultsEq : row.results with _ | ⟨result, rest⟩
    · contradiction
    · refine ⟨row.source, result, ?_⟩
      simp only [steps, List.mem_flatMap]
      refine ⟨row, rowMember, ?_⟩
      simp [actionEq, resultsEq]
}

/-- Package the exact derived machine through the existing ordinary Target authoring boundary. -/
def draftModel [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Observation]
    (validated : CheckedTable Setup State Action Outcome Observation)
    (definition : TableModelSpec)
    (composition : Providers LawStatement := .empty) :
    DraftModel LawStatement Setup State Action Outcome Observation :=
  let machine := validated.machine definition.metadata
  DraftModel.make {
    id := definition.id
    source := definition.source
    definitions := definition.definitions
    requiredCapabilities := definition.requiredCapabilities
    resolvedSetups := machine.setups
    terminalConditions := validated.table.terminalConditions
    machine := machine.machineAvailability
  } composition machine.authoredPlanning

end CheckedTable

/-- A structurally validated typed table paired with its explicit ModelValue identities. -/
structure CheckedTableModel (Setup State Action Outcome Fact : Type) where
  private mk ::
  table : FiniteTable Setup State Action Outcome Fact
  identity : FiniteModelIdentity Setup State Action Outcome Fact

namespace FiniteTable

private def modelValue
    [DecidableEq α]
    (catalog : FiniteCatalog α)
    (definitionId : α → DefinitionId)
    (field : FiniteTableField)
    (value : α) : Except FiniteTableError ModelValue :=
  match catalog.find? fun entry => decide (entry.value = value) with
  | some entry => .ok (ModelValue.named (definitionId value) entry.key)
  | none => .error (.outOfDomain field)

private def modelSetup
    [DecidableEq State]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (setup : Setup) : Except FiniteTableError (List RoleBinding) :=
  identity.setupBindings setup |>.mapM fun binding => do
    let state ← modelValue table.states identity.stateId .state binding.state
    pure (⟨binding.roleId, state⟩ : Umpire.RoleBinding)

private def modelResult
    [DecidableEq State] [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (result : Step State Outcome Fact) :
    Except FiniteTableError (Step ModelValue ModelValue ModelValue) := do
  let state ← modelValue table.states identity.stateId .resultState result.state
  let outcome ← modelValue table.outcomes identity.outcomeId .outcome result.outcome
  let facts ← result.facts.mapM fun fact =>
    modelValue table.facts identity.factId .fact fact
  pure { state, outcome, facts }

private def modelTable
    [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact) :
    Except FiniteTableError
      (FiniteTable (List RoleBinding) ModelValue ModelValue ModelValue ModelValue) := do
  let setups ← table.setups.mapM fun entry => do
    let value ← modelSetup table identity entry.value
    pure { value, key := entry.key }
  let initial ← table.initial.mapM fun row => do
    let setup ← modelSetup table identity row.setup
    let states ← row.states.mapM fun state => modelValue table.states identity.stateId .initial state
    pure { setup, states }
  let terminalConditions ← table.terminalConditions.mapM fun states =>
    states.mapM (modelValue table.states identity.stateId .state)
  let transitions ← table.transitions.mapM fun row => do
    let source ← modelValue table.states identity.stateId .source row.source
    let action ← modelValue table.actions identity.actionId .action row.action
    let results ← row.results.mapM (modelResult table identity)
    pure { key := row.key, source, action, results }
  pure {
    setups
    states := table.states.map fun entry =>
      { value := ModelValue.named (identity.stateId entry.value) entry.key, key := entry.key }
    actions := table.actions.map fun entry =>
      { value := ModelValue.named (identity.actionId entry.value) entry.key, key := entry.key }
    outcomes := table.outcomes.map fun entry =>
      { value := ModelValue.named (identity.outcomeId entry.value) entry.key, key := entry.key }
    facts := table.facts.map fun entry =>
      { value := ModelValue.named (identity.factId entry.value) entry.key, key := entry.key }
    initial
    transitions
    terminalConditions
  }

/-- Validate the typed catalogs and rows before exposing stable-key value resolution. -/
def checkIdentity [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact) :
    Except FiniteTableError (CheckedTableModel Setup State Action Outcome Fact) := do
  let validated ← table.validate
  pure ⟨validated.table, identity⟩

end FiniteTable

namespace CheckedTableModel

/-- Resolve one authored state through its validated catalog key and explicit field identity. -/
def stateValue
    [DecidableEq State]
    (model : CheckedTableModel Setup State Action Outcome Fact)
    (state : State) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.states model.identity.stateId .state state

/-- Resolve one authored Action through its validated catalog key and explicit field identity. -/
def actionValue
    [DecidableEq Action]
    (model : CheckedTableModel Setup State Action Outcome Fact)
    (action : Action) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.actions model.identity.actionId .action action

/-- Resolve one authored Model Outcome through its catalog key and explicit field identity. -/
def outcomeValue
    [DecidableEq Outcome]
    (model : CheckedTableModel Setup State Action Outcome Fact)
    (outcome : Outcome) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.outcomes model.identity.outcomeId .outcome outcome

/-- Resolve one authored fact through its validated catalog key and explicit field identity. -/
def factValue
    [DecidableEq Fact]
    (model : CheckedTableModel Setup State Action Outcome Fact)
    (fact : Fact) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.facts model.identity.factId .fact fact

/-- Lower one authored setup using its explicit role bindings and the state catalog. -/
def setupValue
    [DecidableEq Setup] [DecidableEq State]
    (model : CheckedTableModel Setup State Action Outcome Fact)
    (setup : Setup) : Except FiniteTableError (List RoleBinding) :=
  if setup ∈ model.table.setups.values then
    FiniteTable.modelSetup model.table model.identity setup
  else
    .error (.outOfDomain .setup)

end CheckedTableModel

namespace FiniteTable

/-- Validate typed author data before lowering its stable keys to the existing ModelValue Target. -/
def checkModel [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (definition : TableModelSpec)
    (composition : Providers LawStatement := .empty) :
    Except TableAdmissionError
      (CheckedModel LawStatement (List RoleBinding) ModelValue ModelValue ModelValue ModelValue) := do
  let _ ← table.checkIdentity identity |>.mapError TableAdmissionError.invalidTable
  let lowered ← modelTable table identity |>.mapError TableAdmissionError.invalidTable
  let validated ← FiniteTable.validate
    (Setup := List RoleBinding)
    (State := ModelValue)
    (Action := ModelValue)
    (Outcome := ModelValue)
    (Fact := ModelValue)
    lowered |>.mapError TableAdmissionError.invalidTable
  Umpire.checkModel (validated.draftModel definition composition)
    |>.mapError TableAdmissionError.invalidTarget

/-- Check a table whose carriers stay typed, without the ModelValue lowering `checkModel` applies. -/
def checkTypedModel [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (definition : TableModelSpec)
    (composition : Providers LawStatement := .empty) :
    Except TableAdmissionError
      (CheckedModel LawStatement Setup State Action Outcome Fact) := do
  let validated ← table.validate |>.mapError TableAdmissionError.invalidTable
  Umpire.checkModel (validated.draftModel definition composition)
    |>.mapError TableAdmissionError.invalidTarget

end FiniteTable

end Umpire
