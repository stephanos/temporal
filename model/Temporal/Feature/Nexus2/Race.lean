import Temporal.Feature.Nexus2.Lifecycle

/-!
# Abstract cancellation/completion race

This separate Target models one already-started operation. A cancellation request records a
nonterminal request, then one abstract environment-progress Action resolves to either canceled or
succeeded. The model intentionally omits completion before the request, late events, repeated
requests, retries, caller closure, and multiple operations. Missing rows and vocabulary reject
those inputs; the model makes no fairness, runtime-delivery, or runtime-timeout claim.
-/

namespace Temporal.Feature.Nexus2.Race

open Umpire

private def id (value : String) : DefinitionId := Temporal.Shared.definitionId value

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus2/Race.lean"

def targetId : DefinitionId := id "temporal.nexus2.cancellation-race.target"
def kernelId : DefinitionId := id "temporal.nexus2.cancellation-race.kernel"
def capabilityId : DefinitionId := id "temporal.nexus2.cancellation-race.capability"
def providerId : DefinitionId := id "temporal.nexus2.cancellation-race.provider"
def lawId : DefinitionId := id "temporal.nexus2.cancellation-race.law.authoritative-table"
def operationStateId : DefinitionId := id "temporal.nexus2.cancellation-race.state.operation"
def requestCancelActionId : DefinitionId := id "temporal.nexus2.cancellation-race.action.request-cancel"
def resolveActionId : DefinitionId := id "temporal.nexus2.cancellation-race.action.resolve"
def transitionOutcomeId : DefinitionId := id "temporal.nexus2.cancellation-race.outcome.transition"
def cancelRequestedFactId : DefinitionId := id "temporal.nexus2.cancellation-race.fact.cancel-requested"
def lifecycleFactId : DefinitionId := id "temporal.nexus2.cancellation-race.fact.lifecycle"
def terminalFactId : DefinitionId := id "temporal.nexus2.cancellation-race.fact.terminal"
def operationRoleId : DefinitionId := id "temporal.nexus2.cancellation-race.role.operation"

inductive Setup where
  | started
  deriving BEq, DecidableEq, Repr

inductive State where
  | started
  | cancelRequested
  | canceled
  | succeeded
  deriving BEq, DecidableEq, Repr

inductive Action where
  | requestCancel
  | resolve
  deriving BEq, DecidableEq, Repr

inductive Outcome where
  | cancellationRequested
  | canceled
  | succeeded
  deriving BEq, DecidableEq, Repr

inductive Fact where
  | cancelRequested
  | lifecycleCanceled
  | lifecycleSucceeded
  | terminal
  deriving BEq, DecidableEq, Repr

def cancelRequestedResult : Step State Outcome Fact := {
  outcome := .cancellationRequested
  state := .cancelRequested
  facts := [.cancelRequested]
}

def canceledResult : Step State Outcome Fact := {
  outcome := .canceled
  state := .canceled
  facts := [.lifecycleCanceled, .terminal]
}

def succeededResult : Step State Outcome Fact := {
  outcome := .succeeded
  state := .succeeded
  facts := [.lifecycleSucceeded, .terminal]
}

/-- The complete race model; `resolve` alternatives are Target-owned outcomes. -/
def table : FiniteTable Setup State Action Outcome Fact := {
  setups := [{ value := .started, key := "started" }]
  states := [
    { value := .started, key := "started" },
    { value := .cancelRequested, key := "cancel-requested" },
    { value := .canceled, key := "canceled" },
    { value := .succeeded, key := "succeeded" }
  ]
  actions := [
    { value := .requestCancel, key := "request-cancel" },
    { value := .resolve, key := "resolve" }
  ]
  outcomes := [
    { value := .cancellationRequested, key := "cancellation-requested" },
    { value := .canceled, key := "canceled" },
    { value := .succeeded, key := "succeeded" }
  ]
  facts := [
    { value := .cancelRequested, key := "cancel-requested" },
    { value := .lifecycleCanceled, key := "canceled" },
    { value := .lifecycleSucceeded, key := "succeeded" },
    { value := .terminal, key := "true" }
  ]
  initial := [{ setup := .started, states := [.started] }]
  transitions := [
    { key := "request-cancel", source := .started, action := .requestCancel,
      results := [cancelRequestedResult] },
    { key := "resolve", source := .cancelRequested, action := .resolve,
      results := [canceledResult, succeededResult] }
  ]
}

