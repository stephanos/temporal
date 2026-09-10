import Umpire.Target.Tests.Fixtures

/-! Contract tests for complete finite-machine Target authoring. -/

namespace Umpire.TargetTests.FiniteMachine

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
  metadata := TargetTests.testKernel.metadata
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

def definition : TargetDefinition Unit Bool Bool Bool Bool := {
  id := TargetTests.testTarget.id
  source := TargetTests.testTarget.source
  definitions := TargetTests.testTarget.definitions
  requiredCapabilities := []
  resolvedSetups := [()]
  kernel := machine.machineAvailability
}

def authored : AuthoredTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool :=
  AuthoredTarget.make definition (planning := machine.authoredPlanning)

def assemblyOccurrence : AuthoringOccurrence := {
  id := {
    sourcePath := "Umpire/Target/Tests/FiniteMachine.lean"
    line := 1
    column := 0
    endLine := 1
    endColumn := 1
    localOrdinal := 0
  }
  definitionId := TargetTests.testTarget.id
  path := {
    role := .targetDefinition
    owner := TargetTests.testTarget.id
  }
}

def explicitAssemblyDefinition : TargetDefinition Unit Bool Bool Bool Bool := {
  id := TargetTests.testTarget.id
  source := TargetTests.testTarget.source
  definitions := TargetTests.testTarget.definitions
  requiredCapabilities := TargetTests.testTarget.requiredCapabilities
  resolvedSetups := machine.setups
  kernel := machine.machineAvailability
}

def explicitAssembly : AuthoredTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool :=
  AuthoredTarget.make
    explicitAssemblyDefinition
    (TargetTests.targetCompositionOf TargetTests.testTarget)
    machine.authoredPlanning
    [assemblyOccurrence]

def assembledDefinition : TargetDefinition Unit Bool Bool Bool Bool :=
  machine.targetDefinition
    TargetTests.testTarget.id
    TargetTests.testTarget.source
    TargetTests.testTarget.definitions
    TargetTests.testTarget.requiredCapabilities

def assembledAuthored : AuthoredTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool :=
  machine.authoredTarget
    TargetTests.testTarget.id
    TargetTests.testTarget.source
    TargetTests.testTarget.definitions
    TargetTests.testTarget.requiredCapabilities
    (TargetTests.targetCompositionOf TargetTests.testTarget)
    [assemblyOccurrence]

/-- The smaller seam preserves the exact old definition and complete authored/checker inputs. -/
example : assembledDefinition = explicitAssemblyDefinition ∧ assembledAuthored = explicitAssembly ∧
    checkTarget assembledAuthored = checkTarget explicitAssembly := by
  exact ⟨rfl, rfl, rfl⟩

example : (checkTarget assembledAuthored).isOk = true := by
  native_decide

/-- Ordinary Target authoring accepts the exact kernel and dependent planning derived here. -/
example : (checkTarget authored).isOk = true := by
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
  metadata := TargetTests.testKernel.metadata
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
    List (AuthoredTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool) :=
  (List.range count).map fun _ => machine.authoredTarget
    TargetTests.testTarget.id
    TargetTests.testTarget.source
    TargetTests.testTarget.definitions
    []
    TargetComposition.empty
    []

/-- Independent declarations perform one constructor assembly each, without checker work. -/
example : (assembledTargets 1).length = 1 ∧ (assembledTargets 10).length = 10 := by
  decide

def collidingEncodingMachine : FiniteMachine Unit Bool Bool Bool Bool := {
  machine with encodeState := fun _ => "state"
}

def collidingEncodingDefinition : TargetDefinition Unit Bool Bool Bool Bool := {
  definition with kernel := collidingEncodingMachine.machineAvailability
}

def collidingEncodingAuthoring :
    AuthoredTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool :=
  AuthoredTarget.make collidingEncodingDefinition

def collidingEncodingErrorKind : Option DefinitionErrorKind :=
  match checkTarget collidingEncodingAuthoring with
  | .ok _ => none
  | .error diagnostic => some diagnostic.error.kind

/-- The adapter retains the existing typed diagnostic for colliding canonical encodings. -/
example : collidingEncodingErrorKind = some .incompleteBehaviorDomain := by
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

def definition : FiniteTargetDefinition := {
  id := TargetTests.testTarget.id
  source := TargetTests.testTarget.source
  definitions := TargetTests.testTarget.definitions
  requiredCapabilities := TargetTests.testTarget.requiredCapabilities
  metadata := TargetTests.testKernel.metadata
}

def composition : TargetComposition TargetTests.TestLawStatement :=
  TargetTests.targetCompositionOf TargetTests.testTarget

def admitted := FiniteTable.checkTarget extendedTable definition composition

#guard (FiniteTable.checkTarget
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

#guard (FiniteTable.checkModelTarget
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
  encodedInitial : List TargetInitialStateRow
  encodedTransitions : List TargetTransitionRow
  deriving DecidableEq

def checkedEnumeration? : Option CheckedEnumeration :=
  match admitted with
  | .error _ => none
  | .ok target =>
    match target.kernel.behaviorDomain with
    | .missing => none
    | .incomplete _ => none
    | .complete domain => some {
        setups := domain.setups
        states := domain.states
        actions := domain.actions
        outcomes := domain.outcomes
        facts := domain.observations
        initial := target.kernel.initialStates ()
        idleResults := target.kernel.steps .idle true
        runningResults := target.kernel.steps .running true
        completedResults := target.kernel.steps .completed true
        plannedActions := match target.planning with
          | .unavailable => []
          | .available planning => planning.actions
        encodedSetups := target.behaviorDescription.setups
        encodedStates := target.behaviorDescription.states
        encodedActions := target.behaviorDescription.actions
        encodedOutcomes := target.behaviorDescription.outcomes
        encodedFacts := target.behaviorDescription.observations
        encodedInitial := target.behaviorDescription.initialStates
        encodedTransitions := target.behaviorDescription.transitions
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
  match FiniteTable.checkTarget { extendedTable with transitions := [] } definition composition with
  | .error (.invalidTable error) => some error
  | _ => none

example : structuralFailure? = some .actionWithoutRow := by
  native_decide

def semanticFailureKind : Option DefinitionErrorKind :=
  match FiniteTable.checkTarget extendedTable definition
      (TargetComposition.empty : TargetComposition TargetTests.TestLawStatement) with
  | .error (FiniteTargetAdmissionError.invalidTarget diagnostic) => some diagnostic.error.kind
  | _ => none

example : semanticFailureKind = some .missingProvider := by
  native_decide

def conflictingComposition : TargetComposition TargetTests.TestLawStatement :=
  TargetComposition.empty
    |>.provide TargetTests.primaryProvider
    |>.provide TargetTests.secondaryProvider

def conflictingFailureKind : Option DefinitionErrorKind :=
  match FiniteTable.checkTarget extendedTable definition conflictingComposition with
  | .error (FiniteTargetAdmissionError.invalidTarget diagnostic) => some diagnostic.error.kind
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
  FiniteTable.checkTarget (linearTable stateCount) definition composition

/-- The same generic admission path handles the baseline and ten-times-larger table. -/
example : (linearAdmission 3).isOk = true ∧ (linearAdmission 30).isOk = true := by
  native_decide

#print axioms Umpire.ValidatedFiniteTable.machine
#print axioms Umpire.ValidatedFiniteTable.authoredTarget
#print axioms Umpire.FiniteTable.checkTarget
#print axioms Umpire.FiniteMachine.targetDefinition
#print axioms Umpire.FiniteMachine.authoredTarget

end Admission

end Umpire.TargetTests.FiniteMachine
