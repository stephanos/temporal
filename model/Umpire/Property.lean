import Umpire.Value.Field
import Umpire.Operation.Action
import Umpire.Model.Check
import Umpire.Id

/-!
Closed same-step field operands, the portable Property language, and the ordinary-Lean
constructors authors use to build one. Structural paths retain the complete operation schema and
payload side; concrete scalars enter only through checked cursors. Literal admission preserves
integer kind and range, enum identity, and exact bytes. Floating-point evaluation remains
unsupported.
-/
namespace Umpire
open Operation Value Value.Encoding


/-- The five immutable same-step model payload coordinates. -/
inductive PropertyFieldRoot where
  | request | priorState | resultingState | outcome | event
  deriving BEq, DecidableEq, Repr

/-- The exact earlier occurrence a capture operand names: its declared capture and the ordinal of
the occurrence within the reading operation. Ordinals are assigned in admission order, so a later
occurrence is a new ordinal rather than a replacement of an earlier one. -/
structure PropertyFieldCaptureKey where
  name : DefinitionId
  ordinal : Nat
  deriving BEq, DecidableEq, Repr

/-- Structural coordinates for a field operand, independent of generated Lean spelling. A `capture`
key names one retained earlier occurrence instead of the operand's own step; same-step operands
leave it absent and keep their existing identity. -/
structure PropertyFieldPath where
  root : PropertyFieldRoot
  reference : DefinitionId
  schema : RpcSchema
  side : Side
  steps : List Field.Step
  type : Singular
  capture : Option PropertyFieldCaptureKey := none
  deriving BEq, DecidableEq, Repr

/-- An exact scalar literal or a schema-qualified selection with its own diagnostic source. -/
inductive PropertyFieldOperand where
  | literal (value : Scalar) (source : SourceLocation)
  | field (path : PropertyFieldPath) (source : SourceLocation)
  deriving BEq, DecidableEq, Repr

/-- Exact equality and declared integer ordering; unsupported scalar orders are rejected. -/
inductive PropertyFieldOperator where
  | equal | notEqual | less | atMost | greater | atLeast
  deriving BEq, DecidableEq, Repr

/-- Source-owned field admission failure. -/
structure PropertyFieldError where
  source : SourceLocation
  reason : String
  deriving BEq, DecidableEq, Repr

/-- Serializable comparison data; the Property checker rechecks its operands before evaluation. -/
structure PropertyFieldComparison where
  operator : PropertyFieldOperator
  left : PropertyFieldOperand
  right : PropertyFieldOperand
  source : SourceLocation
  deriving BEq, DecidableEq, Repr

/-- The source attached to the responsible operand. -/
def PropertyFieldOperand.source : PropertyFieldOperand → SourceLocation
  | .literal _ source | .field _ source => source

/-- Exact scalar type, with no coercion across integer kinds or enum declarations. -/
def PropertyFieldOperand.type : PropertyFieldOperand → Singular
  | .literal value _ => Field.scalarType value
  | .field path _ => path.type

private def scalarSupported : Singular → Bool
  | .boolean | .text | .bytes | .integer _ | .enumeration _ => true
  | _ => false

private def literalValid : Scalar → Bool
  | .integer kind value =>
    let low : Int := if kind.signed then -(Int.ofNat (2 ^ (kind.bits - 1))) else 0
    let high : Int := Int.ofNat (2 ^ (kind.bits - if kind.signed then 1 else 0)) - 1
    decide (low ≤ value ∧ value ≤ high)
  | .enumeration name value => !name.isEmpty && decide (-2147483648 ≤ value ∧ value ≤ 2147483647)
  | .floating _ _ => false
  | _ => true

