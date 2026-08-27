import Umpire.Planning

/-! The focused Nexus operation lifecycle and its checked Umpire planning target. -/

namespace Temporal.Feature.Nexus.Lifecycle

open Umpire

private def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Feature/Nexus/Lifecycle.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId : DeclarationId := id "temporal.nexus.basic-lifecycle.target"
def kernelId : DeclarationId := id "temporal.nexus.basic-lifecycle.kernel"
def lifecycleCapabilityId : DeclarationId := id "temporal.nexus.basic-lifecycle.capability"
def lifecycleProviderId : DeclarationId := id "temporal.nexus.basic-lifecycle.provider"
def lifecycleLawId : DeclarationId := id "temporal.nexus.basic-lifecycle.law.authoritative-step"
def operationStateId : DeclarationId := id "temporal.nexus.basic-lifecycle.state.operation"
def startActionId : DeclarationId := id "temporal.nexus.basic-lifecycle.action.start"
def cancelActionId : DeclarationId := id "temporal.nexus.basic-lifecycle.action.cancel"
def reportSuccessActionId : DeclarationId := id "temporal.nexus.basic-lifecycle.action.succeed"
def transitionOutcomeId : DeclarationId := id "temporal.nexus.basic-lifecycle.outcome.transition"
def lifecycleObservationId : DeclarationId := id "temporal.nexus.basic-lifecycle.observation.state"
def operationRoleId : DeclarationId := id "temporal.nexus.basic-lifecycle.role.operation"

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
def LawStatement (lawId : DeclarationId) : Prop :=
  lawId = lifecycleLawId ∧
    step .scheduled .start = some .started ∧
    step .started .cancel = some .canceled ∧
    step .started .succeed = some .succeeded

def lifecycleLaw : LawRequirement := {
  id := lifecycleLawId
  semanticDigest := "temporal-nexus-basic-lifecycle-authoritative-step/v2"
}

theorem lifecycleLawProof : LawStatement lifecycleLaw.id := by
  exact ⟨rfl, rfl, rfl, rfl⟩

private def metadata
    (declarationId : DeclarationId)
    (kind : DeclarationKind)
    (contractDigest : String) : DeclarationMetadata := {
  id := declarationId
  kind
  source
  contractDigest
}

def scheduledState : SemanticValue := { identity := operationStateId, value := "scheduled" }
def startedState : SemanticValue := { identity := operationStateId, value := "started" }
def canceledState : SemanticValue := { identity := operationStateId, value := "canceled" }
def succeededState : SemanticValue := { identity := operationStateId, value := "succeeded" }

def startAction : SemanticValue := { identity := startActionId, value := "start" }
def cancelAction : SemanticValue := { identity := cancelActionId, value := "cancel" }

/-- Successful completion is handler-reported lifecycle progress, not a caller command. -/
def reportSuccessAction : SemanticValue := {
  identity := reportSuccessActionId
  value := "handler-reports-success"
}

def startedOutcome : SemanticValue := { identity := transitionOutcomeId, value := "started" }
def canceledOutcome : SemanticValue := { identity := transitionOutcomeId, value := "canceled" }
def succeededOutcome : SemanticValue := { identity := transitionOutcomeId, value := "succeeded" }
def startedObservation : SemanticValue := {
  identity := lifecycleObservationId
  value := "started"
}
def canceledObservation : SemanticValue := {
  identity := lifecycleObservationId
  value := "canceled"
}
def succeededObservation : SemanticValue := {
  identity := lifecycleObservationId
  value := "succeeded"
}

def scheduledSetup : List RoleBinding := [{ role := operationRoleId, value := scheduledState }]
def startedSetup : List RoleBinding := [{ role := operationRoleId, value := startedState }]

def startedResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := startedOutcome
  resultingState := startedState
  observations := [startedObservation]
}

def canceledResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := canceledOutcome
  resultingState := canceledState
  observations := [canceledObservation]
}

def succeededResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := succeededOutcome
  resultingState := succeededState
  observations := [succeededObservation]
}

def lifecycleState? (state : SemanticValue) : Option OperationState :=
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

def lifecycleEvent? (action : SemanticValue) : Option OperationEvent :=
  if action = startAction then
    some .start
  else if action = cancelAction then
    some .cancel
  else if action = reportSuccessAction then
    some .succeed
  else
    none

def transitionResult? : OperationState → Option
    (TransitionResult SemanticValue SemanticValue SemanticValue)
  | .started => some startedResult
  | .canceled => some canceledResult
  | .succeeded => some succeededResult
  | _ => none

/-- Enumerate only exposed results reached through the focused `step` relation. -/
def stepResult? (state action : SemanticValue) : Option
    (TransitionResult SemanticValue SemanticValue SemanticValue) := do
  let lifecycleState ← lifecycleState? state
  let lifecycleEvent ← lifecycleEvent? action
  let resultingState ← step lifecycleState lifecycleEvent
  transitionResult? resultingState

def initialState? (setup : List RoleBinding) : Option SemanticValue :=
  if setup = scheduledSetup then
    some scheduledState
  else if setup = startedSetup then
    some startedState
  else
    none

def initialStates (setup : List RoleBinding) : List SemanticValue :=
  (initialState? setup).toList

def authoritativeInitial (setup : List RoleBinding) (state : SemanticValue) : Prop :=
  state ∈ initialStates setup

