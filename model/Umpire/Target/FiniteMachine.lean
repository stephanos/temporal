import Umpire.Target.FiniteTable
import Umpire.Target.Language

/-! Complete finite-machine authoring for ordinary Umpire Targets. -/

namespace Umpire

/--
An enumerator-authoritative finite Target. Authors provide the semantic enumerators and the
evidence that their emitted values stay in the declared domains; Target derives the routine
membership relations, exhaustive-domain plumbing, kernel, and finite planning capability.
-/
structure FiniteMachine (Setup State Action Outcome Observation : Type) where
  metadata : KernelMetadata
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
  steps : State → Action → List (TransitionResult State Outcome Observation)
  setupCoverage : ∀ setup state, state ∈ initialStates setup → setup ∈ setups
  initialStateCoverage : ∀ setup state, state ∈ initialStates setup → state ∈ states
  transitionSourceCoverage : ∀ state action result,
    result ∈ steps state action → state ∈ states
  actionCoverage : ∀ state action result, result ∈ steps state action → action ∈ actions
  resultingStateCoverage : ∀ state action result,
    result ∈ steps state action → result.resultingState ∈ states
  outcomeCoverage : ∀ state action result,
    result ∈ steps state action → result.modelOutcome ∈ outcomes
  observationCoverage : ∀ state action result value,
    result ∈ steps state action → value ∈ result.observations → value ∈ observations
  actionExecutable : ∀ action, action ∈ actions →
    ∃ state result, result ∈ steps state action

/-- Target metadata authored alongside a finite table; behavioral domains and setups come from
the table's successful validation branch. -/
structure FiniteTargetDefinition where
  id : DefinitionId
  source : SourceLocation
  definitions : List DefinitionMetadata
  requiredCapabilities : List DefinitionId
  metadata : KernelMetadata

