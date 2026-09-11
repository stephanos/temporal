import Umpire.Case.Compiler
import Umpire.Case.Correlated
import Umpire.Case.Projection.Coverage
import Umpire.Query

/-!
# The generic Case Producer

One checked Model, one selected witness, and one named realization become one Case. Everything the
Contract says is derived from the checked values:

* Each `require` clause of the checked Property becomes one operation-correlated bounded-response
  clause. The clause the Model wrote decides the clause the Case carries, so editing a `require`
  line changes the Case bytes rather than any file here.
* The Contract carries no monitor rule at all. Its whole content is the correlated capability that
  `Umpire.Case.Correlated.lower` produced, whose `Lowered` value is the correspondence certificate
  between the checked Model and the wire bytes.
* The evidence those clauses read is lifted out of the recorded history by a declaration on the
  realization's history read: one rule per admitted evidence kind, keyed by whatever path the
  realization says names the operation.

Nothing here names a protocol, a history attribute, or a role. Those live in the realization the
caller supplies, which is why this module holds under SCP-02 while producing Temporal Cases.

What still rejects, and why: a witness the Query did not select (a Case realizes one selected
trace, so a `verify`-form Query has none), an evidence line naming a kind the realization does not
admit, a selected Action with no evidence line, an evidence line for an Action the witness never
selects, a clause whose shape no correlated predicate can carry, and a requested clause the
lowering did not produce. None of those is waivable by a Known Gap.
-/

namespace Umpire.Case.Producer

open Umpire
open Umpire.Case.Compiler
open temporal.server.api.testpilot.v1

/-! ### The checked authoring bundle

The Producer never sees an authoring surface. A caller converts its own bundle into these
Umpire-owned values at the call site, because a `.umpire` module may not import a feature
namespace. -/

/-- The checked member values of one Model, in declaration order. -/
structure Vocabulary where
  states : List ModelValue
  actions : List ModelValue
  outcomes : List ModelValue
  facts : List ModelValue
  deriving BEq, Repr

/-- The Model Value an out-of-catalog member resolves to; a declared member never reaches it. -/
def unknownValue : ModelValue := ModelValue.named (DefinitionId.of "") ""

namespace Vocabulary

def stateAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.states[index]?).getD unknownValue

def actionAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.actions[index]?).getD unknownValue

def outcomeAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.outcomes[index]?).getD unknownValue

def factAt (values : Vocabulary) (index : Nat) : ModelValue :=
  (values.facts[index]?).getD unknownValue

/-! Selection by declared spelling. An unknown spelling resolves to the unknown Model Value, whose
Definition ID no Target provides, so the clause referencing it is rejected at admission. -/

def named (values : List ModelValue) (spelling : String) : ModelValue :=
  (values.find? (·.value == spelling)).getD unknownValue