def stepResults
    (state action : SemanticValue) :
    List (TransitionResult SemanticValue SemanticValue SemanticValue) :=
  (stepResult? state action).toList

def authoritativeStep
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue) : Prop :=
  result ∈ stepResults state action

theorem initialStates_sound
    (setup : List RoleBinding)
    (state : SemanticValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state :=
  member

theorem initialStates_complete
    (setup : List RoleBinding)
    (state : SemanticValue)
    (admitted : authoritativeInitial setup state) :
    state ∈ initialStates setup :=
  admitted

theorem stepResults_sound
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (member : result ∈ stepResults state action) :
    authoritativeStep state action result :=
  member

theorem stepResults_complete
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action :=
  admitted

theorem initialStates_length_le_one (setup : List RoleBinding) :
    (initialStates setup).length ≤ 1 := by
  cases selected : initialState? setup <;> simp [initialStates, selected]

theorem stepResults_length_le_one (state action : SemanticValue) :
    (stepResults state action).length ≤ 1 := by
  cases selected : stepResult? state action <;> simp [stepResults, selected]

theorem step_action_exposed
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (member : result ∈ stepResults state action) :
    action = cancelAction ∨ action = startAction ∨ action = reportSuccessAction := by
  by_cases isCancel : action = cancelAction
  · exact .inl isCancel
  · by_cases isStart : action = startAction
    · exact .inr (.inl isStart)
    · by_cases isSuccess : action = reportSuccessAction
      · exact .inr (.inr isSuccess)
      · simp [stepResults, stepResult?, lifecycleEvent?, isStart, isCancel, isSuccess] at member

def transitionKernel : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := kernelId
    contractDigest := "temporal-nexus-basic-lifecycle-kernel/v2"
    source
  }
  initialStates
  authoritativeInitial
  initialSound := initialStates_sound
  initialComplete := initialStates_complete
  steps := stepResults
  authoritativeStep
  stepSound := stepResults_sound
  stepComplete := stepResults_complete
}

def lifecycleProvider : CapabilityProvider LawStatement := {
  id := lifecycleProviderId
  source
  contract := {
    id := lifecycleCapabilityId
    semanticDigest := "temporal-nexus-basic-lifecycle/v2"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { declaration := operationStateId, kind := .state,
      semanticDigest := "temporal-nexus-basic-lifecycle-state/v2" },
    { declaration := startActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-start/v1" },
    { declaration := cancelActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-cancel/v1" },
    { declaration := reportSuccessActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-report-success/v1" },
    { declaration := transitionOutcomeId, kind := .outcome,
      semanticDigest := "temporal-nexus-basic-lifecycle-outcome/v2" },
    { declaration := lifecycleObservationId, kind := .observation,
      semanticDigest := "temporal-nexus-basic-lifecycle-observation/v2" }
  ]
  lawWitnesses := [{ requirement := lifecycleLaw, proof := lifecycleLawProof }]
}

def declarations : List DeclarationMetadata := [
  metadata targetId .target "temporal-nexus-basic-lifecycle-target/v2",
  metadata kernelId .kernel "temporal-nexus-basic-lifecycle-kernel/v2",
  metadata lifecycleCapabilityId .capability "temporal-nexus-basic-lifecycle/v2",
  metadata lifecycleProviderId .provider "temporal-nexus-basic-lifecycle-provider/v2",
  metadata lifecycleLawId .law lifecycleLaw.semanticDigest,
  metadata operationStateId .state "temporal-nexus-basic-lifecycle-state/v2",
  metadata startActionId .action "temporal-nexus-basic-lifecycle-start/v1",
  metadata cancelActionId .action "temporal-nexus-basic-lifecycle-cancel/v1",
  metadata reportSuccessActionId .action "temporal-nexus-basic-lifecycle-report-success/v1",
  metadata transitionOutcomeId .outcome "temporal-nexus-basic-lifecycle-outcome/v2",
  metadata lifecycleObservationId .observation "temporal-nexus-basic-lifecycle-observation/v2"
]

def roleAssignments : List (List RoleBinding) := [scheduledSetup, startedSetup]
def actionDomain : List SemanticValue := [cancelAction, startAction, reportSuccessAction]

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep := {
  actions := actionDomain
  roleDomainDigest := "temporal-nexus-basic-lifecycle-role-domain/v1"
  actionDomainDigest := "temporal-nexus-basic-lifecycle-action-domain/v2"
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
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  id := targetId
  source
  declarations
  requiredCapabilities := [lifecycleCapabilityId]
  resolvedSetups := roleAssignments
  kernel := .checked transitionKernel
}

def targetComposition : TargetComposition LawStatement :=
  TargetComposition.empty |>.provide lifecycleProvider

def targetAuthoring : AuthoredTarget LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue :=
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

def bounds : QueryBounds := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 8, unit := .candidateEvaluations }
}

def policy : PlannerPolicy := {
  strategy := .shortest
  seed := 29
  tieBreak := .semanticIdentity
}

def queryContext : QueryCheckContext LawStatement := .ofTarget target

/-- Put checked queries back on the shared target so downstream planner proofs stay small. -/
def materializeQuery (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  checked with
  target
  completeness := (CheckedQueryTarget.ofTarget target).completeness
}

end Temporal.Feature.Nexus.Lifecycle
