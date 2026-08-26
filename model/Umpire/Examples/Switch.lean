import Umpire.Planning

namespace Umpire.Examples.Switch

private def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Examples/Switch.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId : DeclarationId := id "switch.target.two-state"
def kernelId : DeclarationId := id "switch.kernel.two-state"
def switchCapabilityId : DeclarationId := id "switch.capability.state"
def switchProviderId : DeclarationId := id "switch.provider.state"
def flipLawId : DeclarationId := id "switch.law.flip-preserves-domain"
def powerStateId : DeclarationId := id "switch.state.power"
def flipActionId : DeclarationId := id "switch.action.flip"
def appliedOutcomeId : DeclarationId := id "switch.outcome.applied"
def deferredOutcomeId : DeclarationId := id "switch.outcome.deferred"
def powerObservationId : DeclarationId := id "switch.observation.power"
def switchRoleId : DeclarationId := id "switch.role.subject"
def flipPropertyId : DeclarationId := id "switch.property.flip-turns-on"
def exploratoryBehaviorId : DeclarationId := id "switch.behavior.exploratory"
def exactActionBehaviorId : DeclarationId := id "switch.behavior.exact-action"
def exactTraceBehaviorId : DeclarationId := id "switch.behavior.exact-trace"
def exploratoryQueryId : DeclarationId := id "switch.query.explore"
def exactActionQueryId : DeclarationId := id "switch.query.exact-action"
def exactTraceQueryId : DeclarationId := id "switch.query.exact-trace"

inductive Position where
  | off
  | on
  deriving BEq, DecidableEq, Repr

def Position.flip : Position → Position
  | .off => .on
  | .on => .off

def LawStatement (lawId : DeclarationId) : Prop :=
  lawId = flipLawId ∧ Position.flip (Position.flip .off) = .off

def flipLaw : LawRequirement := {
  id := flipLawId
  semanticDigest := "switch-flip-preserves-domain-law/v1"
}

theorem flipLawProof : LawStatement flipLaw.id := by
  exact ⟨rfl, rfl⟩

private def metadata
    (declarationId : DeclarationId)
    (kind : DeclarationKind)
    (contractDigest : String) : DeclarationMetadata := {
  id := declarationId
  kind
  source
  contractDigest
}

def offState : SemanticValue := { identity := powerStateId, value := "off" }
def onState : SemanticValue := { identity := powerStateId, value := "on" }
def flipAction : SemanticValue := { identity := flipActionId, value := "flip" }
def appliedOutcome : SemanticValue := { identity := appliedOutcomeId, value := "applied" }
def deferredOutcome : SemanticValue := { identity := deferredOutcomeId, value := "deferred" }
def powerOffObservation : SemanticValue := { identity := powerObservationId, value := "off" }
def powerOnObservation : SemanticValue := { identity := powerObservationId, value := "on" }

theorem offState_ne_onState : offState ≠ onState := by
  decide

theorem onState_ne_offState : onState ≠ offState := by
  decide

def switchSetup : List RoleBinding := [{ role := switchRoleId, value := offState }]

def appliedResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := appliedOutcome
  resultingState := onState
  observations := [powerOnObservation]
}

def deferredResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := deferredOutcome
  resultingState := offState
  observations := [powerOffObservation]
}

def appliedFromOnResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := appliedOutcome
  resultingState := offState
  observations := [powerOffObservation]
}

def deferredFromOnResult : TransitionResult SemanticValue SemanticValue SemanticValue := {
  modelOutcome := deferredOutcome
  resultingState := onState
  observations := [powerOnObservation]
}

theorem appliedResult_ordered :
    transitionResultOrderKey appliedResult ≤ transitionResultOrderKey deferredResult := by
  decide

theorem appliedFromOnResult_ordered :
    transitionResultOrderKey appliedFromOnResult ≤
      transitionResultOrderKey deferredFromOnResult := by
  decide

def initialStates (setup : List RoleBinding) : List SemanticValue :=
  if setup = switchSetup then [offState] else []

def authoritativeInitial (setup : List RoleBinding) (state : SemanticValue) : Prop :=
  setup = switchSetup ∧ state = offState

