import Temporal.Feature.Nexus.Operations
import Temporal.Feature.Nexus2.Cancellation
import Temporal.Feature.Nexus2.Race
import Temporal.Feature.Nexus2.Authoring
import Temporal.Feature.Nexus2.AuthoringTests
import Temporal.Feature.Nexus2.AuthoringEditProbe
import Umpire.CoreImportTests

/-! Dedicated admission, behavior-equivalence, planning, and negative tests for the Nexus2 baseline. -/

namespace Temporal.Feature.Nexus2.Tests

open Umpire
open Temporal.Feature.Nexus2

namespace Baseline

open Lifecycle
open Cancellation

/-! `native_decide` below executes closed test fixtures only. None of these test theorems is
imported by the model declarations or used to construct a checked Target, Property, Behavior, or
Query; production admission remains on the successful `Except` branch and its kernel proofs. -/

private abbrev TransitionShape := State × Action × State × Outcome × List Fact
private abbrev InitialShape := Setup × State

private def stateOf (model : ModelVocabulary) (value : ModelValue) : Option State :=
  if value = model.scheduledState then some .scheduled
  else if value = model.startedState then some .started
  else if value = model.canceledState then some .canceled
  else if value = model.succeededState then some .succeeded
  else none

private def actionOf (model : ModelVocabulary) (value : ModelValue) : Option Action :=
  if value = model.cancelAction then some .cancel
  else if value = model.startAction then some .start
  else if value = model.reportSuccessAction then some .reportSuccess
  else none

private def outcomeOf (model : ModelVocabulary) (value : ModelValue) : Option Outcome :=
  if value = model.startedOutcome then some .started
  else if value = model.canceledOutcome then some .canceled
  else if value = model.succeededOutcome then some .succeeded
  else none

private def factOf (model : ModelVocabulary) (value : ModelValue) : Option Fact :=
  if value = model.startedFact then some .started
  else if value = model.canceledFact then some .canceled
  else if value = model.succeededFact then some .succeeded
  else none

private def setupOf (model : ModelVocabulary) (value : List RoleBinding) : Option Setup :=
  if value = model.scheduledSetup then some .scheduled
  else if value = model.startedSetup then some .started
  else none

