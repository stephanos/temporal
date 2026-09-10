import Umpire.Model.Tests.Fixtures

/-! Contract tests for complete finite-machine Target authoring. -/

namespace Umpire.ModelTests.FiniteMachine

open Umpire

def transition (state action : Bool) : Step Bool Bool Bool := {
  outcome := action
  state := action
  facts := [state]
}

def alternateTransition (state action : Bool) : Step Bool Bool Bool := {
  outcome := !action
  state := state
  facts := [action]
}

def machine : FiniteMachine Unit Bool Bool Bool Bool := {
  metadata := ModelTests.testKernel.metadata
  setups := [()]
  states := [false, true]
  actions := [true, false]
  outcomes := [false, true]
  observations := [false, true]
  encodeSetup := fun _ => "unit"
  encodeState := toString
  encodeAction := toString
  encodeOutcome := toString
  encodeObservation := toString
  initialStates := fun _ => [false]
  steps := fun state action => [transition state action, alternateTransition state action]
  setupCoverage := by intro setup; cases setup; simp
  initialStateCoverage := by intro _ state _; cases state <;> simp
  transitionSourceCoverage := by intro state _ _ _; cases state <;> simp
  actionCoverage := by intro _ action _ _; cases action <;> simp
  resultingStateCoverage := by intro _ _ result _; cases result.state <;> simp
  outcomeCoverage := by intro _ _ result _; cases result.outcome <;> simp
  observationCoverage := by intro _ _ _ value _ _; cases value <;> simp
  actionExecutable := by
    intro action _
    exact ⟨false, transition false action, by simp⟩
}

def definition : ModelSpec ModelTests.TestLawStatement Unit Bool Bool Bool Bool := {
  id := ModelTests.testTarget.id
  source := ModelTests.testTarget.source
  definitions := ModelTests.testTarget.definitions
  requiredCapabilities := []
  resolvedSetups := [()]
  machine := machine.machineAvailability
}

def authored : DraftModel ModelTests.TestLawStatement Unit Bool Bool Bool Bool :=
  DraftModel.make definition (planning := machine.authoredPlanning)

def assemblyOccurrence : SourceRef := {
  id := {
    sourcePath := "Umpire/Model/Tests/FiniteMachine.lean"
    line := 1
    column := 0
    endLine := 1
    endColumn := 1
    localOrdinal := 0
  }
  definitionId := ModelTests.testTarget.id
  path := {
    role := .modelSpec
    owner := ModelTests.testTarget.id
  }
}

def explicitAssemblyDefinition : ModelSpec ModelTests.TestLawStatement Unit Bool Bool Bool Bool := {
  id := ModelTests.testTarget.id
  source := ModelTests.testTarget.source
  definitions := ModelTests.testTarget.definitions
  requiredCapabilities := ModelTests.testTarget.requiredCapabilities
  resolvedSetups := machine.setups
  machine := machine.machineAvailability
}

def explicitAssembly : DraftModel ModelTests.TestLawStatement Unit Bool Bool Bool Bool :=
  DraftModel.make
    explicitAssemblyDefinition
    (ModelTests.providersOf ModelTests.testTarget)
    machine.authoredPlanning
    [assemblyOccurrence]

def assembledDefinition : ModelSpec ModelTests.TestLawStatement Unit Bool Bool Bool Bool :=
  machine.modelSpec
    ModelTests.testTarget.id
    ModelTests.testTarget.source
    ModelTests.testTarget.definitions
    ModelTests.testTarget.requiredCapabilities

def assembledAuthored : DraftModel ModelTests.TestLawStatement Unit Bool Bool Bool Bool :=
  machine.draftModel
    ModelTests.testTarget.id
    ModelTests.testTarget.source
    ModelTests.testTarget.definitions
    ModelTests.testTarget.requiredCapabilities
    (ModelTests.providersOf ModelTests.testTarget)
    [assemblyOccurrence]

/-- The smaller seam preserves the exact old definition and complete authored/checker inputs. -/
example : assembledDefinition = explicitAssemblyDefinition ∧ assembledAuthored = explicitAssembly ∧
    checkModel assembledAuthored = checkModel explicitAssembly := by
  exact ⟨rfl, rfl, rfl⟩

example : (checkModel assembledAuthored).isOk = true := by
  native_decide

/-- Ordinary Target authoring accepts the exact kernel and dependent planning derived here. -/
example : (checkModel authored).isOk = true := by
  native_decide

example : machine.kernel.initialStates () = [false] := rfl