/-- Check the declared comparison fragment without approximating any operand's value. -/
def PropertyFieldComparison.check (operator : PropertyFieldOperator)
    (left right : PropertyFieldOperand) (source : SourceLocation) :
    Except PropertyFieldError PropertyFieldComparison := do
  for operand in [left, right] do
    if !scalarSupported operand.type then
      throw ⟨operand.source, "unsupported scalar comparison type"⟩
    if let .literal value _ := operand then
      if !literalValid value then throw ⟨operand.source, "scalar kind or range mismatch"⟩
  if left.type != right.type then throw ⟨right.source, "incompatible operand types"⟩
  match operator, left.type with
  | .equal, _ | .notEqual, _ | _, .integer _ => pure ()
  | _, _ => throw ⟨source, "unsupported scalar ordering"⟩
  pure ⟨operator, left, right, source⟩

private def fieldStepData : Field.Step → Raw
  | .field containing number => sequence [.atom 0, textData containing, .atom number]
  | .establish => .atom 1
  | .select group => sequence [.atom 2, textData group]
  | .index index => sequence [.atom 3, .atom index]
  | .cardinality => .atom 4
  | .key key => sequence [.atom 5, literal key]
  | .present => .atom 6

private def fieldTypeData : Singular → Raw
  | .boolean => .atom 0
  | .text => .atom 1
  | .bytes => .atom 2
  | .integer kind => sequence [.atom 3, literal (.integer kind 0)]
  | .enumeration name => sequence [.atom 4, textData name]
  | .message name => sequence [.atom 5, textData name]
  | .floating double => sequence [.atom 6, literal (.boolean double)]
  | .unsupported reason => sequence [.atom 7, textData reason]

private def fieldRootData : PropertyFieldRoot → Nat
  | .request => 0
  | .priorState => 1
  | .resultingState => 2
  | .outcome => 3
  | .event => 4

/-- A same-step path keeps its existing encoding; a capture key is appended only when present, so
declarations written before keyed captures existed retain their exact canonical bytes. -/
private def fieldPathData (path : PropertyFieldPath) : List Raw :=
  [.atom (fieldRootData path.root), textData path.reference.value,
    Canonical.rpcSchema path.schema, .atom (if path.side == .request then 0 else 1),
    sequence (path.steps.map fieldStepData), fieldTypeData path.type] ++
    path.capture.toList.map fun key => sequence [textData key.name.value, .atom key.ordinal]

private def fieldOperandData : PropertyFieldOperand → Raw
  | .literal value _ => sequence [.atom 0, literal value]
  | .field path _ => sequence (.atom 1 :: fieldPathData path)

/-- Versioned structural identity of one operand path; diagnostic sources have no semantic meaning. -/
def PropertyFieldPath.canonical (path : PropertyFieldPath) : String :=
  "property-field-path/v1:" ++ Canonical.key (sequence (fieldPathData path))

/-- New comparison identity is versioned and exact; diagnostic sources have no semantic meaning. -/
def PropertyFieldComparison.canonical (comparison : PropertyFieldComparison) : String :=
  let operator := match comparison.operator with
    | .equal => 0 | .notEqual => 1 | .less => 2 | .atMost => 3 | .greater => 4 | .atLeast => 5
  "property-fields/v1:" ++ Canonical.key (sequence [.atom operator,
    fieldOperandData comparison.left, fieldOperandData comparison.right])

instance : Ord PropertyFieldComparison where
  compare a b := compare a.canonical b.canonical

/-- Mathematical comparison of exact scalar values, used only after operand admission. -/
def PropertyFieldOperator.matches (operator : PropertyFieldOperator) (left right : Scalar) : Bool :=
  match operator with
  | .equal => decide (left = right)
  | .notEqual => decide (left ≠ right)
  | .less | .atMost | .greater | .atLeast =>
    match left, right with
    | .integer first a, .integer second b => decide (first = second) && match operator with
      | .less => decide (a < b)
      | .atMost => decide (a ≤ b)
      | .greater => decide (b < a)
      | .atLeast => decide (b ≤ a)
      | _ => false
    | _, _ => false

/-- Independent propositional interpretation of exact scalar equality and integer ordering. -/
def PropertyFieldOperator.denotes (operator : PropertyFieldOperator) (left right : Scalar) : Prop :=
  match operator with
  | .equal => left = right
  | .notEqual => left ≠ right
  | .less | .atMost | .greater | .atLeast =>
    match left, right with
    | .integer first a, .integer second b => first = second ∧ match operator with
      | .less => a < b
      | .atMost => a ≤ b
      | .greater => b < a
      | .atLeast => b ≤ a
      | _ => False
    | _, _ => False