private def normalizeResult
    (model : ModelVocabulary)
    (source : ModelValue)
    (action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : Option TransitionShape := do
  let source ← stateOf model source
  let action ← actionOf model action
  let resultingState ← stateOf model result.state
  let outcome ← outcomeOf model result.outcome
  let facts ← result.facts.mapM (factOf model)
  pure (source, action, resultingState, outcome, facts)

private def normalizeTarget
    (target : QueryModel LawStatement)
    (model : ModelVocabulary) : Option (List TransitionShape) := do
  let domain ← match target.machine.vocabulary with
    | .complete domain => some domain
    | _ => none
  let setups ← domain.setups.mapM (setupOf model)
  let states ← domain.states.mapM (stateOf model)
  let actions ← domain.actions.mapM (actionOf model)
  let outcomes ← domain.outcomes.mapM (outcomeOf model)
  let facts ← domain.observations.mapM (factOf model)
  if setups != table.setups.values || states != table.states.values ||
      actions != table.actions.values || outcomes != table.outcomes.values ||
      facts != table.facts.values then
    none
  else
    let rows := domain.states.flatMap fun state => domain.actions.flatMap fun action =>
      target.machine.steps state action |>.map fun result => (state, action, result)
    rows.mapM fun row => normalizeResult model row.1 row.2.1 row.2.2

private def admittedTransitions : Option (List TransitionShape) := do
  let baseline ← checkBaseline.toOption
  normalizeTarget baseline.target baseline.model

private def normalizeInitialStates
    (target : QueryModel LawStatement)
    (model : ModelVocabulary) : Option (List InitialShape) := do
  let domain ← match target.machine.vocabulary with
    | .complete domain => some domain
    | _ => none
  let setups ← domain.setups.mapM (setupOf model)
  let states ← domain.states.mapM (stateOf model)
  if setups != table.setups.values || states != table.states.values then
    none
  else
    domain.setups.flatMap (fun setup =>
      target.machine.initialStates setup |>.map fun state => (setup, state)) |>.mapM fun row => do
        pure (← setupOf model row.1, ← stateOf model row.2)

private def establishedStateOf (value : ModelValue) : Option State :=
  if value = Temporal.Feature.Nexus.Lifecycle.scheduledState then some .scheduled
  else if value = Temporal.Feature.Nexus.Lifecycle.startedState then some .started
  else if value = Temporal.Feature.Nexus.Lifecycle.canceledState then some .canceled
  else if value = Temporal.Feature.Nexus.Lifecycle.succeededState then some .succeeded
  else none

private def establishedActionOf (value : ModelValue) : Option Action :=
  if value = Temporal.Feature.Nexus.Lifecycle.cancelAction then some .cancel
  else if value = Temporal.Feature.Nexus.Lifecycle.startAction then some .start
  else if value = Temporal.Feature.Nexus.Lifecycle.reportSuccessAction then some .reportSuccess
  else none

private def establishedOutcomeOf (value : ModelValue) : Option Outcome :=
  if value = Temporal.Feature.Nexus.Lifecycle.startedOutcome then some .started
  else if value = Temporal.Feature.Nexus.Lifecycle.canceledOutcome then some .canceled
  else if value = Temporal.Feature.Nexus.Lifecycle.succeededOutcome then some .succeeded
  else none

private def establishedFactOf (value : ModelValue) : Option Fact :=
  if value = Temporal.Feature.Nexus.Lifecycle.startedObservation then some .started
  else if value = Temporal.Feature.Nexus.Lifecycle.canceledObservation then some .canceled
  else if value = Temporal.Feature.Nexus.Lifecycle.succeededObservation then some .succeeded
  else none

private def establishedSetupOf (value : List RoleBinding) : Option Setup :=
  if value = Temporal.Feature.Nexus.Lifecycle.scheduledSetup then some .scheduled
  else if value = Temporal.Feature.Nexus.Lifecycle.startedSetup then some .started
  else none

private def normalizeEstablishedResult
    (source action : ModelValue)
    (result : Step ModelValue ModelValue ModelValue) : Option TransitionShape := do
  let source ← establishedStateOf source
  let action ← establishedActionOf action
  let resultingState ← establishedStateOf result.state
  let outcome ← establishedOutcomeOf result.outcome
  let facts ← result.facts.mapM establishedFactOf
  pure (source, action, resultingState, outcome, facts)

private def establishedTransitions : Option (List TransitionShape) := do
  let target := Temporal.Feature.Nexus.Lifecycle.target
  let domain ← match target.machine.vocabulary with
    | .complete domain => some domain
    | _ => none
  let setups ← domain.setups.mapM establishedSetupOf
  let states ← domain.states.mapM establishedStateOf
  let actions ← domain.actions.mapM establishedActionOf
  let outcomes ← domain.outcomes.mapM establishedOutcomeOf
  let facts ← domain.observations.mapM establishedFactOf
  if setups != table.setups.values || states != table.states.values ||
      actions != table.actions.values || outcomes != table.outcomes.values ||
      facts != table.facts.values then
    none
  else
    let rows := domain.states.flatMap fun state => domain.actions.flatMap fun action =>
      target.machine.steps state action |>.map fun result => (state, action, result)
    rows.mapM fun row => normalizeEstablishedResult row.1 row.2.1 row.2.2

private def normalizeEstablishedInitialStates : Option (List InitialShape) := do
  let target := Temporal.Feature.Nexus.Lifecycle.target
  let domain ← match target.machine.vocabulary with
    | .complete domain => some domain
    | _ => none
  let setups ← domain.setups.mapM establishedSetupOf
  let states ← domain.states.mapM establishedStateOf
  if setups != table.setups.values || states != table.states.values then
    none
  else
    domain.setups.flatMap (fun setup =>
      target.machine.initialStates setup |>.map fun state => (setup, state)) |>.mapM fun row => do
        pure (← establishedSetupOf row.1, ← establishedStateOf row.2)

private def expectedTransitions : List TransitionShape := [
  (.scheduled, .start, .started, .started, [.started]),
  (.started, .cancel, .canceled, .canceled, [.canceled]),
  (.started, .reportSuccess, .succeeded, .succeeded, [.succeeded])
]

private def expectedInitialStates : List InitialShape := [
  (.scheduled, .scheduled),
  (.started, .started)
]

theorem exactCatalogAndRowCounts :
    (table.states.length, table.actions.length, table.setups.length,
      table.transitions.length) = (4, 3, 2, 3) := by
  native_decide

theorem explicitIdentityMappingCoversVocabularyAndSetups :
    table.states.values.map establishedState = [
        .scheduled, .started, .canceled, .succeeded] ∧
      table.actions.values.map establishedAction = [.cancel, .start, .succeed] ∧
      table.outcomes.values.map establishedOutcomeKey = ["started", "canceled", "succeeded"] ∧
      table.facts.values.map establishedFactKey = ["started", "canceled", "succeeded"] ∧
      table.setups.values.map establishedSetup = [
        Temporal.Feature.Nexus.Lifecycle.scheduledSetup,
        Temporal.Feature.Nexus.Lifecycle.startedSetup] := by
  native_decide

private def changedCancellationTable : FiniteTable Setup State Action Outcome Fact :=
  { table with transitions := table.transitions.map fun
    (row : FiniteTransitionRow State Action Outcome Fact) =>
      if row.action = Action.cancel then { row with results := [succeededResult] } else row }

theorem capabilityRequirementRejectsChangedCancellationMeaning :
    satisfiesLifecycleRequirement changedCancellationTable.transitions = false := by
  native_decide

private def structurallyInvalidResult : Option FiniteTableError :=
  let missingSucceeded := { table with states := table.states.take 3 }
  match missingSucceeded.validate with
  | .error error => some error
  | .ok _ => none

theorem undeclaredResultStateIsStructuralFailure :
    structurallyInvalidResult = some (.outOfDomain .resultState) := by
  native_decide

/-- Both independent identity roots admit exactly the same complete transition relation. -/
theorem completeTransitionsMatchEstablished :
    (admittedTransitions == some expectedTransitions) = true ∧
      (establishedTransitions == some expectedTransitions) = true := by
  native_decide

/-- Both independent identity roots admit exactly the same complete setup-to-initial-state relation. -/
theorem completeInitialStatesMatchEstablished :
    (checkBaseline.toOption.bind fun baseline =>
      normalizeInitialStates baseline.target baseline.model) = some expectedInitialStates ∧
      normalizeEstablishedInitialStates = some expectedInitialStates := by
  native_decide

private def unknownNormalizationFailures : Option (Bool × Bool) := do
  let baseline ← checkBaseline.toOption
  let unknownStateResult : Step ModelValue ModelValue ModelValue := {
    outcome := baseline.model.canceledOutcome
    state := Temporal.Feature.Nexus.Lifecycle.scheduledState
    facts := [baseline.model.canceledFact]
  }
  let unknownFactResult : Step ModelValue ModelValue ModelValue := {
    outcome := baseline.model.canceledOutcome
    state := baseline.model.canceledState
    facts := [Temporal.Feature.Nexus.Lifecycle.startedObservation]
  }
  pure (
    (normalizeResult baseline.model baseline.model.startedState baseline.model.cancelAction
      unknownStateResult).isNone,
    (normalizeResult baseline.model baseline.model.startedState baseline.model.cancelAction
      unknownFactResult).isNone)

private def extraTransitionCannotCompareEqual : Option Bool := do
  let extraRow : FiniteTransitionRow State Action Outcome Fact := {
    key := "scheduled-cancel"
    source := State.scheduled
    action := Action.cancel
    results := [canceledResult]
  }
  let extraTable := { table with transitions := table.transitions ++ [extraRow] }
  let target ← (extraTable.checkModel identity modelSpec modelProviders).toOption
  let model ← modelVocabulary.toOption
  pure (normalizeTarget target model != some expectedTransitions)

private def extraInitialAlternativeCannotCompareEqual : Option Bool := do
  let extraInitialTable := { table with initial := table.initial.map fun
    (row : FiniteSetupRow Setup State) =>
    if row.setup = Setup.scheduled then
      { row with states := [State.scheduled, State.started] }
    else
      row
  }
  let target ← (extraInitialTable.checkModel identity modelSpec modelProviders).toOption
  let model ← modelVocabulary.toOption
  pure (normalizeInitialStates target model != some expectedInitialStates)

theorem comparisonNormalizationFailsClosed :
    unknownNormalizationFailures = some (true, true) ∧
      extraTransitionCannotCompareEqual = some true ∧
      extraInitialAlternativeCannotCompareEqual = some true := by
  native_decide

private def planSemantics (run : PlanResult) : Option (List String × List String × List String ×
    List (List String)) :=
  run.artifact.map fun artifact =>
    (artifact.plan.requestedActions.map ModelValue.value,
      artifact.plan.modelOutcomes.map ModelValue.value,
      artifact.plan.resultingStates.map ModelValue.value,
      artifact.plan.checkpoints.map fun checkpoint => checkpoint.observations.map ModelValue.value)

/-- Explicit stable-key identity mapping makes all three bounded traces equal to established Nexus. -/
theorem exactBoundedTracesMatchEstablished :
    (checkBaseline.toOption.map (fun baseline =>
      (planSemantics baseline.start.run,
        planSemantics baseline.cancel.run,
        planSemantics baseline.success.run)) ==
    some (
      Temporal.Feature.Nexus.Operations.AsyncStart.run.toOption.bind planSemantics,
      Temporal.Feature.Nexus.Operations.Cancellation.run.toOption.bind planSemantics,
      Temporal.Feature.Nexus.Operations.SuccessfulCompletion.run.toOption.bind planSemantics)) = true := by
  native_decide

theorem declarationsAdmitAndRunWithExplicitInputs :
    (checkBaseline.toOption.map (fun baseline =>
      (baseline.start.run.result.outcome.name,
        baseline.cancel.run.result.outcome.name,
        baseline.success.run.result.outcome.name,
        baseline.start.query.form.quantifier,
        baseline.cancel.query.form.quantifier,
        baseline.success.query.form.quantifier,
        baseline.start.query.limits,
        baseline.cancel.query.limits,
        baseline.success.query.limits,
        baseline.start.query.policy,
        baseline.cancel.query.policy,
        baseline.success.query.policy)) == some (
      "found", "found", "found",
      QueryQuantifier.existential, QueryQuantifier.existential, QueryQuantifier.existential,
      { behavior := {
          transitions := { value := 1, unit := .semanticTransitions },
          selectedActions := { value := 1, unit := .selectedActions }
        }, search := { value := 8, unit := .candidateEvaluations } },
      { behavior := {
          transitions := { value := 1, unit := .semanticTransitions },
          selectedActions := { value := 1, unit := .selectedActions }
        }, search := { value := 8, unit := .candidateEvaluations } },
      { behavior := {
          transitions := { value := 1, unit := .semanticTransitions },
          selectedActions := { value := 1, unit := .selectedActions }
        }, search := { value := 8, unit := .candidateEvaluations } },
      { strategy := .shortest, seed := 17, tieBreak := .definitionId },
      { strategy := .shortest, seed := 17, tieBreak := .definitionId },
      { strategy := .shortest, seed := 17, tieBreak := .definitionId })) = true := by
  native_decide

private def targetErrorKind
    (result : Except TableAdmissionError (QueryModel LawStatement)) :
    Option DefinitionErrorKind :=
  match result with
  | .error (.invalidTarget diagnostic) => some diagnostic.error.kind
  | _ => none

theorem missingProviderIsTargetFailure :
    targetErrorKind (table.checkModel identity modelSpec Providers.empty) =
      some .missingProvider := by
  native_decide

private def competingProviderId : DefinitionId :=
  Temporal.Shared.definitionId "temporal.nexus2.basic-lifecycle.provider.competing"

private def competingProvider : Provider LawStatement :=
  { lifecycleProvider with
    id := competingProviderId
    meanings := lifecycleProvider.meanings.map fun meaning =>
      if meaning.definitionId = operationStateId then
        { meaning with behaviorVersion := "temporal-nexus2-conflicting-state/v1" }
      else
        meaning
  }

private def competingDefinition : TableModelSpec := {
  modelSpec with
  definitions := modelSpec.definitions ++ [
    Temporal.Shared.definitionMetadata competingProviderId .provider Lifecycle.source
      "temporal-nexus2-basic-lifecycle-provider/v1"
  ]
}

private def competingComposition : Providers LawStatement :=
  Providers.empty |>.provide lifecycleProvider |>.provide competingProvider

theorem competingProvidersAreTargetFailure :
    targetErrorKind (table.checkModel identity competingDefinition competingComposition) =
      some .conflictingProviders := by
  native_decide

private def propertyErrorKind : Option PropertyErrorKind := do
  let baseline ← checkBaseline.toOption
  let unknown := Temporal.Shared.definitionId "temporal.nexus2.basic-lifecycle.action.unknown"
  let declaration := { Start.authoredProperty baseline.model with
    clauses := [.transitionContract
      (Temporal.Shared.definitionId "temporal.nexus2.basic-lifecycle.property.bad-reference")
      (PropertyPattern.exact .selectedAction unknown "unknown")
      (PropertyPattern.exact .resultingState operationStateId baseline.model.startedState.value)] }
  match Property.check (PropertyCheckContext.ofTarget baseline.target) (declaration) with
  | .error error => some error.kind
  | .ok _ => none

theorem invalidReferenceIsPropertyFailure : propertyErrorKind = some .unknownReference := by
  native_decide

private def missingCapabilityErrorKind : Option PropertyErrorKind := do
  let baseline ← checkBaseline.toOption
  let context := { PropertyCheckContext.ofTarget baseline.target with providers := [] }
  match Property.check context ((Start.authoredProperty baseline.model)) with
  | .error error => some error.kind
  | .ok _ => none

theorem undeclaredCapabilityUseIsPropertyFailure :
    missingCapabilityErrorKind = some .missingCapability := by
  native_decide

private def contradictoryBehaviorErrorKind : Option ScenarioErrorKind := do
  let baseline ← checkBaseline.toOption
  let declaration := { Cancel.authoredScenario baseline.model with
    forbiddenActions := [cancelActionId] }
  match Scenario.check (.ofTarget baseline.target) declaration with
  | .error error => some error.kind
  | .ok _ => none

theorem contradictoryBehaviorIsRejectedByBehavior :
    contradictoryBehaviorErrorKind = some .forbiddenRequired := by
  native_decide

private def queryErrorKinds : Option (QueryErrorKind × QueryErrorKind) := do
  let baseline ← checkBaseline.toOption
  let declaration := Cancel.queryDeclaration baseline.cancel.property baseline.cancel.behavior
  let zero := { declaration with limits := {
    declaration.limits with search := { value := 0, unit := .candidateEvaluations } } }
  let wrongUnit := { declaration with limits := {
    declaration.limits with behavior := {
      declaration.limits.behavior with
      transitions := { value := 1, unit := .selectedActions }
    } } }
  let zeroKind ← match checkQuery (.ofTarget baseline.target) zero with
    | .error error => some error.kind | .ok _ => none
  let unitKind ← match checkQuery (.ofTarget baseline.target) wrongUnit with
    | .error error => some error.kind | .ok _ => none
  pure (zeroKind, unitKind)

theorem invalidAndWrongUnitLimitsAreQueryFailures :
    queryErrorKinds = some (.invalidLimit, .unitMismatch) := by
  native_decide

#guard_msgs (error, substring := true) in
def omittedLimitUnit : Limit := { value := 1 }

#guard_msgs (error, substring := true) in
def omittedQueryLimits (baseline : CheckedBaseline) : QueryDeclaration := {
  id := Cancel.queryId
  source := Cancellation.source
  target := targetId
  form := .witness baseline.cancel.property
  behavior := baseline.cancel.behavior
  policy := { strategy := .shortest, seed := 17, tieBreak := .definitionId }
}

private def unsatisfiablePlannerStatus : Option (ScenarioStatus × String) := do
  let baseline ← checkBaseline.toOption
  let differentId :=
    Temporal.Shared.definitionId "temporal.nexus2.basic-lifecycle.setup.started-different"
  let declaration := { Cancel.authoredScenario baseline.model with
    setup := (Cancel.authoredScenario baseline.model).setup ++ [{
      id := differentId
      relation := .different
      left := .role operationRoleId
      right := .value baseline.model.startedState
    }] }
  let behavior ← (Scenario.check (.ofTarget baseline.target) declaration).toOption
  let query ← (checkQuery (.ofTarget baseline.target)
    (Cancel.queryDeclaration baseline.cancel.property behavior)).toOption
  let kernel ← (SearchView.ofCheckedQuery baseline.target.id query).toOption
  let run ← (plan query kernel).toOption
  pure (behavior.spaceStatus, run.result.outcome.name)

theorem unsatisfiableScenarioRemainsPlannerStatus :
    unsatisfiablePlannerStatus = some (.unsatisfiable, "unsatisfiable") := by
  native_decide

private def guardedCancelDeclaration
    (model : ModelVocabulary)
    (guardAction : DefinitionId := cancelActionId)
    (guardValue : String := model.cancelAction.value) : Property := {
  Cancel.authoredProperty model with
  version := 2
  clauses := [.branches {
    id := Temporal.Shared.definitionId
      "temporal.nexus2.basic-lifecycle.property.cancel.same-step"
    source := Cancellation.source
    guard := .atom {
      field := .selectedAction
      reference := guardAction
      constraint := .equals (.text guardValue)
    }
    cases := [{
      id := Temporal.Shared.definitionId
        "temporal.nexus2.basic-lifecycle.property.cancel.same-step.case"
      source := Cancellation.source
      guard := .atom {
        field := .priorState
        reference := operationStateId
        constraint := .equals (.text model.startedState.value)
      }
      clauses := [
        {
          id := Temporal.Shared.definitionId
            "temporal.nexus2.basic-lifecycle.property.cancel.same-step.state"
          source := Cancellation.source
          expectation := .atom {
            field := .resultingState
            reference := operationStateId
            constraint := .equals (.text model.canceledState.value)
          }
        },
        {
          id := Temporal.Shared.definitionId
            "temporal.nexus2.basic-lifecycle.property.cancel.same-step.outcome"
          source := Cancellation.source
          expectation := .atom {
            field := .outcome
            reference := transitionOutcomeId
            constraint := .equals (.text model.canceledOutcome.value)
          }
        },
        {
          id := Temporal.Shared.definitionId
            "temporal.nexus2.basic-lifecycle.property.cancel.same-step.fact"
          source := Cancellation.source
          expectation := .atom {
            field := .expectationFact
            reference := lifecycleFactId
            constraint := .equals (.text model.canceledFact.value)
          }
        }
      ]
    }]
    complete := true
    exclusive := true
  }]
}

private def guardedPlannerOutcome
    (missingInput : Bool) : Option (String × Option QueryErrorKind) := do
  let baseline ← checkBaseline.toOption
  let declaration := if missingInput then
    guardedCancelDeclaration baseline.model startActionId baseline.model.startAction.value
  else
    guardedCancelDeclaration baseline.model
  let propertyContext := PropertyCheckContext.ofTarget baseline.target
  let propertyContext := if missingInput then {
    propertyContext with meanings := propertyContext.meanings.filter fun entry =>
      entry.2.definitionId != cancelActionId
  } else propertyContext
  let property ← (Property.check propertyContext (declaration)).toOption
  let query ← (checkQuery (.ofTarget baseline.target)
    (Cancel.queryDeclaration property baseline.cancel.behavior)).toOption
  let kernel ← (SearchView.ofCheckedQuery baseline.target.id query).toOption
  let run ← (plan query kernel).toOption
  let outcome := run.result.outcome
  pure (outcome.name, match outcome with
    | .invalid error => some error.kind
    | _ => none)

/-- Guarded Properties remain plannable, while incomplete exact inputs terminate as invalid. -/
theorem guardedPlannerPreservesSuccessAndTypedInputFailure :
    (guardedPlannerOutcome false, guardedPlannerOutcome true) =
      (some ("found", none), some ("invalid", some .propertyEvaluationFailure)) := by
  native_decide

private def guardedTemporalCancelDeclaration
    (model : ModelVocabulary)
    (guardAction : DefinitionId := cancelActionId)
    (guardValue : String := model.cancelAction.value) : Property := {
  Cancel.authoredProperty model with
  version := 2
  clauses := [.eventuallyWithin
      (id := (Temporal.Shared.definitionId
      "temporal.nexus2.basic-lifecycle.property.cancel.guarded-temporal"))
      (source := Cancellation.source)
      (guard := some (.atom {
      field := .selectedAction
      reference := guardAction
      constraint := .equals (.text guardValue)
    }))
      (exception := none)
      (trigger := {
      field := .observation
      reference := lifecycleFactId
      constraint := .equals model.canceledFact.value
    })
      (response := {
      field := .outcome
      reference := transitionOutcomeId
      constraint := .equals model.canceledOutcome.value
    })
      (limit := { value := 0, unit := .semanticTransitions })]
}

private def guardedTemporalPlannerOutcome
    (missingInput : Bool) : Option (String × Option QueryErrorKind) := do
  let baseline ← checkBaseline.toOption
  let declaration := if missingInput then
    guardedTemporalCancelDeclaration baseline.model startActionId baseline.model.startAction.value
  else
    guardedTemporalCancelDeclaration baseline.model
  let propertyContext := PropertyCheckContext.ofTarget baseline.target
  let propertyContext := if missingInput then {
    propertyContext with meanings := propertyContext.meanings.filter fun entry =>
      entry.2.definitionId != cancelActionId
  } else propertyContext
  let property ← (Property.check propertyContext (declaration)).toOption
  let query ← (checkQuery (.ofTarget baseline.target)
    (Cancel.queryDeclaration property baseline.cancel.behavior)).toOption
  let kernel ← (SearchView.ofCheckedQuery baseline.target.id query).toOption
  let run ← (plan query kernel).toOption
  let outcome := run.result.outcome
  pure (outcome.name, match outcome with
    | .invalid error => some error.kind
    | _ => none)

/-- Planning evaluates admitted trigger-frozen clauses and retains incomplete exact-input errors. -/
theorem guardedTemporalPlannerPreservesSuccessAndTypedInputFailure :
    (guardedTemporalPlannerOutcome false, guardedTemporalPlannerOutcome true) =
      (some ("found", none), some ("invalid", some .propertyEvaluationFailure)) := by
  native_decide

private def noncanonicalPlannerError : Option FinitePlannerAdmissionErrorKind := do
  let noncanonicalTable := { table with actions := [
    { value := Action.start, key := "start" },
    { value := Action.cancel, key := "cancel" },
    { value := Action.reportSuccess, key := "handler-reports-success" }
  ] }
  let target ← (noncanonicalTable.checkModel identity modelSpec modelProviders).toOption
  let model ← modelVocabulary.toOption
  let property ← (Property.check (PropertyCheckContext.ofTarget target)
    ((Cancel.authoredProperty model))).toOption
  let behavior ← (Scenario.check (.ofTarget target) (Cancel.authoredScenario model)).toOption
  let query ← (checkQuery (.ofTarget target) (Cancel.queryDeclaration property behavior)).toOption
  match SearchView.ofCheckedQuery target.id query with
  | .error error => some error.kind
  | .ok _ => none

theorem noncanonicalAuthoredActionSequenceIsRejected :
    noncanonicalPlannerError = some .noncanonicalActionOrder := by
  native_decide

private def noncanonicalStepPlannerError : Option FinitePlannerAdmissionErrorKind := do
  let noncanonicalTable := { table with transitions := table.transitions.map fun
    (row : FiniteTransitionRow State Action Outcome Fact) =>
      if row.action = Action.cancel then
        { row with results := [succeededResult, canceledResult] }
      else
        row
  }
  let target ← (noncanonicalTable.checkModel identity modelSpec modelProviders).toOption
  let model ← modelVocabulary.toOption
  let property ← (Property.check (PropertyCheckContext.ofTarget target)
    ((Cancel.authoredProperty model))).toOption
  let behavior ← (Scenario.check (.ofTarget target) (Cancel.authoredScenario model)).toOption
  let query ← (checkQuery (.ofTarget target) (Cancel.queryDeclaration property behavior)).toOption
  match SearchView.ofCheckedQuery target.id query with
  | .error error => some error.kind
  | .ok _ => none

theorem noncanonicalAuthoredResultSequenceIsRejected :
    noncanonicalStepPlannerError = some .noncanonicalStepOrder := by
  native_decide

private def admittedPlannerSequence : Option
    (List ModelValue × List ModelValue ×
      List (Step ModelValue ModelValue ModelValue)) := do
  let baseline ← checkBaseline.toOption
  let kernel ← (SearchView.ofCheckedQuery baseline.target.id baseline.cancel.query).toOption
  pure (
    (List.range kernel.actionLimit).filterMap kernel.actionAt,
    (List.range (kernel.initialLimit baseline.model.startedSetup)).filterMap
      (kernel.initialAt baseline.model.startedSetup),
    (List.range (kernel.stepLimit baseline.model.startedState baseline.model.cancelAction)).filterMap
      (kernel.stepAt baseline.model.startedState baseline.model.cancelAction))

theorem admittedPlannerPreservesExactAuthoredSequences :
    admittedPlannerSequence = checkBaseline.toOption.map (fun baseline =>
      let result : Step ModelValue ModelValue ModelValue := {
        outcome := baseline.model.canceledOutcome
        state := baseline.model.canceledState
        facts := [baseline.model.canceledFact]
      }
      ([baseline.model.cancelAction, baseline.model.startAction,
          baseline.model.reportSuccessAction],
        [baseline.model.startedState],
        [result])) := by
  native_decide

private def malformedModelValidation : Option FiniteTableError :=
  let malformed := { table with states :=
    { value := State.scheduled, key := "scheduled" } :: table.states }
  match (malformed.validate.map (·.withIdentity identity)) with
  | .error error => some error
  | .ok _ => none

private def undeclaredSetupResolution : Option FiniteTableError := do
  let reduced := { table with
    setups := [{ value := Setup.scheduled, key := "scheduled" }]
    initial := [{ setup := Setup.scheduled, states := [State.scheduled] }]
  }
  let model ← ((reduced.validate.map (·.withIdentity identity))).toOption
  match model.setupValue .started with
  | .error error => some error
  | .ok _ => none

theorem checkedValueResolutionRejectsMalformedCatalogAndUndeclaredSetup :
    malformedModelValidation = some (.duplicateKey .state) ∧
      undeclaredSetupResolution = some (.outOfDomain .setup) := by
  native_decide

end Baseline

namespace AbstractRace

open Race

private def admittedOutcomeNames : Option (String × String × String × String) := do
  let checked ← checkRace.toOption
  pure (
    checked.verify.run.result.outcome.name,
    checked.canceledWitness.run.result.outcome.name,
    checked.succeededWitness.run.result.outcome.name,
    checked.cancellationAlwaysWins.run.result.outcome.name)

/-- Verification and existential searches preserve the Target-owned race alternatives. -/
theorem terminalResponseAndBothOutcomesAreExercised :
    admittedOutcomeNames = some ("verified-within-limits", "found", "found", "found") := by
  native_decide

theorem exactRequestThenResolveQueryKeepsExplicitBoundsAndForm :
    checkRace.toOption.map (fun checked =>
      (checked.verify.behavior.actionsExactly,
        checked.verify.query.limits.behavior.transitions,
        checked.verify.query.limits.behavior.selectedActions,
        checked.verify.query.quantifier,
        checked.verify.query.claim)) = modelVocabulary.toOption.map (fun model =>
      (some [model.requestCancelAction.definitionId, model.resolveAction.definitionId],
        { value := 2, unit := .semanticTransitions },
        { value := 2, unit := .selectedActions },
        QueryQuantifier.universal,
        QueryClaim.verifiedWithinLimits)) := by
  native_decide

private def selectedOutcome
    (question : CheckedQuestion) : Option ModelValue := do
  let artifact ← question.run.artifact
  artifact.plan.modelOutcomes.getLast?

theorem witnessesAndCounterexampleSelectIndependentOutcomes :
    checkRace.toOption.map (fun checked =>
      (selectedOutcome checked.canceledWitness,
        selectedOutcome checked.succeededWitness,
        selectedOutcome checked.cancellationAlwaysWins)) =
      modelVocabulary.toOption.map (fun model =>
        (some model.canceledOutcome, some model.succeededOutcome, some model.succeededOutcome)) := by
  native_decide

theorem requestAndResolutionHaveDistinctNonterminalAndTerminalFacts :
    table.transitions = [
      { key := "request-cancel", source := State.started, action := Action.requestCancel,
        results := [cancelRequestedResult] },
      { key := "resolve", source := State.cancelRequested, action := Action.resolve,
        results := [canceledResult, succeededResult] }
    ] ∧
    cancelRequestedResult.facts = [Fact.cancelRequested] ∧
    canceledResult.facts = [Fact.lifecycleCanceled, Fact.terminal] ∧
    succeededResult.facts = [Fact.lifecycleSucceeded, Fact.terminal] ∧
    table.transitions.all (fun row =>
      row.source != State.canceled && row.source != State.succeeded) = true := by
  native_decide

theorem scenarioStatusesRemainDistinct :
    checkRace.toOption.map (fun checked =>
      (checked.requestOnly.run.result.outcome.name,
        checked.noTrigger.run.result.outcome.name,
        checked.unsatisfiable.run.result.outcome.name)) =
      some ("found", "verified-within-limits", "unsatisfiable") := by
  native_decide

private def conditionalPropertyEvaluations : Option (Bool × Bool) := do
  let checked ← checkRace.toOption
  let requestResult : Step ModelValue ModelValue ModelValue := {
    outcome := checked.model.cancellationRequestedOutcome
    state := checked.model.cancelRequestedState
    facts := [checked.model.cancelRequestedFact]
  }
  let noTrigger : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := checked.model.startedState
    steps := []
  }
  let requestOnly : ModelTrace ModelValue ModelValue ModelValue ModelValue := {
    initialState := checked.model.startedState
    steps := [ModelTraceStep.result checked.model.requestCancelAction requestResult]
  }
  let noTriggerEvaluation ← (evaluatePropertyOnTrace checked.noTrigger.property noTrigger).toOption
  let requestOnlyEvaluation ←
    (evaluatePropertyOnTrace checked.requestOnly.property requestOnly).toOption
  pure (noTriggerEvaluation.satisfied, requestOnlyEvaluation.satisfied)

