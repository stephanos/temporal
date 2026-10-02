import Umpire.Model

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

def lifecycleLaw : Law := {
  id := lifecycleLawId
  body := "temporal-system-nexus-lifecycle-authoritative-step/v1"
}

/-- The provider law binds checked System meaning to the pure mechanism transition relation. -/
def LawStatement (law : Law) : Prop :=
  law = lifecycleLaw ∧
    step .queued .dispatch = some .running ∧
    step .running .recordCancellation = some .cancellationRecorded ∧
    step .running .recordCompletion = some .completionRecorded

theorem lifecycleLawProof : LawStatement lifecycleLaw := by
  exact ⟨rfl, rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (behaviorVersion : String) : DefinitionMetadata := {
  id := definitionId
  kind
  source
  behaviorVersion
}

def queuedState : ModelValue := ModelValue.named operationStateId "queued"
def runningState : ModelValue := ModelValue.named operationStateId "running"
def cancellationRecordedState : ModelValue := ModelValue.named operationStateId "cancellation-recorded"
def completionRecordedState : ModelValue := ModelValue.named operationStateId "completion-recorded"

def dispatchAction : ModelValue := ModelValue.named dispatchActionId "dispatch"
def recordCancellationAction : ModelValue :=
  ModelValue.named recordCancellationActionId "record-cancellation"
def recordCompletionAction : ModelValue :=
  ModelValue.named recordCompletionActionId "record-completion"

def dispatchedOutcome : ModelValue := ModelValue.named transitionOutcomeId "running"
def cancellationRecordedOutcome : ModelValue :=
  ModelValue.named transitionOutcomeId "cancellation-recorded"
def completionRecordedOutcome : ModelValue :=
  ModelValue.named transitionOutcomeId "completion-recorded"

def runningObservation : ModelValue := ModelValue.named lifecycleObservationId "running"
def cancellationRecordedObservation : ModelValue :=
  ModelValue.named lifecycleObservationId "cancellation-recorded"
def completionRecordedObservation : ModelValue :=
  ModelValue.named lifecycleObservationId "completion-recorded"

def queuedSetup : ExecutionSetup := .queued
def runningSetup : ExecutionSetup := .running

def dispatchedResult : Step ModelValue ModelValue ModelValue := {
  outcome := dispatchedOutcome
  state := runningState
  facts := [runningObservation]
}

def cancellationRecordedResult : Step ModelValue ModelValue ModelValue := {
  outcome := cancellationRecordedOutcome
  state := cancellationRecordedState
  facts := [cancellationRecordedObservation]
}

