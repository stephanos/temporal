import NexusAutoClose
import Umpire.Planning

namespace Temporal.Umpire.NexusCallerClosure

open _root_.Umpire
open Temporal.Feature.Nexus.AutoClose

private def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Temporal/Umpire/NexusCallerClosure.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId := id "workflow-nexus.target.caller-closure"
def kernelId := id "workflow-nexus.kernel.caller-closure"
def workflowCapabilityId := id "workflow.capability.lifecycle"
def cancellationCapabilityId := id "nexus.capability.cancellation"
def ownershipCapabilityId := id "workflow-nexus.capability.ownership"
def ownershipClaimCapabilityId := id "workflow-nexus.capability.ownership-claim-internal"
def workflowProviderId := id "workflow.provider.lifecycle"
def cancellationProviderId := id "nexus.provider.cancellation"
def workflowOwnershipClaimProviderId := id "workflow.provider.ownership-claim"
def cancellationOwnershipClaimProviderId := id "nexus.provider.ownership-claim"
def ownershipProviderId := id "workflow-nexus.provider.ownership"
def ownershipConnectorId := id "workflow-nexus.connector.ownership"
def lifecycleLawId := id "workflow.law.caller-closure"
def cancellationLawId := id "nexus.law.cancellation-honored"
def ownershipLawId := id "workflow-nexus.law.ownership-reconciled"
def configStateId := id "workflow-nexus.state.config"
def forceCloseActionId := id "workflow.action.force-close"
def upgradedOutcomeId := id "nexus.outcome.cancellation-upgraded"
def deliveredObservationId := id "nexus.observation.cancellation-delivered"
def cancellationCountObservationId := id "nexus.observation.pending-cancellation-count"
def ownershipClaimId := id "workflow-nexus.observation.ownership-claim"
def ownershipRelationId := id "workflow-nexus.relation.owns-operation"
def operationRoleId := id "workflow-nexus.role.operation"
def callerClosurePropertyId := id "workflow-nexus.property.caller-closure"
def exploratoryBehaviorId := id "workflow-nexus.behavior.exploratory"
def exactActionBehaviorId := id "workflow-nexus.behavior.exact-action"
def exactTraceBehaviorId := id "workflow-nexus.behavior.exact-trace"
def verifyQueryId := id "workflow-nexus.query.verify-caller-closure"
def exploratoryQueryId := id "workflow-nexus.query.explore-caller-closure"
def exactActionQueryId := id "workflow-nexus.query.exact-action-caller-closure"
def exactTraceQueryId := id "workflow-nexus.query.model-only-caller-closure"

def lifecycleLaw : LawRequirement := {
  id := lifecycleLawId
  semanticDigest := "workflow-caller-closure-law/v1"
}

def cancellationLaw : LawRequirement := {
  id := cancellationLawId
  semanticDigest := "nexus-cancellation-honored-law/v1"
}

def ownershipLaw : LawRequirement := {
  id := ownershipLawId
  semanticDigest := "workflow-nexus-ownership-law/v1"
}

def CallerOwnsOperation (caller operation : Config) : Prop :=
  caller.callerOpen = true ∧
    operation.callerOpen = false ∧
    operation.op = caller.op ∧ operation.policy = caller.policy

instance (caller operation : Config) : Decidable (CallerOwnsOperation caller operation) := by
  unfold CallerOwnsOperation
  infer_instance

theorem autoClosePreservesCallerOwnership
    (resolution : Resolution)
    (caller : Config)
    (callerOpen : caller.callerOpen = true)
    (requestCancel : caller.policy = .requestCancel)
    (operationStarted : caller.op = .started) :
    CallerOwnsOperation caller (autoClose resolution caller) := by
  exact ⟨callerOpen,
    by simp [autoClose_of_guard requestCancel operationStarted],
    autoClose_op resolution caller,
    autoClose_policy resolution caller⟩

theorem clashOwnershipProof : CallerOwnsOperation wClash (autoClose .upgrade wClash) := by
  exact autoClosePreservesCallerOwnership .upgrade wClash rfl rfl rfl

def OwnershipReconciled : Prop :=
  Reachable .upgrade wClash ∧
    CallerOwnsOperation wClash (autoClose .upgrade wClash)

def LawStatement (lawId : DeclarationId) : Prop :=
  if lawId = lifecycleLaw.id then
    Reachable .upgrade wClash
  else if lawId = cancellationLaw.id then
    Honored (autoClose .upgrade wClash)
  else if lawId = ownershipLaw.id then
    OwnershipReconciled
  else
    False