/-- A conditional pass without a trigger is distinct from the exercised request-only violation. -/
theorem noTriggerIsVacuousWhileRequestOnlyIsExercised :
    conditionalPropertyEvaluations = some (true, false) := by
  native_decide

theorem budgetExhaustionDiffersFromExhaustiveAbsence :
    checkRace.toOption.map (fun checked =>
      (checked.verify.query.limits.search,
        checked.verify.run.result.metadata.explored.traces,
        checked.exhaustiveAbsence.run.result.outcome.name,
        checked.exhaustiveAbsence.run.result.metadata.completeness.established,
        checked.limitReached.run.result.outcome.name,
        checked.limitReached.run.result.metadata.completeness.established)) = some (
      { value := 32, unit := .candidateEvaluations }, 4,
      "no-such-trace-within-complete-limits", true,
      "limit-reached", false) := by
  native_decide

/-- The prototype excludes completion before request, terminal late events, repeated requests,
retries, caller closure, and multiple operations instead of supplying unstated behavior. -/
theorem unsupportedScopeHasNoInventedTransitionsOrVocabulary :
    table.actions.values = [.requestCancel, .resolve] ∧
    table.setups.values = [.started] ∧
    table.transitions.all (fun row =>
      (row.source == State.started && row.action == Action.requestCancel) ||
      (row.source == State.cancelRequested && row.action == Action.resolve)) = true ∧
    table.transitions.any (fun row =>
      row.source == State.started && row.action == Action.resolve) = false ∧
    table.transitions.any (fun row =>
      row.source == State.cancelRequested && row.action == Action.requestCancel) = false := by
  native_decide