def stepResults
    (state action : SemanticValue) :
    List (TransitionResult SemanticValue SemanticValue SemanticValue) :=
  if action = flipAction then
    if state = offState then
      [appliedResult, deferredResult]
    else if state = onState then
      [appliedFromOnResult, deferredFromOnResult]
    else
      []
  else
    []

def authoritativeStep
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue) : Prop :=
  action = flipAction ∧
    ((state = offState ∧ (result = appliedResult ∨ result = deferredResult)) ∨
      (state = onState ∧
        (result = appliedFromOnResult ∨ result = deferredFromOnResult)))

theorem initialStates_sound
    (setup : List RoleBinding)
    (state : SemanticValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state := by
  by_cases selected : setup = switchSetup
  · subst setup
    simp [initialStates, authoritativeInitial] at member ⊢
    exact member
  · simp [initialStates, selected] at member

theorem initialStates_complete
    (setup : List RoleBinding)
    (state : SemanticValue)
    (admitted : authoritativeInitial setup state) :
    state ∈ initialStates setup := by
  rcases admitted with ⟨rfl, rfl⟩
  simp [initialStates]

theorem stepResults_sound
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (member : result ∈ stepResults state action) :
    authoritativeStep state action result := by
  by_cases selectedAction : action = flipAction
  · subst action
    by_cases selectedOff : state = offState
    · subst state
      simp [stepResults, authoritativeStep, offState, onState] at member ⊢
      exact member
    · by_cases selectedOn : state = onState
      · subst state
        simp [stepResults, authoritativeStep, offState, onState] at member ⊢
        exact member
      · simp [stepResults, selectedOff, selectedOn] at member
  · simp [stepResults, selectedAction] at member

theorem stepResults_complete
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action := by
  rcases admitted with ⟨rfl, admitted⟩
  rcases admitted with ⟨rfl, admitted⟩ | ⟨rfl, admitted⟩
  · rcases admitted with rfl | rfl <;> simp [stepResults, offState]
  · rcases admitted with rfl | rfl <;> simp [stepResults, offState, onState]

def transitionKernel : TransitionKernel
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  metadata := {
    id := kernelId
    contractDigest := "switch-two-state-kernel/v1"
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

def switchProvider : CapabilityProvider LawStatement := {
  id := switchProviderId
  source
  contract := {
    id := switchCapabilityId
    semanticDigest := "switch-state/v1"
    requiredLaws := [flipLaw]
  }
  meanings := [
    { declaration := powerStateId, kind := .state, semanticDigest := "switch-power-state/v1" },
    { declaration := flipActionId, kind := .action, semanticDigest := "switch-flip-action/v1" },
    { declaration := appliedOutcomeId, kind := .outcome,
      semanticDigest := "switch-applied-outcome/v1" },
    { declaration := deferredOutcomeId, kind := .outcome,
      semanticDigest := "switch-deferred-outcome/v1" },
    { declaration := powerObservationId, kind := .observation,
      semanticDigest := "switch-power-observation/v1" }
  ]
  lawWitnesses := [{ requirement := flipLaw, proof := flipLawProof }]
}

def declarations : List DeclarationMetadata := [
  metadata targetId .target "switch-two-state-target/v1",
  metadata kernelId .kernel "switch-two-state-kernel/v1",
  metadata switchCapabilityId .capability "switch-state/v1",
  metadata switchProviderId .provider "switch-state-provider/v1",
  metadata flipLawId .law flipLaw.semanticDigest,
  metadata powerStateId .state "switch-power-state/v1",
  metadata flipActionId .action "switch-flip-action/v1",
  metadata appliedOutcomeId .outcome "switch-applied-outcome/v1",
  metadata deferredOutcomeId .outcome "switch-deferred-outcome/v1",
  metadata powerObservationId .observation "switch-power-observation/v1"
]

def targetDeclaration : TargetDeclaration LawStatement
    (List RoleBinding) SemanticValue SemanticValue SemanticValue SemanticValue := {
  id := targetId
  source
  declarations
  requiredCapabilities := [switchCapabilityId]
  providers := [switchProvider]
  connectors := []
  resolvedSetups := [switchSetup]
  kernel := .checked transitionKernel
}

def targetResult : Except DeclarationError (QueryTarget LawStatement) :=
  composeTarget targetDeclaration

private theorem targetResult_isSome : targetResult.toOption.isSome = true := by
  native_decide

private def composedTarget : QueryTarget LawStatement :=
  targetResult.toOption.get targetResult_isSome

/-- Re-ascribe the source kernel after checked composition so its proof relation remains reducible. -/
def target : QueryTarget LawStatement := {
  composedTarget with kernel := transitionKernel
}

theorem target_resolvedSetups : target.resolvedSetups = [switchSetup] := by
  native_decide

theorem target_initial
    (setup : List RoleBinding)
    (state : SemanticValue)
    (admitted : target.kernel.authoritativeInitial setup state) :
    setup = switchSetup ∧ state = offState := by
  exact admitted

theorem target_step
    (state action : SemanticValue)
    (result : TransitionResult SemanticValue SemanticValue SemanticValue)
    (admitted : target.kernel.authoritativeStep state action result) :
    authoritativeStep state action result := by
  exact admitted

theorem target_off_flip_applied_authoritative :
    target.kernel.authoritativeStep offState flipAction appliedResult := by
  change authoritativeStep offState flipAction appliedResult
  exact ⟨rfl, .inl ⟨rfl, .inl rfl⟩⟩

def propertyDeclaration : PropertyDeclaration := {
  id := flipPropertyId
  source
  requires := [switchCapabilityId]
  clauses := [
    .transitionContract (id "switch.property.clause.flip-turns-on")
      { field := .selectedAction, reference := flipActionId,
        constraint := .equals flipAction.value }
      { field := .resultingState, reference := powerStateId,
        constraint := .equals onState.value }
  ]
  documentation := "A selected flip has an outcome that turns the switch on."
}

def propertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def flipProperty : CheckedProperty :=
  propertyResult.toOption.get propertyResult_isSome

def switchRole : ResourceRole := { id := switchRoleId, valueKind := .state }

def setupConstraint : SetupConstraint := {
  id := id "switch.setup.subject-is-off"
  relation := .equal
  left := .role switchRoleId
  right := .value offState
}

def exploratoryBehaviorDeclaration : BehaviorDeclaration := {
  id := exploratoryBehaviorId
  source
  requires := [switchCapabilityId]
  roles := [switchRole]
  setup := [setupConstraint]
  allowedActions := [flipActionId]
  requiredOccurrences := [{ id := id "switch.occurrence.flip", action := flipActionId }]
  occurrenceBounds := [OccurrenceBound.exactly flipActionId 1]
  documentation := "Explore the finite switch outcomes for one selected flip."
}

def exactActionBehaviorDeclaration : BehaviorDeclaration := {
  exploratoryBehaviorDeclaration with
  id := exactActionBehaviorId
  actionsExactly := some [flipActionId]
  documentation := "Select one flip while leaving its outcome to the switch model."
}

def exactTrace : AuthoredExactTrace := {
  setup := switchSetup
  initialState := some offState
  steps := [{
    selectedAction := some flipAction
    modelOutcome := some appliedOutcome
    resultingState := some onState
    observations := some appliedResult.observations
  }]
}

def exactTraceBehaviorDeclaration : BehaviorDeclaration := {
  exactActionBehaviorDeclaration with
  id := exactTraceBehaviorId
  traceExactly := some exactTrace
  documentation := "Select the complete applied flip trace."
}

private def checkBehaviorDeclaration
    (declaration : BehaviorDeclaration) : Except BehaviorError CheckedBehavior :=
  checkBehavior { declarations := target.declarations } declaration

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

def appliedTrace : BehaviorTrace := {
  setup := switchSetup
  trace := {
    initialState := offState
    steps := [{
      selectedAction := flipAction
      modelOutcome := appliedOutcome
      resultingState := onState
      observations := appliedResult.observations
    }]
  }
}

def deferredTrace : BehaviorTrace := {
  setup := switchSetup
  trace := {
    initialState := offState
    steps := [{
      selectedAction := flipAction
      modelOutcome := deferredOutcome
      resultingState := offState
      observations := deferredResult.observations
    }]
  }
}

def completeness : FiniteCompletenessEvidence LawStatement target := {
  roleAssignments := [switchSetup]
  actions := [flipAction]
  roleDomainDigest := "switch-role-domain/v1"
  actionDomainDigest := "switch-action-domain/v1"
  roleSound := by simp [target_resolvedSetups]
  roleComplete := by simp [target_resolvedSetups]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨offState, appliedResult, target_off_flip_applied_authoritative⟩
  actionComplete := by
    intro state action result admitted
    have selected := (target_step state action result admitted).1
    simp [selected]
}

def bounds : QueryBounds := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 8, unit := .candidateEvaluations }
}