theorem ownershipReconciledProof : OwnershipReconciled := by
  exact ⟨wClash_reachable .upgrade, clashOwnershipProof⟩

theorem lifecycleLawProof : LawStatement lifecycleLaw.id := by
  simpa [LawStatement, lifecycleLaw, lifecycleLawId, id, DeclarationId.of] using
    wClash_reachable .upgrade

theorem cancellationLawProof : LawStatement cancellationLaw.id := by
  simpa [LawStatement, lifecycleLaw, cancellationLaw, lifecycleLawId, cancellationLawId, id,
    DeclarationId.of] using upgrade_honors_delivery wClash

theorem ownershipLawProof : LawStatement ownershipLaw.id := by
  simpa [LawStatement, lifecycleLaw, cancellationLaw, ownershipLaw, lifecycleLawId,
    cancellationLawId, ownershipLawId, id, DeclarationId.of] using ownershipReconciledProof

private def witness
    (requirement : LawRequirement)
    (proof : LawStatement requirement.id) : LawWitness LawStatement := {
  requirement
  proof
}

private def metadata
    (declarationId : DeclarationId)
    (kind : DeclarationKind)
    (contractDigest : String) : DeclarationMetadata := {
  id := declarationId
  kind
  source
  contractDigest
}

def clashState : SemanticValue := {
  identity := configStateId
  value := reprStr wClash
}

def closedConfig : Config := autoClose .upgrade wClash

def closedState : SemanticValue := {
  identity := configStateId
  value := reprStr closedConfig
}

def forceCloseAction : SemanticValue := {
  identity := forceCloseActionId
  value := "force-close"
}

def upgradedOutcome : SemanticValue := {
  identity := upgradedOutcomeId
  value := "upgrade"
}

def deliveredObservation : SemanticValue := {
  identity := deliveredObservationId
  value := toString (delivers closedConfig)
}

def cancellationCountObservation : SemanticValue := {
  identity := cancellationCountObservationId
  value := toString closedConfig.cancels.length
}

def ownershipObservation : SemanticValue := {
  identity := ownershipRelationId
  value := toString (decide (CallerOwnsOperation wClash closedConfig))
}

def clashSetup : List RoleBinding := [{ role := operationRoleId, value := clashState }]

def forceCloseResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := upgradedOutcome
  resultingState := closedState
  observations := [deliveredObservation, cancellationCountObservation, ownershipObservation]
}

def authoritativeInitial (setup : List RoleBinding) (state : SemanticValue) : Prop :=
  setup = clashSetup ∧ state = clashState ∧ Reachable .upgrade wClash

def authoritativeStep
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue) : Prop :=
  state = clashState ∧ action = forceCloseAction ∧ result = forceCloseResult ∧
    Honored closedConfig ∧ AtMostOneEvent closedConfig ∧ OwnershipReconciled

def transitionKernel : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := kernelId
    contractDigest := "workflow-nexus-caller-closure-kernel/v1"
    source
  }
  initialStates := fun setup => if setup = clashSetup then [clashState] else []
  authoritativeInitial
  initialSound := by
    intro setup state member
    by_cases selected : setup = clashSetup
    · simp [selected] at member
      subst state
      exact ⟨selected, rfl, wClash_reachable .upgrade⟩
    · simp [selected] at member
  initialComplete := by
    intro setup state admitted
    rcases admitted with ⟨rfl, rfl, _⟩
    simp
  steps := fun state action =>
    if state = clashState ∧ action = forceCloseAction then [forceCloseResult] else []
  authoritativeStep
  stepSound := by
    intro state action result member
    by_cases selected : state = clashState ∧ action = forceCloseAction
    · simp [selected] at member
      subst result
      exact ⟨selected.1, selected.2, rfl,
        upgrade_honors_delivery wClash,
        upgrade_preserves_uniqueness wClash (wClash_reachable .upgrade),
        ownershipReconciledProof⟩
    · simp [selected] at member
  stepComplete := by
    intro state action result admitted
    rcases admitted with ⟨rfl, rfl, rfl, _, _, _⟩
    simp
}

def workflowProvider : CapabilityProvider LawStatement := {
  id := workflowProviderId
  source
  contract := {
    id := workflowCapabilityId
    semanticDigest := "workflow-lifecycle/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { declaration := configStateId, kind := .state,
      semanticDigest := "workflow-config-state/v1" }
  ]
  lawWitnesses := [witness lifecycleLaw lifecycleLawProof]
}

