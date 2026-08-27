import Umpire.Target

/-!
# Temporal Nexus system lifecycle

Pure mechanism meaning for dispatching, canceling, and completing one Nexus operation. This module
does not import Feature meaning or runtime/evidence adapters; the focused Implementation Link leaf
owns the correspondence to the product lifecycle.
-/

namespace Temporal.System.Nexus

open Umpire

private def id (value : String) : DefinitionId := DefinitionId.of value

def source : SourceLocation := {
  path := "Temporal/System/Nexus/Core.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId : DefinitionId := id "temporal.system.nexus.lifecycle.target"
def kernelId : DefinitionId := id "temporal.system.nexus.lifecycle.kernel"
def lifecycleCapabilityId : DefinitionId := id "temporal.system.nexus.lifecycle.capability"
def lifecycleProviderId : DefinitionId := id "temporal.system.nexus.lifecycle.provider"
def lifecycleLawId : DefinitionId := id "temporal.system.nexus.lifecycle.law.authoritative-step"
def operationStateId : DefinitionId := id "temporal.system.nexus.lifecycle.state.operation"
def dispatchActionId : DefinitionId := id "temporal.system.nexus.lifecycle.action.dispatch"
def recordCancellationActionId : DefinitionId :=
  id "temporal.system.nexus.lifecycle.action.record-cancellation"
def recordCompletionActionId : DefinitionId :=
  id "temporal.system.nexus.lifecycle.action.record-completion"
def transitionOutcomeId : DefinitionId := id "temporal.system.nexus.lifecycle.outcome.transition"
def lifecycleObservationId : DefinitionId :=
  id "temporal.system.nexus.lifecycle.observation.state"

/-- The mechanism-owned states required for the ordinary Nexus lifecycle. -/
inductive ExecutionState where
  | queued
  | running
  | cancellationRecorded
  | completionRecorded
  deriving BEq, DecidableEq, Repr

/-- The two supported mechanism entry points for one Nexus operation. -/
inductive ExecutionSetup where
  | queued
  | running
  deriving BEq, DecidableEq, Repr

/-- The mechanism events that advance the ordinary Nexus lifecycle. -/
inductive ExecutionEvent where
  | dispatch
  | recordCancellation
  | recordCompletion
  deriving BEq, DecidableEq, Repr

/-- The complete pure mechanism transition relation. -/
def step : ExecutionState → ExecutionEvent → Option ExecutionState
  | .queued, .dispatch => some .running
  | .running, .recordCancellation => some .cancellationRecorded
  | .running, .recordCompletion => some .completionRecorded
  | _, _ => none

def lifecycleLaw : LawDefinition := {
  id := lifecycleLawId
  body := "temporal-system-nexus-lifecycle-authoritative-step/v1"
}

/-- The provider law binds checked System meaning to the pure mechanism transition relation. -/
def LawStatement (law : LawDefinition) : Prop :=
  law = lifecycleLaw ∧
    step .queued .dispatch = some .running ∧
    step .running .recordCancellation = some .cancellationRecorded ∧
    step .running .recordCompletion = some .completionRecorded

theorem lifecycleLawProof : LawStatement lifecycleLaw := by
  exact ⟨rfl, rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata := {
  id := definitionId
  kind
  source
  canonicalBehavior
}

def queuedState : ModelValue := { definitionId := operationStateId, value := "queued" }
def runningState : ModelValue := { definitionId := operationStateId, value := "running" }
def cancellationRecordedState : ModelValue := {
  definitionId := operationStateId
  value := "cancellation-recorded"
}
def completionRecordedState : ModelValue := {
  definitionId := operationStateId
  value := "completion-recorded"
}

def dispatchAction : ModelValue := { definitionId := dispatchActionId, value := "dispatch" }
def recordCancellationAction : ModelValue := {
  definitionId := recordCancellationActionId
  value := "record-cancellation"
}
def recordCompletionAction : ModelValue := {
  definitionId := recordCompletionActionId
  value := "record-completion"
}

def dispatchedOutcome : ModelValue := { definitionId := transitionOutcomeId, value := "running" }
def cancellationRecordedOutcome : ModelValue := {
  definitionId := transitionOutcomeId
  value := "cancellation-recorded"
}
def completionRecordedOutcome : ModelValue := {
  definitionId := transitionOutcomeId
  value := "completion-recorded"
}

def runningObservation : ModelValue := {
  definitionId := lifecycleObservationId
  value := "running"
}
def cancellationRecordedObservation : ModelValue := {
  definitionId := lifecycleObservationId
  value := "cancellation-recorded"
}
def completionRecordedObservation : ModelValue := {
  definitionId := lifecycleObservationId
  value := "completion-recorded"
}

def queuedSetup : ExecutionSetup := .queued
def runningSetup : ExecutionSetup := .running

def dispatchedResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := dispatchedOutcome
  resultingState := runningState
  observations := [runningObservation]
}

def cancellationRecordedResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := cancellationRecordedOutcome
  resultingState := cancellationRecordedState
  observations := [cancellationRecordedObservation]
}

def completionRecordedResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := completionRecordedOutcome
  resultingState := completionRecordedState
  observations := [completionRecordedObservation]
}

private def executionState? (state : ModelValue) : Option ExecutionState :=
  if state = queuedState then
    some .queued
  else if state = runningState then
    some .running
  else if state = cancellationRecordedState then
    some .cancellationRecorded
  else if state = completionRecordedState then
    some .completionRecorded
  else
    none

private def executionEvent? (action : ModelValue) : Option ExecutionEvent :=
  if action = dispatchAction then
    some .dispatch
  else if action = recordCancellationAction then
    some .recordCancellation
  else if action = recordCompletionAction then
    some .recordCompletion
  else
    none

private def transitionResult? : ExecutionState → Option
    (TransitionResult ModelValue ModelValue ModelValue)
  | .running => some dispatchedResult
  | .cancellationRecorded => some cancellationRecordedResult
  | .completionRecorded => some completionRecordedResult
  | .queued => none

def initialState? (setup : ExecutionSetup) : Option ModelValue :=
  if setup = queuedSetup then
    some queuedState
  else if setup = runningSetup then
    some runningState
  else
    none

def initialStates (setup : ExecutionSetup) : List ModelValue :=
  (initialState? setup).toList

def stepResult? (state action : ModelValue) : Option
    (TransitionResult ModelValue ModelValue ModelValue) := do
  let executionState ← executionState? state
  let executionEvent ← executionEvent? action
  let resultingState ← step executionState executionEvent
  transitionResult? resultingState

def stepResults
    (state action : ModelValue) :
    List (TransitionResult ModelValue ModelValue ModelValue) :=
  (stepResult? state action).toList

def authoritativeInitial (setup : ExecutionSetup) (state : ModelValue) : Prop :=
  state ∈ initialStates setup

def authoritativeStep
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : Prop :=
  result ∈ stepResults state action

