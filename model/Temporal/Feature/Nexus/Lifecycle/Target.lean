import Temporal.Feature.Nexus.Lifecycle.Semantics
import Umpire.Planning

/-!
# Checked Nexus lifecycle target

Read `Temporal.Feature.Nexus.Lifecycle.Semantics` first for the small lifecycle state machine, then
continue here for its ModelValue encoding, authoritative kernel, provider, and planning target.
-/

namespace Temporal.Feature.Nexus.Lifecycle

open Umpire

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata definitionId kind source canonicalBehavior

def scheduledState : ModelValue := ModelValue.named operationStateId "scheduled"
def startedState : ModelValue := ModelValue.named operationStateId "started"
def canceledState : ModelValue := ModelValue.named operationStateId "canceled"
def succeededState : ModelValue := ModelValue.named operationStateId "succeeded"

def startAction : ModelValue := ModelValue.named startActionId "start"
def cancelAction : ModelValue := ModelValue.named cancelActionId "cancel"

/-- Successful completion is handler-reported lifecycle progress, not a caller command. -/
def reportSuccessAction : ModelValue :=
  ModelValue.named reportSuccessActionId "handler-reports-success"

def startedOutcome : ModelValue := ModelValue.named transitionOutcomeId "started"
def canceledOutcome : ModelValue := ModelValue.named transitionOutcomeId "canceled"
def succeededOutcome : ModelValue := ModelValue.named transitionOutcomeId "succeeded"
def startedObservation : ModelValue := ModelValue.named lifecycleObservationId "started"
def canceledObservation : ModelValue := ModelValue.named lifecycleObservationId "canceled"
def succeededObservation : ModelValue := ModelValue.named lifecycleObservationId "succeeded"

def scheduledSetup : List RoleBinding := [{ role := operationRoleId, value := scheduledState }]
def startedSetup : List RoleBinding := [{ role := operationRoleId, value := startedState }]

def startedResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := startedOutcome
  resultingState := startedState
  observations := [startedObservation]
}

def canceledResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := canceledOutcome
  resultingState := canceledState
  observations := [canceledObservation]
}

def succeededResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := succeededOutcome
  resultingState := succeededState
  observations := [succeededObservation]
}

def lifecycleState? (state : ModelValue) : Option OperationState :=
  if state = scheduledState then
    some .scheduled
  else if state = startedState then
    some .started
  else if state = canceledState then
    some .canceled
  else if state = succeededState then
    some .succeeded
  else
    none

def lifecycleEvent? (action : ModelValue) : Option OperationEvent :=
  if action = startAction then
    some .start
  else if action = cancelAction then
    some .cancel
  else if action = reportSuccessAction then
    some .succeed
  else
    none

def transitionResult? : OperationState → Option
    (TransitionResult ModelValue ModelValue ModelValue)
  | .started => some startedResult
  | .canceled => some canceledResult
  | .succeeded => some succeededResult
  | _ => none

/-- Enumerate only exposed results reached through the focused `step` relation. -/
def stepResult? (state action : ModelValue) : Option
    (TransitionResult ModelValue ModelValue ModelValue) := do
  let lifecycleState ← lifecycleState? state
  let lifecycleEvent ← lifecycleEvent? action
  let resultingState ← step lifecycleState lifecycleEvent
  transitionResult? resultingState

def initialState? (setup : List RoleBinding) : Option ModelValue :=
  if setup = scheduledSetup then
    some scheduledState
  else if setup = startedSetup then
    some startedState
  else
    none

def initialStates (setup : List RoleBinding) : List ModelValue :=
  (initialState? setup).toList

def stepResults
    (state action : ModelValue) :
    List (TransitionResult ModelValue ModelValue ModelValue) :=
  (stepResult? state action).toList

theorem initialStates_length_le_one (setup : List RoleBinding) :
    (initialStates setup).length ≤ 1 := by
  cases selected : initialState? setup <;> simp [initialStates, selected]

theorem stepResults_length_le_one (state action : ModelValue) :
    (stepResults state action).length ≤ 1 := by
  cases selected : stepResult? state action <;> simp [stepResults, selected]