def cancellationProvider : CapabilityProvider LawStatement := {
  id := cancellationProviderId
  source
  contract := {
    id := cancellationCapabilityId
    semanticDigest := "nexus-cancellation/v1"
    requiredLaws := [cancellationLaw]
  }
  meanings := [
    { declaration := forceCloseActionId, kind := .action,
      semanticDigest := "workflow-force-close-action/v1" },
    { declaration := upgradedOutcomeId, kind := .outcome,
      semanticDigest := "nexus-upgraded-cancellation-outcome/v1" },
    { declaration := deliveredObservationId, kind := .observation,
      semanticDigest := "nexus-cancellation-delivery-observation/v1" },
    { declaration := cancellationCountObservationId, kind := .observation,
      semanticDigest := "nexus-cancellation-count-observation/v1" }
  ]
  lawWitnesses := [witness cancellationLaw cancellationLawProof]
}

def workflowOwnershipClaimProvider : CapabilityProvider LawStatement := {
  id := workflowOwnershipClaimProviderId
  source
  contract := {
    id := ownershipClaimCapabilityId
    semanticDigest := "workflow-nexus-ownership-claim-internal/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [{
    declaration := ownershipClaimId
    kind := .observation
    semanticDigest := "workflow-operation-ownership-claim/v1"
  }]
  lawWitnesses := [witness lifecycleLaw lifecycleLawProof]
}

def cancellationOwnershipClaimProvider : CapabilityProvider LawStatement := {
  id := cancellationOwnershipClaimProviderId
  source
  contract := {
    id := ownershipClaimCapabilityId
    semanticDigest := "workflow-nexus-ownership-claim-internal/v1"
    requiredLaws := [cancellationLaw]
  }
  meanings := [{
    declaration := ownershipClaimId
    kind := .observation
    semanticDigest := "nexus-operation-ownership-claim/v1"
  }]
  lawWitnesses := [witness cancellationLaw cancellationLawProof]
}

def ownershipProvider : CapabilityProvider LawStatement := {
  id := ownershipProviderId
  source
  contract := {
    id := ownershipCapabilityId
    semanticDigest := "workflow-nexus-ownership/v1"
    requiredLaws := [ownershipLaw]
  }
  meanings := [{
    declaration := ownershipRelationId
    kind := .observation
    semanticDigest := "workflow-nexus-operation-ownership/v1"
  }]
  lawWitnesses := [witness ownershipLaw ownershipLawProof]
}

def ownershipConnector : CapabilityConnector LawStatement := {
  id := ownershipConnectorId
  source
  semanticDigest := "workflow-nexus-ownership-connector/v1"
  reconciliations := [{
    declaration := ownershipClaimId
    kind := .observation
    providers := [workflowOwnershipClaimProviderId, cancellationOwnershipClaimProviderId]
    semanticDigest := "workflow-nexus-operation-ownership-claim/v1"
  }]
  requiredLaws := [ownershipLaw]
  lawWitnesses := [witness ownershipLaw ownershipLawProof]
}

def declarations : List DeclarationMetadata := [
  metadata targetId .target "workflow-nexus-caller-closure-target/v1",
  metadata kernelId .kernel "workflow-nexus-caller-closure-kernel/v1",
  metadata workflowCapabilityId .capability "workflow-lifecycle/v1",
  metadata cancellationCapabilityId .capability "nexus-cancellation/v1",
  metadata ownershipCapabilityId .capability "workflow-nexus-ownership/v1",
  metadata ownershipClaimCapabilityId .capability
    "workflow-nexus-ownership-claim-internal/v1",
  metadata workflowProviderId .provider "workflow-lifecycle-provider/v1",
  metadata cancellationProviderId .provider "nexus-cancellation-provider/v1",
  metadata workflowOwnershipClaimProviderId .provider "workflow-ownership-claim-provider/v1",
  metadata cancellationOwnershipClaimProviderId .provider "nexus-ownership-claim-provider/v1",
  metadata ownershipProviderId .provider "workflow-nexus-ownership-provider/v1",
  metadata ownershipConnectorId .connector "workflow-nexus-ownership-connector/v1",
  metadata lifecycleLawId .law lifecycleLaw.semanticDigest,
  metadata cancellationLawId .law cancellationLaw.semanticDigest,
  metadata ownershipLawId .law ownershipLaw.semanticDigest,
  metadata configStateId .state "workflow-config-state/v1",
  metadata forceCloseActionId .action "workflow-force-close-action/v1",
  metadata upgradedOutcomeId .outcome "nexus-upgraded-cancellation-outcome/v1",
  metadata deliveredObservationId .observation "nexus-cancellation-delivery-observation/v1",
  metadata cancellationCountObservationId .observation "nexus-cancellation-count-observation/v1",
  metadata ownershipClaimId .observation "workflow-nexus-operation-ownership-claim/v1",
  metadata ownershipRelationId .observation "workflow-nexus-operation-ownership/v1"
]

def targetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  id := targetId
  source
  declarations
  requiredCapabilities := [
    workflowCapabilityId,
    cancellationCapabilityId,
    ownershipCapabilityId
  ]
  providers := [
    workflowProvider,
    cancellationProvider,
    workflowOwnershipClaimProvider,
    cancellationOwnershipClaimProvider,
    ownershipProvider
  ]
  connectors := [ownershipConnector]
  resolvedSetups := [clashSetup]
  kernel := .checked transitionKernel
}

def targetResult := composeTarget targetDeclaration

private theorem targetResult_isSome : targetResult.toOption.isSome = true := by
  native_decide

private def composedTarget : QueryTarget LawStatement :=
  targetResult.toOption.get targetResult_isSome

/-- Re-ascribe the source kernel after checked composition so its proof relation remains reducible. -/
def target : QueryTarget LawStatement := {
  composedTarget with kernel := transitionKernel
}

theorem target_initial
    (setup : List RoleBinding)
    (state : SemanticValue)
    (admitted : target.kernel.authoritativeInitial setup state) :
    setup = clashSetup ∧ state = clashState := by
  change authoritativeInitial setup state at admitted
  exact ⟨admitted.1, admitted.2.1⟩

theorem target_step
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (admitted : target.kernel.authoritativeStep state action result) :
    state = clashState ∧ action = forceCloseAction ∧ result = forceCloseResult := by
  change authoritativeStep state action result at admitted
  exact ⟨admitted.1, admitted.2.1, admitted.2.2.1⟩

theorem target_force_close_is_authoritative :
    target.kernel.authoritativeStep clashState forceCloseAction forceCloseResult := by
  change authoritativeStep clashState forceCloseAction forceCloseResult
  exact ⟨rfl, rfl, rfl,
    upgrade_honors_delivery wClash,
    upgrade_preserves_uniqueness wClash (wClash_reachable .upgrade),
    ownershipReconciledProof⟩

def propertyDeclaration : PropertyDeclaration := {
  id := callerClosurePropertyId
  source
  requires := [workflowCapabilityId, cancellationCapabilityId, ownershipCapabilityId]
  clauses := [
    .transitionContract (id "workflow-nexus.property.clause.delivery")
      { field := .selectedAction, reference := forceCloseActionId,
        constraint := .equals forceCloseAction.value }
      { field := .observation, reference := deliveredObservationId,
        constraint := .equals "true" },
    .inputOutput (id "workflow-nexus.property.clause.uniqueness")
      { field := .selectedAction, reference := forceCloseActionId,
        constraint := .equals forceCloseAction.value }
      { field := .observation, reference := cancellationCountObservationId,
        constraint := .naturalAtMost 1 },
    .inputOutput (id "workflow-nexus.property.clause.ownership")
      { field := .selectedAction, reference := forceCloseActionId,
        constraint := .equals forceCloseAction.value }
      { field := .observation, reference := ownershipRelationId,
        constraint := .equals "true" }
  ]
  documentation := "A force-closed caller retains one owned, deliverable Nexus cancellation."
}

def propertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def callerClosureProperty : CheckedProperty :=
  propertyResult.toOption.get propertyResult_isSome

def operationRole : ResourceRole := { id := operationRoleId, valueKind := .state }

def setupConstraint : SetupConstraint := {
  id := id "workflow-nexus.setup.operation-is-clash"
  relation := .equal
  left := .role operationRoleId
  right := .value clashState
}

def exploratoryBehaviorDeclaration : BehaviorDeclaration := {
  id := exploratoryBehaviorId
  source
  requires := [workflowCapabilityId, cancellationCapabilityId]
  roles := [operationRole]
  setup := [setupConstraint]
  allowedActions := [forceCloseActionId]
  occurrenceBounds := [OccurrenceBound.atMost forceCloseActionId 1]
  documentation := "Explore the bounded caller-closure model space."
}

