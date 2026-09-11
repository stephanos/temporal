import Temporal.Shared
import Umpire.Search
import Umpire.Search.Branches
import Umpire.Model.Table
import Umpire.Property.Elab
import Umpire.Scenario.Elab
import Umpire.Query.Elab
import Umpire.Case.Producer

/-!
The Nexus success-slice construction and admission layer. `Syntax` emits ordinary declarations
that call this module, which owns all Umpire records and checked planning. Every model member is
held in an ordered list parallel to its name and Definition ID list, so the declared arity is the
declaration's, not this module's.
-/

namespace Temporal.Feature.Nexus.Success.Authoring

open Umpire

/-- Where one declaration comes from: the semantic family its Definition IDs hang off, and the file
that elaborated it. Both are derived per file by the commands -- the family from the enclosing
namespace, the source from the elaborating file -- so two Models that name the same declaration in
different files carry distinct Definition IDs and distinct Provenance sources. -/
structure Origin where
  family : DefinitionFamily
  source : SourceLocation
  deriving BEq, Repr

namespace Origin

/-- The origin of a declaration in `semanticFamily`, elaborated from `path`. -/
def of (semanticFamily path : String) : Origin := {
  family := Temporal.Shared.definitionFamily semanticFamily
  source := Temporal.Shared.sourceLocation path
}

def ownedId (origin : Origin) (kind owner member : String) : DefinitionId :=
  origin.family.id kind (owner ++ "." ++ member)

def metadata
    (origin : Origin)
    (id : DefinitionId)
    (kind : DefinitionKind) : DefinitionMetadata :=
  Temporal.Shared.definitionMetadata id kind origin.source id.value

end Origin

