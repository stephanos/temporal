import Temporal.Feature.Nexus3.Authoring
import Umpire.Property.Authoring
import Umpire.Behavior.Authoring
import Umpire.Query.Authoring
/-!
# Compact Nexus success model

This executable slice models only scheduled → started → succeeded. `awaitStart` and `awaitSuccess`
represent waits for recorded Temporal outcomes; they do not manufacture those outcomes. Cancellation
and operation-scoped progress remain unsupported design sketches in `Nexus.md`.
-/
namespace Temporal.Feature.Nexus3
open Umpire
abbrev family := Authoring.family
abbrev source := Authoring.source
inductive Setup where | scheduled deriving BEq, DecidableEq, Repr
inductive State where | scheduled | started | succeeded deriving BEq, DecidableEq, Repr
inductive Action where | awaitStart | awaitSuccess deriving BEq, DecidableEq, Repr
inductive Outcome where | acknowledged | completed deriving BEq, DecidableEq, Repr
inductive Fact where | started | succeeded deriving BEq, DecidableEq, Repr
def targetId := Authoring.declarationId "target"
def kernelId := Authoring.declarationId "kernel"
def lifecycleCapabilityId := Authoring.declarationId "capability"
def lifecycleProviderId := Authoring.declarationId "provider"
def lifecycleLawId := Authoring.declarationId "law"
def operationStateId := Authoring.declarationId "state"
def awaitStartActionId := Authoring.memberId "action" ``Action.awaitStart
def awaitSuccessActionId := Authoring.memberId "action" ``Action.awaitSuccess
def transitionOutcomeId := Authoring.declarationId "outcome"
def lifecycleFactId := Authoring.declarationId "observation"
def operationRoleId := Authoring.declarationId "role"
def startedResult : TransitionResult State Outcome Fact :=
  { modelOutcome := .acknowledged, resultingState := .started, observations := [.started] }
def succeededResult : TransitionResult State Outcome Fact :=
  { modelOutcome := .completed, resultingState := .succeeded, observations := [.succeeded] }