private def unsupportedActionError : Option ScenarioErrorKind := do
  let checked ← checkRace.toOption
  let unknown := Temporal.Shared.definitionId "temporal.nexus2.race.action.caller-close"
  let declaration := { exactBehaviorDeclaration checked.model with
    allowedActions := (exactBehaviorDeclaration checked.model).allowedActions ++ [unknown] }
  match Scenario.check (.ofTarget checked.target) declaration with
  | .error error => some error.kind
  | .ok _ => none

theorem unsupportedActionReferenceIsRejected :
    unsupportedActionError = some .unknownReference := by
  native_decide

private def raceId (value : String) : DefinitionId := Temporal.Shared.definitionId value

private def raceAtom
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (literal : PropertyLiteral) : PropertyPredicate :=
  .atom { field, reference, constraint := .equals literal }

private def raceExpectation
    (clauseId : String)
    (field : PropertyPredicateField)
    (reference : DefinitionId)
    (literal : PropertyLiteral) : PropertySameStepClause := {
  id := raceId clauseId
  source
  expectation := raceAtom field reference literal
}

private def requestCase (model : ModelVocabulary) : PropertyBranch := {
  id := raceId "temporal.nexus2.cancellation-race.case.request"
  source
  guard := raceAtom .selectedAction requestCancelActionId (.text model.requestCancelAction.value)
  clauses := [raceExpectation "temporal.nexus2.cancellation-race.case.request.state"
    .resultingState operationStateId (.text model.cancelRequestedState.value)]
  temporalClauses := [.eventuallyWithin
    (raceId "temporal.nexus2.cancellation-race.case.request.terminal") source
    (PropertyPattern.exact .selectedAction requestCancelActionId model.requestCancelAction.value)
    (PropertyPattern.exact .observation terminalFactId model.terminalFact.value)
    { value := 1, unit := .semanticTransitions }]
}

