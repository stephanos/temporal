import Temporal.Feature.Nexus.AutoClose
import Umpire.Planning

namespace Temporal.Feature.Nexus.Examples.BasicLifecycle

open Umpire

private def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Feature/Nexus/Examples/BasicLifecycle.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId := id "temporal.nexus.basic-lifecycle.target"
def kernelId := id "temporal.nexus.basic-lifecycle.kernel"
def lifecycleCapabilityId := id "temporal.nexus.basic-lifecycle.capability"
def lifecycleProviderId := id "temporal.nexus.basic-lifecycle.provider"
def lifecycleLawId := id "temporal.nexus.basic-lifecycle.law.authoritative-step"
def operationStateId := id "temporal.nexus.basic-lifecycle.state.operation"
def startActionId := id "temporal.nexus.basic-lifecycle.action.start"
def reportSuccessActionId := id "temporal.nexus.basic-lifecycle.action.succeed"
def transitionOutcomeId := id "temporal.nexus.basic-lifecycle.outcome.transition"
def lifecycleObservationId := id "temporal.nexus.basic-lifecycle.observation.state"
def operationRoleId := id "temporal.nexus.basic-lifecycle.role.operation"

/-- The provider law ties the teaching surface to the authoritative Nexus lifecycle. -/
def LawStatement (lawId : DeclarationId) : Prop :=
  lawId = lifecycleLawId ∧
    AutoClose.step .scheduled .start = some .started ∧
    AutoClose.step .started .succeed = some .succeeded

def lifecycleLaw : LawRequirement := {
  id := lifecycleLawId
  semanticDigest := "temporal-nexus-basic-lifecycle-authoritative-step/v1"
}

theorem lifecycleLawProof : LawStatement lifecycleLaw.id := by
  exact ⟨rfl, rfl, rfl⟩

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
def succeededState : SemanticValue := { identity := operationStateId, value := "succeeded" }

def startAction : SemanticValue := { identity := startActionId, value := "start" }

/-- Successful completion is handler-reported lifecycle progress, not a caller command. -/
def reportSuccessAction : SemanticValue := {
  identity := reportSuccessActionId
  value := "handler-reports-success"
}

def startedOutcome : SemanticValue := { identity := transitionOutcomeId, value := "started" }
def succeededOutcome : SemanticValue := { identity := transitionOutcomeId, value := "succeeded" }
def startedObservation : SemanticValue := {
  identity := lifecycleObservationId
  value := "started"
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

def succeededResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := succeededOutcome
  resultingState := succeededState
  observations := [succeededObservation]
}

def lifecycleState? (state : SemanticValue) : Option AutoClose.OpState :=
  if state = scheduledState then
    some .scheduled
  else if state = startedState then
    some .started
  else if state = succeededState then
    some .succeeded
  else
    none

def lifecycleEvent? (action : SemanticValue) : Option AutoClose.OpEvent :=
  if action = startAction then
    some .start
  else if action = reportSuccessAction then
    some .succeed
  else
    none

def transitionResult? : AutoClose.OpState → Option
    (TransitionResult SemanticValue SemanticValue SemanticValue)
  | .started => some startedResult
  | .succeeded => some succeededResult
  | _ => none

/-- Enumerate only exposed results reached through the authoritative `AutoClose.step`. -/
def stepResult? (state action : SemanticValue) : Option
    (TransitionResult SemanticValue SemanticValue SemanticValue) := do
  let lifecycleState ← lifecycleState? state
  let lifecycleEvent ← lifecycleEvent? action
  let resultingState ← AutoClose.step lifecycleState lifecycleEvent
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
    action = startAction ∨ action = reportSuccessAction := by
  by_cases isStart : action = startAction
  · exact .inl isStart
  · by_cases isSuccess : action = reportSuccessAction
    · exact .inr isSuccess
    · simp [stepResults, stepResult?, lifecycleEvent?, isStart, isSuccess] at member

def transitionKernel : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := kernelId
    contractDigest := "temporal-nexus-basic-lifecycle-kernel/v1"
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
    semanticDigest := "temporal-nexus-basic-lifecycle/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { declaration := operationStateId, kind := .state,
      semanticDigest := "temporal-nexus-basic-lifecycle-state/v1" },
    { declaration := startActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-start/v1" },
    { declaration := reportSuccessActionId, kind := .action,
      semanticDigest := "temporal-nexus-basic-lifecycle-report-success/v1" },
    { declaration := transitionOutcomeId, kind := .outcome,
      semanticDigest := "temporal-nexus-basic-lifecycle-outcome/v1" },
    { declaration := lifecycleObservationId, kind := .observation,
      semanticDigest := "temporal-nexus-basic-lifecycle-observation/v1" }
  ]
  lawWitnesses := [{ requirement := lifecycleLaw, proof := lifecycleLawProof }]
}