def completionRecordedResult : Step ModelValue ModelValue ModelValue := {
  outcome := completionRecordedOutcome
  state := completionRecordedState
  facts := [completionRecordedObservation]
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

private def modelStep? : ExecutionState → Option
    (Step ModelValue ModelValue ModelValue)
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
    (Step ModelValue ModelValue ModelValue) := do
  let executionState ← executionState? state
  let executionEvent ← executionEvent? action
  let resultingState ← step executionState executionEvent
  modelStep? resultingState

def stepResults
    (state action : ModelValue) :
    List (Step ModelValue ModelValue ModelValue) :=
  (stepResult? state action).toList

private theorem initialStates_cases
    (setup : ExecutionSetup)
    (state : ModelValue)
    (member : state ∈ initialStates setup) :
    (setup = queuedSetup ∧ state = queuedState) ∨
      (setup = runningSetup ∧ state = runningState) := by
  cases setup <;>
    simp_all [initialStates, initialState?, queuedSetup, runningSetup]

private theorem stepResults_cases
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    (state = queuedState ∧ action = dispatchAction ∧ result = dispatchedResult) ∨
      (state = runningState ∧ action = recordCancellationAction ∧
        result = cancellationRecordedResult) ∨
      (state = runningState ∧ action = recordCompletionAction ∧
        result = completionRecordedResult) := by
  have running_ne_queued : runningState ≠ queuedState := by native_decide
  have canceled_ne_queued : cancellationRecordedState ≠ queuedState := by native_decide
  have canceled_ne_running : cancellationRecordedState ≠ runningState := by native_decide
  have completed_ne_queued : completionRecordedState ≠ queuedState := by native_decide
  have completed_ne_running : completionRecordedState ≠ runningState := by native_decide
  have completed_ne_canceled : completionRecordedState ≠ cancellationRecordedState := by
    native_decide
  have cancel_ne_dispatch : recordCancellationAction ≠ dispatchAction := by native_decide
  have complete_ne_dispatch : recordCompletionAction ≠ dispatchAction := by native_decide
  have complete_ne_cancel : recordCompletionAction ≠ recordCancellationAction := by
    native_decide
  by_cases queued : state = queuedState
  · subst state
    by_cases dispatch : action = dispatchAction
    · subst action
      left
      refine ⟨rfl, rfl, ?_⟩
      simpa [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step]
        using member
    · by_cases cancel : action = recordCancellationAction
      · subst action
        simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
          cancel_ne_dispatch] at member
      · by_cases complete : action = recordCompletionAction
        · subst action
          simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
            complete_ne_dispatch, complete_ne_cancel] at member
        · simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
            dispatch, cancel, complete] at member
  · by_cases running : state = runningState
    · subst state
      by_cases cancel : action = recordCancellationAction
      · subst action
        right; left
        refine ⟨rfl, rfl, ?_⟩
        simpa [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
          running_ne_queued, cancel_ne_dispatch] using member
      · by_cases complete : action = recordCompletionAction
        · subst action
          right; right
          refine ⟨rfl, rfl, ?_⟩
          simpa [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
            running_ne_queued, complete_ne_dispatch, complete_ne_cancel] using member
        · by_cases dispatch : action = dispatchAction
          · subst action
            simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?,
              step, running_ne_queued] at member
          · simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?,
              step, running_ne_queued, dispatch, cancel, complete] at member
    · by_cases canceled : state = cancellationRecordedState
      · subst state
        simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
          canceled_ne_queued, canceled_ne_running] at member
      · by_cases completed : state = completionRecordedState
        · subst state
          simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
            completed_ne_queued, completed_ne_running, completed_ne_canceled] at member
        · simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
            queued, running, canceled, completed] at member

def setups : List ExecutionSetup := [queuedSetup, runningSetup]
def states : List ModelValue :=
  [queuedState, runningState, cancellationRecordedState, completionRecordedState]
def actions : List ModelValue :=
  [dispatchAction, recordCancellationAction, recordCompletionAction]
def outcomes : List ModelValue :=
  [dispatchedOutcome, cancellationRecordedOutcome, completionRecordedOutcome]
def observations : List ModelValue :=
  [runningObservation, cancellationRecordedObservation, completionRecordedObservation]

def finiteMachine : FiniteMachine
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue := {
  metadata := { id := kernelId, source }
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
  initialStates := initialStates
  steps := stepResults
  setupCoverage := by
    intro setup state member
    rcases initialStates_cases setup state member with ⟨rfl, rfl⟩ | ⟨rfl, rfl⟩ <;>
      simp [setups]
  initialStateCoverage := by
    intro setup state member
    rcases initialStates_cases setup state member with ⟨rfl, rfl⟩ | ⟨rfl, rfl⟩ <;>
      simp [states]
  transitionSourceCoverage := by
    intro state action result member
    rcases stepResults_cases state action result member with
      ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;> simp [states]
  actionCoverage := by
    intro state action result member
    rcases stepResults_cases state action result member with
      ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;> simp [actions]
  resultingStateCoverage := by
    intro state action result member
    rcases stepResults_cases state action result member with
      ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;>
      simp [states, dispatchedResult, cancellationRecordedResult, completionRecordedResult]
  outcomeCoverage := by
    intro state action result member
    rcases stepResults_cases state action result member with
      ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;>
      simp [outcomes, dispatchedResult, cancellationRecordedResult, completionRecordedResult]
  observationCoverage := by
    intro state action result observation member observationMember
    rcases stepResults_cases state action result member with
      ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ | ⟨rfl, rfl, rfl⟩ <;>
      simp_all [observations, dispatchedResult, cancellationRecordedResult,
        completionRecordedResult]
  actionExecutable := by
    intro action member
    simp [actions] at member
    rcases member with rfl | rfl | rfl
    · exact ⟨queuedState, dispatchedResult, by
        change dispatchedResult ∈ stepResults queuedState dispatchAction
        simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
          queuedState, dispatchAction]⟩
    · exact ⟨runningState, cancellationRecordedResult, by
        change cancellationRecordedResult ∈
          stepResults runningState recordCancellationAction
        simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
          queuedState, runningState, dispatchAction, recordCancellationAction, ModelValue.named]⟩
    · exact ⟨runningState, completionRecordedResult, by
        change completionRecordedResult ∈ stepResults runningState recordCompletionAction
        simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
          queuedState, runningState, dispatchAction, recordCancellationAction,
          recordCompletionAction, ModelValue.named]⟩
}