private def resolveCase (model : ModelVocabulary) : PropertyBranch := {
  id := raceId "temporal.nexus2.cancellation-race.case.resolve"
  source
  guard := raceAtom .selectedAction resolveActionId (.text model.resolveAction.value)
  clauses := [raceExpectation "temporal.nexus2.cancellation-race.case.resolve.terminal"
    .expectationFact terminalFactId (.text model.terminalFact.value)]
}

private def raceCaseGroup
    (model : ModelVocabulary)
    (cases : List PropertyBranch) : PropertyBranches := {
  id := raceId "temporal.nexus2.cancellation-race.case-group.lifecycle"
  source
  guard := .any [
    raceAtom .selectedAction requestCancelActionId (.text model.requestCancelAction.value),
    raceAtom .selectedAction resolveActionId (.text model.resolveAction.value)
  ]
  cases
  complete := true
  exclusive := true
}

private def raceCasePropertyDeclaration
    (model : ModelVocabulary)
    (cases : List PropertyBranch) : Property := {
  id := raceId "temporal.nexus2.cancellation-race.property.cases"
  source
  version := 2
  requires := [capabilityId]
  clauses := [.branches (raceCaseGroup model cases)]
}

private def caseAnalysisFor?
    (cases : List PropertyBranch)
    (budget : Nat := 32) : Option BranchAnalysisResult := do
  let checked ← checkRace.toOption
  let property ← (Property.check (PropertyCheckContext.ofTarget checked.target)
    ((raceCasePropertyDeclaration checked.model cases))).toOption
  let behavior ← (Scenario.check (.ofTarget checked.target)
    (exactBehaviorDeclaration checked.model)).toOption
  let query ← (checkQuery (.ofTarget checked.target) {
    id := raceId "temporal.nexus2.cancellation-race.query.case-analysis"
    source
    target := targetId
    form := .select [property]
    behavior
    limits := QueryLimits.bounded 2 2 budget
    policy := .exhaustive
  }).toOption
  let kernel ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure (analyzeBranches query kernel)

