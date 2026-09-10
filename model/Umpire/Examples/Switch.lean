import Umpire.Search
import Umpire.Search.Branches
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

def LawStatement (law : Law) : Prop :=
  law.id = flipLawId ∧ law.body = "switch-flip-preserves-domain-law/v1" ∧
    Position.flip (Position.flip .off) = .off

def flipLaw : Law := {
  id := flipLawId
  body := "switch-flip-preserves-domain-law/v1"
}

theorem flipLawProof : LawStatement flipLaw := by
  exact ⟨rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (behaviorVersion : String) : DefinitionMetadata :=
  Shared.definitionMetadata definitionId kind source 1 behaviorVersion ""

def offState : ModelValue := ModelValue.named powerStateId "off"
def onState : ModelValue := ModelValue.named powerStateId "on"
def flipAction : ModelValue := ModelValue.named flipActionId "flip"
def appliedOutcome : ModelValue := ModelValue.named appliedOutcomeId "applied"
def deferredOutcome : ModelValue := ModelValue.named deferredOutcomeId "deferred"
def powerOffObservation : ModelValue := ModelValue.named powerObservationId "off"
def powerOnObservation : ModelValue := ModelValue.named powerObservationId "on"

theorem offState_ne_onState : offState ≠ onState := by
  decide

theorem onState_ne_offState : onState ≠ offState := by
  decide

def switchSetup : List RoleBinding := [{ role := switchRoleId, value := offState }]

def appliedResult : Step ModelValue ModelValue ModelValue := {
  outcome := appliedOutcome
  state := onState
  facts := [powerOnObservation]
}

def deferredResult : Step ModelValue ModelValue ModelValue := {
  outcome := deferredOutcome
  state := offState
  facts := [powerOffObservation]
}

def appliedFromOnResult : Step ModelValue ModelValue ModelValue := {
  outcome := appliedOutcome
  state := offState
  facts := [powerOffObservation]
}

def deferredFromOnResult : Step ModelValue ModelValue ModelValue := {
  outcome := deferredOutcome
  state := onState
  facts := [powerOnObservation]
}

theorem appliedResult_ordered :
    stepOrderKey appliedResult ≤ stepOrderKey deferredResult := by
  decide

theorem appliedFromOnResult_ordered :
    stepOrderKey appliedFromOnResult ≤
      stepOrderKey deferredFromOnResult := by
  decide

def initialStates (setup : List RoleBinding) : List ModelValue :=
  if setup = switchSetup then [offState] else []

def authoritativeInitial (setup : List RoleBinding) (state : ModelValue) : Prop :=
  setup = switchSetup ∧ state = offState

def stepResults
    (state action : ModelValue) :
    List (Step ModelValue ModelValue ModelValue) :=
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
    (result : Step ModelValue ModelValue ModelValue) : Prop :=
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
    (result : Step ModelValue ModelValue ModelValue)
    (member : result ∈ stepResults state action) :
    authoritativeStep state action result := by
  by_cases selectedAction : action = flipAction
  · subst action
    by_cases selectedOff : state = offState
    · subst state
      simp [stepResults, authoritativeStep, offState, onState, ModelValue.named] at member ⊢
      exact member
    · by_cases selectedOn : state = onState
      · subst state
        simp [stepResults, authoritativeStep, offState, onState, ModelValue.named] at member ⊢
        exact member
      · simp [stepResults, selectedOff, selectedOn] at member
  · simp [stepResults, selectedAction] at member

theorem stepResults_complete
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (admitted : authoritativeStep state action result) :
    result ∈ stepResults state action := by
  rcases admitted with ⟨rfl, admitted⟩
  rcases admitted with ⟨rfl, admitted⟩ | ⟨rfl, admitted⟩
  · rcases admitted with rfl | rfl <;> simp [stepResults, offState, ModelValue.named]
  · rcases admitted with rfl | rfl <;> simp [stepResults, offState, onState, ModelValue.named]

