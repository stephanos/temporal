import Umpire.Shared
import Umpire.Search
import Umpire.Search.Branches
import Umpire.Model.Table
import Umpire.Property.Elab
import Umpire.Scenario.Elab
import Umpire.Query.Elab
import Umpire.Case.Producer

/-!
# What a declared Model is, before any command

The construction and admission layer behind the Model commands. `Umpire.Command.Syntax` emits
ordinary declarations that call this module, which owns all Umpire records and checked planning.
Every member is held in an ordered list parallel to its name and Definition ID list, so the declared
arity is the declaration's, not this module's.

Nothing here names a feature. A declaration's semantic family and source come from its `Origin`,
which the commands derive from where it was written.
-/

namespace Umpire.Command

open Umpire

/-- Where one declaration comes from: the semantic family its Definition IDs hang off, and the file
that elaborated it. Both are derived per file by the commands -- the family from the enclosing
namespace, the source from the elaborating file -- so two Models that name the same declaration in
different files carry distinct Definition IDs and distinct Provenance sources. -/
structure Origin where
  family : DefinitionFamily
  source : SourceLocation
  deriving BEq, Repr

/-- Join a definition root to a semantic family. -/
private def qualify (root semanticFamily : String) : String :=
  if root.isEmpty then semanticFamily
  else if semanticFamily.isEmpty then root
  else root ++ "." ++ semanticFamily

namespace Origin

/-- The origin of a declaration whose Definition IDs hang off `root.semanticFamily`, elaborated
from `path`. An empty root leaves the family as the semantic family alone, which is what a
declaration outside any owning namespace gets. -/
def of (root semanticFamily path : String) : Origin := {
  family := { root := Umpire.Shared.definitionId (qualify root semanticFamily) }
  source := Umpire.Shared.sourceLocation path 1 1 "lean-model"
}

def ownedId (origin : Origin) (kind owner member : String) : DefinitionId :=
  origin.family.id kind (owner ++ "." ++ member)

def metadata
    (origin : Origin)
    (id : DefinitionId)
    (kind : DefinitionKind) : DefinitionMetadata :=
  Umpire.Shared.definitionMetadata id kind origin.source 1 id.value ""

end Origin

def meaning (id : DefinitionId) (kind : DefinitionKind) : Meaning := {
  definitionId := id
  kind
  behaviorVersion := id.value
}

/-- The ordered member names of one declared success model, in declaration order. -/
structure DeclaredNames where
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