private def caseAnalysis? (budget : Nat := 32) : Option BranchAnalysisResult := do
  let checked ← checkRace.toOption
  caseAnalysisFor? [requestCase checked.model, resolveCase checked.model] budget

private abbrev CaseAnalysisSummary :=
  DefinitionId × DefinitionId × String × Bool ×
    Option (Bool × List DefinitionId × List DefinitionId)

private def caseAnalysisSummary : Option CaseAnalysisSummary :=
  caseAnalysis?.map (fun result =>
      (result.scope.targetId,
        result.scope.behaviorId,
        result.status.name,
        result.findings.isEmpty,
        result.requirements.head?.map fun requirement =>
          (requirement.parentExercised,
            requirement.exercisedCaseIds,
            requirement.clauses.map PropertyClauseIdentity.clauseId)))

/-! The real finite race exercises both named cases and retains same-step plus temporal scope. -/
theorem finiteRaceCaseCoverageIsExhaustive :
    (caseAnalysisSummary == some (
      targetId,
      raceId "temporal.nexus2.cancellation-race.behavior.request-then-resolve",
      "exhaustive",
      true,
      some (true, [
        raceId "temporal.nexus2.cancellation-race.case.request",
        raceId "temporal.nexus2.cancellation-race.case.resolve"
      ], [
        raceId "temporal.nexus2.cancellation-race.case.request.state",
        raceId "temporal.nexus2.cancellation-race.case.request.terminal",
        raceId "temporal.nexus2.cancellation-race.case.resolve.terminal"
      ]))) = true := by
  native_decide

