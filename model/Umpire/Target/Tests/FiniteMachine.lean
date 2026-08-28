import Umpire.Target.Tests.Fixtures

/-! Contract tests for complete finite-machine Target authoring. -/

namespace Umpire.TargetTests.FiniteMachine

open Umpire

def transition (state action : Bool) : TransitionResult Bool Bool Bool := {
  modelOutcome := action
  resultingState := action
  observations := [state]
}

def alternateTransition (state action : Bool) : TransitionResult Bool Bool Bool := {
  modelOutcome := !action
  resultingState := state
  observations := [action]
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
  resultingStateCoverage := by intro _ _ result _; cases result.resultingState <;> simp
  outcomeCoverage := by intro _ _ result _; cases result.modelOutcome <;> simp
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
  kernel := machine.kernelAvailability
}

def authored : AuthoredTarget TargetTests.TestLawStatement Unit Bool Bool Bool Bool :=
  AuthoredTarget.make definition (planning := machine.authoredPlanning)

/-- Ordinary Target authoring accepts the exact kernel and dependent planning derived here. -/
example : (checkTarget authored).isOk = true := by
  native_decide

example : machine.kernel.initialStates () = [false] := rfl

/-- Stable public rewrites expose membership authority without private adapter unfolding. -/
example (setup : Unit) (state action outcome observation : Bool)
    (result : TransitionResult Bool Bool Bool) :
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
example : emptyMachine.kernelAvailability = .checked emptyMachine.kernel := rfl

example : emptyMachine.planning.actions = [] := rfl

/-- Emitting an undeclared initial state leaves an unsatisfiable closure obligation. -/
example : ¬ (∀ state : Bool, state ∈ [true] → state ∈ [false]) := by
  simp

/-- Advertising an unreachable action leaves an unsatisfiable executable-action obligation. -/
example : ¬ (∀ action : Bool, action ∈ [true] →
    ∃ (_state : Bool) (result : TransitionResult Bool Bool Bool),
      result ∈ ([] : List (TransitionResult Bool Bool Bool))) := by
  simp

def collidingEncodingMachine : FiniteMachine Unit Bool Bool Bool Bool := {
  machine with encodeState := fun _ => "state"
}

def collidingEncodingDefinition : TargetDefinition Unit Bool Bool Bool Bool := {
  definition with kernel := collidingEncodingMachine.kernelAvailability
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

end Umpire.TargetTests.FiniteMachine
