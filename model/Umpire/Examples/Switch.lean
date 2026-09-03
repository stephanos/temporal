import Umpire.Planning
import Umpire.Shared

namespace Umpire.Examples.Switch

private def id (value : String) : DefinitionId := Shared.definitionId value

def source : SourceLocation :=
  Shared.sourceLocation "Umpire/Examples/Switch.lean" 1 1 "lean-model"

def targetId : DefinitionId := id "switch.target.two-state"
def kernelId : DefinitionId := id "switch.kernel.two-state"
def switchCapabilityId : DefinitionId := id "switch.capability.state"
def switchProviderId : DefinitionId := id "switch.provider.state"
def flipLawId : DefinitionId := id "switch.law.flip-preserves-domain"
def powerStateId : DefinitionId := id "switch.state.power"
def flipActionId : DefinitionId := id "switch.action.flip"
def appliedOutcomeId : DefinitionId := id "switch.outcome.applied"
def deferredOutcomeId : DefinitionId := id "switch.outcome.deferred"
def powerObservationId : DefinitionId := id "switch.observation.power"
def switchRoleId : DefinitionId := id "switch.role.subject"
def flipPropertyId : DefinitionId := id "switch.property.flip-turns-on"
def exploratoryBehaviorId : DefinitionId := id "switch.behavior.exploratory"
def exactActionBehaviorId : DefinitionId := id "switch.behavior.exact-action"
def exactTraceBehaviorId : DefinitionId := id "switch.behavior.exact-trace"
def exploratoryQueryId : DefinitionId := id "switch.query.explore"
def exactActionQueryId : DefinitionId := id "switch.query.exact-action"
def exactTraceQueryId : DefinitionId := id "switch.query.exact-trace"

inductive Position where
  | off
  | on
  deriving BEq, DecidableEq, Repr

def Position.flip : Position → Position
  | .off => .on
  | .on => .off

def LawStatement (law : LawDefinition) : Prop :=
  law.id = flipLawId ∧ law.body = "switch-flip-preserves-domain-law/v1" ∧
    Position.flip (Position.flip .off) = .off

def flipLaw : LawDefinition := {
  id := flipLawId
  body := "switch-flip-preserves-domain-law/v1"
}

theorem flipLawProof : LawStatement flipLaw := by
  exact ⟨rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (canonicalBehavior : String) : DefinitionMetadata :=
  Shared.definitionMetadata definitionId kind source 1 canonicalBehavior ""

def offState : ModelValue := { definitionId := powerStateId, value := "off" }
def onState : ModelValue := { definitionId := powerStateId, value := "on" }
def flipAction : ModelValue := { definitionId := flipActionId, value := "flip" }
def appliedOutcome : ModelValue := { definitionId := appliedOutcomeId, value := "applied" }
def deferredOutcome : ModelValue := { definitionId := deferredOutcomeId, value := "deferred" }
def powerOffObservation : ModelValue := { definitionId := powerObservationId, value := "off" }
def powerOnObservation : ModelValue := { definitionId := powerObservationId, value := "on" }

theorem offState_ne_onState : offState ≠ onState := by
  decide

theorem onState_ne_offState : onState ≠ offState := by
  decide

def switchSetup : List RoleBinding := [{ role := switchRoleId, value := offState }]

def appliedResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := appliedOutcome
  resultingState := onState
  observations := [powerOnObservation]
}

def deferredResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := deferredOutcome
  resultingState := offState
  observations := [powerOffObservation]
}

def appliedFromOnResult : TransitionResult ModelValue ModelValue ModelValue := {
  modelOutcome := appliedOutcome
  resultingState := offState
  observations := [powerOffObservation]
}

def deferredFromOnResult : TransitionResult ModelValue ModelValue ModelValue := {
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

def initialStates (setup : List RoleBinding) : List ModelValue :=
  if setup = switchSetup then [offState] else []

def authoritativeInitial (setup : List RoleBinding) (state : ModelValue) : Prop :=
  setup = switchSetup ∧ state = offState

def stepResults
    (state action : ModelValue) :
    List (TransitionResult ModelValue ModelValue ModelValue) :=
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
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue) : Prop :=
  action = flipAction ∧
    ((state = offState ∧ (result = appliedResult ∨ result = deferredResult)) ∨
      (state = onState ∧
        (result = appliedFromOnResult ∨ result = deferredFromOnResult)))

theorem initialStates_sound
    (setup : List RoleBinding)
    (state : ModelValue)
    (member : state ∈ initialStates setup) :
    authoritativeInitial setup state := by
  by_cases selected : setup = switchSetup
  · subst setup
    simp [initialStates, authoritativeInitial] at member ⊢
    exact member
  · simp [initialStates, selected] at member