/-- Every declared Step row must appear in the authored table exactly as declared. -/
def satisfiesTransitionRequirement [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (rows required : List (FiniteTransitionRow State Action Outcome Fact)) : Bool :=
  required.all fun declared => rows.any fun row =>
    row.source == declared.source && row.action == declared.action &&
      row.results == declared.results

def TableLawStatement [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (lawId : DefinitionId)
    (table : FiniteTable Setup State Action Outcome Fact)
    (required : List (FiniteTransitionRow State Action Outcome Fact))
    (law : Law) : Prop :=
  law.id = lawId ∧ law.body = lawId.value ∧
    satisfiesTransitionRequirement table.transitions required = true

/-- One declared Model, held as ordered member lists rather than fixed-arity fields. Every
member list is parallel to the matching name and Definition ID list. -/
structure DeclaredModel (Setup State Action Outcome Fact : Type)
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

namespace DeclaredModel

variable [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]

def stateIdAt (model : DeclaredModel Setup State Action Outcome Fact) (index : Nat) : DefinitionId :=
  (model.stateIds[index]?).getD unknownId

def actionIdAt (model : DeclaredModel Setup State Action Outcome Fact) (index : Nat) :
    DefinitionId :=
  (model.actionIds[index]?).getD unknownId




/-- The role Definition ID a declaration naming this role addresses. A role the model does not
declare yields an ID no Target provides, so the declaration is rejected at admission naming it. -/
def namedRole (model : DeclaredModel Setup State Action Outcome Fact) (spelling : String) :
    DefinitionId :=
  if spelling == model.roleName then model.operationRoleId
  else model.origin.ownedId "role" model.key spelling

/-- The capability a declaration naming this role requires, resolved the same way. -/
def roleCapability (model : DeclaredModel Setup State Action Outcome Fact) (spelling : String) :
    DefinitionId :=
  if spelling == model.roleName then model.capabilityId
  else model.origin.ownedId "capability" model.key spelling

/-- The declared results of one transition row, by declaration position. -/
def resultsAt (model : DeclaredModel Setup State Action Outcome Fact) (index : Nat) :
    List (Step State Outcome Fact) :=
  ((model.table.transitions[index]?).map (·.results)).getD []

end DeclaredModel

/-- One declared transition result: the reached state, its Model Outcome, and the Facts it
records. -/
def step (outcome : Outcome) (state : State) (facts : List Fact) :
    Step State Outcome Fact := {
  outcome := outcome
  state := state
  facts := facts
}

def declaredTable
    (names : DeclaredNames)
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

def declareModel [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (origin : Origin)
    (names : DeclaredNames)
    (setupValue : Setup)
    (states : List State)
    (actions : List Action)
    (outcomes : List Outcome)
    (facts : List Fact)
    (initial terminal : List State)
    (transitions : List (FiniteTransitionRow State Action Outcome Fact))
    (lawProof :
      TableLawStatement (origin.ownedId "law" names.declaration "canonical-table")
        (declaredTable names setupValue states actions outcomes facts initial transitions)
        transitions
        { id := origin.ownedId "law" names.declaration "canonical-table",
          body := (origin.ownedId "law" names.declaration "canonical-table").value }) :
    DeclaredModel Setup State Action Outcome Fact := by
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
  let table := declaredTable names setupValue states actions outcomes facts initial transitions
  let identity : FiniteModelIdentity Setup State Action Outcome Fact := {
    setupBindings := fun _ => initial.map fun value => { roleId := operationRoleId, state := value }
    stateId := catalogId states stateIds
    actionId := catalogId actions actionIds
    outcomeId := catalogId outcomes outcomeIds
    factId := catalogId facts factIds
  }
  let lawStatement := TableLawStatement lawId table transitions
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

/-- The checked member values of one Model, in declaration order. It is the Producer's own
vocabulary record: the Producer reads exactly these four lists, so a second copy of them would only
be a conversion waiting to drift. -/
abbrev ModelVocabulary := Umpire.Case.Producer.Vocabulary

/-- The Model Value an out-of-catalog member resolves to; a declared member never reaches it. -/
abbrev unknownValue := Umpire.Case.Producer.unknownValue


/-- One `require` clause: its label and the member spelling it selects. -/
inductive PropertyRequirement where
  | stateClause (label spelling : String)
  | outcomeClause (label spelling : String)
  | factClause (label spelling : String)

/-- The declared role, the Action every clause is about, and the ordered `require` clauses. -/
structure PropertyNames where
  declaration : String
  roleName : String
  actionSpelling : String
  requirements : List PropertyRequirement

/-- The setup state and the ordered occurrence labels with the Action each one selects. -/
structure ScenarioNames where
  declaration : String
  roleName : String
  setupState : String
  occurrences : List (String × String)

def modelVocabulary [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    [DecidableEq Setup] [DecidableEq State] [DecidableEq Action]
    [DecidableEq Outcome] [DecidableEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact)
    (table : FiniteTable Setup State Action Outcome Fact) : Except FiniteTableError ModelVocabulary := do
  let checked ← (table.validate.map (·.withIdentity model.identity))
  pure {
    states := ← model.states.mapM checked.stateValue
    actions := ← model.actions.mapM checked.actionValue
    outcomes := ← model.outcomes.mapM checked.outcomeValue
    facts := ← model.facts.mapM checked.factValue
  }

def authoredProperty [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    (model : DeclaredModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : PropertyNames) : Property := {
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
    (model : DeclaredModel Setup State Action Outcome Fact)
    (values : ModelVocabulary)
    (names : ScenarioNames) : Scenario :=
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
    (model : DeclaredModel Setup State Action Outcome Fact) where
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
    (model : DeclaredModel Setup State Action Outcome Fact)
    (queryKey : String)
    (limits : Limits)
    (propertyAuthor : ModelVocabulary → Property)
    (behaviorAuthor : ModelVocabulary → Scenario)
    (knownGaps : Except KnownGapError KnownGapSet := .ok KnownGapSet.empty)
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
  let gaps ← knownGaps.mapError .invalidKnownGaps
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

`producerInput` is what the checked declaration cannot be: the Producer needs the operation role and
the declaring file's source, which belong to the declaration rather than to the check, and it needs
the Query flattened to the three fields it actually reads. The vocabulary is shared outright.

The `case` command emits one call to `produceCase`; everything it decides -- the template, the
fixture name, the evidence mapping -- is an argument. -/

def producerInput [BEq Setup] [BEq State] [BEq Action] [BEq Outcome] [BEq Fact]
    {«model» : DeclaredModel Setup State Action Outcome Fact}
    (checked : CheckedModel «model») :
    Umpire.Case.Producer.Input «model».lawStatement := {
  target := checked.target
  vocabulary := checked.vocabulary
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
    {«model» : DeclaredModel Setup State Action Outcome Fact}
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
    {«model» : DeclaredModel Setup State Action Outcome Fact}
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

end Umpire.Command