def declarations : List DeclarationMetadata := [
  metadata targetId .target "temporal-nexus-basic-lifecycle-target/v1",
  metadata kernelId .kernel "temporal-nexus-basic-lifecycle-kernel/v1",
  metadata lifecycleCapabilityId .capability "temporal-nexus-basic-lifecycle/v1",
  metadata lifecycleProviderId .provider "temporal-nexus-basic-lifecycle-provider/v1",
  metadata lifecycleLawId .law lifecycleLaw.semanticDigest,
  metadata operationStateId .state "temporal-nexus-basic-lifecycle-state/v1",
  metadata startActionId .action "temporal-nexus-basic-lifecycle-start/v1",
  metadata reportSuccessActionId .action "temporal-nexus-basic-lifecycle-report-success/v1",
  metadata transitionOutcomeId .outcome "temporal-nexus-basic-lifecycle-outcome/v1",
  metadata lifecycleObservationId .observation "temporal-nexus-basic-lifecycle-observation/v1"
]

def roleAssignments : List (List RoleBinding) := [scheduledSetup, startedSetup]
def actionDomain : List SemanticValue := [startAction, reportSuccessAction]

def targetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  id := targetId
  source
  declarations
  requiredCapabilities := [lifecycleCapabilityId]
  providers := [lifecycleProvider]
  connectors := []
  resolvedSetups := roleAssignments
  kernel := .checked transitionKernel
}

/-- Checked composition remains public so callers can inspect its typed declaration error. -/
def targetResult := composeTarget targetDeclaration

private theorem targetResult_isSome : targetResult.toOption.isSome = true := by
  native_decide

private def composedTarget : QueryTarget LawStatement :=
  targetResult.toOption.get targetResult_isSome

/-- Re-ascribe the source kernel after composition so its proof relation remains reducible. -/
def target : QueryTarget LawStatement := {
  composedTarget with kernel := transitionKernel
}

theorem target_resolvedSetups : target.resolvedSetups = roleAssignments := by
  native_decide

theorem target_scheduled_start_authoritative :
    target.kernel.authoritativeStep scheduledState startAction startedResult := by
  change startedResult ∈ stepResults scheduledState startAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    AutoClose.step, scheduledState, startAction]

theorem target_started_reportSuccess_authoritative :
    target.kernel.authoritativeStep startedState reportSuccessAction succeededResult := by
  change succeededResult ∈ stepResults startedState reportSuccessAction
  simp [stepResults, stepResult?, lifecycleState?, lifecycleEvent?, transitionResult?,
    AutoClose.step, scheduledState, startedState, startAction, reportSuccessAction]

def completeness : FiniteCompletenessEvidence LawStatement target := {
  roleAssignments
  actions := actionDomain
  roleDomainDigest := "temporal-nexus-basic-lifecycle-role-domain/v1"
  actionDomainDigest := "temporal-nexus-basic-lifecycle-action-domain/v1"
  roleSound := by
    intro setup member
    simpa [target_resolvedSetups] using member
  roleComplete := by
    intro setup member
    simpa [target_resolvedSetups] using member
  actionSound := by
    intro action member
    simp [actionDomain] at member
    rcases member with rfl | rfl
    · exact ⟨scheduledState, startedResult, target_scheduled_start_authoritative⟩
    · exact ⟨startedState, succeededResult, target_started_reportSuccess_authoritative⟩
  actionComplete := by
    intro state action result admitted
    change authoritativeStep state action result at admitted
    simpa [actionDomain] using step_action_exposed state action result admitted
}

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

def queryContext : QueryCheckContext LawStatement := {
  target := .checked { target, completeness := some completeness }
}

/-- Put checked queries back on the shared target so downstream planner proofs stay small. -/
def materializeQuery (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  id := checked.id
  source := checked.source
  version := checked.version
  form := checked.form
  quantifier := checked.quantifier
  claim := checked.claim
  behavior := checked.behavior
  target
  bounds := checked.bounds
  policy := checked.policy
  targetComposition := checked.targetComposition
  completeness := some completeness
  documentation := checked.documentation
  canonicalMetadata := checked.canonicalMetadata
  semanticDigest := checked.semanticDigest
}

def incrementalKernel : IncrementalPlannerKernel target :=
  .ofFinite completeness {
    action := by
      simp [completeness, actionDomain]
      native_decide
    initial := by
      intro setup
      simp [target, transitionKernel, initialStates]
    step := by
      intro state action
      simp [target, transitionKernel, stepResults]
  }

def kernelFor
    (query : CheckedQuery LawStatement)
    (agreement : query.target = target) : IncrementalPlannerKernel query.target := by
  rw [agreement]
  exact incrementalKernel

end Temporal.Feature.Nexus.Examples.BasicLifecycle
