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

end Umpire
