import Temporal.Shared
import Umpire.Planning

/-!
The fixed Nexus3 success-slice construction and admission layer. `Syntax` emits ordinary
declarations that call this module, which owns all Umpire records and checked planning.
-/

namespace Temporal.Feature.Nexus3.Authoring

open Umpire

def family : DefinitionFamily := Temporal.Shared.definitionFamily "nexus3"

def source : SourceLocation :=
  Temporal.Shared.sourceLocation "Temporal/Feature/Nexus3/Nexus.lean"

def ownedId (kind owner member : String) : DefinitionId :=
  family.id kind (owner ++ "." ++ member)

def metadata
    (id : DefinitionId)
    (kind : DefinitionKind) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata id kind source id.value

def meaning (id : DefinitionId) (kind : DefinitionKind) : MeaningProvision := {
  definitionId := id
  kind
  canonicalBehavior := id.value
}

structure SuccessModelNames where
  declaration : String
  roleName : String
  setup : String
  scheduledState : String
  startedState : String
  succeededState : String
  awaitStartAction : String
  awaitSuccessAction : String
  acknowledgedOutcome : String
  completedOutcome : String
  startedFact : String
  succeededFact : String
  startRelation : String
  successRelation : String

inductive FiniteAdmissionError where
  | outgoingTerminalTransition
  | noncanonicalTable
  | finite (error : FiniteTargetAdmissionError)