def exactActionBehaviorDeclaration : BehaviorDeclaration := {
  id := exactActionBehaviorId
  source
  requires := [workflowCapabilityId, cancellationCapabilityId]
  roles := [operationRole]
  setup := [setupConstraint]
  allowedActions := [forceCloseActionId]
  requiredOccurrences := [{
    id := id "workflow-nexus.occurrence.force-close"
    action := forceCloseActionId
  }]
  occurrenceBounds := [OccurrenceBound.exactly forceCloseActionId 1]
  actionsExactly := some [forceCloseActionId]
  documentation := "Select exactly the caller force-close action while leaving outcomes to the model."
}

def exactTrace : AuthoredExactTrace := {
  setup := clashSetup
  initialState := some clashState
  steps := [{
    selectedAction := some forceCloseAction
    modelOutcome := some upgradedOutcome
    resultingState := some closedState
    observations := some forceCloseResult.observations
  }]
}

def exactTraceBehaviorDeclaration : BehaviorDeclaration := {
  exactActionBehaviorDeclaration with
  id := exactTraceBehaviorId
  traceExactly := some exactTrace
  documentation := "Replay the complete model-owned caller-closure trace."
}

def checkBehaviorDeclaration
    (declaration : BehaviorDeclaration) : Except BehaviorError CheckedBehavior :=
  checkBehavior { declarations := target.declarations } declaration

def exploratoryBehaviorResult := checkBehaviorDeclaration exploratoryBehaviorDeclaration
def exactActionBehaviorResult := checkBehaviorDeclaration exactActionBehaviorDeclaration
def exactTraceBehaviorResult := checkBehaviorDeclaration exactTraceBehaviorDeclaration

private theorem exploratoryBehaviorResult_isSome :
    exploratoryBehaviorResult.toOption.isSome = true := by native_decide

private theorem exactActionBehaviorResult_isSome :
    exactActionBehaviorResult.toOption.isSome = true := by native_decide

private theorem exactTraceBehaviorResult_isSome :
    exactTraceBehaviorResult.toOption.isSome = true := by native_decide

def exploratoryBehavior : CheckedBehavior :=
  exploratoryBehaviorResult.toOption.get exploratoryBehaviorResult_isSome

def exactActionBehavior : CheckedBehavior :=
  exactActionBehaviorResult.toOption.get exactActionBehaviorResult_isSome

def exactTraceBehavior : CheckedBehavior :=
  exactTraceBehaviorResult.toOption.get exactTraceBehaviorResult_isSome

def completeness : FiniteCompletenessEvidence LawStatement target := {
  roleAssignments := target.resolvedSetups
  actions := [forceCloseAction]
  roleDomainDigest := "workflow-nexus-role-domain/v1"
  actionDomainDigest := "workflow-nexus-action-domain/v1"
  roleSound := by simp
  roleComplete := by simp
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨clashState, forceCloseResult, target_force_close_is_authoritative⟩
  actionComplete := by
    intro state action result admitted
    have selected := (target_step state action result admitted).2.1
    simp [selected]
}

def bounds : QueryBounds := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 8, unit := .candidateEvaluations }
}

def exhaustivePolicy : PlannerPolicy := {
  strategy := .exhaustive
  seed := 17
  tieBreak := .semanticIdentity
}

def shortestPolicy : PlannerPolicy := {
  strategy := .shortest
  seed := 17
  tieBreak := .semanticIdentity
}

def queryContext : QueryCheckContext LawStatement := {
  target := .checked { target, completeness := some completeness }
}

private def queryDeclaration
    (queryId : DeclarationId)
    (form : QueryForm)
    (behavior : CheckedBehavior)
    (policy : PlannerPolicy) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form
  behavior
  bounds
  policy
}

def verifyQueryResult := checkQuery queryContext
  (queryDeclaration verifyQueryId (.verify callerClosureProperty)
    exactActionBehavior exhaustivePolicy)

def exploratoryQueryResult := checkQuery queryContext
  (queryDeclaration exploratoryQueryId (.select [callerClosureProperty])
    exploratoryBehavior shortestPolicy)

def exactActionQueryResult := checkQuery queryContext
  (queryDeclaration exactActionQueryId (.witness callerClosureProperty)
    exactActionBehavior shortestPolicy)

def exactTraceQueryResult := checkQuery queryContext
  (queryDeclaration exactTraceQueryId (.witness callerClosureProperty)
    exactTraceBehavior shortestPolicy)