def shortestPolicy : PlannerPolicy := {
  strategy := .shortest
  seed := 23
  tieBreak := .semanticIdentity
}

def queryContext : QueryCheckContext LawStatement := {
  target := .checked { target, completeness := some completeness }
}

private def queryDeclaration
    (queryId : DeclarationId)
    (form : QueryForm)
    (behavior : CheckedBehavior) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form
  behavior
  bounds
  policy := shortestPolicy
}

def exploratoryQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext
    (queryDeclaration exploratoryQueryId (.select [flipProperty]) exploratoryBehavior)

def exactActionQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext
    (queryDeclaration exactActionQueryId (.witness flipProperty) exactActionBehavior)

def exactTraceQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  checkQuery queryContext
    (queryDeclaration exactTraceQueryId (.witness flipProperty) exactTraceBehavior)

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

def exploratoryQuery : CheckedQuery LawStatement := materializeQuery
  (exploratoryQueryResult.toOption.get exploratoryQueryResult_isSome)

def exactActionQuery : CheckedQuery LawStatement := materializeQuery
  (exactActionQueryResult.toOption.get exactActionQueryResult_isSome)

def exactTraceQuery : CheckedQuery LawStatement := materializeQuery
  (exactTraceQueryResult.toOption.get exactTraceQueryResult_isSome)

