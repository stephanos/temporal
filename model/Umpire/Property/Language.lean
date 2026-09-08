import Umpire.Target.Semantics
import Umpire.Property.Fields

/-! Implementation behind the `Umpire.Property` public facade. -/

namespace Umpire

/-! Portable, pure properties over capability-limited Model Traces. -/

inductive PropertyTraceField where
  | state
  | priorState
  | resultingState
  | selectedAction
  | modelOutcome
  | observation
  | relation
  deriving BEq, DecidableEq, Ord, Repr

def PropertyTraceField.name : PropertyTraceField → String
  | .state => "state"
  | .priorState => "prior-state"
  | .resultingState => "resulting-state"
  | .selectedAction => "selected-action"
  | .modelOutcome => "model-outcome"
  | .observation => "observation"
  | .relation => "relation"

def PropertyTraceField.definitionKind : PropertyTraceField → DefinitionKind
  | .state | .priorState | .resultingState => .state
  | .selectedAction => .action
  | .modelOutcome => .outcome
  | .observation => .observation
  | .relation => .relation

inductive ValueConstraint where
  | present
  | equals (value : String)
  | notEquals (value : String)
  | naturalAtMost (value : Nat)
  | naturalAtLeast (value : Nat)
  deriving BEq, DecidableEq, Ord, Repr

structure PropertyPattern where
  field : PropertyTraceField
  reference : DefinitionId
  constraint : ValueConstraint := .present
  deriving BEq, DecidableEq, Ord, Repr

/-- Match one trace value by Definition ID and exact payload. -/
def PropertyPattern.exact
    (field : PropertyTraceField)
    (reference : DefinitionId)
    (value : String) : PropertyPattern := {
  field
  reference
  constraint := .equals value
}

/-- The two same-step environments supported by portable Boolean Property predicates. Guards read
the triggering prior state and selected Action; expectations read that transition's result. -/
inductive PropertyPredicateContext where
  | guard
  | expectation
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable diagnostic name for a predicate context. -/
def PropertyPredicateContext.name : PropertyPredicateContext → String
  | .guard => "guard"
  | .expectation => "expectation"

/-- A field in one explicit same-step predicate environment. -/
inductive PropertyPredicateField where
  | priorState
  | selectedAction
  | resultingState
  | modelOutcome
  | expectationFact
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable diagnostic name for a predicate field. -/
def PropertyPredicateField.name : PropertyPredicateField → String
  | .priorState => "prior-state"
  | .selectedAction => "selected-action"
  | .resultingState => "resulting-state"
  | .modelOutcome => "model-outcome"
  | .expectationFact => "expectation-fact"

/-- Definition kind required by a predicate field reference. -/
def PropertyPredicateField.definitionKind : PropertyPredicateField → DefinitionKind
  | .priorState | .resultingState => .state
  | .selectedAction => .action
  | .modelOutcome => .outcome
  | .expectationFact => .observation

/-- Whether a field belongs to the context at the same trace step. -/
def PropertyPredicateContext.allows
    (context : PropertyPredicateContext)
    (field : PropertyPredicateField) : Bool :=
  match context, field with
  | .guard, .priorState | .guard, .selectedAction => true
  | .expectation, .resultingState | .expectation, .modelOutcome
  | .expectation, .expectationFact => true
  | _, _ => false

/-- A typed literal in the closed portable Property predicate vocabulary. -/
inductive PropertyLiteral where
  | text (value : String)
  | natural (value : Nat)
  | boolean (value : Bool)
  deriving BEq, DecidableEq, Ord, Repr

/-- Runtime payload category selected by a typed predicate literal. -/
inductive PropertyLiteralType where
  | text
  | natural
  | boolean
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable diagnostic name for a predicate literal type. -/
def PropertyLiteralType.name : PropertyLiteralType → String
  | .text => "text"
  | .natural => "natural"
  | .boolean => "boolean"

/-- Return the payload category carried by a predicate literal. -/
def PropertyLiteral.type : PropertyLiteral → PropertyLiteralType
  | .text _ => .text
  | .natural _ => .natural
  | .boolean _ => .boolean

/-- The atomic comparisons supported by the portable Boolean kernel. `oneOf` compares several
typed literals with the atom's one field and reference, so it cannot become cross-field equality. -/
inductive PropertyAtomConstraint where
  | present
  | equals (value : PropertyLiteral)
  | oneOf (values : List PropertyLiteral)
  | fields (comparison : PropertyFieldComparison)
  deriving BEq, DecidableEq, Ord, Repr