def checkFiniteTarget [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (canonicalTable : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (definition : FiniteTargetDefinition)
    (composition : TargetComposition LawStatement)
    (terminal : State → Bool) : Except FiniteAdmissionError (QueryTarget LawStatement) := do
  let _ ← table.validateModel identity
    |>.mapError (FiniteAdmissionError.finite ∘ FiniteTargetAdmissionError.invalidTable)
  if table.transitions.any fun row => terminal row.source then
    throw .outgoingTerminalTransition
  if table ≠ canonicalTable then
    throw .noncanonicalTable
  table.checkModelTarget identity definition composition |>.mapError .finite

def hasExactTransition [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (source : State)
    (action : Action)
    (result : TransitionResult State Outcome Fact) : Bool :=
  rows.any fun row => row.source == source && row.action == action && row.results == [result]

def satisfiesSuccessRequirement [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (rows : List (FiniteTransitionRow State Action Outcome Fact))
    (scheduled started : State)
    (awaitStart awaitSuccess : Action)
    (startedResult succeededResult : TransitionResult State Outcome Fact) : Bool :=
  hasExactTransition rows scheduled awaitStart startedResult &&
    hasExactTransition rows started awaitSuccess succeededResult

def SuccessLawStatement [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (lawId : DefinitionId)
    (table : FiniteTable Setup State Action Outcome Fact)
    (scheduled started : State)
    (awaitStart awaitSuccess : Action)
    (startedResult succeededResult : TransitionResult State Outcome Fact)
    (law : LawDefinition) : Prop :=
  law.id = lawId ∧ law.body = lawId.value ∧
    satisfiesSuccessRequirement table.transitions scheduled started awaitStart awaitSuccess
      startedResult succeededResult = true

structure SuccessModel (Setup State Action Outcome Fact : Type)
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact] where
  key : String
  scheduledSetup : Setup
  scheduledState : State
  startedState : State
  succeededState : State
  awaitStartAction : Action
  awaitSuccessAction : Action
  acknowledgedOutcome : Outcome
  completedOutcome : Outcome
  startedFact : Fact
  succeededFact : Fact
  targetId : DefinitionId
  kernelId : DefinitionId
  capabilityId : DefinitionId
  providerId : DefinitionId
  lawId : DefinitionId
  operationRoleId : DefinitionId
  scheduledStateId : DefinitionId
  startedStateId : DefinitionId
  succeededStateId : DefinitionId
  awaitStartActionId : DefinitionId
  awaitSuccessActionId : DefinitionId
  acknowledgedOutcomeId : DefinitionId
  completedOutcomeId : DefinitionId
  startedFactId : DefinitionId
  succeededFactId : DefinitionId
  startRelationId : DefinitionId
  successRelationId : DefinitionId
  startedResult : TransitionResult State Outcome Fact
  succeededResult : TransitionResult State Outcome Fact
  table : FiniteTable Setup State Action Outcome Fact
  identity : FiniteModelIdentity Setup State Action Outcome Fact
  lawStatement : LawDefinition → Prop
  law : LawDefinition
  lawProof : lawStatement law
  composition : TargetComposition lawStatement
  targetDefinition : FiniteTargetDefinition

def successResult (outcome : Outcome) (state : State) (fact : Fact) :
    TransitionResult State Outcome Fact := {
  modelOutcome := outcome
  resultingState := state
  observations := [fact]
}

def successTable
    (names : SuccessModelNames)
    (scheduledSetup : Setup)
    (scheduledState startedState succeededState : State)
    (awaitStartAction awaitSuccessAction : Action)
    (acknowledgedOutcome completedOutcome : Outcome)
    (startedFact succeededFact : Fact) : FiniteTable Setup State Action Outcome Fact := {
  setups := [{ value := scheduledSetup, key := names.setup }]
  states := [
    { value := scheduledState, key := names.scheduledState },
    { value := startedState, key := names.startedState },
    { value := succeededState, key := names.succeededState }
  ]
  actions := [
    { value := awaitStartAction, key := names.awaitStartAction },
    { value := awaitSuccessAction, key := names.awaitSuccessAction }
  ]
  outcomes := [
    { value := acknowledgedOutcome, key := names.acknowledgedOutcome },
    { value := completedOutcome, key := names.completedOutcome }
  ]
  facts := [
    { value := startedFact, key := names.startedFact },
    { value := succeededFact, key := names.succeededFact }
  ]
  initial := [{ setup := scheduledSetup, states := [scheduledState] }]
  transitions := [
    { key := names.startRelation, source := scheduledState, action := awaitStartAction,
      results := [successResult acknowledgedOutcome startedState startedFact] },
    { key := names.successRelation, source := startedState, action := awaitSuccessAction,
      results := [successResult completedOutcome succeededState succeededFact] }
  ]
}

def successModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (names : SuccessModelNames)
    (scheduledSetup : Setup)
    (scheduledState startedState succeededState : State)
    (awaitStartAction awaitSuccessAction : Action)
    (acknowledgedOutcome completedOutcome : Outcome)
    (startedFact succeededFact : Fact)
    (lawProof :
      SuccessLawStatement (ownedId "law" names.declaration "canonical-table")
        (successTable names scheduledSetup scheduledState startedState succeededState
          awaitStartAction awaitSuccessAction acknowledgedOutcome completedOutcome
          startedFact succeededFact)
        scheduledState startedState awaitStartAction awaitSuccessAction
        (successResult acknowledgedOutcome startedState startedFact)
        (successResult completedOutcome succeededState succeededFact)
        { id := ownedId "law" names.declaration "canonical-table",
          body := (ownedId "law" names.declaration "canonical-table").value }) :
    SuccessModel Setup State Action Outcome Fact := by
  let ownerKey := names.declaration
  let targetId := family.id "target" ownerKey
  let kernelId := ownedId "kernel" ownerKey "planner"
  let capabilityId := ownedId "capability" ownerKey "transitions"
  let providerId := ownedId "provider" ownerKey "finite-table"
  let lawId := ownedId "law" ownerKey "canonical-table"
  let operationRoleId := ownedId "role" ownerKey names.roleName
  let scheduledStateId := ownedId "state" ownerKey names.scheduledState
  let startedStateId := ownedId "state" ownerKey names.startedState
  let succeededStateId := ownedId "state" ownerKey names.succeededState
  let awaitStartActionId := ownedId "action" ownerKey names.awaitStartAction
  let awaitSuccessActionId := ownedId "action" ownerKey names.awaitSuccessAction
  let acknowledgedOutcomeId := ownedId "outcome" ownerKey names.acknowledgedOutcome
  let completedOutcomeId := ownedId "outcome" ownerKey names.completedOutcome
  let startedFactId := ownedId "fact" ownerKey names.startedFact
  let succeededFactId := ownedId "fact" ownerKey names.succeededFact
  let startRelationId := ownedId "relation" ownerKey names.startRelation
  let successRelationId := ownedId "relation" ownerKey names.successRelation
  let startedResult := successResult acknowledgedOutcome startedState startedFact
  let succeededResult := successResult completedOutcome succeededState succeededFact
  let table := successTable names scheduledSetup scheduledState startedState succeededState
    awaitStartAction awaitSuccessAction acknowledgedOutcome completedOutcome startedFact succeededFact
  let identity : FiniteModelIdentity Setup State Action Outcome Fact := {
    setupBindings := fun _ => [{ roleId := operationRoleId, state := scheduledState }]
    stateId := fun state =>
      if state == scheduledState then scheduledStateId
      else if state == startedState then startedStateId else succeededStateId
    actionId := fun action => if action == awaitStartAction then awaitStartActionId
      else awaitSuccessActionId
    outcomeId := fun outcome => if outcome == acknowledgedOutcome then acknowledgedOutcomeId
      else completedOutcomeId
    factId := fun fact => if fact == startedFact then startedFactId else succeededFactId
  }
  let lawStatement := SuccessLawStatement lawId table scheduledState startedState
    awaitStartAction awaitSuccessAction startedResult succeededResult
  let law : LawDefinition := { id := lawId, body := lawId.value }
  let contract : CapabilityContract := {
    id := capabilityId
    canonicalBehavior := capabilityId.value
    requiredLaws := [law]
  }
  let meanings :=
    table.states.map (fun entry => meaning (identity.stateId entry.value) .state) ++
    table.actions.map (fun entry => meaning (identity.actionId entry.value) .action) ++
    table.outcomes.map (fun entry => meaning (identity.outcomeId entry.value) .outcome) ++
    table.facts.map (fun entry => meaning (identity.factId entry.value) .observation)
  let provider : CapabilityProvider lawStatement := {
    id := providerId
    source
    contract
    meanings
    lawWitnesses := [{ definition := law, proof := lawProof }]
  }
  let definitions :=
    [metadata targetId .target, metadata kernelId .kernel, metadata capabilityId .capability,
      metadata providerId .provider, metadata lawId .law] ++
    (meanings.map fun provided => metadata provided.definitionId provided.kind) ++
    table.transitions.map fun row => metadata (ownedId "relation" ownerKey row.key) .relation
  let targetDefinition : FiniteTargetDefinition := {
    id := targetId
    source
    definitions
    requiredCapabilities := [capabilityId]
    metadata := { id := kernelId, source }
  }
  exact {
    key := ownerKey, scheduledSetup, scheduledState, startedState, succeededState,
    awaitStartAction,
    awaitSuccessAction, acknowledgedOutcome, completedOutcome, startedFact, succeededFact,
    targetId, kernelId, capabilityId, providerId, lawId, operationRoleId, scheduledStateId,
    startedStateId, succeededStateId, awaitStartActionId, awaitSuccessActionId,
    acknowledgedOutcomeId, completedOutcomeId, startedFactId, succeededFactId, startRelationId,
    successRelationId, startedResult, succeededResult, table, identity, lawStatement, law,
    lawProof, composition := TargetComposition.empty |>.provide provider, targetDefinition
  }

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

structure SuccessPropertyNames where
  declaration : String
  stateClause : String
  outcomeClause : String
  factClause : String

structure SuccessBehaviorNames where
  declaration : String
  roleName : String
  startOccurrence : String
  completionOccurrence : String

def modelVocabulary [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (table : FiniteTable Setup State Action Outcome Fact) : Except FiniteTableError ModelVocabulary := do
  let checked ← table.validateModel model.identity
  pure {
    scheduledState := ← checked.stateValue model.scheduledState
    startedState := ← checked.stateValue model.startedState
    succeededState := ← checked.stateValue model.succeededState
    awaitStartAction := ← checked.actionValue model.awaitStartAction
    awaitSuccessAction := ← checked.actionValue model.awaitSuccessAction
    acknowledgedOutcome := ← checked.outcomeValue model.acknowledgedOutcome
    completedOutcome := ← checked.outcomeValue model.completedOutcome
    startedFact := ← checked.factValue model.startedFact
    succeededFact := ← checked.factValue model.succeededFact
  }

def propertySpec [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : SuccessPropertyNames) : PropertySpec := {
  family
  key := names.declaration
  source
  requires := [model.capabilityId]
  clauses := [
    .transitionContract (ownedId "property" names.declaration names.stateClause)
      (.selectedAction values.awaitSuccessAction) (.resultingState values.succeededState),
    .transitionContract (ownedId "property" names.declaration names.outcomeClause)
      (.selectedAction values.awaitSuccessAction) (.modelOutcome values.completedOutcome),
    .inputOutput (ownedId "property" names.declaration names.factClause)
      (.selectedAction values.awaitSuccessAction) (.fact values.succeededFact)
  ]
}

def behaviorSpec [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : SuccessBehaviorNames) : ExactSequenceSpec := {
  family
  key := names.declaration
  source
  requires := [model.capabilityId]
  roles := [{ id := model.operationRoleId, valueKind := .state }]
  setup := [SetupConstraint.roleEquals (ownedId "setup" names.declaration names.roleName)
    model.operationRoleId values.scheduledState]
  occurrences := [
    { key := names.declaration ++ "." ++ names.startOccurrence,
      action := values.awaitStartAction.definitionId },
    { key := names.declaration ++ "." ++ names.completionOccurrence,
      action := values.awaitSuccessAction.definitionId }
  ]
}

def withStates
    (table : FiniteTable Setup State Action Outcome Fact)
    (states : List (FiniteCatalogEntry State)) : FiniteTable Setup State Action Outcome Fact :=
  { table with states }

def withTransitions
    (table : FiniteTable Setup State Action Outcome Fact)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact)) :
    FiniteTable Setup State Action Outcome Fact :=
  { table with transitions }

def transitionRow
    (key : String)
    (source : State)
    (selectedAction : Action)
    (results : List (TransitionResult State Outcome Fact)) :
    FiniteTransitionRow State Action Outcome Fact :=
  { key, source, action := selectedAction, results }

def withOccurrences (spec : ExactSequenceSpec) (occurrences : List SequenceOccurrence) :
    ExactSequenceSpec :=
  { spec with occurrences }

def occurrence (key : String) (selectedAction : DefinitionId) : SequenceOccurrence :=
  { key, action := selectedAction }

def withClauses (spec : PropertySpec) (clauses : List PropertyClause) : PropertySpec :=
  { spec with clauses }

def reorderedAndDocumented (spec : PropertySpec) (documentation : String) : PropertySpec :=
  { spec with clauses := spec.clauses.reverse, documentation }

def cancellationKnownGap : KnownGap := {
  kind := .capabilityContract
  code := DefinitionId.of "temporal.nexus3.known-gap.cancellation"
  subject := some (DefinitionId.of "temporal.nexus3.property.cancellationResolves")
  detail := some "Operation-scoped Nexus cancellation is unsupported by the success slice."
}

def operationScopedProgressKnownGap : KnownGap := {
  kind := .capabilityContract
  code := DefinitionId.of "temporal.nexus3.known-gap.operation-scoped-progress"
  subject := some (DefinitionId.of "temporal.nexus3.property.cancellationResolves")
  detail := some "Operation-scoped progress counting is unsupported by the success slice."
}

def completionKnownGaps : Except KnownGapError KnownGapSet :=
  KnownGapSet.checkCanonical [cancellationKnownGap, operationScopedProgressKnownGap]

inductive AdmissionError where
  | invalidTarget (error : FiniteAdmissionError)
  | invalidVocabulary (error : FiniteTableError)
  | invalidProperty (error : PropertyError)
  | invalidBehavior (error : BehaviorError)
  | invalidKnownGaps (error : KnownGapError)
  | invalidQuery (error : QueryError)
  | invalidPlanner (error : FinitePlannerAdmissionError)
  | noWitness

structure CheckedModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact) where
  target : QueryTarget model.lawStatement
  vocabulary : ModelVocabulary
  property : CheckedProperty
  behavior : CheckedBehavior
  query : CheckedQuery model.lawStatement
  kernel : IncrementalPlannerKernel query.target
  run : PlannerRun
  witness : BehaviorTrace

def check [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (queryKey : String)
    (limits : QueryLimitSpec)
    (propertyAuthor : ModelVocabulary → PropertySpec)
    (behaviorAuthor : ModelVocabulary → ExactSequenceSpec)
    (authoredTable : FiniteTable Setup State Action Outcome Fact := model.table)
    (authoredDefinition : FiniteTargetDefinition := model.targetDefinition) :
    Except AdmissionError (CheckedModel model) := do
  let target ← checkFiniteTarget authoredTable model.table model.identity authoredDefinition
    model.composition (· == model.succeededState) |>.mapError .invalidTarget
  let vocabulary ← modelVocabulary model authoredTable |>.mapError .invalidVocabulary
  let property ← (propertyAuthor vocabulary).check (PropertyCheckContext.ofTarget target)
    |>.mapError .invalidProperty
  let behavior ← (behaviorAuthor vocabulary).check (.ofTarget target) |>.mapError .invalidBehavior
  let gaps ← completionKnownGaps |>.mapError .invalidKnownGaps
  let querySpec : QuerySpec := {
    family
    key := queryKey
    source
    target := model.targetId
    form := .witness property
    behavior
    limits
    policy := .shortest
    authoredKnownGaps := gaps
  }
  let query ← querySpec.check target |>.mapError .invalidQuery
  let kernel ← IncrementalPlannerKernel.ofCheckedQuery target.id query |>.mapError .invalidPlanner
  let run ← plan query kernel |>.mapError .invalidKnownGaps
  match run.result.outcome with
  | .found witness .satisfyingWitness =>
      pure { target, vocabulary, property, behavior, query, kernel, run, witness }
  | _ => throw .noWitness

end Temporal.Feature.Nexus3.Authoring