theorem initialStates_complete
    (setup : List RoleBinding)
    (state : ModelValue)
    (admitted : authoritativeInitial setup state) :
    state ∈ initialStates setup := by
  rcases admitted with ⟨rfl, rfl⟩
  simp [initialStates]

theorem stepResults_sound
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
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
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action := by
  rcases admitted with ⟨rfl, admitted⟩
  rcases admitted with ⟨rfl, admitted⟩ | ⟨rfl, admitted⟩
  · rcases admitted with rfl | rfl <;> simp [stepResults, offState]
  · rcases admitted with rfl | rfl <;> simp [stepResults, offState, onState]

def transitionKernel : TransitionKernel
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  metadata := {
    id := kernelId
    source
  }
  setupDomain := fun candidate => candidate = switchSetup
  stateDomain := fun candidate => candidate = offState ∨ candidate = onState
  actionDomain := fun candidate => candidate = flipAction
  outcomeDomain := fun candidate => candidate = appliedOutcome ∨ candidate = deferredOutcome
  observationDomain := fun candidate =>
    candidate = powerOffObservation ∨ candidate = powerOnObservation
  initialStates
  authoritativeInitial
  initialSound := initialStates_sound
  initialComplete := initialStates_complete
  steps := stepResults
  authoritativeStep
  stepSound := stepResults_sound
  stepComplete := stepResults_complete
  behaviorDomain := .complete {
    setups := [switchSetup]
    states := [offState, onState]
    actions := [flipAction]
    outcomes := [appliedOutcome, deferredOutcome]
    observations := [powerOffObservation, powerOnObservation]
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
      by_cases selected : setup = switchSetup
      · simp [selected]
      · simp [initialStates, selected] at member
    initialStateCoverage := by
      intro setup state member
      by_cases selected : setup = switchSetup
      · rw [initialStates, if_pos selected] at member
        simp [List.mem_singleton.mp member]
      · simp [initialStates, selected] at member
    transitionSourceCoverage := by
      intro state action result member
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · simp [selectedOff]
        · by_cases selectedOn : state = onState
          · simp [selectedOn]
          · simp [stepResults, selectedOff, selectedOn] at member
      · simp [stepResults, selectedAction] at member
    actionCoverage := by
      intro state action result member
      by_cases selectedAction : action = flipAction
      · simp [selectedAction]
      · simp [stepResults, selectedAction] at member
    resultingStateCoverage := by
      intro state action result member
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          change result ∈ [appliedResult, deferredResult] at member
          rcases List.mem_cons.mp member with resultEq | tail
          · subst result
            simp [appliedResult]
          · have resultEq := List.mem_singleton.mp tail
            subst result
            simp [deferredResult]
        · by_cases selectedOn : state = onState
          · subst state
            change result ∈ [appliedFromOnResult, deferredFromOnResult] at member
            rcases List.mem_cons.mp member with resultEq | tail
            · subst result
              simp [appliedFromOnResult]
            · have resultEq := List.mem_singleton.mp tail
              subst result
              simp [deferredFromOnResult]
          · simp [stepResults, selectedOff, selectedOn] at member
      · simp [stepResults, selectedAction] at member
    outcomeCoverage := by
      intro state action result member
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          change result ∈ [appliedResult, deferredResult] at member
          rcases List.mem_cons.mp member with resultEq | tail
          · subst result
            simp [appliedResult]
          · have resultEq := List.mem_singleton.mp tail
            subst result
            simp [deferredResult]
        · by_cases selectedOn : state = onState
          · subst state
            change result ∈ [appliedFromOnResult, deferredFromOnResult] at member
            rcases List.mem_cons.mp member with resultEq | tail
            · subst result
              simp [appliedFromOnResult]
            · have resultEq := List.mem_singleton.mp tail
              subst result
              simp [deferredFromOnResult]
          · simp [stepResults, selectedOff, selectedOn] at member
      · simp [stepResults, selectedAction] at member
    observationCoverage := by
      intro state action result observation member observationMember
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          change result ∈ [appliedResult, deferredResult] at member
          rcases List.mem_cons.mp member with resultEq | tail
          · subst result
            exact List.mem_cons.mpr (.inr <| List.mem_singleton.mpr <|
              by simpa [appliedResult] using observationMember)
          · have resultEq := List.mem_singleton.mp tail
            subst result
            exact List.mem_cons.mpr (.inl <| by simpa [deferredResult] using observationMember)
        · by_cases selectedOn : state = onState
          · subst state
            change result ∈ [appliedFromOnResult, deferredFromOnResult] at member
            rcases List.mem_cons.mp member with resultEq | tail
            · subst result
              exact List.mem_cons.mpr (.inl <|
                by simpa [appliedFromOnResult] using observationMember)
            · have resultEq := List.mem_singleton.mp tail
              subst result
              exact List.mem_cons.mpr (.inr <| List.mem_singleton.mpr <|
                by simpa [deferredFromOnResult] using observationMember)
          · simp [stepResults, selectedOff, selectedOn] at member
      · simp [stepResults, selectedAction] at member
  }
}

