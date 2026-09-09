import Umpire.Value.Field
import Umpire.Operation.Action

/-!
Closed same-step field operands. Structural paths retain the complete operation schema and payload
side; concrete scalars enter only through checked cursors. Literal admission preserves integer kind
and range, enum identity, and exact bytes. Floating-point evaluation remains unsupported.
-/
namespace Umpire
open Operation Value Value.Encoding

/-- The five immutable same-step model payload coordinates. -/
inductive PropertyFieldRoot where
  | request | priorState | resultingState | outcome | event
  deriving BEq, DecidableEq, Repr

/-- Structural coordinates for a field operand, independent of generated Lean spelling. -/
structure PropertyFieldPath where
  root : PropertyFieldRoot
  reference : DefinitionId
  schema : RpcSchema
  side : Side
  steps : List Field.Step
  type : Singular
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

private def fieldOperandData : PropertyFieldOperand → Raw
  | .literal value _ => sequence [.atom 0, literal value]
  | .field path _ => sequence [.atom 1, .atom (fieldRootData path.root),
      textData path.reference.value, Canonical.rpcSchema path.schema,
      .atom (if path.side == .request then 0 else 1),
      sequence (path.steps.map fieldStepData), fieldTypeData path.type]

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
  ⟨root, reference, owner.schema witness, payloadSide, cursor.path, valueType⟩

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

end Umpire
