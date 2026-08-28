import Temporal.Feature.Nexus.Experimental.AutoClose
import Temporal.Shared
import Umpire.Planning

/-!
# Experimental caller closure

This advanced module remains one file. Its sections proceed from ownership laws and model values,
through the checked Target and Properties, to Behaviors, Queries, deterministic planner runs, and
the selected Artifact. Start with `Temporal.Feature.Nexus` for the ordinary lifecycle and operation
walkthroughs; use this module only when following the caller-closure composition.
-/

namespace Temporal.Feature.Nexus.Experimental.CallerClosure

open _root_.Umpire
open Temporal.Feature.Nexus.Experimental.AutoClose

private def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus/Experimental/CallerClosure.lean"

def targetId : DefinitionId := id "workflow-nexus.target.caller-closure"
def kernelId : DefinitionId := id "workflow-nexus.kernel.caller-closure"
def workflowCapabilityId : DefinitionId := id "workflow.capability.lifecycle"
def cancellationCapabilityId : DefinitionId := id "nexus.capability.cancellation"
def ownershipCapabilityId : DefinitionId := id "workflow-nexus.capability.ownership"
def ownershipClaimCapabilityId : DefinitionId :=
  id "workflow-nexus.capability.ownership-claim-internal"
def workflowProviderId : DefinitionId := id "workflow.provider.lifecycle"
def cancellationProviderId : DefinitionId := id "nexus.provider.cancellation"
def workflowOwnershipClaimProviderId : DefinitionId := id "workflow.provider.ownership-claim"
def cancellationOwnershipClaimProviderId : DefinitionId := id "nexus.provider.ownership-claim"
def ownershipProviderId : DefinitionId := id "workflow-nexus.provider.ownership"
def ownershipConnectorId : DefinitionId := id "workflow-nexus.connector.ownership"
def lifecycleLawId : DefinitionId := id "workflow.law.caller-closure"
def cancellationLawId : DefinitionId := id "nexus.law.cancellation-honored"
def ownershipLawId : DefinitionId := id "workflow-nexus.law.ownership-reconciled"
def configStateId : DefinitionId := id "workflow-nexus.state.config"
def forceCloseActionId : DefinitionId := id "workflow.action.force-close"
def upgradedOutcomeId : DefinitionId := id "nexus.outcome.cancellation-upgraded"
def deliveredObservationId : DefinitionId := id "nexus.observation.cancellation-delivered"
def cancellationCountObservationId : DefinitionId :=
  id "nexus.observation.pending-cancellation-count"
def ownershipClaimId : DefinitionId := id "workflow-nexus.observation.ownership-claim"
def ownershipRelationId : DefinitionId := id "workflow-nexus.relation.owns-operation"
def operationRoleId : DefinitionId := id "workflow-nexus.role.operation"
def callerClosurePropertyId : DefinitionId := id "workflow-nexus.property.caller-closure"
def exploratoryBehaviorId : DefinitionId := id "workflow-nexus.behavior.exploratory"
def exactActionBehaviorId : DefinitionId := id "workflow-nexus.behavior.exact-action"
def exactTraceBehaviorId : DefinitionId := id "workflow-nexus.behavior.exact-trace"
def verifyQueryId : DefinitionId := id "workflow-nexus.query.verify-caller-closure"
def exploratoryQueryId : DefinitionId := id "workflow-nexus.query.explore-caller-closure"
def exactActionQueryId : DefinitionId := id "workflow-nexus.query.exact-action-caller-closure"
def exactTraceQueryId : DefinitionId := id "workflow-nexus.query.model-only-caller-closure"

def lifecycleLaw : LawDefinition := {
  id := lifecycleLawId
  body := "workflow-caller-closure-law/v1"
}

def cancellationLaw : LawDefinition := {
  id := cancellationLawId
  body := "nexus-cancellation-honored-law/v1"
}