def identity : FiniteModelIdentity Setup State Action Outcome Fact := {
  setupBindings := fun .started => [{ roleId := operationRoleId, state := .started }]
  stateId := fun _ => operationStateId
  actionId := fun
    | .requestCancel => requestCancelActionId
    | .resolve => resolveActionId
  outcomeId := fun _ => transitionOutcomeId
  factId := fun
    | .cancelRequested => cancelRequestedFactId
    | .lifecycleCanceled | .lifecycleSucceeded => lifecycleFactId
    | .terminal => terminalFactId
}

private def hasExactTransition
    (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (source : State)
    (action : Action)
    (results : List (Step State Outcome Fact)) : Bool :=
  rows.any fun row =>
    row.source == source && row.action == action && row.results == results

/-- The law states the two required source/Action meanings independently of unrelated rows. -/
def satisfiesRaceRequirement
    (rows : List (FiniteTransitionRow State Action Outcome Fact)) : Bool :=
  hasExactTransition rows .started .requestCancel [{
    outcome := .cancellationRequested
    state := .cancelRequested
    facts := [.cancelRequested]
  }] &&
  hasExactTransition rows .cancelRequested .resolve [{
    outcome := .canceled
    state := .canceled
    facts := [.lifecycleCanceled, .terminal]
  }, {
    outcome := .succeeded
    state := .succeeded
    facts := [.lifecycleSucceeded, .terminal]
  }]

def LawStatement (law : Law) : Prop :=
  law.id = lawId ∧
    law.body = "temporal-nexus2-cancellation-race-authoritative-table/v1" ∧
    satisfiesRaceRequirement table.transitions = true

def raceLaw : Law := {
  id := lawId
  body := "temporal-nexus2-cancellation-race-authoritative-table/v1"
}

theorem raceLawProof : LawStatement raceLaw := ⟨rfl, rfl, rfl⟩

private def metadata
    (definitionId : DefinitionId)
    (kind : DefinitionKind)
    (behaviorVersion : String) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata definitionId kind source behaviorVersion

def provider : Provider LawStatement := {
  id := providerId
  source
  contract := {
    id := capabilityId
    behaviorVersion := "temporal-nexus2-cancellation-race/v1"
    requiredLaws := [raceLaw]
  }
  meanings := [
    { definitionId := operationStateId, kind := .state,
      behaviorVersion := "temporal-nexus2-cancellation-race-state/v1" },
    { definitionId := requestCancelActionId, kind := .action,
      behaviorVersion := "temporal-nexus2-cancellation-race-request-cancel/v1" },
    { definitionId := resolveActionId, kind := .action,
      behaviorVersion := "temporal-nexus2-cancellation-race-resolve/v1" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      behaviorVersion := "temporal-nexus2-cancellation-race-outcome/v1" },
    { definitionId := cancelRequestedFactId, kind := .fact,
      behaviorVersion := "temporal-nexus2-cancellation-race-request-fact/v1" },
    { definitionId := lifecycleFactId, kind := .fact,
      behaviorVersion := "temporal-nexus2-cancellation-race-lifecycle-fact/v1" },
    { definitionId := terminalFactId, kind := .fact,
      behaviorVersion := "temporal-nexus2-cancellation-race-terminal-fact/v1" }
  ]
  lawProofs := [{ definition := raceLaw, proof := raceLawProof }]
}

def definitions : List DefinitionMetadata := [
  metadata targetId .target "temporal-nexus2-cancellation-race-target/v1",
  metadata kernelId .machine "temporal-nexus2-cancellation-race-kernel/v1",
  metadata capabilityId .capability "temporal-nexus2-cancellation-race/v1",
  metadata providerId .provider "temporal-nexus2-cancellation-race-provider/v1",
  metadata lawId .law raceLaw.body,
  metadata operationStateId .state "temporal-nexus2-cancellation-race-state/v1",
  metadata requestCancelActionId .action "temporal-nexus2-cancellation-race-request-cancel/v1",
  metadata resolveActionId .action "temporal-nexus2-cancellation-race-resolve/v1",
  metadata transitionOutcomeId .outcome "temporal-nexus2-cancellation-race-outcome/v1",
  metadata cancelRequestedFactId .fact "temporal-nexus2-cancellation-race-request-fact/v1",
  metadata lifecycleFactId .fact "temporal-nexus2-cancellation-race-lifecycle-fact/v1",
  metadata terminalFactId .fact "temporal-nexus2-cancellation-race-terminal-fact/v1"
]

def modelSpec : TableModelSpec := {
  id := targetId
  source
  definitions
  requiredCapabilities := [capabilityId]
  metadata := { id := kernelId, source }
}

def modelProviders : Providers LawStatement :=
  Providers.empty |>.provide provider

def targetResult : Except TableAdmissionError (QueryModel LawStatement) :=
  table.checkModel identity modelSpec modelProviders

structure ModelVocabulary where
  startedState : ModelValue
  cancelRequestedState : ModelValue
  canceledState : ModelValue
  succeededState : ModelValue
  requestCancelAction : ModelValue
  resolveAction : ModelValue
  cancellationRequestedOutcome : ModelValue
  canceledOutcome : ModelValue
  succeededOutcome : ModelValue
  cancelRequestedFact : ModelValue
  lifecycleCanceledFact : ModelValue
  lifecycleSucceededFact : ModelValue
  terminalFact : ModelValue
  startedSetup : List RoleBinding
  deriving BEq, DecidableEq, Repr

def modelVocabulary : Except FiniteTableError ModelVocabulary := do
  let model ← (table.validate.map (·.withIdentity identity))
  pure {
    startedState := ← model.stateValue .started
    cancelRequestedState := ← model.stateValue .cancelRequested
    canceledState := ← model.stateValue .canceled
    succeededState := ← model.stateValue .succeeded
    requestCancelAction := ← model.actionValue .requestCancel
    resolveAction := ← model.actionValue .resolve
    cancellationRequestedOutcome := ← model.outcomeValue .cancellationRequested
    canceledOutcome := ← model.outcomeValue .canceled
    succeededOutcome := ← model.outcomeValue .succeeded
    cancelRequestedFact := ← model.factValue .cancelRequested
    lifecycleCanceledFact := ← model.factValue .lifecycleCanceled
    lifecycleSucceededFact := ← model.factValue .lifecycleSucceeded
    terminalFact := ← model.factValue .terminal
    startedSetup := ← model.setupValue .started
  }

def operationRole : Scenario.Role := { id := operationRoleId, valueKind := .state }

def terminalResponsePropertyId : DefinitionId :=
  id "temporal.nexus2.cancellation-race.property.terminal-response"
def canceledResolutionPropertyId : DefinitionId :=
  id "temporal.nexus2.cancellation-race.property.canceled-resolution"
def succeededResolutionPropertyId : DefinitionId :=
  id "temporal.nexus2.cancellation-race.property.succeeded-resolution"

def terminalResponsePropertyDeclaration (model : ModelVocabulary) : Property := {
  id := terminalResponsePropertyId
  source
  requires := [capabilityId]
  clauses := [.eventuallyWithin
    (id "temporal.nexus2.cancellation-race.property.terminal-response.clause")
    (PropertyPattern.exact .selectedAction requestCancelActionId model.requestCancelAction.value)
    (PropertyPattern.exact .observation terminalFactId model.terminalFact.value)
    { value := 1, unit := .steps }]
  documentation := "A cancellation request receives a terminal model response within one additional semantic transition."
}

private def resolutionPropertyDeclaration
    (propertyId clauseId : DefinitionId)
    (model : ModelVocabulary)
    (outcome : ModelValue)
    (documentation : String) : Property := {
  id := propertyId
  source
  requires := [capabilityId]
  clauses := [.transitionContract clauseId
    (PropertyPattern.exact .selectedAction resolveActionId model.resolveAction.value)
    (PropertyPattern.exact .outcome transitionOutcomeId outcome.value)]
  documentation
}

def canceledResolutionPropertyDeclaration (model : ModelVocabulary) : Property :=
  resolutionPropertyDeclaration canceledResolutionPropertyId
    (id "temporal.nexus2.cancellation-race.property.canceled-resolution.clause")
    model model.canceledOutcome "The abstract resolution may select the canceled outcome."

def succeededResolutionPropertyDeclaration (model : ModelVocabulary) : Property :=
  resolutionPropertyDeclaration succeededResolutionPropertyId
    (id "temporal.nexus2.cancellation-race.property.succeeded-resolution.clause")
    model model.succeededOutcome "The abstract resolution may select the succeeded outcome."

def requestOccurrenceId : DefinitionId :=
  id "temporal.nexus2.cancellation-race.occurrence.request"
def resolutionOccurrenceId : DefinitionId :=
  id "temporal.nexus2.cancellation-race.occurrence.resolution"
def setupConstraintId : DefinitionId :=
  id "temporal.nexus2.cancellation-race.setup.started"

private def startedSetupConstraint (model : ModelVocabulary) : SetupConstraint :=
  SetupConstraint.roleEquals setupConstraintId operationRoleId model.startedState

/-- Select exactly request then abstract resolution; neither Action chooses the terminal outcome. -/
def exactBehaviorDeclaration (model : ModelVocabulary) : Scenario := {
  id := id "temporal.nexus2.cancellation-race.behavior.request-then-resolve"
  source
  requires := [capabilityId]
  roles := [operationRole]
  setup := [startedSetupConstraint model]
  allowedActions := [requestCancelActionId, resolveActionId]
  requiredOccurrences := [
    { id := requestOccurrenceId, action := requestCancelActionId },
    { id := resolutionOccurrenceId, action := resolveActionId }
  ]
  occurrenceBounds := [
    Scenario.Count.exactly requestCancelActionId 1,
    Scenario.Count.exactly resolveActionId 1
  ]
  ordering := [{ before := requestOccurrenceId, after := resolutionOccurrenceId }]
  actionsExactly := some [requestCancelActionId, resolveActionId]
  documentation := "Select one cancellation request followed by one abstract resolution."
}

private def requestOnlyBehaviorDeclaration (model : ModelVocabulary) : Scenario := {
  id := id "temporal.nexus2.cancellation-race.behavior.request-only"
  source
  requires := [capabilityId]
  roles := [operationRole]
  setup := [startedSetupConstraint model]
  allowedActions := [requestCancelActionId]
  requiredOccurrences := [{ id := requestOccurrenceId, action := requestCancelActionId }]
  occurrenceBounds := [Scenario.Count.exactly requestCancelActionId 1]
  actionsExactly := some [requestCancelActionId]
  documentation := "Stop the finite trace after the nonterminal cancellation request."
}

private def noTriggerBehaviorDeclaration (model : ModelVocabulary) : Scenario := {
  id := id "temporal.nexus2.cancellation-race.behavior.no-trigger"
  source
  requires := [capabilityId]
  roles := [operationRole]
  setup := [startedSetupConstraint model]
  actionsExactly := some []
  documentation := "Select the initial state without exercising the conditional request trigger."
}

private def unsatisfiableBehaviorDeclaration (model : ModelVocabulary) : Scenario := {
  exactBehaviorDeclaration model with
  id := id "temporal.nexus2.cancellation-race.behavior.unsatisfiable"
  setup := (exactBehaviorDeclaration model).setup ++ [{
    id := id "temporal.nexus2.cancellation-race.setup.not-started"
    relation := .different
    left := .role operationRoleId
    right := .value model.startedState
  }]
  documentation := "Contradictory setup constraints admit no finite Model Trace."
}

inductive RaceAdmissionError where
  | invalidTarget (error : TableAdmissionError)
  | invalidVocabulary (error : FiniteTableError)
  | invalidProperty (error : PropertyError)
  | invalidBehavior (error : ScenarioError)
  | invalidQuery (error : QueryError)
  | invalidPlanner (error : FiniteSearchAdmissionError)
  | invalidKnownGap (error : KnownGapError)

structure CheckedQuestion where
  property : CheckedProperty
  behavior : CheckedScenario
  query : CheckedQuery LawStatement
  run : PlanResult

structure CheckedRace where
  target : QueryModel LawStatement
  model : ModelVocabulary
  verify : CheckedQuestion
  canceledWitness : CheckedQuestion
  succeededWitness : CheckedQuestion
  cancellationAlwaysWins : CheckedQuestion
  requestOnly : CheckedQuestion
  noTrigger : CheckedQuestion
  unsatisfiable : CheckedQuestion
  exhaustiveAbsence : CheckedQuestion
  limitReached : CheckedQuestion

private def checkQuestion
    (target : QueryModel LawStatement)
    (queryId : DefinitionId)
    (authoredProperty : Property)
    (authoredScenario : Scenario)
    (form : CheckedProperty → QueryForm)
    (policy : PlannerPolicy)
    (candidateBudget : Nat) : Except RaceAdmissionError CheckedQuestion := do
  let property ← Property.check (PropertyCheckContext.ofTarget target) (authoredProperty)
    |>.mapError RaceAdmissionError.invalidProperty
  let behavior ← Scenario.check (.ofTarget target) authoredScenario
    |>.mapError RaceAdmissionError.invalidBehavior
  let declaration : QueryDeclaration := {
    id := queryId
    source
    target := targetId
    form := form property
    behavior
    limits := QueryLimits.bounded 2 2 candidateBudget
    policy
  }
  let query ← checkQuery (.ofTarget target) declaration
    |>.mapError RaceAdmissionError.invalidQuery
  let kernel ← SearchView.ofCheckedQuery target.id query
    |>.mapError RaceAdmissionError.invalidPlanner
  let run ← search query kernel |>.mapError RaceAdmissionError.invalidKnownGap
  pure { property, behavior, query, run }

/-- Admit the race and its separate bounded questions only through successful checked branches. -/
def checkRace : Except RaceAdmissionError CheckedRace := do
  let target ← targetResult.mapError RaceAdmissionError.invalidTarget
  let model ← modelVocabulary.mapError RaceAdmissionError.invalidVocabulary
  let exact := exactBehaviorDeclaration model
  let terminal := terminalResponsePropertyDeclaration model
  let canceled := canceledResolutionPropertyDeclaration model
  let succeeded := succeededResolutionPropertyDeclaration model
  let verify ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.verify-terminal") terminal exact
    QueryForm.verify .exhaustive 32
  let canceledWitness ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.witness-canceled") canceled exact
    QueryForm.witness .shortest 32
  let succeededWitness ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.witness-succeeded") succeeded exact
    QueryForm.witness .shortest 32
  let cancellationAlwaysWins ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.counterexample-cancellation-always-wins") canceled exact
    QueryForm.counterexample .shortest 32
  let requestOnly ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.request-only") terminal
    (requestOnlyBehaviorDeclaration model) QueryForm.verify .exhaustive 32
  let noTrigger ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.no-trigger") terminal
    (noTriggerBehaviorDeclaration model) QueryForm.verify .exhaustive 32
  let unsatisfiable ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.unsatisfiable") terminal
    (unsatisfiableBehaviorDeclaration model) QueryForm.verify .exhaustive 32
  let exhaustiveAbsence ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.exhaustive-absence") terminal exact
    QueryForm.counterexample .exhaustive 32
  let limitReached ← checkQuestion target
    (id "temporal.nexus2.cancellation-race.query.limit-reached") terminal exact
    QueryForm.counterexample .exhaustive 1
  pure {
    target
    model
    verify
    canceledWitness
    succeededWitness
    cancellationAlwaysWins
    requestOnly
    noTrigger
    unsatisfiable
    exhaustiveAbsence
    limitReached
  }

end Temporal.Feature.Nexus2.Race
