import Temporal.Shared
import Umpire.Planning

/-! The focused Nexus operation lifecycle and its checked Umpire planning target. -/

namespace Temporal.Feature.Nexus.Lifecycle

open Umpire

private def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Lifecycle.lean"

def targetId : DefinitionId := id "temporal.nexus.basic-lifecycle.target"
def kernelId : DefinitionId := id "temporal.nexus.basic-lifecycle.kernel"
def lifecycleCapabilityId : DefinitionId := id "temporal.nexus.basic-lifecycle.capability"
def lifecycleProviderId : DefinitionId := id "temporal.nexus.basic-lifecycle.provider"
def lifecycleLawId : DefinitionId := id "temporal.nexus.basic-lifecycle.law.authoritative-step"
def operationStateId : DefinitionId := id "temporal.nexus.basic-lifecycle.state.operation"
def startActionId : DefinitionId := id "temporal.nexus.basic-lifecycle.action.start"
def cancelActionId : DefinitionId := id "temporal.nexus.basic-lifecycle.action.cancel"
def reportSuccessActionId : DefinitionId := id "temporal.nexus.basic-lifecycle.action.succeed"
def transitionOutcomeId : DefinitionId := id "temporal.nexus.basic-lifecycle.outcome.transition"
def lifecycleObservationId : DefinitionId := id "temporal.nexus.basic-lifecycle.observation.state"
def operationRoleId : DefinitionId := id "temporal.nexus.basic-lifecycle.role.operation"

/-- The four states exposed by the ordinary Nexus operation lifecycle. -/
inductive OperationState where
  | scheduled
  | started
  | canceled
  | succeeded
  deriving DecidableEq, Repr

/-- The three events exposed by the ordinary Nexus operation lifecycle. -/
inductive OperationEvent where
  | start
  | cancel
  | succeed
  deriving DecidableEq, Repr

/-- The complete focused Nexus transition relation. -/
def step : OperationState → OperationEvent → Option OperationState
  | .scheduled, .start => some .started
  | .started, .cancel => some .canceled
  | .started, .succeed => some .succeeded
  | _, _ => none

/-- The provider law ties the teaching surface to the authoritative Nexus lifecycle. -/
def LawStatement (law : LawDefinition) : Prop :=
  law.id = lifecycleLawId ∧
    law.body = "temporal-nexus-basic-lifecycle-authoritative-step/v2" ∧
    step .scheduled .start = some .started ∧
    step .started .cancel = some .canceled ∧
    step .started .succeed = some .succeeded

def lifecycleLaw : LawDefinition := {
  id := lifecycleLawId
  body := "temporal-nexus-basic-lifecycle-authoritative-step/v2"
}

theorem lifecycleLawProof : LawStatement lifecycleLaw := by
  exact ⟨rfl, rfl, rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata definitionId kind source canonicalBehavior

def scheduledState : ModelValue := { definitionId := operationStateId, value := "scheduled" }
def startedState : ModelValue := { definitionId := operationStateId, value := "started" }
def canceledState : ModelValue := { definitionId := operationStateId, value := "canceled" }
def succeededState : ModelValue := { definitionId := operationStateId, value := "succeeded" }

def startAction : ModelValue := { definitionId := startActionId, value := "start" }
def cancelAction : ModelValue := { definitionId := cancelActionId, value := "cancel" }

/-- Successful completion is handler-reported lifecycle progress, not a caller command. -/
def reportSuccessAction : ModelValue := {
  definitionId := reportSuccessActionId
  value := "handler-reports-success"
}

def startedOutcome : ModelValue := { definitionId := transitionOutcomeId, value := "started" }
def canceledOutcome : ModelValue := { definitionId := transitionOutcomeId, value := "canceled" }
def succeededOutcome : ModelValue := { definitionId := transitionOutcomeId, value := "succeeded" }
def startedObservation : ModelValue := {
  definitionId := lifecycleObservationId
  value := "started"
}
def canceledObservation : ModelValue := {
  definitionId := lifecycleObservationId
  value := "canceled"
}
def succeededObservation : ModelValue := {
  definitionId := lifecycleObservationId
  value := "succeeded"
}

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

def authoritativeInitial (setup : List RoleBinding) (state : ModelValue) : Prop :=
  state ∈ initialStates setup

def stepResults
    (state action : ModelValue) :
    List (TransitionResult ModelValue ModelValue ModelValue) :=
  (stepResult? state action).toList