def ownershipLaw : LawDefinition := {
  id := ownershipLawId
  body := "workflow-nexus-ownership-law/v1"
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

def LawStatement (law : LawDefinition) : Prop :=
  if law = lifecycleLaw then
    Reachable .upgrade wClash
  else if law = cancellationLaw then
    Honored (autoClose .upgrade wClash)
  else if law = ownershipLaw then
    OwnershipReconciled
  else
    False

theorem ownershipReconciledProof : OwnershipReconciled := by
  exact ⟨wClash_reachable .upgrade, clashOwnershipProof⟩

theorem lifecycleLawProof : LawStatement lifecycleLaw := by
  simpa [LawStatement, lifecycleLaw, lifecycleLawId, id, DefinitionId.of] using
    wClash_reachable .upgrade

theorem cancellationLawProof : LawStatement cancellationLaw := by
  simpa [LawStatement, lifecycleLaw, cancellationLaw, lifecycleLawId, cancellationLawId, id,
    DefinitionId.of] using upgrade_honors_delivery wClash

theorem ownershipLawProof : LawStatement ownershipLaw := by
  simpa [LawStatement, lifecycleLaw, cancellationLaw, ownershipLaw, lifecycleLawId,
    cancellationLawId, ownershipLawId, id, DefinitionId.of] using ownershipReconciledProof

private def witness
    (definition : LawDefinition)
    (proof : LawStatement definition) : LawWitness LawStatement := {
  definition
  proof
}

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata definitionId kind source canonicalBehavior

private def opStateRepr : OpState → String
  | .unspecified => "NexusAutoClose.OpState.unspecified"
  | .scheduled => "NexusAutoClose.OpState.scheduled"
  | .backingOff => "NexusAutoClose.OpState.backingOff"
  | .started => "NexusAutoClose.OpState.started"
  | .succeeded => "NexusAutoClose.OpState.succeeded"
  | .failed => "NexusAutoClose.OpState.failed"
  | .canceled => "NexusAutoClose.OpState.canceled"
  | .timedOut => "NexusAutoClose.OpState.timedOut"
  | .terminated => "NexusAutoClose.OpState.terminated"
  | .rejected => "NexusAutoClose.OpState.rejected"

private def policyRepr : Policy → String
  | .abandon => "NexusAutoClose.Policy.abandon"
  | .requestCancel => "NexusAutoClose.Policy.requestCancel"

private def initiatorRepr : Initiator → String
  | .user => "NexusAutoClose.Initiator.user"
  | .system => "NexusAutoClose.Initiator.system"

private def initiatorsRepr (initiators : List Initiator) : String :=
  "[" ++ String.intercalate ", " (initiators.map initiatorRepr) ++ "]"

private def configRepr (config : Config) : String :=
  "{ op := " ++ opStateRepr config.op ++ ",\n" ++
  "  policy := " ++ policyRepr config.policy ++ ",\n" ++
  "  cancels := " ++ initiatorsRepr config.cancels ++ ",\n" ++
  "  callerOpen := " ++ toString config.callerOpen ++ ",\n" ++
  "  slack := " ++ toString config.slack ++ " }"

def clashState : ModelValue := {
  definitionId := configStateId
  value := configRepr wClash
}

def closedConfig : Config := autoClose .upgrade wClash

def closedState : ModelValue := {
  definitionId := configStateId
  value := configRepr closedConfig
}

def forceCloseAction : ModelValue := {
  definitionId := forceCloseActionId
  value := "force-close"
}

def upgradedOutcome : ModelValue := {
  definitionId := upgradedOutcomeId
  value := "upgrade"
}

def deliveredObservation : ModelValue := {
  definitionId := deliveredObservationId
  value := toString (delivers closedConfig)
}

def cancellationCountObservation : ModelValue := {
  definitionId := cancellationCountObservationId
  value := toString closedConfig.cancels.length
}

def ownershipObservation : ModelValue := {
  definitionId := ownershipRelationId
  value := toString (decide (CallerOwnsOperation wClash closedConfig))
}

def clashSetup : List RoleBinding := [{ role := operationRoleId, value := clashState }]

def forceCloseResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := upgradedOutcome
  resultingState := closedState
  observations := [deliveredObservation, cancellationCountObservation, ownershipObservation]
}

def authoritativeInitial (setup : List RoleBinding) (state : ModelValue) : Prop :=
  setup = clashSetup ∧ state = clashState ∧ Reachable .upgrade wClash

def authoritativeStep
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : Prop :=
  state = clashState ∧ action = forceCloseAction ∧ result = forceCloseResult ∧
    Honored closedConfig ∧ AtMostOneEvent closedConfig ∧ OwnershipReconciled

def transitionKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := {
    id := kernelId
    source
  }
  setupDomain := fun candidate => candidate = clashSetup
  stateDomain := fun candidate => candidate = clashState ∨ candidate = closedState
  actionDomain := fun candidate => candidate = forceCloseAction
  outcomeDomain := fun candidate => candidate = upgradedOutcome
  observationDomain := fun candidate => candidate = deliveredObservation ∨
    candidate = cancellationCountObservation ∨ candidate = ownershipObservation
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
  behaviorDomain := .complete {
    setups := [clashSetup]
    states := [clashState, closedState]
    actions := [forceCloseAction]
    outcomes := [upgradedOutcome]
    observations := [deliveredObservation, cancellationCountObservation, ownershipObservation]
    encodeSetup := fun bindings => String.intercalate "|" (bindings.map fun binding =>
      binding.role.value ++ "=" ++ binding.value.definitionId.value ++ ":" ++ binding.value.value)
    encodeState := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    encodeAction := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    encodeOutcome := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    encodeObservation := fun modelValue => modelValue.definitionId.value ++ ":" ++ modelValue.value
    setupSound := by intro candidate member; simpa using member
    setupComplete := by intro candidate admitted; simpa using admitted
    stateSound := by intro candidate member; simpa using member
    stateComplete := by intro candidate admitted; simpa using admitted
    actionSound := by intro candidate member; simpa using member
    actionComplete := by intro candidate admitted; simpa using admitted
    outcomeSound := by intro candidate member; simpa using member
    outcomeComplete := by intro candidate admitted; simpa using admitted
    observationSound := by intro candidate member; simpa using member
    observationComplete := by intro candidate admitted; simpa using admitted
    setupCoverage := by
      intro setup state member
      by_cases selected : setup = clashSetup
      · simp [selected]
      · simp [selected] at member
    initialStateCoverage := by
      intro setup state member
      by_cases selected : setup = clashSetup
      · rw [if_pos selected] at member
        simp [List.mem_singleton.mp member]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    transitionSourceCoverage := by
      intro state action result member
      by_cases selected : state = clashState ∧ action = forceCloseAction
      · simp [selected.1]
      · simp [selected] at member
    actionCoverage := by
      intro state action result member
      by_cases selected : state = clashState ∧ action = forceCloseAction
      · simp [selected.2]
      · simp [selected] at member
    resultingStateCoverage := by
      intro state action result member
      by_cases selected : state = clashState ∧ action = forceCloseAction
      · rw [if_pos selected] at member
        simp [List.mem_singleton.mp member, forceCloseResult]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    outcomeCoverage := by
      intro state action result member
      by_cases selected : state = clashState ∧ action = forceCloseAction
      · rw [if_pos selected] at member
        simp [List.mem_singleton.mp member, forceCloseResult]
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
    observationCoverage := by
      intro state action result observation member observationMember
      by_cases selected : state = clashState ∧ action = forceCloseAction
      · rw [if_pos selected] at member
        simpa [List.mem_singleton.mp member, forceCloseResult] using observationMember
      · rw [if_neg selected] at member
        exact (List.not_mem_nil member).elim
  }
}

def workflowProvider : CapabilityProvider LawStatement := {
  id := workflowProviderId
  source
  contract := {
    id := workflowCapabilityId
    canonicalBehavior := "workflow-lifecycle/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { definitionId := configStateId, kind := .state,
      canonicalBehavior := "workflow-config-state/v1" }
  ]
  lawWitnesses := [witness lifecycleLaw lifecycleLawProof]
}

def cancellationProvider : CapabilityProvider LawStatement := {
  id := cancellationProviderId
  source
  contract := {
    id := cancellationCapabilityId
    canonicalBehavior := "nexus-cancellation/v1"
    requiredLaws := [cancellationLaw]
  }
  meanings := [
    { definitionId := forceCloseActionId, kind := .action,
      canonicalBehavior := "workflow-force-close-action/v1" },
    { definitionId := upgradedOutcomeId, kind := .outcome,
      canonicalBehavior := "nexus-upgraded-cancellation-outcome/v1" },
    { definitionId := deliveredObservationId, kind := .observation,
      canonicalBehavior := "nexus-cancellation-delivery-observation/v1" },
    { definitionId := cancellationCountObservationId, kind := .observation,
      canonicalBehavior := "nexus-cancellation-count-observation/v1" }
  ]
  lawWitnesses := [witness cancellationLaw cancellationLawProof]
}

def workflowOwnershipClaimProvider : CapabilityProvider LawStatement := {
  id := workflowOwnershipClaimProviderId
  source
  contract := {
    id := ownershipClaimCapabilityId
    canonicalBehavior := "workflow-nexus-ownership-claim-internal/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [{
    definitionId := ownershipClaimId
    kind := .observation
    canonicalBehavior := "workflow-operation-ownership-claim/v1"
  }]
  lawWitnesses := [witness lifecycleLaw lifecycleLawProof]
}