/-- Exact scalar execution agrees with its mathematical interpretation. -/
theorem PropertyFieldOperator.matches_agrees (operator : PropertyFieldOperator)
    (left right : Scalar) : operator.matches left right = true ↔ operator.denotes left right := by
  cases operator with
  | equal => simp [PropertyFieldOperator.matches, PropertyFieldOperator.denotes]
  | notEqual => simp [PropertyFieldOperator.matches, PropertyFieldOperator.denotes]
  | less | atMost | greater | atLeast =>
    cases left <;> cases right <;>
      simp [PropertyFieldOperator.matches, PropertyFieldOperator.denotes]

/-- Selected schema authority, constructed from a retained payload-indexed witness. -/
structure PropertyFieldBinding where
  private mk ::
  reference : DefinitionId
  schema : RpcSchema
  deriving BEq, DecidableEq, Repr

/-- Bind a model payload definition to the explicitly selected generated operation. -/
def PropertyFieldBinding.ofWitness (owner : RpcOwner) {Request Response : Type}
    (witness : owner.Witness Request Response) (reference : DefinitionId) : PropertyFieldBinding :=
  ⟨reference, owner.schema witness⟩

/-- A request binding takes its identity and generated signature from the checked Action template. -/
def PropertyFieldBinding.ofAction (template : ActionTemplate owner Request Response Failure) :
    PropertyFieldBinding :=
  ofWitness owner template.declaration.reference template.identity

/-- Construct structural operand coordinates from the selected checked cursor. -/
def PropertyFieldPath.ofCursor (root : PropertyFieldRoot) (reference : DefinitionId)
    (cursor : Field.Cursor owner witness payloadSide limits valueType card readiness) : PropertyFieldPath :=
  { root, reference, schema := owner.schema witness, side := payloadSide,
    steps := cursor.path, type := valueType }

private def fieldSchema (path : PropertyFieldPath) : Schema :=
  if path.side == .request then path.schema.request else path.schema.response

/-- Check every structural step, requiring presence facts from this Boolean branch before reads. -/
def PropertyFieldPath.validate (path : PropertyFieldPath) (facts : List PropertyFieldPath)
    (source : SourceLocation) : Except PropertyFieldError Unit := do
  let schema := fieldSchema path
  let mut type : Singular := .message schema.root
  let mut card : Cardinality := .singular
  let mut readiness : Field.Availability := .available
  let mut traversed : List Field.Step := []
  for step in path.steps do
    let fail reason := Except.error (PropertyFieldError.mk source reason)
    match step with
    | .field containing number =>
      if type != .message containing || card != .singular || readiness != .available then
        fail "field requires an available containing message"
      let some (_, field) := (Field.schemaFields schema).find? (fun item =>
        item.1 == containing && item.2.number == number)
        | fail "unknown containing schema or field"
      type := field.type
      card := field.cardinality
      readiness := Field.availability field
      if let .floating _ := type then fail "unsupported floating-point evaluation"
      if let .unsupported reason := type then fail ("unsupported " ++ reason)
    | .establish | .select _ =>
      match step, readiness with
      | .establish, .optional => pure ()
      | .select expected, .oneof actual =>
        if expected != actual then fail "oneof group mismatch"
      | _, _ => fail "presence or oneof selection mismatch"
      let presence := { path with steps := traversed ++ [.present], type := .boolean }
      if !facts.contains presence then fail "presence is not established in this Boolean branch"
      readiness := .available
    | .present =>
      if readiness == .available then fail "field has no optional presence"
      type := .boolean
      card := .singular
      readiness := .available
    | .index _ =>
      if card != .repeated || readiness != .available then fail "index requires an available repeated field"
      card := .singular
    | .key key =>
      let .map keyType := card | fail "lookup requires an available map"
      if readiness != .available || Field.scalarType key != keyType || !literalValid key then
        fail "map key type or range mismatch"
      card := .singular
      readiness := .optional
    | .cardinality => fail "unsupported natural cardinality operand"
    traversed := traversed ++ [step]
  if readiness != .available then throw ⟨source, "operand presence or oneof selection is not established"⟩
  if card != .singular || type != path.type then throw ⟨source, "field type or cardinality mismatch"⟩