/-- Stable public rewrites expose membership authority without private adapter unfolding. -/
example (setup : Unit) (state action outcome observation : Bool)
    (result : Step Bool Bool Bool) :
    (machine.kernel.setupDomain setup ↔ setup ∈ machine.setups) ∧
    (machine.kernel.stateDomain state ↔ state ∈ machine.states) ∧
    (machine.kernel.actionDomain action ↔ action ∈ machine.actions) ∧
    (machine.kernel.outcomeDomain outcome ↔ outcome ∈ machine.outcomes) ∧
    (machine.kernel.observationDomain observation ↔ observation ∈ machine.observations) ∧
    (machine.kernel.authoritativeInitial setup state ↔ state ∈ machine.initialStates setup) ∧
    (machine.kernel.authoritativeStep state action result ↔ result ∈ machine.steps state action) := by
  simp

/-- Authored action order passes through to finite planning without normalization. -/
example : machine.planning.actions = [true, false] := rfl

/-- Multiple transition results retain their authored enumeration order. -/
example : machine.kernel.steps false true =
    [transition false true, alternateTransition false true] := rfl

def emptyMachine : FiniteMachine Empty Empty Empty Empty Empty := {
  metadata := ModelTests.testKernel.metadata
  setups := []
  states := []
  actions := []
  outcomes := []
  observations := []
  encodeSetup := fun setup => nomatch setup
  encodeState := fun state => nomatch state
  encodeAction := fun action => nomatch action
  encodeOutcome := fun outcome => nomatch outcome
  encodeObservation := fun observation => nomatch observation
  initialStates := fun setup => nomatch setup
  steps := fun state => nomatch state
  setupCoverage := by intro setup; exact nomatch setup
  initialStateCoverage := by intro setup; exact nomatch setup
  transitionSourceCoverage := by intro state; exact nomatch state
  actionCoverage := by intro state; exact nomatch state
  resultingStateCoverage := by intro state; exact nomatch state
  outcomeCoverage := by intro state; exact nomatch state
  observationCoverage := by intro state; exact nomatch state
  actionExecutable := by intro action; exact nomatch action
}

/-- Empty proof-valid domains produce a complete checked kernel and vacuous planning. -/
example : emptyMachine.machineAvailability = .checked emptyMachine.kernel := rfl

example : emptyMachine.planning.actions = [] := rfl

/-- Emitting an undeclared initial state leaves an unsatisfiable closure obligation. -/
example : ¬ (∀ state : Bool, state ∈ [true] → state ∈ [false]) := by
  simp

/-- Advertising an unreachable action leaves an unsatisfiable executable-action obligation. -/
example : ¬ (∀ action : Bool, action ∈ [true] →
    ∃ (_state : Bool) (result : Step Bool Bool Bool),
      result ∈ ([] : List (Step Bool Bool Bool))) := by
  simp

#guard_msgs (error, substring := true) in
def missingInitialStateCoverage : FiniteMachine Unit Bool Bool Bool Bool := {
  metadata := machine.metadata
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
  initialStates := machine.initialStates
  steps := machine.steps
  setupCoverage := machine.setupCoverage
  transitionSourceCoverage := machine.transitionSourceCoverage
  actionCoverage := machine.actionCoverage
  resultingStateCoverage := machine.resultingStateCoverage
  outcomeCoverage := machine.outcomeCoverage
  observationCoverage := machine.observationCoverage
  actionExecutable := machine.actionExecutable
}

#guard_msgs (error, substring := true) in
def missingActionExecutable : FiniteMachine Unit Bool Bool Bool Bool := {
  metadata := machine.metadata
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
  initialStates := machine.initialStates
  steps := machine.steps
  setupCoverage := machine.setupCoverage
  initialStateCoverage := machine.initialStateCoverage
  transitionSourceCoverage := machine.transitionSourceCoverage
  actionCoverage := machine.actionCoverage
  resultingStateCoverage := machine.resultingStateCoverage
  outcomeCoverage := machine.outcomeCoverage
  observationCoverage := machine.observationCoverage
}

def assembledTargets (count : Nat) :
    List (DraftModel ModelTests.TestLawStatement Unit Bool Bool Bool Bool) :=
  (List.range count).map fun _ => machine.draftModel
    ModelTests.testTarget.id
    ModelTests.testTarget.source
    ModelTests.testTarget.definitions
    []
    Providers.empty
    []

/-- Independent declarations perform one constructor assembly each, without checker work. -/
example : (assembledTargets 1).length = 1 ∧ (assembledTargets 10).length = 10 := by
  decide