def meaning (id : DefinitionId) (kind : DefinitionKind) : Meaning := {
  definitionId := id
  kind
  behaviorVersion := id.value
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
  | finite (error : TableAdmissionError)

def checkFiniteTarget [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (table : FiniteTable Setup State Action Outcome Fact)
    (canonicalTable : FiniteTable Setup State Action Outcome Fact)
    (identity : FiniteModelIdentity Setup State Action Outcome Fact)
    (definition : TableModelSpec)
    (composition : Providers LawStatement)
    (terminal : State → Bool) : Except FiniteAdmissionError (QueryModel LawStatement) := do
  let _ ← table.validate
    |>.mapError (FiniteAdmissionError.finite ∘ TableAdmissionError.invalidTable)
  if table.transitions.any fun row => terminal row.source then
    throw .outgoingTerminalTransition
  if table ≠ canonicalTable then
    throw .noncanonicalTable
  table.checkModel identity definition composition |>.mapError .finite

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
    (law : Law) : Prop :=
  law.id = lawId ∧ law.body = lawId.value ∧
    satisfiesTransitionRequirement table.transitions required = true

/-- The declared success model, held as ordered member lists rather than fixed-arity fields. Every
member list is parallel to the matching name and Definition ID list. -/
structure SuccessModel (Setup State Action Outcome Fact : Type)
    [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact] where
  origin : Origin
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
  lawStatement : Law → Prop
  law : Law
  lawProof : lawStatement law
  composition : Providers lawStatement
  modelSpec : TableModelSpec

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
  else model.origin.ownedId "role" model.key spelling

/-- The capability a declaration naming this role requires, resolved the same way. -/
def roleCapability (model : SuccessModel Setup State Action Outcome Fact) (spelling : String) :
    DefinitionId :=
  if spelling == model.roleName then model.capabilityId
  else model.origin.ownedId "capability" model.key spelling

/-- The declared results of one transition row, by declaration position. -/
def resultsAt (model : SuccessModel Setup State Action Outcome Fact) (index : Nat) :
    List (Step State Outcome Fact) :=
  ((model.table.transitions[index]?).map (·.results)).getD []

end SuccessModel

/-- One declared transition result: the reached state, its Model Outcome, and the Facts it
records. -/
def step (outcome : Outcome) (state : State) (facts : List Fact) :
    Step State Outcome Fact := {
  outcome := outcome
  state := state
  facts := facts
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
    (origin : Origin)
    (names : SuccessModelNames)
    (setupValue : Setup)
    (states : List State)
    (actions : List Action)
    (outcomes : List Outcome)
    (facts : List Fact)
    (initial terminal : List State)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact))
    (lawProof :
      SuccessLawStatement (origin.ownedId "law" names.declaration "canonical-table")
        (successTable names setupValue states actions outcomes facts initial transitions)
        transitions
        { id := origin.ownedId "law" names.declaration "canonical-table",
          body := (origin.ownedId "law" names.declaration "canonical-table").value }) :
    SuccessModel Setup State Action Outcome Fact := by
  let ownerKey := names.declaration
  let targetId := origin.family.id "target" ownerKey
  let kernelId := origin.ownedId "kernel" ownerKey "planner"
  let capabilityId := origin.ownedId "capability" ownerKey "transitions"
  let providerId := origin.ownedId "provider" ownerKey "finite-table"
  let lawId := origin.ownedId "law" ownerKey "canonical-table"
  let operationRoleId := origin.ownedId "role" ownerKey names.roleName
  let stateIds := names.stateKeys.map (origin.ownedId "state" ownerKey)
  let actionIds := names.actionKeys.map (origin.ownedId "action" ownerKey)
  let outcomeIds := names.outcomeKeys.map (origin.ownedId "outcome" ownerKey)
  let factIds := names.factKeys.map (origin.ownedId "fact" ownerKey)
  let relationIds := transitions.map fun row => origin.ownedId "relation" ownerKey row.key
  let table := successTable names setupValue states actions outcomes facts initial transitions
  let identity : FiniteModelIdentity Setup State Action Outcome Fact := {
    setupBindings := fun _ => initial.map fun value => { roleId := operationRoleId, state := value }
    stateId := catalogId states stateIds
    actionId := catalogId actions actionIds
    outcomeId := catalogId outcomes outcomeIds
    factId := catalogId facts factIds
  }
  let lawStatement := SuccessLawStatement lawId table transitions
  let law : Law := { id := lawId, body := lawId.value }
  let contract : Capability := {
    id := capabilityId
    behaviorVersion := capabilityId.value
    requiredLaws := [law]
  }
  let meanings :=
    table.states.map (fun entry => meaning (identity.stateId entry.value) .state) ++
    table.actions.map (fun entry => meaning (identity.actionId entry.value) .action) ++
    table.outcomes.map (fun entry => meaning (identity.outcomeId entry.value) .outcome) ++
    table.facts.map (fun entry => meaning (identity.factId entry.value) .fact)
  let provider : Provider lawStatement := {
    id := providerId
    source := origin.source
    contract
    meanings
    lawProofs := [{ definition := law, proof := lawProof }]
  }
  let definitions :=
    [origin.metadata targetId .target, origin.metadata kernelId .machine,
      origin.metadata capabilityId .capability,
      origin.metadata providerId .provider, origin.metadata lawId .law] ++
    (meanings.map fun provided => origin.metadata provided.definitionId provided.kind) ++
    table.transitions.map fun row =>
      origin.metadata (origin.ownedId "relation" ownerKey row.key) .relation
  let modelSpec : TableModelSpec := {
    id := targetId
    source := origin.source
    definitions
    requiredCapabilities := [capabilityId]
    metadata := { id := kernelId, source := origin.source }
  }
  exact {
    origin, key := ownerKey, roleName := names.roleName, setupValue,
    states, actions, outcomes, facts, initial, terminal,
    targetId, kernelId, capabilityId, providerId, lawId, operationRoleId,
    stateIds, actionIds, outcomeIds, factIds, relationIds,
    table, identity, lawStatement, law, lawProof,
    composition := Providers.empty |>.provide provider, modelSpec
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
  let checked ← (table.validate.map (·.withIdentity model.identity))
  pure {
    states := ← model.states.mapM checked.stateValue
    actions := ← model.actions.mapM checked.actionValue
    outcomes := ← model.outcomes.mapM checked.outcomeValue
    facts := ← model.facts.mapM checked.factValue
  }

def authoredProperty [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : SuccessPropertyNames) : Property := {
  id := model.origin.family.id "property" names.declaration
  source := model.origin.source
  requires := [model.roleCapability names.roleName]
  clauses :=
    let selected := PropertyPattern.selectedAction (values.namedAction names.actionSpelling)
    names.requirements.map fun requirement =>
    match requirement with
    | .stateClause label spelling =>
        .transitionContract (model.origin.ownedId "property" names.declaration label) selected
          (.resultingState (values.namedState spelling))
    | .outcomeClause label spelling =>
        .transitionContract (model.origin.ownedId "property" names.declaration label) selected
          (.outcome (values.namedOutcome spelling))
    | .factClause label spelling =>
        .inputOutput (model.origin.ownedId "property" names.declaration label) selected
          (.fact (values.namedFact spelling))
}

def authoredScenario [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : SuccessBehaviorNames) : Scenario :=
  Scenario.exactly
    (family := model.origin.family)
    (key := names.declaration)
    (source := model.origin.source)
    (requires := [model.roleCapability names.roleName])
    (roles := [{ id := model.namedRole names.roleName, valueKind := .state }])
    (setup := [SetupConstraint.roleEquals
      (model.origin.ownedId "setup" names.declaration names.roleName)
      (model.namedRole names.roleName) (values.namedState names.setupState)])
    (occurrences := names.occurrences.map fun occurrence =>
      { key := names.declaration ++ "." ++ occurrence.1,
        action := (values.namedAction occurrence.2).definitionId })

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
    (results : List (Step State Outcome Fact)) :
    FiniteTransitionRow State Action Outcome Fact :=
  { key, source, action := selectedAction, results }

def withOccurrences (spec : Scenario) (occurrences : List Scenario.Step) : Scenario :=
  spec.withSteps occurrences

def Origin.occurrence (origin : Origin) (key : String) (selectedAction : DefinitionId) :
    Scenario.Step :=
  { id := origin.family.id "occurrence" key, action := selectedAction }

def withClauses (spec : Property) (clauses : List PropertyClause) : Property :=
  { spec with clauses }

def reorderedAndDocumented (spec : Property) (documentation : String) : Property :=
  { spec with clauses := spec.clauses.reverse, documentation }

def cancellationKnownGap : KnownGap := {
  kind := .capability
  code := DefinitionId.of "temporal.nexus.success.known-gap.cancellation"
  subject := some (DefinitionId.of "temporal.nexus.success.property.cancellationResolves")
  detail := some "Operation-correlated Nexus cancellation is unsupported by the success slice."
}

def operationCorrelatedProgressKnownGap : KnownGap := {
  kind := .capability
  code := DefinitionId.of "temporal.nexus.success.known-gap.operation-correlated-progress"
  subject := some (DefinitionId.of "temporal.nexus.success.property.cancellationResolves")
  detail := some "Operation-correlated progress counting is unsupported by the success slice."
}

def completionKnownGaps : Except KnownGapError KnownGapSet :=
  KnownGapSet.checkCanonical [cancellationKnownGap, operationCorrelatedProgressKnownGap]

inductive AdmissionError where
  | invalidTarget (error : FiniteAdmissionError)
  | invalidVocabulary (error : FiniteTableError)
  | invalidProperty (error : PropertyError)
  | invalidBehavior (error : ScenarioError)
  | invalidKnownGaps (error : KnownGapError)
  | invalidQuery (error : QueryError)
  | invalidPlanner (error : FiniteSearchAdmissionError)
  /-- Planning ran but did not deliver what the Query form claimed; the outcome says what it did
  deliver, so an unsatisfiable Behavior is distinguishable from an exhausted limit. -/
  | notSelected (outcome : PlanningOutcome)

/-- Which claim a Query makes: select one satisfying witness, or verify the requirement over every
trace the Behavior admits within the declared limits. -/
inductive QueryFormKind where
  | selectWitness
  | verifyClaim
  deriving BEq, DecidableEq, Repr

structure CheckedModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact) where
  target : QueryModel model.lawStatement
  vocabulary : ModelVocabulary
  property : CheckedProperty
  behavior : CheckedScenario
  query : CheckedQuery model.lawStatement
  kernel : SearchView query.target
  run : PlanResult
  /-- The selected trace, present only for a witness Query. A verify Query establishes its claim
  over every admitted trace and selects none. -/
  witness : Option Scenario.Trace

def check [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (model : SuccessModel Setup State Action Outcome Fact)
    (queryKey : String)
    (limits : Limits)
    (propertyAuthor : ModelVocabulary → Property)
    (behaviorAuthor : ModelVocabulary → Scenario)
    (form : QueryFormKind := .selectWitness)
    (authoredTable : FiniteTable Setup State Action Outcome Fact := model.table)
    (authoredDefinition : TableModelSpec := model.modelSpec) :
    Except AdmissionError (CheckedModel model) := do
  let target ← checkFiniteTarget authoredTable model.table model.identity authoredDefinition
    model.composition (fun value => model.terminal.contains value) |>.mapError .invalidTarget
  let vocabulary ← modelVocabulary model authoredTable |>.mapError .invalidVocabulary
  let property ← (propertyAuthor vocabulary).check (PropertyCheckContext.ofTarget target)
    |>.mapError .invalidProperty
  let behavior ← (behaviorAuthor vocabulary).check (.ofTarget target) |>.mapError .invalidBehavior
  let gaps ← completionKnownGaps |>.mapError .invalidKnownGaps
  let authoredQuery : Query := {
    id := model.origin.family.id "query" queryKey
    source := model.origin.source
    target := model.targetId
    form := match form with
      | .selectWitness => .find property
      | .verifyClaim => .verify property
    behavior
    limits
    policy := match form with
      | .selectWitness => .shortest
      | .verifyClaim => .exhaustive
    authoredKnownGaps := gaps
  }
  let query ← Query.check (.ofTarget target) authoredQuery |>.mapError .invalidQuery
  let kernel ← SearchView.ofCheckedQuery target.id query |>.mapError .invalidPlanner
  let run ← search query kernel |>.mapError .invalidKnownGaps
  match form, run.result.outcome with
  | .selectWitness, .found witness .satisfyingWitness =>
      pure { target, vocabulary, property, behavior, query, kernel, run, witness := some witness }
  | .verifyClaim, .verified =>
      pure { target, vocabulary, property, behavior, query, kernel, run, witness := none }
  | _, outcome => throw (.notSelected outcome)


/-! ### Producing a Case

A `.umpire` module may not import this namespace, so the conversion from the checked authoring
bundle to the Umpire-owned Producer input lives here. The `case` command emits one call to
`produceCase`; everything it decides -- the template, the fixture name, the evidence mapping -- is
an argument. -/

def producerInput [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : SuccessModel Setup State Action Outcome Fact}
    (checked : CheckedModel «model») :
    Umpire.Case.Producer.Input «model».lawStatement := {
  target := checked.target
  vocabulary := {
    «states» := checked.vocabulary.states
    «actions» := checked.vocabulary.actions
    «outcomes» := checked.vocabulary.outcomes
    «facts» := checked.vocabulary.facts }
  «property» := checked.property
  «scenario» := checked.behavior
  «witness» := checked.witness
  operationRole := «model».operationRoleId
  queryId := checked.query.id
  querySource := checked.query.source
  queryFingerprint := checked.query.behaviorFingerprint.render
  knownGaps := checked.query.authoredKnownGaps
  source := «model».origin.source }

/-- Lower one checked Model into a Case through a named realization. The checked values are carried,
never compared against an expected Model: a different Machine, Scenario, Query or Property produces
different Case bytes.

`required` names clauses the caller requires the Case to carry, beyond the ones the checked Property
already names. Coverage is always requested explicitly, never left to a default. -/
def produce [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : SuccessModel Setup State Action Outcome Fact}
    (checked : CheckedModel «model»)
    (identity : Umpire.Case.Producer.Identity)
    (realization : Umpire.Case.Producer.Realization)
    (evidence : Umpire.Case.Producer.Vocabulary → List Umpire.Case.Producer.EvidenceMapping)
    (required : List DefinitionId := []) :
    Except Umpire.Case.Compiler.Error temporal.server.api.testpilot.v1.Case :=
  let input := producerInput checked
  Umpire.Case.Producer.produce input identity realization (evidence input.vocabulary) required

/-- The same, starting from the Query's own admission result. A Model the Query did not admit
rejects as `checked-model` against the Case's own identity, because there is nothing else to name
at that point. -/
def produceCase [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : SuccessModel Setup State Action Outcome Fact}
    (admitted : Except AdmissionError (CheckedModel «model»))
    (identity : Umpire.Case.Producer.Identity)
    (realization : Umpire.Case.Producer.Realization)
    (evidence : Umpire.Case.Producer.Vocabulary → List Umpire.Case.Producer.EvidenceMapping)
    (required : List DefinitionId := []) :
    Except Umpire.Case.Compiler.Error temporal.server.api.testpilot.v1.Case := do
  let checked ← admitted.mapError fun _ => {
    sourceDefinitionId := identity.caseId
    source := «model».origin.source
    construct := "checked-model" }
  produce checked identity realization evidence required

end Temporal.Feature.Nexus.Success.Authoring