/-- Every presence fact a value at these coordinates carries with it: one `present` path for each
`establish` or `select` step it traverses. A checked cursor reaches those steps only over a payload
that actually supplied the optional value or selected the oneof member, so an occurrence retained
under these coordinates had its presence decided at the step that admitted it rather than in the
Boolean branch that later reads it. -/
def PropertyFieldPath.retainedFacts (path : PropertyFieldPath) : List PropertyFieldPath :=
  let rec collect (traversed : List Field.Step) : List Field.Step → List PropertyFieldPath
    | [] => []
    | step :: rest =>
      let learned := match step with
        | .establish | .select _ => [{ path with steps := traversed ++ [.present], type := .boolean }]
        | _ => []
      learned ++ collect (traversed ++ [step]) rest
  collect [] path.steps

/-- Presence learned by a true atom is local to its enclosing conjunction. -/
def PropertyFieldComparison.established (comparison : PropertyFieldComparison) : List PropertyFieldPath :=
  let learn (operand : PropertyFieldOperand) (value : PropertyFieldOperand) :=
    match operand, value with
    | .field path _, .literal (.boolean expected) _ =>
      if path.steps.getLast? == some .present &&
          ((comparison.operator == .equal && expected) ||
            (comparison.operator == .notEqual && !expected)) then [path] else []
    | _, _ => []
  learn comparison.left comparison.right ++ learn comparison.right comparison.left


/-! Portable, pure properties over capability-limited Model Traces. -/


inductive PropertyTraceField where
  | state
  | priorState
  | resultingState
  | selectedAction
  | outcome
  | observation
  | relation
  deriving BEq, DecidableEq, Ord, Repr

def PropertyTraceField.name : PropertyTraceField → String
  | .state => "state"
  | .priorState => "prior-state"
  | .resultingState => "resulting-state"
  | .selectedAction => "selected-action"
  | .outcome => "outcome"
  | .observation => "observation"
  | .relation => "relation"

def PropertyTraceField.definitionKind : PropertyTraceField → DefinitionKind
  | .state | .priorState | .resultingState => .state
  | .selectedAction => .action
  | .outcome => .outcome
  | .observation => .fact
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
  | before
  | after
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable diagnostic name for a predicate context. -/
def PropertyPredicateContext.name : PropertyPredicateContext → String
  | .before => "before"
  | .after => "after"

/-- A field in one explicit same-step predicate environment. -/
inductive PropertyPredicateField where
  | priorState
  | selectedAction
  | resultingState
  | outcome
  | expectationFact
  deriving BEq, DecidableEq, Ord, Repr

/-- Stable diagnostic name for a predicate field. -/
def PropertyPredicateField.name : PropertyPredicateField → String
  | .priorState => "prior-state"
  | .selectedAction => "selected-action"
  | .resultingState => "resulting-state"
  | .outcome => "outcome"
  | .expectationFact => "expectation-fact"

/-- Definition kind required by a predicate field reference. -/
def PropertyPredicateField.definitionKind : PropertyPredicateField → DefinitionKind
  | .priorState | .resultingState => .state
  | .selectedAction => .action
  | .outcome => .outcome
  | .expectationFact => .fact

/-- Whether a field belongs to the context at the same trace step. -/
def PropertyPredicateContext.allows
    (context : PropertyPredicateContext)
    (field : PropertyPredicateField) : Bool :=
  match context, field with
  | .before, .priorState | .before, .selectedAction => true
  | .after, .resultingState | .after, .outcome
  | .after, .expectationFact => true
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