/-- The only initial state and the complete two-row success transition relation. -/
def table : FiniteTable Setup State Action Outcome Fact := {
  setups := [{ value := .scheduled, key := Authoring.declarationKey none ``Setup.scheduled }]
  states := [
    { value := .scheduled, key := Authoring.declarationKey none ``State.scheduled },
    { value := .started, key := Authoring.declarationKey none ``State.started },
    { value := .succeeded, key := Authoring.declarationKey none ``State.succeeded }
  ]
  actions := [
    { value := .awaitStart, key := Authoring.declarationKey none ``Action.awaitStart },
    { value := .awaitSuccess, key := Authoring.declarationKey none ``Action.awaitSuccess }
  ]
  outcomes := [
    { value := .acknowledged, key := Authoring.declarationKey none ``Outcome.acknowledged },
    { value := .completed, key := Authoring.declarationKey none ``Outcome.completed }
  ]
  facts := [
    { value := .started, key := Authoring.declarationKey none ``Fact.started },
    { value := .succeeded, key := Authoring.declarationKey none ``Fact.succeeded }
  ]
  initial := [{ setup := .scheduled, states := [.scheduled] }]
  transitions := [
    { key := "awaitStart", source := .scheduled, action := .awaitStart, results := [startedResult] },
    { key := "awaitSuccess", source := .started, action := .awaitSuccess,
      results := [succeededResult] }
  ]
}
def identity : FiniteModelIdentity Setup State Action Outcome Fact := {
  setupBindings := fun _ => [{ roleId := operationRoleId, state := .scheduled }]
  stateId := fun _ => operationStateId
  actionId := fun
    | .awaitStart => awaitStartActionId
    | .awaitSuccess => awaitSuccessActionId
  outcomeId := fun _ => transitionOutcomeId
  factId := fun _ => lifecycleFactId
}
private def hasExactTransition
    (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (source : State) (action : Action) (result : TransitionResult State Outcome Fact) : Bool :=
  rows.any fun row => row.source == source && row.action == action && row.results == [result]
def satisfiesSuccessRequirement
    (rows : List (FiniteTransitionRow State Action Outcome Fact)) : Bool :=
  hasExactTransition rows .scheduled .awaitStart startedResult &&
    hasExactTransition rows .started .awaitSuccess succeededResult
def LawStatement (law : LawDefinition) : Prop :=
  law.id = lifecycleLawId ∧ law.body = "temporal-nexus3-success-table/v1" ∧
    satisfiesSuccessRequirement table.transitions = true
def lifecycleLaw : LawDefinition :=
  { id := lifecycleLawId, body := "temporal-nexus3-success-table/v1" }
theorem lifecycleLawProof : LawStatement lifecycleLaw := ⟨rfl, rfl, rfl⟩
def lifecycleProvider : CapabilityProvider LawStatement := {
  id := lifecycleProviderId
  source
  contract := {
    id := lifecycleCapabilityId
    canonicalBehavior := "temporal-nexus3-success/v1"
    requiredLaws := [lifecycleLaw]
  }
  meanings := [
    { definitionId := operationStateId, kind := .state,
      canonicalBehavior := "temporal-nexus3-operation-state/v1" },
    { definitionId := awaitStartActionId, kind := .action,
      canonicalBehavior := "temporal-nexus3-await-start/v1" },
    { definitionId := awaitSuccessActionId, kind := .action,
      canonicalBehavior := "temporal-nexus3-await-success/v1" },
    { definitionId := transitionOutcomeId, kind := .outcome,
      canonicalBehavior := "temporal-nexus3-transition-outcome/v1" },
    { definitionId := lifecycleFactId, kind := .observation,
      canonicalBehavior := "temporal-nexus3-lifecycle-fact/v1" }
  ]
  lawWitnesses := [{ definition := lifecycleLaw, proof := lifecycleLawProof }]
}
def definitions : List DefinitionMetadata := [
  Authoring.metadata targetId .target "temporal-nexus3-target/v1",
  Authoring.metadata kernelId .kernel "temporal-nexus3-kernel/v1",
  Authoring.metadata lifecycleCapabilityId .capability "temporal-nexus3-success/v1",
  Authoring.metadata lifecycleProviderId .provider "temporal-nexus3-provider/v1",
  Authoring.metadata lifecycleLawId .law lifecycleLaw.body,
  Authoring.metadata operationStateId .state "temporal-nexus3-operation-state/v1",
  Authoring.metadata awaitStartActionId .action "temporal-nexus3-await-start/v1",
  Authoring.metadata awaitSuccessActionId .action "temporal-nexus3-await-success/v1",
  Authoring.metadata transitionOutcomeId .outcome "temporal-nexus3-transition-outcome/v1",
  Authoring.metadata lifecycleFactId .observation "temporal-nexus3-lifecycle-fact/v1"
]
def targetDefinition : FiniteTargetDefinition := {
  id := targetId
  source
  definitions
  requiredCapabilities := [lifecycleCapabilityId]
  metadata := { id := kernelId, source }
}
def targetComposition : TargetComposition LawStatement :=
  TargetComposition.empty |>.provide lifecycleProvider
structure ModelVocabulary where
  scheduledState : ModelValue
  startedState : ModelValue
  succeededState : ModelValue
  awaitStartAction : ModelValue
  awaitSuccessAction : ModelValue
  acknowledgedOutcome : ModelValue
  completedOutcome : ModelValue
  startedFact : ModelValue
  succeededFact : ModelValue
def modelVocabulary (authoredTable : FiniteTable Setup State Action Outcome Fact) :
    Except FiniteTableError ModelVocabulary := do
  let model ← authoredTable.validateModel identity
  pure {
    scheduledState := ← model.stateValue .scheduled
    startedState := ← model.stateValue .started
    succeededState := ← model.stateValue .succeeded
    awaitStartAction := ← model.actionValue .awaitStart
    awaitSuccessAction := ← model.actionValue .awaitSuccess
    acknowledgedOutcome := ← model.outcomeValue .acknowledged
    completedOutcome := ← model.outcomeValue .completed
    startedFact := ← model.factValue .started
    succeededFact := ← model.factValue .succeeded
  }
def propertySpec
    (_model : ModelVocabulary)
    (action state outcome fact : ModelValue)
    (override : Option String := none)
    (declarationName : Lean.Name := by exact decl_name%) : PropertySpec := {
  family
  key := Authoring.declarationKey override declarationName
  source
  requires := [lifecycleCapabilityId]
  clauses := transitionResultClauses family (Authoring.declarationKey override declarationName)
    action state outcome fact
}
/-- `awaitSuccess` must produce the Target-owned succeeded state, completed outcome, and fact. -/
def successfulResult (model : ModelVocabulary) (override : Option String := none) : PropertySpec :=
  propertySpec model model.awaitSuccessAction model.succeededState model.completedOutcome
    model.succeededFact override
/-- Select exactly the two waits from the scheduled setup. -/
def successfulCompletion (model : ModelVocabulary) (override : Option String := none) :
    ExactSequenceSpec := {
  family
  key := Authoring.declarationKey override (by exact decl_name%)
  source
  requires := [lifecycleCapabilityId]
  roles := [{ id := operationRoleId, valueKind := .state }]
  setup := [SetupConstraint.roleEquals (family.id "setup" "scheduled")
    operationRoleId model.scheduledState]
  occurrences := [
    { key := Authoring.declarationKey none ``Action.awaitStart,
      action := model.awaitStartAction.definitionId },
    { key := Authoring.declarationKey none ``Action.awaitSuccess,
      action := model.awaitSuccessAction.definitionId }
  ]
}
/-- Ask for one successful witness within the exact two-transition bound. -/
def completion
    (property : CheckedProperty)
    (behavior : CheckedBehavior)
    (override : Option String := none) : QuerySpec := {
  family
  key := Authoring.declarationKey override (by exact decl_name%)
  source
  target := targetId
  form := .witness property
  behavior
  limits := { transitions := 2, selectedActions := 2, candidateEvaluations := 16 }
  policy := .shortest
}
inductive AdmissionError where
  | invalidTarget (error : Authoring.FiniteAdmissionError)
  | invalidVocabulary (error : FiniteTableError)
  | invalidProperty (error : PropertyError)
  | invalidBehavior (error : BehaviorError)
  | invalidQuery (error : QueryError)
  | invalidPlanner (error : FinitePlannerAdmissionError)
  | invalidKnownGaps (error : KnownGapError)
  | noWitness
structure CheckedModel where
  target : QueryTarget LawStatement
  model : ModelVocabulary
  property : CheckedProperty
  behavior : CheckedBehavior
  query : CheckedQuery LawStatement
  kernel : IncrementalPlannerKernel query.target
  run : PlannerRun
  witness : BehaviorTrace
/-- Publish the model only after Target, Property, Behavior, Query, planner, and witness checks. -/
def check
    (authoredTable : FiniteTable Setup State Action Outcome Fact := table)
    (authoredDefinition : FiniteTargetDefinition := targetDefinition)
    (propertyAuthor : ModelVocabulary → PropertySpec := successfulResult)
    (behaviorAuthor : ModelVocabulary → ExactSequenceSpec := successfulCompletion)
    (queryAuthor : CheckedProperty → CheckedBehavior → QuerySpec := completion) :
    Except AdmissionError CheckedModel := do
  let target ← Authoring.checkFiniteTarget authoredTable table identity authoredDefinition
    targetComposition (· == State.succeeded) |>.mapError .invalidTarget
  let model ← modelVocabulary authoredTable |>.mapError .invalidVocabulary
  let property ← (propertyAuthor model).check (PropertyCheckContext.ofTarget target)
    |>.mapError .invalidProperty
  let behavior ← (behaviorAuthor model).check (.ofTarget target) |>.mapError .invalidBehavior
  let query ← (queryAuthor property behavior).check target |>.mapError .invalidQuery
  let kernel ← IncrementalPlannerKernel.ofCheckedQuery target.id query |>.mapError .invalidPlanner
  let run ← plan query kernel |>.mapError .invalidKnownGaps
  match run.result.outcome with
  | .found witness .satisfyingWitness =>
      pure { target, model, property, behavior, query, kernel, run, witness }
  | _ => throw .noWitness
end Temporal.Feature.Nexus3