def switchProvider : CapabilityProvider LawStatement := {
  id := switchProviderId
  source
  contract := {
    id := switchCapabilityId
    canonicalBehavior := "switch-state/v1"
    requiredLaws := [flipLaw]
  }
  meanings := [
    { definitionId := powerStateId, kind := .state, canonicalBehavior := "switch-power-state/v1" },
    { definitionId := flipActionId, kind := .action, canonicalBehavior := "switch-flip-action/v1" },
    { definitionId := appliedOutcomeId, kind := .outcome,
      canonicalBehavior := "switch-applied-outcome/v1" },
    { definitionId := deferredOutcomeId, kind := .outcome,
      canonicalBehavior := "switch-deferred-outcome/v1" },
    { definitionId := powerObservationId, kind := .observation,
      canonicalBehavior := "switch-power-observation/v1" }
  ]
  lawWitnesses := [{ definition := flipLaw, proof := flipLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "switch-two-state-target/v1",
  metadata kernelId .kernel "switch-two-state-kernel/v1",
  metadata switchCapabilityId .capability "switch-state/v1",
  metadata switchProviderId .provider "switch-state-provider/v1",
  metadata flipLawId .law flipLaw.body,
  metadata powerStateId .state "switch-power-state/v1",
  metadata flipActionId .action "switch-flip-action/v1",
  metadata appliedOutcomeId .outcome "switch-applied-outcome/v1",
  metadata deferredOutcomeId .outcome "switch-deferred-outcome/v1",
  metadata powerObservationId .observation "switch-power-observation/v1"
]

def finitePlanning : FinitePlanningCapability transitionKernel.authoritativeStep := {
  actions := [flipAction]
  actionSound := by
    intro action member
    simp only [List.mem_cons, List.not_mem_nil, or_false] at member
    subst action
    exact ⟨offState, appliedResult, ⟨rfl, .inl ⟨rfl, .inl rfl⟩⟩⟩
  actionComplete := by
    intro state action result admitted
    simp [admitted.1]
}

def targetDefinition : TargetDefinition
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [switchCapabilityId]
  resolvedSetups := [switchSetup]
  kernel := .checked transitionKernel
}

def targetComposition : TargetComposition LawStatement :=
  TargetComposition.empty |>.provide switchProvider

def targetAuthoring : AuthoredTarget LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  AuthoredTarget.make targetDefinition targetComposition
    (.available transitionKernel rfl finitePlanning)

/-- Re-ascribe the source kernel after checked composition so its proof relation remains reducible. -/
def target : QueryTarget LawStatement := checkedTarget targetAuthoring

theorem target_resolvedSetups : target.resolvedSetups = [switchSetup] := by
  native_decide

theorem target_initial
    (setup : List RoleBinding)
    (state : ModelValue)
    (admitted : target.kernel.authoritativeInitial setup state) :
    setup = switchSetup ∧ state = offState := by
  exact admitted

theorem target_step
    (state action : ModelValue)
    (result : TransitionResult ModelValue ModelValue ModelValue)
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
      (PropertyPattern.exact .selectedAction flipActionId flipAction.value)
      (PropertyPattern.exact .resultingState powerStateId onState.value)
  ]
  documentation := "A selected flip has an outcome that turns the switch on."
}

def propertyResult : Except PropertyError CheckedProperty :=
  checkProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def flipProperty : CheckedProperty :=
  checkedProperty (PropertyCheckContext.ofTarget target) (.portable propertyDeclaration)
    propertyResult_isSome

def switchRole : ResourceRole := { id := switchRoleId, valueKind := .state }

def setupConstraint : SetupConstraint :=
  SetupConstraint.roleEquals (id "switch.setup.subject-is-off") switchRoleId offState

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

def exactActionBehaviorDeclaration : BehaviorDeclaration :=
  BehaviorDeclaration.exactlyOneAction exactActionBehaviorId source
    { id := id "switch.occurrence.flip", action := flipActionId }
    (requires := [switchCapabilityId])
    (roles := [switchRole])
    (setup := [setupConstraint])
    (documentation := "Select one flip while leaving its outcome to the switch model.")

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
  checkedBehavior (.ofTarget target) exploratoryBehaviorDeclaration
    exploratoryBehaviorResult_isSome

def exactActionBehavior : CheckedBehavior :=
  checkedBehavior (.ofTarget target) exactActionBehaviorDeclaration
    exactActionBehaviorResult_isSome