/-- Every field operand this predicate's atoms compare, in declaration order. Consumers that lower
or cover field operands read exactly the operands the checked evaluator resolves. -/
def PropertyPredicate.fieldOperands : PropertyPredicate → List PropertyFieldOperand
  | .atom value =>
      (value.fieldComparison.map fun comparison => [comparison.left, comparison.right]).getD []
  | .all items | .any items => items.flatMap PropertyPredicate.fieldOperands
  | .not item => PropertyPredicate.fieldOperands item

/-- Raw values for one predicate coordinate. `none` means the producer cannot supply a complete
field; `some []` records the known absence of an expectation fact. -/
structure PropertyPredicateInput where
  context : PropertyPredicateContext
  priorState : Option ModelValue := none
  selectedAction : Option ModelValue := none
  resultingState : Option ModelValue := none
  outcome : Option ModelValue := none
  facts : Option (List ModelValue) := none
  deriving BEq, DecidableEq, Repr

/-- A named condition that removes one parent or case applicability context. -/
structure PropertyUnless where
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

/-- One legacy single-pattern bounded obligation owned by a named Property case. -/
inductive PropertyTemporalClause where
  | eventuallyWithin
      (id : DefinitionId)
      (source : SourceLocation)
      (trigger response : PropertyPattern)
      (limit : Limit)
  | neverWithin
      (id : DefinitionId)
      (source : SourceLocation)
      (trigger forbidden : PropertyPattern)
      (limit : Limit)
  deriving BEq, DecidableEq, Repr

def PropertyTemporalClause.id : PropertyTemporalClause → DefinitionId
  | .eventuallyWithin id _ _ _ _ | .neverWithin id _ _ _ _ => id

def PropertyTemporalClause.source : PropertyTemporalClause → SourceLocation
  | .eventuallyWithin _ source _ _ _ | .neverWithin _ source _ _ _ => source

/-- One named branch in a same-step Property group. Matching branches are all applied. -/
structure PropertyBranch where
  id : DefinitionId
  source : SourceLocation
  guard : PropertyPredicate
  exception : Option PropertyUnless := none
  clauses : List PropertySameStepClause
  temporalClauses : List PropertyTemporalClause := []
  deriving BEq, DecidableEq, Repr

/-- A parent applicability condition and its explicitly checked named cases. -/
structure PropertyBranches where
  id : DefinitionId
  source : SourceLocation
  guard : PropertyPredicate
  exception : Option PropertyUnless := none
  cases : List PropertyBranch
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
      (unit : LimitUnit := .steps)
  | eventuallyWithin
      (id : DefinitionId)
      (trigger response : PropertyPattern)
      (limit : Limit)
      (guard : Option PropertyPredicate := none)
      (exception : Option PropertyUnless := none)
      (source : SourceLocation := { path := "" })
  | neverWithin
      (id : DefinitionId)
      (trigger forbidden : PropertyPattern)
      (limit : Limit)
      (guard : Option PropertyPredicate := none)
      (exception : Option PropertyUnless := none)
      (source : SourceLocation := { path := "" })
  | branches (group : PropertyBranches)
  deriving BEq, DecidableEq, Repr

def PropertyClause.id : PropertyClause → DefinitionId
  | .stateInvariant id _
  | .transitionContract id _ _
  | .identityRelation id _
  | .inputOutput id _ _
  | .ordered id _ _ _
  | .eventuallyWithin id _ _ _ _ _ _
  | .neverWithin id _ _ _ _ _ _ => id
  | .branches group => group.id

/-- Closing an incomplete prefix does not invent missing deadline evidence. -/
inductive PropertyScopedEndpoint where
  | final
  | «partial»
  deriving BEq, DecidableEq, Repr

/-- One named per-operation capture: the operation key it belongs to, the checked field coordinates
whose exact value each occurrence retains, and how many occurrences that operation may retain.
Occurrences are numbered from zero in admission order, so a correlation operand names an exact
earlier occurrence rather than an implicit latest match. -/
structure PropertyScopedCapture where
  name : DefinitionId
  key : DefinitionId
  path : PropertyFieldPath
  lifetime : Nat
  deriving BEq, DecidableEq, Repr