def namedState (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.states spelling
def namedAction (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.actions spelling
def namedOutcome (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.outcomes spelling
def namedFact (values : Vocabulary) (spelling : String) : ModelValue :=
  named values.facts spelling

end Vocabulary

/-- Everything the Producer reads out of a checked authoring bundle. `witness` stays optional
because a Query that verifies rather than selects has none, and rejecting that at production is
what keeps the diagnostic on the Case rather than on the Query. -/
structure Input (LawStatement : Law → Prop) where
  target : QueryModel LawStatement
  vocabulary : Vocabulary
  property : CheckedProperty
  scenario : CheckedScenario
  witness : Option Scenario.Trace
  operationRole : DefinitionId
  queryId : DefinitionId
  querySource : SourceLocation
  queryFingerprint : String
  knownGaps : KnownGapSet
  /-- Where a rejection points when the construct it names is the Model's, not the Query's. -/
  source : SourceLocation

/-! ### Identity

`fixture` names the checked-in file, and every other identity derives from it. The defaults are the
derivation; a caller that must keep older bytes passes the field it needs explicitly. -/

structure Identity where
  caseId : String
  fixture : String
  programId : String := caseId ++ ".program"
  contractId : String := caseId ++ ".contract"
  /-- The one Run coordinate recorded history does not carry: every event a Case lifts belongs to
  the single Run it executes, so the Case declares that scope rather than reading it. -/
  runScope : String := fixture
  deriving BEq, Repr

/-- The Case ID a fixture name derives. -/
def Identity.ofFixture (fixture : String) : Identity :=
  { caseId := "temporal.case." ++ fixture, fixture }

/-! ### Realization

A realization is the Program a Case runs plus the coordinates a Contract needs to read it back. It
is a value, not syntax: adding a shape is one declaration in the owning feature namespace. -/

inductive HookPlacement where
  | before
  | after
  deriving BEq, DecidableEq, Repr

/-- A named point of a realization's Program a fault line may be placed against. -/
structure Hook where
  name : String
  instruction : InstructionRef

/-- What a realization knows about one admitted evidence kind. `eventKind` is the spelling an
author writes; everything else is how the realization reads that kind back out of a response. -/
structure EvidenceSource where
  eventKind : String
  attributesField : String
  operationKeyPath : FieldPath
  kindId : DefinitionId
  sourceId : DefinitionId

/-- One resolved (Action, source) pair: the Action a recorded event confirms, and how to read it. -/
structure EvidenceRule where
  action : ModelValue
  source : EvidenceSource

/-- One `evidence` line: the Action the Scenario selects, and the event kind that confirms it. -/
structure EvidenceMapping where
  action : ModelValue
  eventKind : String

/-- The two fault kinds a Scenario may declare. -/
inductive FaultKind where
  | workerStop
  | workerResume
  deriving BEq, DecidableEq, Repr

/-- One `fault` line of a Scenario, resolved against the realization's hooks. -/
structure FaultLine where
  kind : FaultKind
  hook : String
  placement : HookPlacement
  deriving BEq, DecidableEq, Repr

structure Realization where
  /-- The Program, as a function of the identity it carries and the evidence rules it must lift. -/
  program : Identity → List EvidenceRule → Program
  producerId : String
  producerVersion : String := "1"
  projectionId : DefinitionId
  /-- The Run coordinate the projection scopes evidence by. -/
  scopeField : DefinitionId
  /-- The coordinate that names one operation across every admitted evidence kind. -/
  operationKey : DefinitionId
  historyObservation : String
  correlatedObservation : String
  /-- The role a fault line's outage is injected against. -/
  taskQueueRole : String
  /-- The Contract rule the Producer adds to order a Scenario's `fault` lines. It is read only when
  the Scenario carries some, so a Case with no fault line never carries this ID. -/
  faultRuleId : String
  hooks : List Hook := []
  sources : List EvidenceSource
  contractLimits : ContractLimits
  projectionLimits : Case.Projection.Limits
  /-- Evaluation ceilings for the correlated consumer, separate from the semantic window. -/
  runLimits : Property.Correlated.Limits

/-! ### The derived correlated Property

A `require` clause says what must hold at the step that selects one Action. The checked Scenario
says where that Action sits in the operation's own sequence. Together they are a bounded response:
from the operation's first selected Action, the required value is due within exactly as many
semantic transitions as the Scenario places between them.

That is what makes the derived Contract discriminating rather than vacuous. A same-step clause
triggered on its own Action would answer satisfied for an operation that never reached the Action
at all, because nothing triggered; triggering on the operation's first Action instead leaves the
obligation open until the operation either reaches the required value or the window closes. -/

private def loweringError (source : SourceLocation) (definitionId construct : String) : Error := {
  sourceDefinitionId := definitionId
  source
  construct
}

/-- The predicate field one modeled trace field names in a same-step predicate environment. A trace
field with no same-step predicate is a clause this lowering cannot express. -/
private def predicateField : PropertyTraceField → Option PropertyPredicateField
  | .priorState => some .priorState
  | .selectedAction => some .selectedAction
  | .resultingState => some .resultingState
  | .outcome => some .outcome
  | .observation => some .expectationFact
  | .state | .relation => none

private def predicateOf (pattern : PropertyPattern) : Option PropertyPredicate := do
  let field ← predicateField pattern.field
  let constraint ← match pattern.constraint with
    | .present => some PropertyAtomConstraint.present
    | .equals value => some (.equals (.text value))
    | _ => none
  pure (.atom { field, reference := pattern.reference, constraint })

/-- Whether one modeled step already carries a pattern's value. -/
private def patternHolds
    (pattern : PropertyPattern)
    (step : ModelTraceStep ModelValue ModelValue ModelValue ModelValue) : Bool :=
  let carries := fun (value : ModelValue) =>
    pattern.reference == value.definitionId &&
      match pattern.constraint with
      | .present => true
      | .equals expected => expected == value.value
      | _ => false
  match pattern.field with
  | .selectedAction => carries step.selectedAction
  | .resultingState => carries step.state
  | .outcome => carries step.outcome
  | .observation => step.facts.any carries
  | _ => false

/-- One `require` clause as an operation-correlated clause, placed by the checked Scenario. A
same-step clause is the only form with a trigger and a response to carry across; a value constraint
the portable predicate vocabulary has no spelling for, a trigger that is not an Action, and an
Action the Scenario never selects each reject by clause name rather than being narrowed or guessed.

The window is inclusive of its trigger step, so a required value that the selected trace already
carries somewhere before the Action the clause names would answer the clause without that Action
ever being observed. That is the vacuity this trigger choice exists to avoid, so a Property whose
response holds earlier rejects rather than lowering a clause a shorter trace could satisfy. -/
private def scopedClauseOf
    (source : SourceLocation)
    (scopeField operationKey : DefinitionId)
    (occurrences : List DefinitionId) (opening : ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (clause : CheckedPropertyClause) : Except Error PropertyCorrelatedClause :=
  let unexpressible := fun construct =>
    Except.error (loweringError source clause.id.value construct)
  match clause with
  | .transitionContract id trigger response
  | .inputOutput id trigger response =>
      if trigger.field != .selectedAction then
        unexpressible "property.clause-shape"
      else
        match occurrences.idxOf? trigger.reference, predicateOf response with
        | some bound, some lowered =>
            if (steps.take bound).any (patternHolds response) then
              unexpressible "property.clause-early-response"
            else
              .ok {
                id, source
                trigger := .selectedActionIs opening, response := lowered
                scope := [scopeField], key := operationKey
                bound, ending := .«partial» }
        | none, _ => unexpressible "property.clause-occurrence"
        | _, none => unexpressible "property.clause-shape"
  | _ => unexpressible "property.clause-form"

/-! ### The declared projection

Which Step a recorded event confirms is read from the checked Machine along the witness trace, so
an author states the mapping once, in the `steps` block, rather than repeating the state, outcome
and facts beside every event kind. -/

private def stepOf
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (action : ModelValue) : Option (Step ModelValue ModelValue ModelValue) :=
  (steps.find? fun step => step.selectedAction == action).map fun step =>
    { «state» := step.state, «outcome» := step.outcome, «facts» := step.facts }

private def projectionDeclaration
    (realization : Realization)
    (rules : List (EvidenceRule × Step ModelValue ModelValue ModelValue)) :
    Case.Projection.Declaration ModelValue ModelValue ModelValue ModelValue := {
  id := realization.projectionId
  scopeFields := [realization.scopeField]
  operationField := realization.operationKey
  sources := (rules.map (·.1.source.sourceId)).eraseDups
  rules := rules.map fun entry =>
    { kind := entry.1.source.kindId
      meaning := .confirmed none [(entry.1.action, entry.2)] }
  «limits» := realization.projectionLimits }

/-! ### Evidence resolution

An `evidence` line names an event kind; the realization says which kinds it admits. A kind outside
that list, a selected Action with no line, and a line for an Action the witness never selects each
reject by name. -/

private def resolveEvidence
    (source : SourceLocation)
    (realization : Realization)
    (selected : List ModelValue)
    (steps : List (ModelTraceStep ModelValue ModelValue ModelValue ModelValue))
    (evidence : List EvidenceMapping) :
    Except Error (List (EvidenceRule × Step ModelValue ModelValue ModelValue)) := do
  let resolved ← evidence.mapM fun mapping => do
    let admitted ← match realization.sources.find? (·.eventKind == mapping.eventKind) with
      | some admitted => pure admitted
      | none => throw (loweringError source mapping.eventKind "evidence.kind-unknown")
    unless selected.any (· == mapping.action) do
      throw (loweringError source mapping.action.definitionId.value
        "evidence.action-unselected")
    match stepOf steps mapping.action with
    | some step => pure (({ action := mapping.action, source := admitted } : EvidenceRule), step)
    | none =>
        throw (loweringError source mapping.action.definitionId.value
          "evidence.action-unwitnessed")
  for action in selected do
    unless evidence.any (·.action == action) do
      throw (loweringError source action.definitionId.value "evidence.action-unmapped")
  pure resolved

/-! ### Production -/

/-- Lower one checked Model into a Case through a named realization. The checked values are
carried, never compared against an expected Model: a different Machine, Scenario, Query or Property
produces different Case bytes. Only a claim this Producer cannot realize rejects.

`required` names clauses the caller requires the Case to carry, beyond the ones the checked
Property already names. Coverage is always requested explicitly, never left to a default. -/
def produce {LawStatement : Law → Prop}
    (input : Input LawStatement)
    (identity : Identity)
    (realization : Realization)
    (evidence : List EvidenceMapping)
    (required : List DefinitionId := []) :
    Except Error temporal.server.api.testpilot.v1.Case := do
  let rejects := loweringError input.source
  -- A Case realizes one selected trace, so a Query that verifies rather than selects has no
  -- witness to realize and rejects here. A Known Gap does not admit it.
  let selected ← match input.witness with
    | some selected => pure selected
    | none => throw (rejects input.queryId.value "witness.absent")
  -- The operation's own sequence in trace order: what it does first, and where each later Action
  -- sits after it. Only an `exactly` Scenario fixes that order, and the derivation needs it, so a
  -- Scenario that only bounds occurrences rejects rather than being read in canonical key order.
  let occurrences ← match input.scenario.actionsExactly with
    | some occurrences => pure occurrences
    | none => throw (rejects input.scenario.id.value "behavior.sequence.absent")
  let selectedActions := occurrences.filterMap fun occurrence =>
    input.vocabulary.actions.find? fun value => value.definitionId == occurrence
  let opening ← match occurrences.head? with
    | some first =>
        match input.vocabulary.actions.find? fun value => value.definitionId == first with
        | some opening => pure opening
        | none => throw (rejects first.value "behavior.action.undeclared")
    | none => throw (rejects input.scenario.id.value "behavior.sequence.absent")
  let evidenceRules ← resolveEvidence input.source realization selectedActions
    selected.trace.steps evidence
  let correlatedRules ← input.property.clauses.mapM
    (scopedClauseOf input.source realization.scopeField realization.operationKey
      occurrences opening selected.trace.steps)
  if correlatedRules.isEmpty then
    throw (rejects input.property.id.value "property.clauses.absent")
  let correlatedProperty ← (Property.check (.ofTarget input.target) ({
      id := input.property.id
      source := input.property.source
      version := input.property.version
      requires := input.property.requires
      clauses := []
      correlatedRules })).mapError fun _ =>
    rejects input.property.id.value "property.correlated-admission"
  let compiled ← (Property.Correlated.compile input.target correlatedProperty
    [realization.scopeField] realization.operationKey realization.runLimits).mapError fun _ =>
    rejects input.property.id.value "property.correlated-compile"
  let setup : List RoleBinding :=
    [{ «role» := input.operationRole, value := input.vocabulary.stateAt 0 }]
  let plan ← (Case.Projection.check input.target
    (projectionDeclaration realization evidenceRules) setup
    (input.vocabulary.stateAt 0)).mapError fun _ =>
    rejects realization.projectionId.value "projection.admission"
  let lowered ← Umpire.Case.Correlated.lower plan compiled realization.correlatedObservation
    (Case.Projection.Coverage.empty plan)
  compile {
    version := { major := 1 }
    caseId := identity.caseId
    producerId := realization.producerId
    producerVersion := realization.producerVersion
    definitions := [
      { definitionId := input.target.id.value
        behaviorFingerprint := input.target.behaviorFingerprint.render, kind := .target },
      { definitionId := input.scenario.id.value
        behaviorFingerprint := input.scenario.behaviorFingerprint.render, kind := .«scenario» },
      { definitionId := input.queryId.value
        behaviorFingerprint := input.queryFingerprint, kind := .«query» },
      { definitionId := correlatedProperty.id.value
        behaviorFingerprint := correlatedProperty.behaviorFingerprint.render, kind := .«property» }]
    sources := [input.target.source, input.scenario.source, input.querySource,
      input.property.source]
    knownGaps := input.knownGaps.toProvenanceGaps
    program := realization.program identity (evidenceRules.map (·.1))
    contractId := identity.contractId
    properties := [lowered.contractLowering]
    contractLimits := realization.contractLimits
    -- Every clause the Model wrote must appear among the lowered ones, so a clause silently lost
    -- between the checked Property and the Contract rejects here, before any Driver I/O. A caller
    -- may name further clauses it requires; one this Case does not carry rejects the same way.
    coverage := { clauses := (correlatedRules.map (·.id) ++ required).eraseDups }
  }

end Umpire.Case.Producer