/-- One typed comparison against a named value in a same-step predicate environment. -/
structure PropertyAtom where
  field : PropertyPredicateField
  reference : DefinitionId
  constraint : PropertyAtomConstraint := .present
  deriving BEq, DecidableEq, Ord, Repr

/-- The versioned field extension leaves all legacy atom constructor forms intact. -/
def PropertyAtom.fieldComparison (atom : PropertyAtom) : Option PropertyFieldComparison :=
  match atom.constraint with
  | .fields comparison => some comparison
  | _ => none

/-- Closed, serializable Boolean predicate data. Empty groups remain representable so checked
admission can return a source-owned diagnostic; callbacks and cross-field operands are absent. -/
inductive PropertyPredicate where
  | atom (value : PropertyAtom)
  | all (items : List PropertyPredicate)
  | any (items : List PropertyPredicate)
  | not (item : PropertyPredicate)
  deriving BEq, Repr

mutual
  private def PropertyPredicate.decEq
      (left right : PropertyPredicate) : Decidable (left = right) :=
    match left, right with
    | .atom first, .atom second =>
        match decEq first second with
        | isTrue equal => isTrue (by cases equal; rfl)
        | isFalse different => isFalse (by intro equal; cases equal; exact different rfl)
    | .all first, .all second | .any first, .any second =>
        match propertyPredicateListDecEq first second with
        | isTrue equal => isTrue (by cases equal; rfl)
        | isFalse different => isFalse (by intro equal; cases equal; exact different rfl)
    | .not first, .not second =>
        match PropertyPredicate.decEq first second with
        | isTrue equal => isTrue (by cases equal; rfl)
        | isFalse different => isFalse (by intro equal; cases equal; exact different rfl)
    | .atom _, .all _ | .atom _, .any _ | .atom _, .not _
    | .all _, .atom _ | .all _, .any _ | .all _, .not _
    | .any _, .atom _ | .any _, .all _ | .any _, .not _
    | .not _, .atom _ | .not _, .all _ | .not _, .any _ =>
        isFalse (by intro equal; cases equal)
  termination_by sizeOf left + sizeOf right

  private def propertyPredicateListDecEq
      (left right : List PropertyPredicate) : Decidable (left = right) :=
    match left, right with
    | [], [] => isTrue rfl
    | [], _ :: _ | _ :: _, [] => isFalse (by intro equal; cases equal)
    | first :: rest, second :: tail =>
        match PropertyPredicate.decEq first second, propertyPredicateListDecEq rest tail with
        | isTrue headEqual, isTrue tailEqual =>
            isTrue (by cases headEqual; cases tailEqual; rfl)
        | isFalse different, _ =>
            isFalse (by intro equal; cases equal; exact different rfl)
        | _, isFalse different =>
            isFalse (by intro equal; cases equal; exact different rfl)
  termination_by sizeOf left + sizeOf right
end

instance : DecidableEq PropertyPredicate := PropertyPredicate.decEq

/-- Raw values for one predicate coordinate. `none` means the producer cannot supply a complete
field; `some []` records the known absence of an expectation fact. -/
structure PropertyPredicateInput where
  context : PropertyPredicateContext
  priorState : Option ModelValue := none
  selectedAction : Option ModelValue := none
  resultingState : Option ModelValue := none
  modelOutcome : Option ModelValue := none
  facts : Option (List ModelValue) := none
  fieldValues : List PropertyFieldValue := []
  deriving BEq, DecidableEq, Repr

structure PropertyLimitProfile where
  id : DefinitionId
  source : SourceLocation
  limit : Limit
  deriving BEq, DecidableEq, Repr

/-- A named condition that removes one parent or case applicability context. -/
structure PropertyException where
  id : DefinitionId
  source : SourceLocation
  condition : PropertyPredicate
  deriving BEq, DecidableEq, Repr

/-- One same-step expectation with its own stable identity and author source. -/
structure PropertySameStepClause where
  id : DefinitionId
  source : SourceLocation
  expectation : PropertyPredicate
  deriving BEq, DecidableEq, Repr

inductive PropertyLimit where
  | exact (limit : Limit)
  | named (profile : DefinitionId) (expectedUnit : LimitUnit)
  deriving BEq, DecidableEq, Repr

