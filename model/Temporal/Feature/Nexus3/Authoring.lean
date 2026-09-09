import Temporal.Shared
import Umpire.Planning
import Umpire.Target.FiniteMachine
import Umpire.Property.Authoring
import Umpire.Behavior.Authoring
import Umpire.Query.Authoring

/-!
The Nexus3 success-slice construction and admission layer. `Syntax` emits ordinary declarations
that call this module, which owns all Umpire records and checked planning. Every model member is
held in an ordered list parallel to its name and Definition ID list, so the declared arity is the
declaration's, not this module's.
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

/-- The ordered member names of one declared success model, in declaration order. -/
structure SuccessModelNames where
  declaration : String
  roleName : String
  setup : String
  stateKeys : List String
  actionKeys : List String
  outcomeKeys : List String
  factKeys : List String

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

/-- Every declared transition row must appear in the authored table exactly as declared. -/
def satisfiesTransitionRequirement [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (rows required : List (FiniteTransitionRow State Action Outcome Fact)) : Bool :=
  required.all fun declared => rows.any fun row =>
    row.source == declared.source && row.action == declared.action &&
      row.results == declared.results

def SuccessLawStatement [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (lawId : DefinitionId)
    (table : FiniteTable Setup State Action Outcome Fact)
    (required : List (FiniteTransitionRow State Action Outcome Fact))
    (law : LawDefinition) : Prop :=
  law.id = lawId ∧ law.body = lawId.value ∧
    satisfiesTransitionRequirement table.transitions required = true

/-- The declared success model, held as ordered member lists rather than fixed-arity fields. Every
member list is parallel to the matching name and Definition ID list. -/
structure SuccessModel (Setup State Action Outcome Fact : Type)
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact] where
  key : String
  roleName : String
  setupValue : Setup
  states : List State
  actions : List Action
  outcomes : List Outcome
  facts : List Fact
  initial : List State
  terminal : List State
  targetId : DefinitionId
  kernelId : DefinitionId
  capabilityId : DefinitionId
  providerId : DefinitionId
  lawId : DefinitionId
  operationRoleId : DefinitionId
  stateIds : List DefinitionId
  actionIds : List DefinitionId
  outcomeIds : List DefinitionId
  factIds : List DefinitionId
  relationIds : List DefinitionId
  table : FiniteTable Setup State Action Outcome Fact
  identity : FiniteModelIdentity Setup State Action Outcome Fact
  lawStatement : LawDefinition → Prop
  law : LawDefinition
  lawProof : lawStatement law
  composition : TargetComposition lawStatement
  targetDefinition : FiniteTargetDefinition

/-- The Definition ID an out-of-catalog member resolves to; a declared member never reaches it. -/
def unknownId : DefinitionId := DefinitionId.of ""

/-- Resolve a member's Definition ID through its own catalog position. -/
def catalogId [BEq α] (values : List α) (ids : List DefinitionId) (value : α) : DefinitionId :=
  let fallback := (ids.getLast?).getD unknownId
  match values.findIdx? (· == value) with
  | some index => (ids[index]?).getD fallback
  | none => fallback

namespace SuccessModel

variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]

def stateIdAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) : DefinitionId :=
  (model.stateIds[index]?).getD unknownId

def actionIdAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) :
    DefinitionId :=
  (model.actionIds[index]?).getD unknownId

def outcomeIdAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) :
    DefinitionId :=
  (model.outcomeIds[index]?).getD unknownId

def factIdAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) : DefinitionId :=
  (model.factIds[index]?).getD unknownId

def relationIdAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) :
    DefinitionId :=
  (model.relationIds[index]?).getD unknownId

/-- The role Definition ID a declaration naming this role addresses. A role the model does not
declare yields an ID no Target provides, so the declaration is rejected at admission naming it. -/
def namedRole (model : SuccessModel Setup State Action Outcome Fact) (spelling : String) :
    DefinitionId :=
  if spelling == model.roleName then model.operationRoleId
  else ownedId "role" model.key spelling

/-- The capability a declaration naming this role requires, resolved the same way. -/
def roleCapability (model : SuccessModel Setup State Action Outcome Fact) (spelling : String) :
    DefinitionId :=
  if spelling == model.roleName then model.capabilityId
  else ownedId "capability" model.key spelling

/-- The declared results of one transition row, by declaration position. -/
def resultsAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) :
    List (TransitionResult State Outcome Fact) :=
  ((model.table.transitions[index]?).map (·.results)).getD []

end SuccessModel

/-- One declared transition result: the reached state, its Model Outcome, and the Facts it
records. -/
def transitionResult (outcome : Outcome) (state : State) (facts : List Fact) :
    TransitionResult State Outcome Fact := {
  modelOutcome := outcome
  resultingState := state
  observations := facts
}

def successTable
    (names : SuccessModelNames)
    (setupValue : Setup)
    (states : List State)
    (actions : List Action)
    (outcomes : List Outcome)
    (facts : List Fact)
    (initial : List State)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact)) :
    FiniteTable Setup State Action Outcome Fact := {
  setups := [{ value := setupValue, key := names.setup }]
  states := (states.zip names.stateKeys).map fun entry => { value := entry.1, key := entry.2 }
  actions := (actions.zip names.actionKeys).map fun entry => { value := entry.1, key := entry.2 }
  outcomes :=
    (outcomes.zip names.outcomeKeys).map fun entry => { value := entry.1, key := entry.2 }
  facts := (facts.zip names.factKeys).map fun entry => { value := entry.1, key := entry.2 }
  initial := [{ setup := setupValue, states := initial }]
  transitions
}

def successModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (names : SuccessModelNames)
    (setupValue : Setup)
    (states : List State)
    (actions : List Action)
    (outcomes : List Outcome)
    (facts : List Fact)
    (initial terminal : List State)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact))
    (lawProof :
      SuccessLawStatement (ownedId "law" names.declaration "canonical-table")
        (successTable names setupValue states actions outcomes facts initial transitions)
        transitions
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
  let stateIds := names.stateKeys.map (ownedId "state" ownerKey)
  let actionIds := names.actionKeys.map (ownedId "action" ownerKey)
  let outcomeIds := names.outcomeKeys.map (ownedId "outcome" ownerKey)
  let factIds := names.factKeys.map (ownedId "fact" ownerKey)
  let relationIds := transitions.map fun row => ownedId "relation" ownerKey row.key
  let table := successTable names setupValue states actions outcomes facts initial transitions
  let identity : FiniteModelIdentity Setup State Action Outcome Fact := {
    setupBindings := fun _ => initial.map fun value => { roleId := operationRoleId, state := value }
    stateId := catalogId states stateIds
    actionId := catalogId actions actionIds
    outcomeId := catalogId outcomes outcomeIds
    factId := catalogId facts factIds
  }
  let lawStatement := SuccessLawStatement lawId table transitions
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
    key := ownerKey, roleName := names.roleName, setupValue,
    states, actions, outcomes, facts, initial, terminal,
    targetId, kernelId, capabilityId, providerId, lawId, operationRoleId,
    stateIds, actionIds, outcomeIds, factIds, relationIds,
    table, identity, lawStatement, law, lawProof,
    composition := TargetComposition.empty |>.provide provider, targetDefinition
  }

/-- The checked member values of one model, in declaration order. -/
structure ModelVocabulary where
  states : List ModelValue
  actions : List ModelValue
  outcomes : List ModelValue
  facts : List ModelValue

/-- The Model Value an out-of-catalog member resolves to; a declared member never reaches it. -/
def unknownValue : ModelValue := ModelValue.named unknownId ""

namespace ModelVocabulary

def stateAt (values : ModelVocabulary) (index : Nat) : ModelValue :=
  (values.states[index]?).getD unknownValue

def actionAt (values : ModelVocabulary) (index : Nat) : ModelValue :=
  (values.actions[index]?).getD unknownValue

def outcomeAt (values : ModelVocabulary) (index : Nat) : ModelValue :=
  (values.outcomes[index]?).getD unknownValue

def factAt (values : ModelVocabulary) (index : Nat) : ModelValue :=
  (values.facts[index]?).getD unknownValue

/-! Selection by declared spelling. An unknown spelling resolves to the unknown Model Value, whose
Definition ID no Target provides, so the clause referencing it is rejected at Property admission. -/

def named (values : List ModelValue) (spelling : String) : ModelValue :=
  (values.find? (·.value == spelling)).getD unknownValue

def namedState (values : ModelVocabulary) (spelling : String) : ModelValue :=
  named values.states spelling
def namedAction (values : ModelVocabulary) (spelling : String) : ModelValue :=
  named values.actions spelling
def namedOutcome (values : ModelVocabulary) (spelling : String) : ModelValue :=
  named values.outcomes spelling
def namedFact (values : ModelVocabulary) (spelling : String) : ModelValue :=
  named values.facts spelling

end ModelVocabulary

/-- One `require` clause: its label and the member spelling it selects. -/
inductive PropertyRequirement where
  | stateClause (label spelling : String)
  | outcomeClause (label spelling : String)
  | factClause (label spelling : String)

/-- The declared role, the Action every clause is about, and the ordered `require` clauses. -/
structure SuccessPropertyNames where
  declaration : String
  roleName : String
  actionSpelling : String
  requirements : List PropertyRequirement

/-- The setup state and the ordered occurrence labels with the Action each one selects. -/
structure SuccessBehaviorNames where
  declaration : String
  roleName : String
  setupState : String
  occurrences : List (String × String)

def modelVocabulary [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (table : FiniteTable Setup State Action Outcome Fact) : Except FiniteTableError ModelVocabulary := do
  let checked ← table.validateModel model.identity
  pure {
    states := ← model.states.mapM checked.stateValue
    actions := ← model.actions.mapM checked.actionValue
    outcomes := ← model.outcomes.mapM checked.outcomeValue
    facts := ← model.facts.mapM checked.factValue
  }

def propertySpec [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : SuccessPropertyNames) : PropertySpec := {
  family
  key := names.declaration
  source
  requires := [model.roleCapability names.roleName]
  clauses := names.requirements.map fun requirement =>
    let selected := PropertyPattern.selectedAction (values.namedAction names.actionSpelling)
    match requirement with
    | .stateClause label spelling =>
        .transitionContract (ownedId "property" names.declaration label) selected
          (.resultingState (values.namedState spelling))
    | .outcomeClause label spelling =>
        .transitionContract (ownedId "property" names.declaration label) selected
          (.modelOutcome (values.namedOutcome spelling))
    | .factClause label spelling =>
        .inputOutput (ownedId "property" names.declaration label) selected
          (.fact (values.namedFact spelling))
}

def behaviorSpec [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : SuccessBehaviorNames) : ExactSequenceSpec := {
  family
  key := names.declaration
  source
  requires := [model.roleCapability names.roleName]
  roles := [{ id := model.namedRole names.roleName, valueKind := .state }]
  setup := [SetupConstraint.roleEquals (ownedId "setup" names.declaration names.roleName)
    (model.namedRole names.roleName) (values.namedState names.setupState)]
  occurrences := names.occurrences.map fun occurrence =>
    { key := names.declaration ++ "." ++ occurrence.1,
      action := (values.namedAction occurrence.2).definitionId }
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
    model.composition (fun value => model.terminal.contains value) |>.mapError .invalidTarget
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