private theorem verifyQueryResult_isSome : verifyQueryResult.toOption.isSome = true := by native_decide
private theorem exploratoryQueryResult_isSome :
    exploratoryQueryResult.toOption.isSome = true := by native_decide
private theorem exactActionQueryResult_isSome :
    exactActionQueryResult.toOption.isSome = true := by native_decide
private theorem exactTraceQueryResult_isSome :
    exactTraceQueryResult.toOption.isSome = true := by native_decide

private def materializeQuery (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
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

def verifyQuery := materializeQuery
  (verifyQueryResult.toOption.get verifyQueryResult_isSome)

def exploratoryQuery := materializeQuery
  (exploratoryQueryResult.toOption.get exploratoryQueryResult_isSome)

def exactActionQuery := materializeQuery
  (exactActionQueryResult.toOption.get exactActionQueryResult_isSome)

def exactTraceQuery := materializeQuery
  (exactTraceQueryResult.toOption.get exactTraceQueryResult_isSome)

def incrementalKernel : IncrementalPlannerKernel target := {
  actionLimit := 1
  actionAt := fun index => if index = 0 then some forceCloseAction else none
  initialLimit := fun setup => if setup = clashSetup then 1 else 0
  initialAt := fun setup index =>
    if setup = clashSetup ∧ index = 0 then some clashState else none
  stepLimit := fun state action =>
    if state = clashState ∧ action = forceCloseAction then 1 else 0
  stepAt := fun state action index =>
    if state = clashState ∧ action = forceCloseAction ∧ index = 0 then
      some forceCloseResult
    else
      none
  actionSound := by
    intro index action inBounds emitted
    simp only [Nat.lt_one_iff] at inBounds
    subst index
    simp at emitted
    subst action
    exact ⟨clashState, forceCloseResult, target_force_close_is_authoritative⟩
  actionComplete := by
    intro state action result admitted
    have selected := (target_step state action result admitted).2.1
    exact ⟨0, by simp, by simp [selected]⟩
  initialSound := by
    intro setup index state _ emitted
    by_cases selected : setup = clashSetup ∧ index = 0
    · simp [selected] at emitted
      subst state
      change authoritativeInitial setup clashState
      exact ⟨selected.1, rfl, wClash_reachable .upgrade⟩
    · simp [selected] at emitted
  initialComplete := by
    intro setup state admitted
    have selected := target_initial setup state admitted
    exact ⟨0, by simp [selected.1], by simp [selected.1, selected.2]⟩
  stepSound := by
    intro state action index result _ emitted
    by_cases selected : state = clashState ∧ action = forceCloseAction ∧ index = 0
    · simp [selected] at emitted
      subst result
      rw [selected.1, selected.2.1]
      exact target_force_close_is_authoritative
    · simp [selected] at emitted
  stepComplete := by
    intro state action result admitted
    have selected := target_step state action result admitted
    exact ⟨0, by simp [selected.1, selected.2.1],
      by simp [selected.1, selected.2.1, selected.2.2]⟩
  actionOrdered := by intros; simp_all [semanticValueOrderKey]
  initialOrdered := by intros; simp_all [semanticValueOrderKey]
  stepOrdered := by intros; simp_all [transitionResultOrderKey]
}

private def kernelFor
    (query : CheckedQuery LawStatement)
    (agreement : query.target = target) : IncrementalPlannerKernel query.target := by
  rw [agreement]
  exact incrementalKernel

theorem verifyQuery_target : verifyQuery.target = target := by rfl
theorem exploratoryQuery_target : exploratoryQuery.target = target := by rfl
theorem exactActionQuery_target : exactActionQuery.target = target := by rfl
theorem exactTraceQuery_target : exactTraceQuery.target = target := by rfl

def verifyRun : PlannerRun := plan verifyQuery (kernelFor verifyQuery verifyQuery_target)

def exploratoryRun : PlannerRun :=
  plan exploratoryQuery (kernelFor exploratoryQuery exploratoryQuery_target)

def exactActionRun : PlannerRun :=
  plan exactActionQuery (kernelFor exactActionQuery exactActionQuery_target)

def exactTraceRun : PlannerRun :=
  plan exactTraceQuery (kernelFor exactTraceQuery exactTraceQuery_target)

def artifact : Option ExperimentSpec := exactActionRun.artifact

private theorem artifact_isSome : artifact.isSome = true := by native_decide

def compiledArtifact : ExperimentSpec := artifact.get artifact_isSome

end Temporal.Umpire.NexusCallerClosure