theorem finiteRaceCaseBudgetExhaustionIsInconclusive :
    (caseAnalysis? 1).map (fun result =>
      (result.status.name, result.metadata.completeness.established,
        result.scope.limits.search)) = some (
      "limit-reached",
      false,
      { value := 1, unit := .candidateEvaluations }) := by
  native_decide

/-! Joint temporal evidence retains the request trigger coordinate and its original bound. -/
private def jointTemporalSummary :=
  ((caseAnalysis? 32).bind fun result => result.joint.triggers.head?).map fun trigger =>
    (trigger.trigger.modeledPrefix.trace.steps.length,
      trigger.trigger.transitionPosition,
      trigger.trigger.limits.behavior.transitions,
      trigger.expectations.filterMap fun expectation => match expectation.formula with
        | .guardedTemporal _ _ _ limit => some (expectation.triggerCoordinate, limit)
        | .sameStep _ => none,
      trigger.admittedContinuations.length,
      trigger.satisfyingContinuations.length)

theorem finiteRaceJointTemporalScopeIsFrozenAtTrigger :
    (jointTemporalSummary == some (
      0,
      1,
      ({ value := 2, unit := .semanticTransitions } : Limit),
      [(1, ({ value := 1, unit := .semanticTransitions } : Limit))],
      2,
      2)) = true := by
  native_decide

private def requestCaseWithLaterException (model : ModelVocabulary) : PropertyBranch := {
  requestCase model with
  exception := some {
    id := raceId "temporal.nexus2.cancellation-race.case.request.exception.later-state"
    source
    condition := raceAtom .priorState operationStateId (.text model.cancelRequestedState.value)
  }
}

private def laterExceptionSummary := do
  let checked ← checkRace.toOption
  let result ← caseAnalysisFor?
    [requestCaseWithLaterException checked.model, resolveCase checked.model]
  let trigger ← result.joint.triggers.head?
  let expectation ← trigger.expectations.find? fun (expectation : OverlapExpectationEvidence) =>
    match expectation.formula with
    | JointObligationFormula.guardedTemporal .. => true
    | JointObligationFormula.sameStep _ => false
  let exception ← expectation.exceptions.head?
  let input ← (checkPropertyPredicateInput exception.condition {
    context := .before
    priorState := some checked.model.cancelRequestedState
    selectedAction := some checked.model.resolveAction
  }).toOption
  let limit ← match expectation.formula with
    | .guardedTemporal _ _ _ limit => some limit
    | .sameStep _ => none
  pure (result.joint.status.name,
    expectation.transitionPosition,
    expectation.triggerCoordinate,
    limit,
    exception.id,
    evaluatePropertyPredicate exception.condition input,
    trigger.satisfyingContinuations.length)

/-! An exception that matches a later step cannot withdraw the request-time temporal obligation. -/
theorem finiteRaceLaterExceptionCannotWithdrawTemporalObligation :
    (laterExceptionSummary == modelVocabulary.toOption.map (fun _ => (
      "compatible-within-limits",
      1,
      1,
      ({ value := 1, unit := .semanticTransitions } : Limit),
      raceId "temporal.nexus2.cancellation-race.case.request.exception.later-state",
      true,
      2))) = true := by
  native_decide

private def jointPropertyDeclaration
    (key : String)
    (action : ModelValue)
    (expectation : PropertyPredicate) : Property := {
  id := raceId ("temporal.nexus2.cancellation-race.property.joint." ++ key)
  source
  version := 2
  requires := [capabilityId]
  clauses := [.branches {
    id := raceId ("temporal.nexus2.cancellation-race.property.joint." ++ key ++ ".group")
    source
    guard := raceAtom .selectedAction action.definitionId (.text action.value)
    cases := [{
      id := raceId ("temporal.nexus2.cancellation-race.property.joint." ++ key ++ ".case")
      source
      guard := raceAtom .selectedAction action.definitionId (.text action.value)
      clauses := [{
        id := raceId ("temporal.nexus2.cancellation-race.property.joint." ++ key ++ ".clause")
        source
        expectation
      }]
    }]
  }]
}

private def jointRaceAnalysis?
    (declarations : List Property) : Option BranchAnalysisResult := do
  let checked ← checkRace.toOption
  let properties ← declarations.mapM fun declaration =>
    (Property.check (PropertyCheckContext.ofTarget checked.target)
      (declaration)).toOption
  let behavior ← (Scenario.check (.ofTarget checked.target)
    (exactBehaviorDeclaration checked.model)).toOption
  let form := QueryForm.select properties
  let query ← (checkQuery (.ofTarget checked.target) {
    id := raceId "temporal.nexus2.cancellation-race.query.joint-analysis"
    source
    target := targetId
    form
    behavior
    limits := QueryLimits.bounded 2 2 32
    policy := .exhaustive
  }).toOption
  let kernel ← (SearchView.ofCheckedQuery query.target.id query).toOption
  pure (analyzeBranches query kernel)

private def observationTriggerPropertyDeclaration
    (key : String)
    (action : ModelValue)
    (trigger response : ModelValue) : Property := {
  id := raceId ("temporal.nexus2.cancellation-race.property.observation-trigger." ++ key)
  source
  version := 2
  requires := [capabilityId]
  clauses := [.eventuallyWithin
      (id := (raceId ("temporal.nexus2.cancellation-race.property.observation-trigger." ++ key ++ ".clause")))
      (source := source)
      (guard := some (raceAtom .selectedAction action.definitionId (.text action.value)))
      (exception := none)
      (trigger := (PropertyPattern.exact .observation trigger.definitionId trigger.value))
      (response := (PropertyPattern.exact .observation response.definitionId response.value))
      (limit := { value := 1, unit := .observationPositions })]
}