def exactTraceBehavior : CheckedBehavior :=
  checkedBehavior (.ofTarget target) exactTraceBehaviorDeclaration
    exactTraceBehaviorResult_isSome

def appliedTrace : BehaviorTrace :=
  BehaviorTrace.singleStep switchSetup offState flipAction appliedResult

def deferredTrace : BehaviorTrace :=
  BehaviorTrace.singleStep switchSetup offState flipAction deferredResult

def limits : QueryLimits := {
  behavior := {
    transitions := { value := 1, unit := .semanticTransitions }
    selectedActions := { value := 1, unit := .selectedActions }
  }
  search := { value := 8, unit := .candidateEvaluations }
}

def shortestPolicy : PlannerPolicy := PlannerPolicy.shortest

def queryContext : QueryCheckContext LawStatement := .ofTarget target

private def queryDeclaration
    (queryId : DefinitionId)
    (form : QueryForm)
    (behavior : CheckedBehavior) : QueryDeclaration := {
  id := queryId
  source
  target := target.id
  form
  behavior
  limits
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

def exploratoryQuery : CheckedQuery LawStatement :=
  checkedQuery target
    (queryDeclaration exploratoryQueryId (.select [flipProperty]) exploratoryBehavior)
    exploratoryQueryResult_isSome

def exactActionQuery : CheckedQuery LawStatement :=
  checkedQuery target
    (queryDeclaration exactActionQueryId (.witness flipProperty) exactActionBehavior)
    exactActionQueryResult_isSome

def exactTraceQuery : CheckedQuery LawStatement :=
  checkedQuery target
    (queryDeclaration exactTraceQueryId (.witness flipProperty) exactTraceBehavior)
    exactTraceQueryResult_isSome

theorem stepResults_length_le_two (state action : ModelValue) :
    (stepResults state action).length ≤ 2 := by
  by_cases selectedAction : action = flipAction
  · subst action
    by_cases selectedOff : state = offState
    · subst state
      simp [stepResults]
    · by_cases selectedOn : state = onState
      · subst state
        simp [stepResults, selectedOff]
      · simp [stepResults, selectedOff, selectedOn]
  · simp [stepResults, selectedAction]

private def incrementalKernel? : Option (IncrementalPlannerKernel exactActionQuery.target) :=
  IncrementalPlannerKernel.ofCheckedQuery? exactActionQuery
    (by
      intro evidence evidenceEq
      simp [exactActionQuery, checkedQuery, CheckedQueryTarget.ofTarget, target,
        checkedTarget, targetAuthoring, AuthoredTarget.make, targetDefinition] at evidenceEq
      cases Option.some.inj evidenceEq
      simp [finitePlanning])
    (by
      intro _ _ setup
      simp only [exactActionQuery, checkedQuery, target, checkedTarget, targetAuthoring,
        AuthoredTarget.make, targetDefinition,
        transitionKernel, initialStates]
      split <;> simp)
    (by
      intro _ _ state action
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          simpa [exactActionQuery, checkedQuery, target, checkedTarget, targetAuthoring,
            AuthoredTarget.make, targetDefinition,
            transitionKernel, stepResults] using appliedResult_ordered
        · by_cases selectedOn : state = onState
          · subst state
            simpa [exactActionQuery, checkedQuery, target, checkedTarget, targetAuthoring,
              AuthoredTarget.make, targetDefinition,
              transitionKernel, stepResults, onState_ne_offState] using
              appliedFromOnResult_ordered
          · simp [exactActionQuery, checkedQuery, target, checkedTarget, targetAuthoring,
              AuthoredTarget.make, targetDefinition,
              transitionKernel, stepResults, selectedOff, selectedOn]
      · simp [exactActionQuery, checkedQuery, target, checkedTarget, targetAuthoring,
          AuthoredTarget.make, targetDefinition,
          transitionKernel, stepResults, selectedAction])

private theorem incrementalKernel?_isSome : incrementalKernel?.isSome = true := by
  rfl

def incrementalKernel : IncrementalPlannerKernel target :=
  incrementalKernel?.get incrementalKernel?_isSome

theorem exploratoryQuery_target : exploratoryQuery.target = target := by rfl
theorem exactActionQuery_target : exactActionQuery.target = target := by rfl
theorem exactTraceQuery_target : exactTraceQuery.target = target := by rfl

def exploratoryRun : PlannerRun :=
  plan exploratoryQuery incrementalKernel

def exactActionRun : PlannerRun :=
  plan exactActionQuery incrementalKernel

def exactTraceRun : PlannerRun :=
  plan exactTraceQuery incrementalKernel

def artifact : Option ExperimentSpec := exactActionRun.artifact

private theorem artifact_isSome : artifact.isSome = true := by
  native_decide

def compiledArtifact : ExperimentSpec := artifact.get artifact_isSome

end Umpire.Examples.Switch