def machine : Machine
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
  vocabulary := .complete {
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

def switchProvider : Provider LawStatement := {
  id := switchProviderId
  source
  contract := {
    id := switchCapabilityId
    behaviorVersion := "switch-state/v1"
    requiredLaws := [flipLaw]
  }
  meanings := [
    { definitionId := powerStateId, kind := .state, behaviorVersion := "switch-power-state/v1" },
    { definitionId := flipActionId, kind := .action, behaviorVersion := "switch-flip-action/v1" },
    { definitionId := appliedOutcomeId, kind := .outcome,
      behaviorVersion := "switch-applied-outcome/v1" },
    { definitionId := deferredOutcomeId, kind := .outcome,
      behaviorVersion := "switch-deferred-outcome/v1" },
    { definitionId := powerObservationId, kind := .fact,
      behaviorVersion := "switch-power-observation/v1" }
  ]
  lawProofs := [{ definition := flipLaw, proof := flipLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "switch-two-state-target/v1",
  metadata kernelId .machine "switch-two-state-kernel/v1",
  metadata switchCapabilityId .capability "switch-state/v1",
  metadata switchProviderId .provider "switch-state-provider/v1",
  metadata flipLawId .law flipLaw.body,
  metadata powerStateId .state "switch-power-state/v1",
  metadata flipActionId .action "switch-flip-action/v1",
  metadata appliedOutcomeId .outcome "switch-applied-outcome/v1",
  metadata deferredOutcomeId .outcome "switch-deferred-outcome/v1",
  metadata powerObservationId .fact "switch-power-observation/v1"
]

def finitePlanning : FinitePlanningCapability machine.authoritativeStep := {
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

def modelSpec : ModelSpec LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue := {
  id := targetId
  source
  definitions
  requiredCapabilities := [switchCapabilityId]
  resolvedSetups := [switchSetup]
  machine := .checked machine
}

def modelProviders : Providers LawStatement :=
  Providers.empty |>.provide switchProvider

def targetAuthoring : DraftModel LawStatement
    (List RoleBinding) ModelValue ModelValue ModelValue ModelValue :=
  DraftModel.make modelSpec modelProviders
    (.available machine rfl finitePlanning)

/-- Re-ascribe the source kernel after checked composition so its proof relation remains reducible. -/
def target : QueryModel LawStatement := model targetAuthoring

theorem target_resolvedSetups : target.resolvedSetups = [switchSetup] := by
  native_decide

theorem target_initial
    (setup : List RoleBinding)
    (state : ModelValue)
    (admitted : target.machine.authoritativeInitial setup state) :
    setup = switchSetup ∧ state = offState := by
  exact admitted

theorem target_step
    (state action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue)
    (admitted : target.machine.authoritativeStep state action result) :
    authoritativeStep state action result := by
  exact admitted

theorem target_off_flip_applied_authoritative :
    target.machine.authoritativeStep offState flipAction appliedResult := by
  change authoritativeStep offState flipAction appliedResult
  exact ⟨rfl, .inl ⟨rfl, .inl rfl⟩⟩

def authoredProperty : Property := {
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
  Property.check (PropertyCheckContext.ofTarget target) (authoredProperty)

private theorem propertyResult_isSome : propertyResult.toOption.isSome = true := by
  native_decide

def flipProperty : CheckedProperty :=
  Property.checked (PropertyCheckContext.ofTarget target) (authoredProperty)
    propertyResult_isSome

def switchRole : Scenario.Role := { id := switchRoleId, valueKind := .state }

def setupConstraint : SetupConstraint :=
  SetupConstraint.roleEquals (id "switch.setup.subject-is-off") switchRoleId offState

def exploratoryBehaviorDeclaration : Scenario := {
  id := exploratoryBehaviorId
  source
  requires := [switchCapabilityId]
  roles := [switchRole]
  setup := [setupConstraint]
  allowedActions := [flipActionId]
  requiredOccurrences := [{ id := id "switch.occurrence.flip", action := flipActionId }]
  occurrenceBounds := [Scenario.Count.exactly flipActionId 1]
  documentation := "Explore the finite switch outcomes for one selected flip."
}

def exactActionBehaviorDeclaration : Scenario :=
  Scenario.exactlyOneAction exactActionBehaviorId source
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
    outcome := some appliedOutcome
    resultingState := some onState
    observations := some appliedResult.facts
  }]
}

def exactTraceBehaviorDeclaration : Scenario := {
  exactActionBehaviorDeclaration with
  id := exactTraceBehaviorId
  traceExactly := some exactTrace
  documentation := "Select the complete applied flip trace."
}

private def checkBehaviorDeclaration
    (declaration : Scenario) : Except ScenarioError CheckedScenario :=
  Scenario.check (.ofTarget target) declaration

def exploratoryBehaviorResult : Except ScenarioError CheckedScenario :=
  checkBehaviorDeclaration exploratoryBehaviorDeclaration
def exactActionBehaviorResult : Except ScenarioError CheckedScenario :=
  checkBehaviorDeclaration exactActionBehaviorDeclaration
def exactTraceBehaviorResult : Except ScenarioError CheckedScenario :=
  checkBehaviorDeclaration exactTraceBehaviorDeclaration

private theorem exploratoryBehaviorResult_isSome :
    exploratoryBehaviorResult.toOption.isSome = true := by native_decide

private theorem exactActionBehaviorResult_isSome :
    exactActionBehaviorResult.toOption.isSome = true := by native_decide

private theorem exactTraceBehaviorResult_isSome :
    exactTraceBehaviorResult.toOption.isSome = true := by native_decide

def exploratoryBehavior : CheckedScenario :=
  Scenario.checked (.ofTarget target) exploratoryBehaviorDeclaration
    exploratoryBehaviorResult_isSome

def exactActionBehavior : CheckedScenario :=
  Scenario.checked (.ofTarget target) exactActionBehaviorDeclaration
    exactActionBehaviorResult_isSome

def exactTraceBehavior : CheckedScenario :=
  Scenario.checked (.ofTarget target) exactTraceBehaviorDeclaration
    exactTraceBehaviorResult_isSome

def appliedTrace : Scenario.Trace :=
  Scenario.Trace.singleStep switchSetup offState flipAction appliedResult

def deferredTrace : Scenario.Trace :=
  Scenario.Trace.singleStep switchSetup offState flipAction deferredResult

def limits : Limits := Limits.bounded 1 1 8

def shortestPolicy : PlannerPolicy := PlannerPolicy.shortest

def queryContext : QueryCheckContext LawStatement := .ofTarget target

private def authoredQuery
    (queryId : DefinitionId)
    (form : Query.Form)
    (behavior : CheckedScenario) : Query := {
  id := queryId
  source
  target := target.id
  form
  behavior
  limits
  policy := shortestPolicy
}

def exploratoryQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  Query.check queryContext
    (authoredQuery exploratoryQueryId (.pick [flipProperty]) exploratoryBehavior)

def exactActionQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  Query.check queryContext
    (authoredQuery exactActionQueryId (.find flipProperty) exactActionBehavior)

def exactTraceQueryResult : Except QueryError (CheckedQuery LawStatement) :=
  Query.check queryContext
    (authoredQuery exactTraceQueryId (.find flipProperty) exactTraceBehavior)

private theorem exploratoryQueryResult_isSome :
    exploratoryQueryResult.toOption.isSome = true := by native_decide

private theorem exactActionQueryResult_isSome :
    exactActionQueryResult.toOption.isSome = true := by native_decide

private theorem exactTraceQueryResult_isSome :
    exactTraceQueryResult.toOption.isSome = true := by native_decide

def exploratoryQuery : CheckedQuery LawStatement :=
  Query.checked target
    (authoredQuery exploratoryQueryId (.pick [flipProperty]) exploratoryBehavior)
    exploratoryQueryResult_isSome

def exactActionQuery : CheckedQuery LawStatement :=
  Query.checked target
    (authoredQuery exactActionQueryId (.find flipProperty) exactActionBehavior)
    exactActionQueryResult_isSome

def exactTraceQuery : CheckedQuery LawStatement :=
  Query.checked target
    (authoredQuery exactTraceQueryId (.find flipProperty) exactTraceBehavior)
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

private def incrementalKernel? : Option (SearchView exactActionQuery.target) :=
  SearchView.ofCheckedQuery? exactActionQuery
    (by
      intro evidence evidenceEq
      simp [exactActionQuery, Query.checked, ModelCompleteness.ofTarget, target,
        model, targetAuthoring, DraftModel.make, modelSpec] at evidenceEq
      cases Option.some.inj evidenceEq
      simp [finitePlanning])
    (by
      intro _ _ setup
      simp only [exactActionQuery, Query.checked, target, model, targetAuthoring,
        DraftModel.make, modelSpec,
        machine, initialStates]
      split <;> simp)
    (by
      intro _ _ state action
      by_cases selectedAction : action = flipAction
      · subst action
        by_cases selectedOff : state = offState
        · subst state
          simpa [exactActionQuery, Query.checked, target, model, targetAuthoring,
            DraftModel.make, modelSpec,
            machine, stepResults] using appliedResult_ordered
        · by_cases selectedOn : state = onState
          · subst state
            simpa [exactActionQuery, Query.checked, target, model, targetAuthoring,
              DraftModel.make, modelSpec,
              machine, stepResults, onState_ne_offState] using
              appliedFromOnResult_ordered
          · simp [exactActionQuery, Query.checked, target, model, targetAuthoring,
              DraftModel.make, modelSpec,
              machine, stepResults, selectedOff, selectedOn]
      · simp [exactActionQuery, Query.checked, target, model, targetAuthoring,
          DraftModel.make, modelSpec,
          machine, stepResults, selectedAction])

private theorem incrementalKernel?_isSome : incrementalKernel?.isSome = true := by
  rfl

def incrementalKernel : SearchView target :=
  incrementalKernel?.get incrementalKernel?_isSome

theorem exploratoryQuery_target : exploratoryQuery.target = target := by rfl
theorem exactActionQuery_target : exactActionQuery.target = target := by rfl
theorem exactTraceQuery_target : exactTraceQuery.target = target := by rfl

def exploratoryRun : Except KnownGapError PlanResult :=
  search exploratoryQuery incrementalKernel

def exactActionRunResult : Except KnownGapError PlanResult :=
  search exactActionQuery incrementalKernel

def exactTraceRun : Except KnownGapError PlanResult :=
  search exactTraceQuery incrementalKernel

def artifact : Option Plan := exactActionRunResult.toOption.bind PlanResult.artifact

private theorem artifact_isSome : artifact.isSome = true := by
  native_decide

private theorem exactActionRunResult_isSome : exactActionRunResult.toOption.isSome = true := by
  cases selected : exactActionRunResult with
  | error error =>
      have artifactExists := artifact_isSome
      simp [artifact, selected] at artifactExists
      contradiction
  | ok run => rfl

def exactActionRun : PlanResult :=
  exactActionRunResult.toOption.get exactActionRunResult_isSome

def compiledArtifact : Plan := artifact.get artifact_isSome

end Umpire.Examples.Switch