def collidingEncodingMachine : FiniteMachine Unit Bool Bool Bool Bool := {
  machine with encodeState := fun _ => "state"
}

def collidingEncodingDefinition : ModelSpec ModelTests.TestLawStatement Unit Bool Bool Bool Bool := {
  definition with machine := collidingEncodingMachine.machineAvailability
}

def collidingEncodingAuthoring :
    DraftModel ModelTests.TestLawStatement Unit Bool Bool Bool Bool :=
  DraftModel.make collidingEncodingDefinition

def collidingEncodingErrorKind : Option DefinitionErrorKind :=
  match checkModel collidingEncodingAuthoring with
  | .ok _ => none
  | .error diagnostic => some diagnostic.error.kind

/-- The adapter retains the existing typed diagnostic for colliding canonical encodings. -/
example : collidingEncodingErrorKind = some .incompleteVocabulary := by
  native_decide

namespace Admission

inductive State where
  | idle
  | running
  | completed
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | started
  | completed
  deriving BEq, DecidableEq, Repr

inductive Fact where
  | started
  | completed
  deriving BEq, DecidableEq, Repr

def startResult : Step State Outcome Fact := {
  outcome := .started
  state := .running
  facts := [.started]
}

def completeResult : Step State Outcome Fact := {
  outcome := .completed
  state := .completed
  facts := [.completed]
}

def retryResult : Step State Outcome Fact := {
  outcome := .started
  state := .running
  facts := [.started]
}

def baseTable : FiniteTable Unit State Bool Outcome Fact := {
  setups := [⟨(), "default"⟩]
  states := [⟨.idle, "idle"⟩, ⟨.running, "running"⟩]
  actions := [⟨true, "advance"⟩]
  outcomes := [⟨.started, "started"⟩, ⟨.completed, "completed"⟩]
  facts := [⟨.started, "started"⟩, ⟨.completed, "completed"⟩]
  initial := [⟨(), [.idle]⟩]
  transitions := [⟨"start", .idle, true, [startResult]⟩]
}

/-- The feature edit is only a state entry and one row; generic admission owns every proof. -/
def extendedTable : FiniteTable Unit State Bool Outcome Fact := {
  baseTable with
  states := baseTable.states ++ [⟨.completed, "completed"⟩]
  transitions := baseTable.transitions ++ [
    ⟨"finish-or-retry", .running, true, [completeResult, retryResult]⟩
  ]
}

def definition : TableModelSpec := {
  id := ModelTests.testTarget.id
  source := ModelTests.testTarget.source
  definitions := ModelTests.testTarget.definitions
  requiredCapabilities := ModelTests.testTarget.requiredCapabilities
  metadata := ModelTests.testKernel.metadata
}

def composition : Providers ModelTests.TestLawStatement :=
  ModelTests.providersOf ModelTests.testTarget

def admitted := FiniteTable.checkTypedModel extendedTable definition composition

#guard (FiniteTable.checkTypedModel
    { extendedTable with terminalConditions := [[.completed], [.running, .completed]] }
    definition composition).toOption.map (fun target =>
      (target.isTerminal .running, target.isTerminal .completed)) == some (false, true)

private def terminalIdentity : FiniteModelIdentity Unit State Bool Outcome Fact := {
  setupBindings := fun _ => []
  stateId := fun _ => DefinitionId.of "test.state.finite"
  actionId := fun _ => DefinitionId.of "test.action.finite"
  outcomeId := fun _ => DefinitionId.of "test.outcome.finite"
  factId := fun _ => DefinitionId.of "test.observation.finite"
}

#guard (FiniteTable.checkModel
    { extendedTable with terminalConditions := [[.completed], [.running, .completed]] }
    terminalIdentity definition composition).toOption.map (fun target =>
      (target.isTerminal (.named (DefinitionId.of "test.state.finite") "running"),
        target.isTerminal (.named (DefinitionId.of "test.state.finite") "completed"))) ==
    some (false, true)

structure CheckedEnumeration where
  setups : List Unit
  states : List State
  actions : List Bool
  outcomes : List Outcome
  facts : List Fact
  initial : List State
  idleResults : List (Step State Outcome Fact)
  runningResults : List (Step State Outcome Fact)
  completedResults : List (Step State Outcome Fact)
  plannedActions : List Bool
  encodedSetups : List String
  encodedStates : List String
  encodedActions : List String
  encodedOutcomes : List String
  encodedFacts : List String
  encodedInitial : List BehaviorInitialStateRow
  encodedTransitions : List BehaviorTransitionRow
  deriving DecidableEq