theorem step_action_exposed
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    action = cancelAction ∨ action = startAction ∨ action = reportSuccessAction := by
  by_cases isCancel : action = cancelAction
  · exact .inl isCancel
  · by_cases isStart : action = startAction
    · exact .inr (.inl isStart)
    · by_cases isSuccess : action = reportSuccessAction
      · exact .inr (.inr isSuccess)
      · simp [stepResults, stepResult?, lifecycleEvent?, isStart, isCancel, isSuccess] at member

theorem step_result_exposed
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    result = startedResult ∨ result = canceledResult ∨ result = succeededResult := by
  have started_ne_scheduled : startedState ≠ scheduledState := by native_decide
  have canceled_ne_scheduled : canceledState ≠ scheduledState := by native_decide
  have canceled_ne_started : canceledState ≠ startedState := by native_decide
  have succeeded_ne_scheduled : succeededState ≠ scheduledState := by native_decide
  have succeeded_ne_started : succeededState ≠ startedState := by native_decide
  have succeeded_ne_canceled : succeededState ≠ canceledState := by native_decide
  have cancel_ne_start : cancelAction ≠ startAction := by native_decide
  have success_ne_start : reportSuccessAction ≠ startAction := by native_decide
  have success_ne_cancel : reportSuccessAction ≠ cancelAction := by native_decide
  by_cases scheduled : state = scheduledState
  · subst state
    by_cases start : action = startAction
    · subst action
      left
      simpa [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?, step]
        using member
    · by_cases cancel : action = cancelAction
      · subst action
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?, step,
          cancel_ne_start] at member
      · by_cases reportSuccess : action = reportSuccessAction
        · subst action
          simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?, step,
            success_ne_start, success_ne_cancel] at member
        · simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
            step, start, cancel, reportSuccess] at member
  · by_cases started : state = startedState
    · subst state
      by_cases cancel : action = cancelAction
      · subst action
        right; left
        simpa [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?, step,
          started_ne_scheduled, cancel_ne_start] using member
      · by_cases reportSuccess : action = reportSuccessAction
        · subst action
          right; right
          simpa [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
            step, started_ne_scheduled, success_ne_start, success_ne_cancel] using member
        · by_cases start : action = startAction
          · subst action
            simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
              step, started_ne_scheduled] at member
          · simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
              step, started_ne_scheduled, start, cancel, reportSuccess] at member
    · by_cases canceled : state = canceledState
      · subst state
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?, step,
          canceled_ne_scheduled, canceled_ne_started] at member
      · by_cases succeeded : state = succeededState
        · subst state
          simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?, step,
            succeeded_ne_scheduled, succeeded_ne_started, succeeded_ne_canceled] at member
        · simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
            step, scheduled, started, canceled, succeeded] at member

def roleAssignments : List (List RoleBinding) := [scheduledSetup, startedSetup]
def actionDomain : List ModelValue := [cancelAction, startAction, reportSuccessAction]