def authoritativeStep
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : Prop :=
  result ∈ stepResults state action

theorem initialStates_sound
    (setup : List RoleBinding)
    (state : ModelValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state :=
  member

theorem initialStates_complete
    (setup : List RoleBinding)
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

def transitionKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := {
    id := kernelId
    source
  }
  setupDomain := fun candidate => candidate = scheduledSetup ∨ candidate = startedSetup
  stateDomain := fun candidate => candidate = scheduledState ∨ candidate = startedState ∨
    candidate = canceledState ∨ candidate = succeededState
  actionDomain := fun candidate => candidate = cancelAction ∨ candidate = startAction ∨
    candidate = reportSuccessAction
  outcomeDomain := fun candidate => candidate = startedOutcome ∨ candidate = canceledOutcome ∨
    candidate = succeededOutcome
  observationDomain := fun candidate => candidate = startedObservation ∨
    candidate = canceledObservation ∨ candidate = succeededObservation
  initialStates
  authoritativeInitial
  initialSound := initialStates_sound
  initialComplete := initialStates_complete
  steps := stepResults
  authoritativeStep
  stepSound := stepResults_sound
  stepComplete := stepResults_complete
  behaviorDomain := .complete {
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
    setupSound := by intro candidate member; simpa [roleAssignments] using member
    setupComplete := by intro candidate admitted; simpa [roleAssignments] using admitted
    stateSound := by intro candidate member; simpa using member
    stateComplete := by intro candidate admitted; simpa using admitted
    actionSound := by intro candidate member; simpa [actionDomain] using member
    actionComplete := by intro candidate admitted; simpa [actionDomain] using admitted
    outcomeSound := by intro candidate member; simpa using member
    outcomeComplete := by intro candidate admitted; simpa using admitted
    observationSound := by intro candidate member; simpa using member
    observationComplete := by intro candidate admitted; simpa using admitted
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
            scheduledState, startedState] at member
          simp [member, scheduledState, startedState, canceledState, succeededState]
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
              step, scheduledState, startedState, canceledState] at member
          · by_cases succeeded : state = succeededState
            · subst state
              simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
                step, scheduledState, startedState, canceledState, succeededState] at member
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
  }
}

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

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep := {
  actions := actionDomain
  actionSound := by
    intro action member
    simp [actionDomain] at member
    rcases member with rfl | rfl | rfl
    · exact ⟨startedState, canceledResult, by
        change canceledResult ∈ stepResults startedState cancelAction
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
          step, scheduledState, startedState, startAction, cancelAction]⟩
    · exact ⟨scheduledState, startedResult, by
        change startedResult ∈ stepResults scheduledState startAction
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
          step, scheduledState, startAction]⟩
    · exact ⟨startedState, succeededResult, by
        change succeededResult ∈ stepResults startedState reportSuccessAction
        simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
          step, scheduledState, startedState, startAction, cancelAction,
          reportSuccessAction]⟩
  actionComplete := by
    intro state action result admitted
    change authoritativeStep state action result at admitted
    simpa [actionDomain] using step_action_exposed state action result admitted
}

def targetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := roleAssignments
  kernel := .checked transitionKernel
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

theorem target_scheduled_start_authoritative :
    target.kernel.authoritativeStep scheduledState startAction startedResult := by
  change startedResult ∈ stepResults scheduledState startAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    step, scheduledState, startAction]

theorem target_started_cancel_authoritative :
    target.kernel.authoritativeStep startedState cancelAction canceledResult := by
  change canceledResult ∈ stepResults startedState cancelAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    step, scheduledState, startedState, startAction, cancelAction]

theorem target_started_reportSuccess_authoritative :
    target.kernel.authoritativeStep startedState reportSuccessAction succeededResult := by
  change succeededResult ∈ stepResults startedState reportSuccessAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    step, scheduledState, startedState, startAction, cancelAction,
    reportSuccessAction]

def limits : QueryLimits := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 8, unit := .candidateEvaluations }
}

def policy : PlannerPolicy := {
  strategy := .shortest
  seed := 29
  tieBreak := .definitionId
}

def queryContext : QueryCheckContext LawStatement := .ofTarget target

/-- Put checked queries back on the shared target so downstream planner proofs stay small. -/
def materializeQuery (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  checked with
  target
  completeness := (CheckedQueryTarget.ofTarget target).completeness
}

end Temporal.Feature.Nexus.Lifecycle