def checkedEnumeration? : Option CheckedEnumeration :=
  match admitted with
  | .error _ => none
  | .ok target =>
    match target.machine.vocabulary with
    | .missing => none
    | .incomplete _ => none
    | .complete domain => some {
        setups := domain.setups
        states := domain.states
        actions := domain.actions
        outcomes := domain.outcomes
        facts := domain.observations
        initial := target.machine.initialStates ()
        idleResults := target.machine.steps .idle true
        runningResults := target.machine.steps .running true
        completedResults := target.machine.steps .completed true
        plannedActions := match target.planning with
          | .unavailable => []
          | .available planning => planning.actions
        encodedSetups := target.behaviorTable.setups
        encodedStates := target.behaviorTable.states
        encodedActions := target.behaviorTable.actions
        encodedOutcomes := target.behaviorTable.outcomes
        encodedFacts := target.behaviorTable.observations
        encodedInitial := target.behaviorTable.initialStates
        encodedTransitions := target.behaviorTable.transitions
      }

/-- The successful branch exposes every authored enumeration and the canonical checked view. -/
example : checkedEnumeration? = some {
    setups := [()]
    states := [.idle, .running, .completed]
    actions := [true]
    outcomes := [.started, .completed]
    facts := [.started, .completed]
    initial := [.idle]
    idleResults := [startResult]
    runningResults := [completeResult, retryResult]
    completedResults := []
    plannedActions := [true]
    encodedSetups := ["default"]
    encodedStates := ["completed", "idle", "running"]
    encodedActions := ["advance"]
    encodedOutcomes := ["completed", "started"]
    encodedFacts := ["completed", "started"]
    encodedInitial := [⟨"default", "idle"⟩]
    encodedTransitions := [
      ⟨"idle", "advance", "started", "running", ["started"]⟩,
      ⟨"running", "advance", "completed", "completed", ["completed"]⟩,
      ⟨"running", "advance", "started", "running", ["started"]⟩
    ]
  } := by
  native_decide

/-- Raw tables and checked Targets remain observably distinct on both failure layers. -/
def structuralFailure? : Option FiniteTableError :=
  match FiniteTable.checkTypedModel { extendedTable with transitions := [] } definition composition with
  | .error (.invalidTable error) => some error
  | _ => none

example : structuralFailure? = some .actionWithoutRow := by
  native_decide

def semanticFailureKind : Option DefinitionErrorKind :=
  match FiniteTable.checkTypedModel extendedTable definition
      (Providers.empty : Providers ModelTests.TestLawStatement) with
  | .error (TableAdmissionError.invalidTarget diagnostic) => some diagnostic.error.kind
  | _ => none

example : semanticFailureKind = some .missingProvider := by
  native_decide

def conflictingComposition : Providers ModelTests.TestLawStatement :=
  Providers.empty
    |>.provide ModelTests.primaryProvider
    |>.provide ModelTests.secondaryProvider

def conflictingFailureKind : Option DefinitionErrorKind :=
  match FiniteTable.checkTypedModel extendedTable definition conflictingComposition with
  | .error (TableAdmissionError.invalidTarget diagnostic) => some diagnostic.error.kind
  | _ => none

example : conflictingFailureKind = some .conflictingProviders := by
  native_decide

def linearTable (stateCount : Nat) : FiniteTable Unit Nat Unit Unit Unit := {
  setups := [⟨(), "default"⟩]
  states := (List.range stateCount).map fun state => ⟨state, s!"state-{state}"⟩
  actions := [⟨(), "advance"⟩]
  outcomes := [⟨(), "advanced"⟩]
  facts := [⟨(), "advanced"⟩]
  initial := [⟨(), [0]⟩]
  transitions := (List.range stateCount).map fun state =>
    ⟨s!"step-{state}", state, (), [⟨(), (state + 1) % stateCount, [()]⟩]⟩
}

def linearAdmission (stateCount : Nat) :=
  FiniteTable.checkTypedModel (linearTable stateCount) definition composition

/-- The same generic admission path handles the baseline and ten-times-larger table. -/
example : (linearAdmission 3).isOk = true ∧ (linearAdmission 30).isOk = true := by
  native_decide

#print axioms Umpire.CheckedTable.machine
#print axioms Umpire.CheckedTable.draftModel
#print axioms Umpire.FiniteTable.checkTypedModel
#print axioms Umpire.FiniteMachine.modelSpec
#print axioms Umpire.FiniteMachine.draftModel

end Admission

end Umpire.ModelTests.FiniteMachine