def finiteMachine : FiniteMachine
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := {
    id := kernelId
    source
  }
  setups := roleAssignments
  states := [scheduledState, startedState, canceledState, succeededState]
  actions := actionDomain
  outcomes := [startedOutcome, canceledOutcome, succeededOutcome]
  observations := [startedObservation, canceledObservation, succeededObservation]
  encodeSetup := fun bindings => String.intercalate "|" (bindings.map fun binding =>
    binding.role.value ++ "=" ++ binding.value.definitionId.value ++ ":" ++ binding.value.value)
  encodeState := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
  encodeAction := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
  encodeOutcome := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
  encodeObservation := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
  initialStates := initialStates
  steps := stepResults
  setupCoverage := by
    intro setup state member
    by_cases scheduled : setup = scheduledSetup
    · simp [roleAssignments, scheduled]
    · by_cases started : setup = startedSetup
      · simp [roleAssignments, started]
      · simp [initialStates, initialState?, scheduled, started] at member
  initialStateCoverage := by
    intro setup state member
    by_cases scheduled : setup = scheduledSetup
    · simp [initialStates, initialState?, scheduled] at member
      simp [member]
    · by_cases started : setup = startedSetup
      · subst setup
        simp [initialStates, initialState?, scheduledSetup, startedSetup,
          scheduledState, startedState, ModelValue.named] at member
        simp [member, scheduledState, startedState, canceledState, succeededState, ModelValue.named]
      · simp [initialStates, initialState?, scheduled, started] at member
  transitionSourceCoverage := by
    intro state action result member
    by_cases scheduled : state = scheduledState
    · simp [scheduled]
    · by_cases started : state = startedState
      · simp [started]
      · by_cases canceled : state = canceledState
        · subst state
          simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
            step, scheduledState, startedState, canceledState, ModelValue.named] at member
        · by_cases succeeded : state = succeededState
          · subst state
            simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
              step, scheduledState, startedState, canceledState, succeededState,
                ModelValue.named] at member
          · simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
              step, scheduled, started, canceled, succeeded] at member
  actionCoverage := by
    intro state action result member
    simpa [actionDomain] using step_action_exposed state action result member
  resultingStateCoverage := by
    intro state action result member
    rcases step_result_exposed state action result member with rfl | rfl | rfl <;>
      simp [startedResult, canceledResult, succeededResult]
  outcomeCoverage := by
    intro state action result member
    rcases step_result_exposed state action result member with rfl | rfl | rfl <;>
      simp [startedResult, canceledResult, succeededResult]
  observationCoverage := by
    intro state action result observation member observationMember
    rcases step_result_exposed state action result member with rfl | rfl | rfl
    · exact List.mem_cons.mpr (.inl <| by simpa [startedResult] using observationMember)
    · exact List.mem_cons.mpr (.inr <| List.mem_cons.mpr (.inl <|
        by simpa [canceledResult] using observationMember))
    · exact List.mem_cons.mpr (.inr <| List.mem_cons.mpr (.inr <|
        List.mem_singleton.mpr <| by simpa [succeededResult] using observationMember))
  actionExecutable := by
    intro action member
    simp [actionDomain] at member
    rcases member with rfl | rfl | rfl
    · exact ⟨startedState, canceledResult, by
        change canceledResult ∈ stepResults startedState cancelAction
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
          step, scheduledState, startedState, startAction, cancelAction, ModelValue.named]⟩
    · exact ⟨scheduledState, startedResult, by
        change startedResult ∈ stepResults scheduledState startAction
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
          step, scheduledState, startAction]⟩
    · exact ⟨startedState, succeededResult, by
        change succeededResult ∈ stepResults startedState reportSuccessAction
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
          step, scheduledState, startedState, startAction, cancelAction,
          reportSuccessAction, ModelValue.named]⟩
}

def authoritativeInitial (setup : List RoleBinding) (state : ModelValue) : Prop :=
  finiteMachine.kernel.authoritativeInitial setup state

def authoritativeStep
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : Prop :=
  finiteMachine.kernel.authoritativeStep state action result

