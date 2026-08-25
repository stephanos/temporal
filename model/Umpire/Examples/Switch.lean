import Umpire.Planning

namespace Umpire.Examples.Switch

private def id (value : String) : DeclarationId := DeclarationId.of value

def source : SemanticSource := {
  path := "Umpire/Examples/Switch.lean"
  line := 1
  column := 1
  provenance := "lean-model"
}

def targetId := id "switch.target.two-state"
def kernelId := id "switch.kernel.two-state"
def switchCapabilityId := id "switch.capability.state"
def switchProviderId := id "switch.provider.state"
def flipLawId := id "switch.law.flip-preserves-domain"
def powerStateId := id "switch.state.power"
def flipActionId := id "switch.action.flip"
def appliedOutcomeId := id "switch.outcome.applied"
def deferredOutcomeId := id "switch.outcome.deferred"
def powerObservationId := id "switch.observation.power"
def switchRoleId := id "switch.role.subject"
def flipPropertyId := id "switch.property.flip-turns-on"
def exploratoryBehaviorId := id "switch.behavior.exploratory"
def exactActionBehaviorId := id "switch.behavior.exact-action"
def exactTraceBehaviorId := id "switch.behavior.exact-trace"
def exploratoryQueryId := id "switch.query.explore"
def exactActionQueryId := id "switch.query.exact-action"
def exactTraceQueryId := id "switch.query.exact-trace"

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
  native_decide

theorem onState_ne_offState : onState ≠ offState := by
  native_decide

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
  native_decide

theorem appliedFromOnResult_ordered :
    transitionResultOrderKey appliedFromOnResult ≤
      transitionResultOrderKey deferredFromOnResult := by
  native_decide

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

def targetResult := composeTarget targetDeclaration

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

def exploratoryQueryResult := checkQuery queryContext
  (queryDeclaration exploratoryQueryId (.select [flipProperty]) exploratoryBehavior)

def exactActionQueryResult := checkQuery queryContext
  (queryDeclaration exactActionQueryId (.witness flipProperty) exactActionBehavior)

def exactTraceQueryResult := checkQuery queryContext
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

def exploratoryQuery := materializeQuery
  (exploratoryQueryResult.toOption.get exploratoryQueryResult_isSome)

def exactActionQuery := materializeQuery
  (exactActionQueryResult.toOption.get exactActionQueryResult_isSome)

def exactTraceQuery := materializeQuery
  (exactTraceQueryResult.toOption.get exactTraceQueryResult_isSome)

def incrementalStepLimit (state action : SemanticValue) : Nat :=
  (stepResults state action).length

def incrementalStepAt
    (state action : SemanticValue)
    (index : Nat) : Option (TransitionResult SemanticValue SemanticValue SemanticValue) :=
  (stepResults state action)[index]?

theorem stepResults_length_le_two (state action : SemanticValue) :
    (stepResults state action).length ≤ 2 := by
  by_cases action = flipAction <;>
    by_cases state = offState <;>
    by_cases state = onState <;>
    simp_all [stepResults]

def incrementalKernel : IncrementalPlannerKernel target := {
  actionLimit := 1
  actionAt := fun index => if index = 0 then some flipAction else none
  initialLimit := fun setup => if setup = switchSetup then 1 else 0
  initialAt := fun setup index =>
    if setup = switchSetup ∧ index = 0 then some offState else none
  stepLimit := incrementalStepLimit
  stepAt := incrementalStepAt
  actionSound := by
    intro index action inBounds emitted
    simp only [Nat.lt_one_iff] at inBounds
    subst index
    simp at emitted
    subst action
    exact ⟨offState, appliedResult, target_off_flip_applied_authoritative⟩
  actionComplete := by
    intro state action result admitted
    have selected := (target_step state action result admitted).1
    exact ⟨0, by simp, by simp [selected]⟩
  initialSound := by
    intro setup index state _ emitted
    by_cases selected : setup = switchSetup ∧ index = 0
    · simp [selected] at emitted
      subst state
      change authoritativeInitial setup offState
      exact ⟨selected.1, rfl⟩
    · simp [selected] at emitted
  initialComplete := by
    intro setup state admitted
    have selected := target_initial setup state admitted
    exact ⟨0, by simp [selected.1], by simp [selected.1, selected.2]⟩
  stepSound := by
    intro state action index result _ emitted
    rcases List.getElem?_eq_some_iff.mp emitted with ⟨inBounds, selected⟩
    apply stepResults_sound
    rw [List.mem_iff_getElem]
    exact ⟨index, inBounds, selected⟩
  stepComplete := by
    intro state action result admitted
    have member := stepResults_complete state action result admitted
    rw [List.mem_iff_getElem] at member
    rcases member with ⟨index, inBounds, selected⟩
    exact ⟨index, by simpa [incrementalStepLimit],
      List.getElem?_eq_some_iff.mpr ⟨inBounds, selected⟩⟩
  actionOrdered := by intros; simp_all [semanticValueOrderKey]
  initialOrdered := by intros; simp_all [semanticValueOrderKey]
  stepOrdered := by
    intro state action first second left right earlier emittedLeft emittedRight
    rcases List.getElem?_eq_some_iff.mp emittedLeft with ⟨firstBound, selectedLeft⟩
    rcases List.getElem?_eq_some_iff.mp emittedRight with ⟨secondBound, selectedRight⟩
    have lengthBound := stepResults_length_le_two state action
    have firstZero : first = 0 := by omega
    have secondOne : second = 1 := by omega
    subst first
    subst second
    by_cases selectedAction : action = flipAction
    · subst action
      by_cases selectedOff : state = offState
      · subst state
        simp [stepResults] at selectedLeft selectedRight
        subst left
        subst right
        exact appliedResult_ordered
      · by_cases selectedOn : state = onState
        · subst state
          simp [stepResults, onState_ne_offState] at selectedLeft selectedRight
          subst left
          subst right
          exact appliedFromOnResult_ordered
        · simp [stepResults, selectedOff, selectedOn] at firstBound
    · simp [stepResults, selectedAction] at firstBound
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