theorem stepResults_length_le_two (state action : SemanticValue) :
    (stepResults state action).length ≤ 2 := by
  by_cases action = flipAction <;>
    by_cases state = offState <;>
    by_cases state = onState <;>
    simp_all [stepResults]

def incrementalKernel : IncrementalPlannerKernel target :=
  .ofFinite completeness {
    action := by
      simp [completeness]
    initial := by
      intro setup
      simp only [target, transitionKernel, initialStates]
      split <;> simp
    step := by
      intro state action
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          simpa [target, transitionKernel, stepResults] using appliedResult_ordered
        · by_cases selectedOn : state = onState
          · subst state
            simpa [target, transitionKernel, stepResults, onState_ne_offState] using
              appliedFromOnResult_ordered
          · simp [target, transitionKernel, stepResults, selectedOff, selectedOn]
      · simp [target, transitionKernel, stepResults, selectedAction]
  }

theorem exploratoryQuery_target : exploratoryQuery.target = target := by rfl
theorem exactActionQuery_target : exactActionQuery.target = target := by rfl
theorem exactTraceQuery_target : exactTraceQuery.target = target := by rfl

private def kernelFor
    (query : CheckedQuery LawStatement)
    (agreement : query.target = target) : IncrementalPlannerKernel query.target := by
  rw [agreement]
  exact incrementalKernel

def exploratoryRun : PlannerRun :=
  plan exploratoryQuery (kernelFor exploratoryQuery exploratoryQuery_target)

def exactActionRun : PlannerRun :=
  plan exactActionQuery (kernelFor exactActionQuery exactActionQuery_target)

def exactTraceRun : PlannerRun :=
  plan exactTraceQuery (kernelFor exactTraceQuery exactTraceQuery_target)

def artifact : Option ExperimentSpec := exactActionRun.artifact

private theorem artifact_isSome : artifact.isSome = true := by
  native_decide

def compiledArtifact : ExperimentSpec := artifact.get artifact_isSome

end Umpire.Examples.Switch