private def distinctObservationTriggerAnalysis? : Option BranchAnalysisResult := do
  let checked ← checkRace.toOption
  jointRaceAnalysis? [
    observationTriggerPropertyDeclaration "canceled-left"
      checked.model.resolveAction
      checked.model.lifecycleCanceledFact checked.model.terminalFact,
    observationTriggerPropertyDeclaration "canceled-right"
      checked.model.resolveAction
      checked.model.lifecycleCanceledFact checked.model.terminalFact,
    observationTriggerPropertyDeclaration "succeeded-left"
      checked.model.resolveAction
      checked.model.lifecycleSucceededFact checked.model.terminalFact,
    observationTriggerPropertyDeclaration "succeeded-right"
      checked.model.resolveAction
      checked.model.lifecycleSucceededFact checked.model.terminalFact,
    observationTriggerPropertyDeclaration "terminal-left"
      checked.model.resolveAction
      checked.model.terminalFact checked.model.terminalFact,
    observationTriggerPropertyDeclaration "terminal-right"
      checked.model.resolveAction
      checked.model.terminalFact checked.model.terminalFact
  ]

/-! The lifecycle and terminal observations in one transition remain distinct realized triggers.
Alternative lifecycle values at one Action also admit only the continuation where each trigger
occurs, so an absent trigger cannot become incompatibility. -/
theorem finiteRaceSeparatesObservationTriggerOccurrences :
    (distinctObservationTriggerAnalysis?.map (fun result =>
      (result.joint.status.name,
        result.joint.modelIncompatibilities.isEmpty,
        result.joint.triggers.map fun trigger =>
          (trigger.trigger.occurrence.value,
            trigger.trigger.occurrence.coordinateUnit,
            trigger.trigger.occurrence.coordinate,
            trigger.expectations.length,
            trigger.admittedContinuations.length,
            trigger.satisfyingContinuations.length))) == modelVocabulary.toOption.map (fun model =>
      ("compatible-within-limits", true, [
        (some model.lifecycleCanceledFact, LimitUnit.observationPositions, 2, 2, 1, 1),
        (some model.lifecycleSucceededFact, LimitUnit.observationPositions, 2, 2, 1, 1),
        (some model.terminalFact, LimitUnit.observationPositions, 3, 2, 2, 2)
      ]))) = true := by
  native_decide

private def logicalRaceConflict? : Option BranchAnalysisResult := do
  let checked ← checkRace.toOption
  jointRaceAnalysis? [
    jointPropertyDeclaration "request-cancel-requested" checked.model.requestCancelAction
      (raceAtom .resultingState operationStateId (.text checked.model.cancelRequestedState.value)),
    jointPropertyDeclaration "request-succeeded" checked.model.requestCancelAction
      (raceAtom .resultingState operationStateId (.text checked.model.succeededState.value))
  ]

/-! Mutually exclusive scalar expectations name both checked clauses at the common request trigger. -/
private def logicalRaceConflictSummary :=
  logicalRaceConflict?.map fun result =>
    (result.joint.status.name,
      result.joint.modelIncompatibilities.isEmpty,
      result.joint.logicalConflicts.head?.map fun conflict =>
        (conflict.trigger.transitionPosition,
          conflict.trigger.priorState,
          conflict.trigger.selectedAction,
          conflict.expectations.map OverlapExpectationEvidence.clauseId,
          conflict.expectations.map OverlapExpectationEvidence.source))

theorem finiteRaceReportsSourceLinkedLogicalContradiction :
    (logicalRaceConflictSummary == modelVocabulary.toOption.map (fun model =>
        ("logical-conflict", true, some (
          1,
          some model.startedState,
          some model.requestCancelAction,
          [
            raceId "temporal.nexus2.cancellation-race.property.joint.request-cancel-requested.clause",
            raceId "temporal.nexus2.cancellation-race.property.joint.request-succeeded.clause"
          ],
          [source, source])))) = true := by
  native_decide

private def modelRelativeRaceConflict? : Option BranchAnalysisResult := do
  let checked ← checkRace.toOption
  jointRaceAnalysis? [
    jointPropertyDeclaration "resolve-canceled-state" checked.model.resolveAction
      (raceAtom .resultingState operationStateId (.text checked.model.canceledState.value)),
    jointPropertyDeclaration "resolve-succeeded-outcome" checked.model.resolveAction
      (raceAtom .outcome transitionOutcomeId (.text checked.model.succeededOutcome.value))
  ]

/-! Each requirement has a witness on a different race branch, but no admitted branch meets both. -/
private def modelRelativeRaceConflictSummary :=
  modelRelativeRaceConflict?.map fun result =>
    (result.joint.status.name,
      result.joint.logicalConflicts.isEmpty,
      result.joint.modelIncompatibilities.head?.map fun conflict =>
        (conflict.trigger.modeledPrefix.trace.steps.length,
          conflict.trigger.transitionPosition,
          conflict.trigger.limits,
          conflict.admittedContinuations.length,
          conflict.expectations.map OverlapExpectationEvidence.clauseId))

theorem finiteRaceSeparatesModelRelativeFromLogicalIncompatibility :
    (modelRelativeRaceConflictSummary == some ("model-incompatible", true, some (
        1,
        2,
        QueryLimits.bounded 2 2 32,
        2,
        [
          raceId "temporal.nexus2.cancellation-race.property.joint.resolve-canceled-state.clause",
          raceId "temporal.nexus2.cancellation-race.property.joint.resolve-succeeded-outcome.clause"
        ]))) = true := by
  native_decide

private def malformedCasePropertyError : Option PropertyErrorKind := do
  let checked ← checkRace.toOption
  let malformed := raceCasePropertyDeclaration checked.model []
  match Property.check (PropertyCheckContext.ofTarget checked.target) (malformed) with
  | .error error => some error.kind
  | .ok _ => none

private def mismatchedAnalysisTargetError : Option QueryErrorKind := do
  let checked ← checkRace.toOption
  let property ← (Property.check (PropertyCheckContext.ofTarget checked.target)
    ((raceCasePropertyDeclaration checked.model
      [requestCase checked.model, resolveCase checked.model]))).toOption
  let behavior ← (Scenario.check (.ofTarget checked.target)
    (exactBehaviorDeclaration checked.model)).toOption
  match checkQuery (.ofTarget checked.target) {
    id := raceId "temporal.nexus2.cancellation-race.query.case-analysis.bad-target"
    source
    target := raceId "temporal.nexus2.cancellation-race.target.other"
    form := .select [property]
    behavior
    limits := QueryLimits.bounded 2 2 32
    policy := .exhaustive
  } with
  | .error error => some error.kind
  | .ok _ => none

/-! Malformed case data and Target mismatch reject before bounded traversal begins. -/
theorem malformedCaseAnalysisInputsAreRejected :
    (malformedCasePropertyError, mismatchedAnalysisTargetError) =
      (some .emptyCaseGroup, some .targetMismatch) := by
  native_decide

end AbstractRace

end Temporal.Feature.Nexus2.Tests