/-- Finite admission keeps structural table failures separate from semantic Target failures. -/
inductive FiniteTargetAdmissionError where
  | invalidTable (error : FiniteTableError)
  | invalidTarget (diagnostic : AuthoringDiagnostic)

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
    TransitionKernel Setup State Action Outcome Observation := {
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
  behaviorDomain := .complete {
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
def kernelAvailability
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    KernelAvailability Setup State Action Outcome Observation :=
  .checked machine.kernel

/-- Derive finite planning from the same ordered action list and exact kernel relation. -/
def planning
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    FinitePlanningCapability machine.kernel.authoritativeStep := {
  actions := machine.actions
  actionSound := machine.actionExecutable
  actionComplete := machine.actionCoverage
}

/-- The dependent planning input consumed by `AuthoredTarget.make`. -/
def authoredPlanning
    (machine : FiniteMachine Setup State Action Outcome Observation) :
    AuthoredPlanningCapability machine.kernelAvailability :=
  .available machine.kernel rfl machine.planning

/-- Assemble ordinary Target metadata around the finite machine's exact setup list and checked
kernel. The machine remains the evidence boundary: authors still provide every domain, encoder,
enumerator, closure proof, and executable-action proof when constructing it.

This constructor calls only `kernelAvailability` (which calls `kernel`) and projects `setups`; both
functions assemble records without traversal, normalization, validation, or scanning. One
invocation therefore adds one record assembly regardless of domain size, and 1×/10× independent
declaration sets add exactly 1×/10× assemblies. -/
def targetDefinition
    (machine : FiniteMachine Setup State Action Outcome Observation)
    (id : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (requiredCapabilities : List DefinitionId) :
    TargetDefinition Setup State Action Outcome Observation := {
  id
  source
  definitions
  requiredCapabilities
  resolvedSetups := machine.setups
  kernel := machine.kernelAvailability
}

/-- Assemble an authored Target with explicit composition and occurrence inputs, while deriving the
dependent planning witness from the same finite machine. This calls `targetDefinition`,
`AuthoredTarget.make`, and `authoredPlanning`; transitively, `authoredPlanning` calls
`kernelAvailability`, `kernel`, and `planning`. Every call only assembles records or projects the
machine's existing values. The constructor adds no list traversal, normalization, validation, or
nested scan, and does not run `checkTarget`. One call adds one authored assembly per declaration, so
1×/10× independent declaration sets add exactly 1×/10× assemblies before unchanged checker work. -/
def authoredTarget
    (machine : FiniteMachine Setup State Action Outcome Observation)
    (id : DefinitionId)
    (source : SourceLocation)
    (definitions : List DefinitionMetadata)
    (requiredCapabilities : List DefinitionId)
    (composition : TargetComposition LawStatement)
    (occurrences : List AuthoringOccurrence) :
    AuthoredTarget LawStatement Setup State Action Outcome Observation :=
  AuthoredTarget.make
    (machine.targetDefinition id source definitions requiredCapabilities)
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
    (state : State) (action : Action) (result : TransitionResult State Outcome Observation) :
    machine.kernel.authoritativeStep state action result ↔
      result ∈ machine.steps state action :=
  Iff.rfl

end FiniteMachine

namespace ValidatedFiniteTable

private def encode [DecidableEq α] (catalog : FiniteCatalog α) (value : α) : String :=
  (catalog.encode? value).getD ""

private def initialStates [DecidableEq Setup]
    (table : FiniteTable Setup State Action Outcome Observation) (setup : Setup) : List State :=
  table.initial.flatMap fun row => if row.setup = setup then row.states else []

private def steps [DecidableEq State] [DecidableEq Action]
    (table : FiniteTable Setup State Action Outcome Observation)
    (state : State) (action : Action) : List (TransitionResult State Outcome Observation) :=
  table.transitions.flatMap fun row =>
    if row.source = state ∧ row.action = action then row.results else []

/-- Derive every finite enumerator and mechanical witness from the successfully validated rows. -/
def machine [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Observation]
    (validated : ValidatedFiniteTable Setup State Action Outcome Observation)
    (metadata : KernelMetadata) : FiniteMachine Setup State Action Outcome Observation := {
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
def authoredTarget [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Observation]
    (validated : ValidatedFiniteTable Setup State Action Outcome Observation)
    (definition : FiniteTargetDefinition)
    (composition : TargetComposition LawStatement := .empty) :
    AuthoredTarget LawStatement Setup State Action Outcome Observation :=
  let machine := validated.machine definition.metadata
  AuthoredTarget.make {
    id := definition.id
    source := definition.source
    definitions := definition.definitions
    requiredCapabilities := definition.requiredCapabilities
    resolvedSetups := machine.setups
    kernel := machine.kernelAvailability
  } composition machine.authoredPlanning

end ValidatedFiniteTable

/-- A structurally validated typed table paired with its explicit ModelValue identities. -/
structure ValidatedFiniteModel (Setup State Action Outcome Fact : Type) where
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
    (result : TransitionResult State Outcome Fact) :
    Except FiniteTableError (TransitionResult ModelValue ModelValue ModelValue) := do
  let resultingState ← modelValue table.states identity.stateId .resultState result.resultingState
  let modelOutcome ← modelValue table.outcomes identity.outcomeId .outcome result.modelOutcome
  let observations ← result.observations.mapM fun fact =>
    modelValue table.facts identity.factId .fact fact
  pure { resultingState, modelOutcome, observations }

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
  }

/-- Validate the typed catalogs and rows before exposing stable-key value resolution. -/
def validateModel [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact) :
    Except FiniteTableError (ValidatedFiniteModel Setup State Action Outcome Fact) := do
  let validated ← table.validate
  pure ⟨validated.table, identity⟩

end FiniteTable

namespace ValidatedFiniteModel

/-- Resolve one authored state through its validated catalog key and explicit field identity. -/
def stateValue
    [DecidableEq State]
    (model : ValidatedFiniteModel Setup State Action Outcome Fact)
    (state : State) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.states model.identity.stateId .state state

/-- Resolve one authored Action through its validated catalog key and explicit field identity. -/
def actionValue
    [DecidableEq Action]
    (model : ValidatedFiniteModel Setup State Action Outcome Fact)
    (action : Action) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.actions model.identity.actionId .action action

/-- Resolve one authored Model Outcome through its catalog key and explicit field identity. -/
def outcomeValue
    [DecidableEq Outcome]
    (model : ValidatedFiniteModel Setup State Action Outcome Fact)
    (outcome : Outcome) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.outcomes model.identity.outcomeId .outcome outcome

/-- Resolve one authored fact through its validated catalog key and explicit field identity. -/
def factValue
    [DecidableEq Fact]
    (model : ValidatedFiniteModel Setup State Action Outcome Fact)
    (fact : Fact) : Except FiniteTableError ModelValue :=
  FiniteTable.modelValue model.table.facts model.identity.factId .fact fact

/-- Lower one authored setup using its explicit role bindings and the state catalog. -/
def setupValue
    [DecidableEq Setup] [DecidableEq State]
    (model : ValidatedFiniteModel Setup State Action Outcome Fact)
    (setup : Setup) : Except FiniteTableError (List RoleBinding) :=
  if setup ∈ model.table.setups.values then
    FiniteTable.modelSetup model.table model.identity setup
  else
    .error (.outOfDomain .setup)

end ValidatedFiniteModel

namespace FiniteTable

/-- Validate typed author data before lowering its stable keys to the existing ModelValue Target. -/
def checkModelTarget [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (definition : FiniteTargetDefinition)
    (composition : TargetComposition LawStatement := .empty) :
    Except FiniteTargetAdmissionError
      (CheckedTarget LawStatement (List RoleBinding) ModelValue ModelValue ModelValue ModelValue) := do
  let _ ← table.validateModel identity |>.mapError FiniteTargetAdmissionError.invalidTable
  let lowered ← modelTable table identity |>.mapError FiniteTargetAdmissionError.invalidTable
  let validated ← FiniteTable.validate
    (Setup := List RoleBinding)
    (State := ModelValue)
    (Action := ModelValue)
    (Outcome := ModelValue)
    (Fact := ModelValue)
    lowered |>.mapError FiniteTargetAdmissionError.invalidTable
  Umpire.checkTarget (validated.authoredTarget definition composition)
    |>.mapError FiniteTargetAdmissionError.invalidTarget

/-- Validate once, then admit semantic composition through the existing Target checker. -/
def checkTarget [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Observation]
    (table : FiniteTable Setup State Action Outcome Observation)
    (definition : FiniteTargetDefinition)
    (composition : TargetComposition LawStatement := .empty) :
    Except FiniteTargetAdmissionError
      (CheckedTarget LawStatement Setup State Action Outcome Observation) := do
  let validated ← table.validate |>.mapError FiniteTargetAdmissionError.invalidTable
  Umpire.checkTarget (validated.authoredTarget definition composition)
    |>.mapError FiniteTargetAdmissionError.invalidTarget

end FiniteTable

end Umpire