/-- One legacy single-pattern bounded obligation owned by a named Property case. -/
inductive PropertyCaseTemporalClause where
  | eventuallyWithin
      (id : DefinitionId)
      (source : SourceLocation)
      (trigger response : PropertyPattern)
      (limit : PropertyLimit)
  | quiescentWithin
      (id : DefinitionId)
      (source : SourceLocation)
      (trigger forbidden : PropertyPattern)
      (limit : PropertyLimit)
  deriving BEq, DecidableEq, Repr

def PropertyCaseTemporalClause.id : PropertyCaseTemporalClause → DefinitionId
  | .eventuallyWithin id _ _ _ _ | .quiescentWithin id _ _ _ _ => id

def PropertyCaseTemporalClause.source : PropertyCaseTemporalClause → SourceLocation
  | .eventuallyWithin _ source _ _ _ | .quiescentWithin _ source _ _ _ => source

/-- One named branch in a same-step Property group. Matching branches are all applied. -/
structure PropertyCase where
  id : DefinitionId
  source : SourceLocation
  guard : PropertyPredicate
  exception : Option PropertyException := none
  clauses : List PropertySameStepClause
  temporalClauses : List PropertyCaseTemporalClause := []
  deriving BEq, DecidableEq, Repr

/-- A parent applicability condition and its explicitly checked named cases. -/
structure PropertyCaseGroup where
  id : DefinitionId
  source : SourceLocation
  guard : PropertyPredicate
  exception : Option PropertyException := none
  cases : List PropertyCase
  complete : Bool := false
  exclusive : Bool := false
  deriving BEq, DecidableEq, Repr

inductive PropertyClause where
  | stateInvariant (id : DefinitionId) (state : PropertyPattern)
  | transitionContract (id : DefinitionId) (precondition postcondition : PropertyPattern)
  | identityRelation (id : DefinitionId) (relation : PropertyPattern)
  | inputOutput (id : DefinitionId) (input output : PropertyPattern)
  | ordered
      (id : DefinitionId)
      (before after : PropertyPattern)
      (unit : LimitUnit := .semanticTransitions)
  | eventuallyWithin
      (id : DefinitionId)
      (trigger response : PropertyPattern)
      (limit : PropertyLimit)
  | quiescentWithin
      (id : DefinitionId)
      (trigger forbidden : PropertyPattern)
      (limit : PropertyLimit)
  | sameStepCases (group : PropertyCaseGroup)
  | guardedEventuallyWithin
      (id : DefinitionId)
      (source : SourceLocation)
      (guard : PropertyPredicate)
      (exception : Option PropertyException)
      (trigger response : PropertyPattern)
      (limit : PropertyLimit)
  | guardedQuiescentWithin
      (id : DefinitionId)
      (source : SourceLocation)
      (guard : PropertyPredicate)
      (exception : Option PropertyException)
      (trigger forbidden : PropertyPattern)
      (limit : PropertyLimit)
  deriving BEq, DecidableEq, Repr

def PropertyClause.id : PropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _
  | .quiescentWithin id _ _ _
  | .guardedEventuallyWithin id _ _ _ _ _ _
  | .guardedQuiescentWithin id _ _ _ _ _ _ => id
  | .sameStepCases group => group.id

/-- Semantic clocks are distinct from runtime and search-work limits. -/
inductive PropertyScopedClock where
  | operationTransitions
  deriving BEq, DecidableEq, Repr

/-- Closing an incomplete prefix does not invent missing deadline evidence. -/
inductive PropertyScopedEndpoint where
  | deliberatelyClosed
  | runtimePrefix
  deriving BEq, DecidableEq, Repr

/-- A bounded response captures one immutable operation key in declared execution scope. -/
structure PropertyScopedClause where
  id : DefinitionId
  source : SourceLocation
  trigger : PropertyPredicate
  response : PropertyPredicate
  scope : List DefinitionId
  key : DefinitionId
  clock : PropertyScopedClock
  bound : Nat
  endpoint : PropertyScopedEndpoint
  deriving BEq, DecidableEq, Repr

structure PropertyDeclaration where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  requires : List DefinitionId
  clauses : List PropertyClause
  scopedClauses : List PropertyScopedClause := []
  logicalTimeSource : Option DefinitionId := none
  documentation : String := ""
  deriving BEq, DecidableEq, Repr

/-- An opaque expert declaration is recognizable for rejection, but its callback never enters the
portable declaration, checked property, planner input, or artifact types. -/
inductive PropertyAuthoring where
  | portable (declaration : PropertyDeclaration)
  | opaque (id : DefinitionId) (source : SourceLocation)
  deriving BEq, DecidableEq, Repr

end Umpire