def authoritativeInitial (setup : ExecutionSetup) (state : ModelValue) : Prop :=
  finiteMachine.kernel.authoritativeInitial setup state

def authoritativeStep
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : Prop :=
  finiteMachine.kernel.authoritativeStep state action result

theorem initialStates_sound
    (setup : ExecutionSetup)
    (state : ModelValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state := by
  exact finiteMachine.kernel.initialSound setup state member

theorem initialStates_complete
    (setup : ExecutionSetup)
    (state : ModelValue)
    (admitted : authoritativeInitial setup state) :
    state ∈ initialStates setup := by
  exact finiteMachine.kernel.initialComplete setup state admitted

theorem stepResults_sound
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    authoritativeStep state action result := by
  exact finiteMachine.kernel.stepSound state action result member

theorem stepResults_complete
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action := by
  exact finiteMachine.kernel.stepComplete state action result admitted

theorem authoritativeInitial_cases
    (setup : ExecutionSetup)
    (state : ModelValue)
    (admitted : authoritativeInitial setup state) :
    (setup = queuedSetup ∧ state = queuedState) ∨
      (setup = runningSetup ∧ state = runningState) := by
  exact initialStates_cases setup state (initialStates_complete setup state admitted)

theorem authoritativeStep_cases
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    (state = queuedState ∧ action = dispatchAction ∧ result = dispatchedResult) ∨
      (state = runningState ∧ action = recordCancellationAction ∧
        result = cancellationRecordedResult) ∨
      (state = runningState ∧ action = recordCompletionAction ∧
        result = completionRecordedResult) := by
  exact stepResults_cases state action result (stepResults_complete state action result admitted)

private theorem setupDomain_eq :
    (fun candidate => candidate = queuedSetup ∨ candidate = runningSetup) =
      finiteMachine.kernel.setupDomain := by
  funext candidate
  apply propext
  simp [finiteMachine, setups]

private theorem stateDomain_eq :
    (fun candidate => candidate = queuedState ∨ candidate = runningState ∨
      candidate = cancellationRecordedState ∨ candidate = completionRecordedState) =
      finiteMachine.kernel.stateDomain := by
  funext candidate
  apply propext
  simp [finiteMachine, states]

private theorem actionDomain_eq :
    (fun candidate => candidate = dispatchAction ∨
      candidate = recordCancellationAction ∨ candidate = recordCompletionAction) =
      finiteMachine.kernel.actionDomain := by
  funext candidate
  apply propext
  simp [finiteMachine, actions]

private theorem outcomeDomain_eq :
    (fun candidate => candidate = dispatchedOutcome ∨
      candidate = cancellationRecordedOutcome ∨ candidate = completionRecordedOutcome) =
      finiteMachine.kernel.outcomeDomain := by
  funext candidate
  apply propext
  simp [finiteMachine, outcomes]

private theorem observationDomain_eq :
    (fun candidate => candidate = runningObservation ∨
      candidate = cancellationRecordedObservation ∨ candidate = completionRecordedObservation) =
      finiteMachine.kernel.observationDomain := by
  funext candidate
  apply propext
  simp [finiteMachine, observations]

def machine : Machine
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue := {
  metadata := finiteMachine.kernel.metadata
  setupDomain := fun candidate => candidate = queuedSetup ∨ candidate = runningSetup
  stateDomain := fun candidate => candidate = queuedState ∨ candidate = runningState ∨
    candidate = cancellationRecordedState ∨ candidate = completionRecordedState
  actionDomain := fun candidate => candidate = dispatchAction ∨
    candidate = recordCancellationAction ∨ candidate = recordCompletionAction
  outcomeDomain := fun candidate => candidate = dispatchedOutcome ∨
    candidate = cancellationRecordedOutcome ∨ candidate = completionRecordedOutcome
  observationDomain := fun candidate => candidate = runningObservation ∨
    candidate = cancellationRecordedObservation ∨ candidate = completionRecordedObservation
  initialStates := finiteMachine.kernel.initialStates
  authoritativeInitial := finiteMachine.kernel.authoritativeInitial
  initialSound := finiteMachine.kernel.initialSound
  initialComplete := finiteMachine.kernel.initialComplete
  steps := finiteMachine.kernel.steps
  authoritativeStep := finiteMachine.kernel.authoritativeStep
  stepSound := finiteMachine.kernel.stepSound
  stepComplete := finiteMachine.kernel.stepComplete
  vocabulary := by
    rw [setupDomain_eq, stateDomain_eq, actionDomain_eq, outcomeDomain_eq,
      observationDomain_eq]
    exact finiteMachine.kernel.vocabulary
}

@[simp] private theorem finiteMachine_kernel_initialStates (setup : ExecutionSetup) :
    finiteMachine.kernel.initialStates setup = initialStates setup :=
  rfl

@[simp] private theorem finiteMachine_kernel_steps (state action : ModelValue) :
    finiteMachine.kernel.steps state action = stepResults state action :=
  rfl

def lifecycleProvider : Provider LawStatement := {
  id := lifecycleProviderId
  source
  contract := {
    id := lifecycleCapabilityId
    behaviorVersion := "temporal-system-nexus-lifecycle/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { definitionId := operationStateId, kind := .state,
      behaviorVersion := "temporal-system-nexus-lifecycle-state/v1" },
    { definitionId := dispatchActionId, kind := .action,
      behaviorVersion := "temporal-system-nexus-lifecycle-dispatch/v1" },
    { definitionId := recordCancellationActionId, kind := .action,
      behaviorVersion := "temporal-system-nexus-lifecycle-record-cancellation/v1" },
    { definitionId := recordCompletionActionId, kind := .action,
      behaviorVersion := "temporal-system-nexus-lifecycle-record-completion/v1" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      behaviorVersion := "temporal-system-nexus-lifecycle-outcome/v1" },
    { definitionId := lifecycleObservationId, kind := .fact,
      behaviorVersion := "temporal-system-nexus-lifecycle-observation/v1" }
  ]
  lawProofs := [{ definition := lifecycleLaw, proof := lifecycleLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "temporal-system-nexus-lifecycle-target/v1",
  metadata kernelId .machine "temporal-system-nexus-lifecycle-kernel/v1",
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
  metadata lifecycleObservationId .fact
    "temporal-system-nexus-lifecycle-observation/v1"
]

def finitePlanning : FinitePlanningCapability machine.authoritativeStep :=
  finiteMachine.planning

@[simp] private theorem finiteMachine_planning_actions :
    finiteMachine.planning.actions = actions :=
  rfl

def modelSpec : ModelSpec LawStatement
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := setups
  machine := .checked machine
}

def modelProviders : Providers LawStatement :=
  Providers.empty |>.provide lifecycleProvider

def targetAuthoring : DraftModel LawStatement
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make modelSpec modelProviders
    (.available machine rfl finitePlanning)

/-- The independently checked pure Nexus System target. -/
def target : CheckedModel LawStatement
    ExecutionSetup ModelValue ModelValue ModelValue ModelValue :=
  model targetAuthoring

theorem target_queued_initial_authoritative :
    target.machine.authoritativeInitial queuedSetup queuedState := by
  change queuedState ∈ initialStates queuedSetup
  simp [initialStates, initialState?]

theorem target_queued_dispatch_authoritative :
    target.machine.authoritativeStep queuedState dispatchAction dispatchedResult := by
  change dispatchedResult ∈ stepResults queuedState dispatchAction
  simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
    queuedState, dispatchAction]

theorem target_running_cancellation_authoritative :
    target.machine.authoritativeStep runningState recordCancellationAction
      cancellationRecordedResult := by
  change cancellationRecordedResult ∈ stepResults runningState recordCancellationAction
  simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
    queuedState, runningState, dispatchAction, recordCancellationAction, ModelValue.named]

theorem target_running_completion_authoritative :
    target.machine.authoritativeStep runningState recordCompletionAction
      completionRecordedResult := by
  change completionRecordedResult ∈ stepResults runningState recordCompletionAction
  simp [stepResults, stepResult?, executionState?, executionEvent?, modelStep?, step,
    queuedState, runningState, dispatchAction, recordCancellationAction, recordCompletionAction,
      ModelValue.named]

end Temporal.System.Nexus