theorem initialStates_sound
    (setup : ExecutionSetup)
    (state : ModelValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state :=
  member

theorem initialStates_complete
    (setup : ExecutionSetup)
    (state : ModelValue)
    (admitted : authoritativeInitial setup state) :
    state ∈ initialStates setup :=
  admitted

theorem stepResults_sound
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    authoritativeStep state action result :=
  member

theorem stepResults_complete
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action :=
  admitted

theorem authoritativeInitial_cases
    (setup : ExecutionSetup)
    (state : ModelValue)
    (admitted : authoritativeInitial setup state) :
    (setup = queuedSetup ∧ state = queuedState) ∨
      (setup = runningSetup ∧ state = runningState) := by
  cases setup <;>
    simp_all [authoritativeInitial, initialStates, initialState?, queuedSetup, runningSetup]

theorem authoritativeStep_cases
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    (state = queuedState ∧ action = dispatchAction ∧ result = dispatchedResult) ∨
      (state = runningState ∧ action = recordCancellationAction ∧
        result = cancellationRecordedResult) ∨
      (state = runningState ∧ action = recordCompletionAction ∧
        result = completionRecordedResult) := by
  by_cases queued : state = queuedState
  all_goals by_cases running : state = runningState
  all_goals by_cases canceled : state = cancellationRecordedState
  all_goals by_cases completed : state = completionRecordedState
  all_goals by_cases dispatch : action = dispatchAction
  all_goals by_cases cancel : action = recordCancellationAction
  all_goals by_cases complete : action = recordCompletionAction
  all_goals simp_all [authoritativeStep, stepResults, stepResult?, executionState?,
    executionEvent?, transitionResult?, step, queuedState, runningState,
    cancellationRecordedState, completionRecordedState, dispatchAction,
    recordCancellationAction, recordCompletionAction]

def setups : List ExecutionSetup := [queuedSetup, runningSetup]
def states : List ModelValue :=
  [queuedState, runningState, cancellationRecordedState, completionRecordedState]
def actions : List ModelValue :=
  [dispatchAction, recordCancellationAction, recordCompletionAction]
def outcomes : List ModelValue :=
  [dispatchedOutcome, cancellationRecordedOutcome, completionRecordedOutcome]
def observations : List ModelValue :=
  [runningObservation, cancellationRecordedObservation, completionRecordedObservation]

def transitionKernel : TransitionKernel
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue := {
  metadata := { id := kernelId, source }
  setupDomain := fun candidate => candidate = queuedSetup ∨ candidate = runningSetup
  stateDomain := fun candidate => candidate = queuedState ∨ candidate = runningState ∨
    candidate = cancellationRecordedState ∨ candidate = completionRecordedState
  actionDomain := fun candidate => candidate = dispatchAction ∨
    candidate = recordCancellationAction ∨ candidate = recordCompletionAction
  outcomeDomain := fun candidate => candidate = dispatchedOutcome ∨
    candidate = cancellationRecordedOutcome ∨ candidate = completionRecordedOutcome
  observationDomain := fun candidate => candidate = runningObservation ∨
    candidate = cancellationRecordedObservation ∨ candidate = completionRecordedObservation
  initialStates
  authoritativeInitial
  initialSound := initialStates_sound
  initialComplete := initialStates_complete
  steps := stepResults
  authoritativeStep
  stepSound := stepResults_sound
  stepComplete := stepResults_complete
  behaviorDomain := .complete {
    setups
    states
    actions
    outcomes
    observations
    encodeSetup := fun setup => match setup with
      | .queued => "queued"
      | .running => "running"
    encodeState := fun value => value.definitionId.value ++ ":" ++ value.value
    encodeAction := fun value => value.definitionId.value ++ ":" ++ value.value
    encodeOutcome := fun value => value.definitionId.value ++ ":" ++ value.value
    encodeObservation := fun value => value.definitionId.value ++ ":" ++ value.value
    setupSound := by intro candidate member; simpa [setups] using member
    setupComplete := by intro candidate admitted; simpa [setups] using admitted
    stateSound := by intro candidate member; simpa [states] using member
    stateComplete := by intro candidate admitted; simpa [states] using admitted
    actionSound := by intro candidate member; simpa [actions] using member
    actionComplete := by intro candidate admitted; simpa [actions] using admitted
    outcomeSound := by intro candidate member; simpa [outcomes] using member
    outcomeComplete := by intro candidate admitted; simpa [outcomes] using admitted
    observationSound := by intro candidate member; simpa [observations] using member
    observationComplete := by intro candidate admitted; simpa [observations] using admitted
    setupCoverage := by
      intro setup state member
      rcases authoritativeInitial_cases setup state member with ⟨rfl, rfl⟩ | ⟨rfl, rfl⟩ <;>
        simp [setups]
    initialStateCoverage := by
      intro setup state member
      rcases authoritativeInitial_cases setup state member with ⟨rfl, rfl⟩ | ⟨rfl, rfl⟩ <;>
        simp [states]
    transitionSourceCoverage := by
      intro state action result member
      rcases authoritativeStep_cases state action result member with
        ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;> simp [states]
    actionCoverage := by
      intro state action result member
      rcases authoritativeStep_cases state action result member with
        ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;> simp [actions]
    resultingStateCoverage := by
      intro state action result member
      rcases authoritativeStep_cases state action result member with
        ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;>
        simp [states, dispatchedResult, cancellationRecordedResult, completionRecordedResult]
    outcomeCoverage := by
      intro state action result member
      rcases authoritativeStep_cases state action result member with
        ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;>
        simp [outcomes, dispatchedResult, cancellationRecordedResult, completionRecordedResult]
    observationCoverage := by
      intro state action result observation member observationMember
      rcases authoritativeStep_cases state action result member with
        ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;>
        simp_all [observations, dispatchedResult, cancellationRecordedResult,
          completionRecordedResult]
  }
}

def lifecycleProvider : CapabilityProvider LawStatement := {
  id := lifecycleProviderId
  source
  contract := {
    id := lifecycleCapabilityId
    canonicalBehavior := "temporal-system-nexus-lifecycle/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { definitionId := operationStateId, kind := .state,
      canonicalBehavior := "temporal-system-nexus-lifecycle-state/v1" },
    { definitionId := dispatchActionId, kind := .action,
      canonicalBehavior := "temporal-system-nexus-lifecycle-dispatch/v1" },
    { definitionId := recordCancellationActionId, kind := .action,
      canonicalBehavior := "temporal-system-nexus-lifecycle-record-cancellation/v1" },
    { definitionId := recordCompletionActionId, kind := .action,
      canonicalBehavior := "temporal-system-nexus-lifecycle-record-completion/v1" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      canonicalBehavior := "temporal-system-nexus-lifecycle-outcome/v1" },
    { definitionId := lifecycleObservationId, kind := .observation,
      canonicalBehavior := "temporal-system-nexus-lifecycle-observation/v1" }
  ]
  lawWitnesses := [{ definition := lifecycleLaw, proof := lifecycleLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "temporal-system-nexus-lifecycle-target/v1",
  metadata kernelId .kernel "temporal-system-nexus-lifecycle-kernel/v1",
  metadata lifecycleCapabilityId .capability "temporal-system-nexus-lifecycle/v1",
  metadata lifecycleProviderId .provider "temporal-system-nexus-lifecycle-provider/v1",
  metadata lifecycleLawId .law lifecycleLaw.body,
  metadata operationStateId .state "temporal-system-nexus-lifecycle-state/v1",
  metadata dispatchActionId .action "temporal-system-nexus-lifecycle-dispatch/v1",
  metadata recordCancellationActionId .action
    "temporal-system-nexus-lifecycle-record-cancellation/v1",
  metadata recordCompletionActionId .action
    "temporal-system-nexus-lifecycle-record-completion/v1",
  metadata transitionOutcomeId .outcome "temporal-system-nexus-lifecycle-outcome/v1",
  metadata lifecycleObservationId .observation
    "temporal-system-nexus-lifecycle-observation/v1"
]

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep := {
  actions
  actionSound := by
    intro action member
    simp [actions] at member
    rcases member with rfl | rfl | rfl
    · exact ⟨queuedState, dispatchedResult, by
        change authoritativeStep queuedState dispatchAction dispatchedResult
        simp [authoritativeStep, stepResults, stepResult?, executionState?, executionEvent?,
          transitionResult?, step, queuedState, dispatchAction]⟩
    · exact ⟨runningState, cancellationRecordedResult, by
        change authoritativeStep runningState recordCancellationAction cancellationRecordedResult
        simp [authoritativeStep, stepResults, stepResult?, executionState?, executionEvent?,
          transitionResult?, step, queuedState, runningState, dispatchAction,
          recordCancellationAction]⟩
    · exact ⟨runningState, completionRecordedResult, by
        change authoritativeStep runningState recordCompletionAction completionRecordedResult
        simp [authoritativeStep, stepResults, stepResult?, executionState?, executionEvent?,
          transitionResult?, step, queuedState, runningState, dispatchAction,
          recordCancellationAction, recordCompletionAction]⟩
  actionComplete := by
    intro state action result admitted
    change authoritativeStep state action result at admitted
    rcases authoritativeStep_cases state action result admitted with
      ⟨_, rfl, _⟩ | ⟨_, rfl, _⟩ | ⟨_, rfl, _⟩ <;> simp [actions]
}

def targetDefinition : TargetDefinition
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := setups
  kernel := .checked transitionKernel
}

def targetComposition : TargetComposition LawStatement :=
  TargetComposition.empty |>.provide lifecycleProvider

def targetAuthoring : AuthoredTarget LawStatement
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition targetComposition
    (.available transitionKernel rfl finitePlanning)

/-- The independently checked pure Nexus System target. -/
def target : CheckedTarget LawStatement
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue :=
  checkedTarget targetAuthoring

theorem target_queued_initial_authoritative :
    target.kernel.authoritativeInitial queuedSetup queuedState := by
  change queuedState ∈ initialStates queuedSetup
  simp [initialStates, initialState?]

theorem target_queued_dispatch_authoritative :
    target.kernel.authoritativeStep queuedState dispatchAction dispatchedResult := by
  change dispatchedResult ∈ stepResults queuedState dispatchAction
  simp [stepResults, stepResult?, executionState?, executionEvent?, transitionResult?, step,
    queuedState, dispatchAction]

theorem target_running_cancellation_authoritative :
    target.kernel.authoritativeStep runningState recordCancellationAction
      cancellationRecordedResult := by
  change cancellationRecordedResult ∈ stepResults runningState recordCancellationAction
  simp [stepResults, stepResult?, executionState?, executionEvent?, transitionResult?, step,
    queuedState, runningState, dispatchAction, recordCancellationAction]

theorem target_running_completion_authoritative :
    target.kernel.authoritativeStep runningState recordCompletionAction
      completionRecordedResult := by
  change completionRecordedResult ∈ stepResults runningState recordCompletionAction
  simp [stepResults, stepResult?, executionState?, executionEvent?, transitionResult?, step,
    queuedState, runningState, dispatchAction, recordCancellationAction, recordCompletionAction]

end Temporal.System.Nexus