def cancellationOwnershipClaimProvider : CapabilityProvider LawStatement := {
  id := cancellationOwnershipClaimProviderId
  source
  contract := {
    id := ownershipClaimCapabilityId
    canonicalBehavior := "workflow-nexus-ownership-claim-internal/v1"
    requiredLaws := [cancellationLaw]
  }
  meanings := [{
    definitionId := ownershipClaimId
    kind := .observation
    canonicalBehavior := "nexus-operation-ownership-claim/v1"
  }]
  lawWitnesses := [witness cancellationLaw cancellationLawProof]
}

def ownershipProvider : CapabilityProvider LawStatement := {
  id := ownershipProviderId
  source
  contract := {
    id := ownershipCapabilityId
    canonicalBehavior := "workflow-nexus-ownership/v1"
    requiredLaws := [ownershipLaw]
  }
  meanings := [{
    definitionId := ownershipRelationId
    kind := .observation
    canonicalBehavior := "workflow-nexus-operation-ownership/v1"
  }]
  lawWitnesses := [witness ownershipLaw ownershipLawProof]
}

def ownershipConnector : CapabilityConnector LawStatement := {
  id := ownershipConnectorId
  source
  canonicalBehavior := "workflow-nexus-ownership-connector/v1"
  reconciliations := [{
    definitionId := ownershipClaimId
    kind := .observation
    providers := [workflowOwnershipClaimProviderId, cancellationOwnershipClaimProviderId]
    canonicalBehavior := "workflow-nexus-operation-ownership-claim/v1"
  }]
  requiredLaws := [ownershipLaw]
  lawWitnesses := [witness ownershipLaw ownershipLawProof]
}

def definitions : List DefinitionMetadata := [
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
  metadata lifecycleLawId .law lifecycleLaw.body,
  metadata cancellationLawId .law cancellationLaw.body,
  metadata ownershipLawId .law ownershipLaw.body,
  metadata configStateId .state "workflow-config-state/v1",
  metadata forceCloseActionId .action "workflow-force-close-action/v1",
  metadata upgradedOutcomeId .outcome "nexus-upgraded-cancellation-outcome/v1",
  metadata deliveredObservationId .observation "nexus-cancellation-delivery-observation/v1",
  metadata cancellationCountObservationId .observation "nexus-cancellation-count-observation/v1",
  metadata ownershipClaimId .observation "workflow-nexus-operation-ownership-claim/v1",
  metadata ownershipRelationId .observation "workflow-nexus-operation-ownership/v1"
]

def targetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [
    workflowCapabilityId,
    cancellationCapabilityId,
    ownershipCapabilityId
  ]
  resolvedSetups := [clashSetup]
  kernel := .checked transitionKernel
}

def targetComposition : TargetComposition LawStatement :=
  TargetComposition.empty
    |>.provide workflowProvider
    |>.provide cancellationProvider
    |>.provide workflowOwnershipClaimProvider
    |>.provide cancellationOwnershipClaimProvider
    |>.provide ownershipProvider
    |>.connect ownershipConnector

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep := {
  actions := [forceCloseAction]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨clashState, forceCloseResult, by
      change authoritativeStep clashState forceCloseAction forceCloseResult
      exact ⟨rfl, rfl, rfl,
        upgrade_honors_delivery wClash,
        upgrade_preserves_uniqueness wClash (wClash_reachable .upgrade),
        ownershipReconciledProof⟩⟩
  actionComplete := by
    intro state action result admitted
    change authoritativeStep state action result at admitted
    simp [admitted.2.1]
}

def targetAuthoring : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition targetComposition
    (.available transitionKernel rfl finitePlanning)

/-- Re-ascribe the source kernel after checked composition so its proof relation remains reducible. -/
def target : QueryTarget LawStatement := checkedTarget targetAuthoring

theorem target_initial
    (setup : List RoleBinding)
    (state : ModelValue)
    (admitted : target.kernel.authoritativeInitial setup state) :
    setup = clashSetup ∧ state = clashState := by
  change authoritativeInitial setup state at admitted
  exact ⟨admitted.1, admitted.2.1⟩

theorem target_step
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
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
  checkBehavior (.ofTarget target) declaration

def exploratoryBehaviorResult : Except BehaviorError CheckedBehavior :=
  checkBehaviorDeclaration exploratoryBehaviorDeclaration
def exactActionBehaviorResult : Except BehaviorError CheckedBehavior :=
  checkBehaviorDeclaration exactActionBehaviorDeclaration
def exactTraceBehaviorResult : Except BehaviorError CheckedBehavior :=
  checkBehaviorDeclaration exactTraceBehaviorDeclaration

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

def limits : QueryLimits := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 8, unit := .candidateEvaluations }
}