theorem initialStates_sound
    (setup : List RoleBinding)
    (state : ModelValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state := by
  exact finiteMachine.kernel.initialSound setup state member

theorem initialStates_complete
    (setup : List RoleBinding)
    (state : ModelValue)
    (admitted : authoritativeInitial setup state) :
    state ∈ initialStates setup := by
  exact finiteMachine.kernel.initialComplete setup state admitted

theorem stepResults_sound
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    authoritativeStep state action result := by
  exact finiteMachine.kernel.stepSound state action result member

theorem stepResults_complete
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action := by
  exact finiteMachine.kernel.stepComplete state action result admitted

def transitionKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  finiteMachine.kernel

@[simp] private theorem finiteMachine_kernel_initialStates (setup : List RoleBinding) :
    finiteMachine.kernel.initialStates setup = initialStates setup :=
  rfl

@[simp] private theorem finiteMachine_kernel_steps (state action : ModelValue) :
    finiteMachine.kernel.steps state action = stepResults state action :=
  rfl

def lifecycleProvider : CapabilityProvider LawStatement := {
  id := lifecycleProviderId
  source
  contract := {
    id := lifecycleCapabilityId
    canonicalBehavior := "temporal-nexus-basic-lifecycle/v2"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { definitionId := operationStateId, kind := .state,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-state/v2" },
    { definitionId := startActionId, kind := .action,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-start/v1" },
    { definitionId := cancelActionId, kind := .action,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-cancel/v1" },
    { definitionId := reportSuccessActionId, kind := .action,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-report-success/v1" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-outcome/v2" },
    { definitionId := lifecycleObservationId, kind := .observation,
      canonicalBehavior := "temporal-nexus-basic-lifecycle-observation/v2" }
  ]
  lawWitnesses := [{ definition := lifecycleLaw, proof := lifecycleLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "temporal-nexus-basic-lifecycle-target/v2",
  metadata kernelId .kernel "temporal-nexus-basic-lifecycle-kernel/v2",
  metadata lifecycleCapabilityId .capability "temporal-nexus-basic-lifecycle/v2",
  metadata lifecycleProviderId .provider "temporal-nexus-basic-lifecycle-provider/v2",
  metadata lifecycleLawId .law lifecycleLaw.body,
  metadata operationStateId .state "temporal-nexus-basic-lifecycle-state/v2",
  metadata startActionId .action "temporal-nexus-basic-lifecycle-start/v1",
  metadata cancelActionId .action "temporal-nexus-basic-lifecycle-cancel/v1",
  metadata reportSuccessActionId .action "temporal-nexus-basic-lifecycle-report-success/v1",
  metadata transitionOutcomeId .outcome "temporal-nexus-basic-lifecycle-outcome/v2",
  metadata lifecycleObservationId .observation "temporal-nexus-basic-lifecycle-observation/v2"
]

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep :=
  finiteMachine.planning

@[simp] private theorem finiteMachine_planning_actions :
    finiteMachine.planning.actions = actionDomain :=
  rfl

def targetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := roleAssignments
  kernel := finiteMachine.kernelAvailability
}

def targetComposition : TargetComposition LawStatement :=
  TargetComposition.empty |>.provide lifecycleProvider

def targetAuthoring : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition targetComposition
    (.available transitionKernel rfl finitePlanning)

/-- Re-ascribe the source kernel after composition so its proof relation remains reducible. -/
def target : QueryTarget LawStatement := checkedTarget targetAuthoring

theorem target_resolvedSetups : target.resolvedSetups = roleAssignments := by
  native_decide

theorem target_scheduled_initial_authoritative :
    target.kernel.authoritativeInitial scheduledSetup scheduledState := by
  change scheduledState ∈ initialStates scheduledSetup
  simp [initialStates, initialState?]

theorem target_started_initial_authoritative :
    target.kernel.authoritativeInitial startedSetup startedState := by
  change startedState ∈ initialStates startedSetup
  have different : startedSetup ≠ scheduledSetup := by native_decide
  simp [initialStates, initialState?, different]

theorem target_scheduled_start_authoritative :
    target.kernel.authoritativeStep scheduledState startAction startedResult := by
  change startedResult ∈ stepResults scheduledState startAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    step, scheduledState, startAction]

theorem target_started_cancel_authoritative :
    target.kernel.authoritativeStep startedState cancelAction canceledResult := by
  change canceledResult ∈ stepResults startedState cancelAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    step, scheduledState, startedState, startAction, cancelAction, ModelValue.named]

theorem target_started_reportSuccess_authoritative :
    target.kernel.authoritativeStep startedState reportSuccessAction succeededResult := by
  change succeededResult ∈ stepResults startedState reportSuccessAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    step, scheduledState, startedState, startAction, cancelAction,
    reportSuccessAction, ModelValue.named]

def limits : QueryLimits := QueryLimits.bounded 1 1 8

def policy : PlannerPolicy := PlannerPolicy.shortest

def queryContext : QueryCheckContext LawStatement := .ofTarget target

/-- Put checked queries back on the shared target so downstream planner proofs stay small. -/
def materializeQuery (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  checked with
  target
  completeness := (CheckedQueryTarget.ofTarget target).completeness
}

end Temporal.Feature.Nexus.Lifecycle