/-- A bounded response captures one immutable operation key in declared execution scope.
`captures` names the operation-local values retained at each admitted step, and `correlation` is
the precondition — typically relating this step's request fields to an earlier captured occurrence
— that every labeled transition must satisfy to be one of the operation's semantic steps. The
correlation never supplies a trigger or a response; the bounded countdown stays exactly the one
its trigger and response patterns describe. -/
structure PropertyScopedClause where
  id : DefinitionId
  source : SourceLocation
  trigger : PropertyPredicate
  response : PropertyPredicate
  scope : List DefinitionId
  key : DefinitionId
  bound : Nat
  endpoint : PropertyScopedEndpoint
  captures : List PropertyScopedCapture := []
  correlation : Option PropertyPredicate := none
  deriving BEq, DecidableEq, Repr

structure Property where
  id : DefinitionId
  source : SourceLocation
  version : Nat := 1
  requires : List DefinitionId
  clauses : List PropertyClause
  scopedClauses : List PropertyScopedClause := []
  logicalTimeSource : Option DefinitionId := none
  documentation : String := ""
  deriving BEq, DecidableEq, Repr



/-! Narrow ordinary-Lean constructors over the checked Property language. -/

namespace PropertyPattern

def selectedAction (value : ModelValue) : PropertyPattern :=
  .exact .selectedAction value.definitionId value.value

def resultingState (value : ModelValue) : PropertyPattern :=
  .exact .resultingState value.definitionId value.value

def outcome (value : ModelValue) : PropertyPattern :=
  .exact .outcome value.definitionId value.value

def fact (value : ModelValue) : PropertyPattern :=
  .exact .observation value.definitionId value.value

end PropertyPattern

namespace PropertyPredicate

/-- Author an independent field relationship in the existing closed Boolean language. -/
def compareFields (operator : PropertyFieldOperator) (left right : PropertyFieldOperand)
    (source : SourceLocation) : PropertyPredicate := .atom {
  field := .selectedAction
  reference := .of "umpire.property.fields"
  constraint := .fields ⟨operator, left, right, source⟩ }

/-- Surface spelling elaborates to exactly the ordinary typed field comparison constructor. -/
syntax "field_compare%" term:max "with" term:max term:max "at" term:max : term

macro_rules
  | `(field_compare% $left with $operator $right at $source) =>
    `(PropertyPredicate.compareFields $operator $left $right $source)

def priorStateIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .priorState
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def selectedActionIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .selectedAction
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def resultingStateIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .resultingState
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def outcomeIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .outcome
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

def factIs (value : ModelValue) : PropertyPredicate :=
  .atom {
    field := .expectationFact
    reference := value.definitionId
    constraint := .equals (.text value.value)
  }

end PropertyPredicate

/-- Build the three independent baseline obligations for one Target-owned transition result. -/
def stepClauses
    (family : DefinitionFamily)
    (propertyKey : String)
    (action state outcome fact : ModelValue) : List PropertyClause := [
  .transitionContract (family.id "property" (propertyKey ++ ".state"))
    (.selectedAction action) (.resultingState state),
  .transitionContract (family.id "property" (propertyKey ++ ".outcome"))
    (.selectedAction action) (.outcome outcome),
  .inputOutput (family.id "property" (propertyKey ++ ".fact"))
    (.selectedAction action) (.fact fact)
]

/-- Readable bounded response clauses elaborate directly to the typed declaration. Admission,
reference resolution, and canonicalization remain owned by `property%` and `Property.check`. -/
syntax (name := correlatedResponseSyntax)
  "correlated_response%" term:max "at" term:max &"whenever" term:max &"eventually" term:max
  &"within" term:max "scoped" term:max "by" term:max &"closing" term:max : term

macro_rules
  | `(correlated_response% $id at $source whenever $trigger eventually $response
      within $bound scoped $scope by $key closing $endpoint) =>
      `(({ id := $id, source := $source, trigger := $trigger, response := $response,
           bound := $bound, scope := $scope, key := $key,
           endpoint := $endpoint } : PropertyScopedClause))

end Umpire