def exhaustivePolicy : PlannerPolicy := {
  strategy := .exhaustive
  seed := 17
  tieBreak := .definitionId
}

def shortestPolicy : PlannerPolicy := {
  strategy := .shortest
  seed := 17
  tieBreak := .definitionId
}

def queryContext : QueryCheckContext LawStatement := .ofTarget target

private def queryDeclaration
    (queryId : DefinitionId)
    (form : QueryForm)
    (behavior : CheckedBehavior)
    (policy : PlannerPolicy) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form
  behavior
  limits
  policy
}

def verifyQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (queryDeclaration verifyQueryId (.verify callerClosureProperty)
    exactActionBehavior exhaustivePolicy)

def exploratoryQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (queryDeclaration exploratoryQueryId (.select [callerClosureProperty])
    exploratoryBehavior shortestPolicy)

def exactActionQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (queryDeclaration exactActionQueryId (.witness callerClosureProperty)
    exactActionBehavior shortestPolicy)

def exactTraceQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext (queryDeclaration exactTraceQueryId (.witness callerClosureProperty)
    exactTraceBehavior shortestPolicy)

private theorem verifyQueryResult_isSome : verifyQueryResult.toOption.isSome = true := by native_decide
private theorem exploratoryQueryResult_isSome :
    exploratoryQueryResult.toOption.isSome = true := by native_decide
private theorem exactActionQueryResult_isSome :
    exactActionQueryResult.toOption.isSome = true := by native_decide
private theorem exactTraceQueryResult_isSome :
    exactTraceQueryResult.toOption.isSome = true := by native_decide

private def materializeQuery (checked : CheckedQuery LawStatement) : CheckedQuery LawStatement := {
  checked with
  target
  completeness := (CheckedQueryTarget.ofTarget target).completeness
}

def verifyQuery : CheckedQuery LawStatement := materializeQuery
  (verifyQueryResult.toOption.get verifyQueryResult_isSome)

def exploratoryQuery : CheckedQuery LawStatement := materializeQuery
  (exploratoryQueryResult.toOption.get exploratoryQueryResult_isSome)

def exactActionQuery : CheckedQuery LawStatement := materializeQuery
  (exactActionQueryResult.toOption.get exactActionQueryResult_isSome)

def exactTraceQuery : CheckedQuery LawStatement := materializeQuery
  (exactTraceQueryResult.toOption.get exactTraceQueryResult_isSome)

private def incrementalKernel? : Option (IncrementalPlannerKernel exactActionQuery.target) :=
  IncrementalPlannerKernel.ofCheckedQuery? exactActionQuery
    (by
      intro evidence evidenceEq
      simp [exactActionQuery, materializeQuery, CheckedQueryTarget.ofTarget, target,
        checkedTarget, targetAuthoring, AuthoredTarget.make, targetDefinition] at evidenceEq
      cases Option.some.inj evidenceEq
      simp [finitePlanning])
    (by
      intro _ _ setup
      simp only [exactActionQuery, materializeQuery, target, checkedTarget, targetAuthoring,
        AuthoredTarget.make, targetDefinition,
        transitionKernel]
      split <;> simp)
    (by
      intro _ _ state action
      simp only [exactActionQuery, materializeQuery, target, checkedTarget, targetAuthoring,
        AuthoredTarget.make, targetDefinition,
        transitionKernel]
      split <;> simp)

private theorem incrementalKernel?_isSome : incrementalKernel?.isSome = true := by
  rfl

def incrementalKernel : IncrementalPlannerKernel target :=
  incrementalKernel?.get incrementalKernel?_isSome

theorem verifyQuery_target : verifyQuery.target = target := by rfl
theorem exploratoryQuery_target : exploratoryQuery.target = target := by rfl
theorem exactActionQuery_target : exactActionQuery.target = target := by rfl
theorem exactTraceQuery_target : exactTraceQuery.target = target := by rfl

def verifyRun : PlannerRun := plan verifyQuery incrementalKernel

def exploratoryRun : PlannerRun :=
  plan exploratoryQuery incrementalKernel

def exactActionRun : PlannerRun :=
  plan exactActionQuery incrementalKernel

def exactTraceRun : PlannerRun :=
  plan exactTraceQuery incrementalKernel

def artifact : Option ExperimentSpec := exactActionRun.artifact

private theorem artifact_isSome : artifact.isSome = true := by native_decide

def compiledArtifact : ExperimentSpec := artifact.get artifact_isSome

end Temporal.Feature.Nexus.Experimental.CallerClosure
