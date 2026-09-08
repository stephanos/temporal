import Umpire.Value

/-!
Schema-indexed access to admitted operation values. References enumerate the complete generated
closure and retain its owner, payload witness, side, and containing message. Cursors start only at
`Checked` values. Optional fields and map lookups require `establish`; oneof members require
`select` before dependent access. Each operation constructs a structural denotation derivation over
the original concrete tree, preserving exact values without reinterpreting descriptor text as types.
-/
set_option backward.match.sparseCases false

namespace Umpire.Value.Field
open Operation Encoding

/-- Access failures retain the exact owning author location and structural field path. -/
structure Error where
  source : Umpire.SourceLocation
  path : String
  reason : String
  deriving BEq, DecidableEq, Repr

/-- All fields in the selected descriptor closure, including recursive and unsupported forms. -/
def schemaFields (schema : Schema) : List (String × ValueField) :=
  schema.nodes.flatMap fun node => match node.valueShape with
    | some (.message fields _) => fields.map (node.name, ·)
    | _ => []

/-- Structural membership in the selected owner's schema; names alone cannot construct a reference. -/
structure Reference (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (containing : String) where
  private mk ::
  field : ValueField
  member : (containing, field) ∈ schemaFields (signature owner reference side)

/-- Discover every field without a concrete recursion bound or sample-field whitelist. -/
def references (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) :
    List ((containing : String) × Reference owner reference side containing) :=
  (schemaFields (signature owner reference side)).attach.map fun item =>
    ⟨item.val.1, ⟨item.val.2, item.property⟩⟩

/-- Discovery returns the entire generated field surface without dropping unsupported fields. -/
theorem references_complete (owner : RpcOwner) {Request Response : Type}
    (witness : owner.Witness Request Response) (side : Side) :
    (references owner witness side).map (fun ref => (ref.1, ref.2.field)) =
      schemaFields (signature owner witness side) := by
  simp only [references, List.map_map]
  exact List.attach_map_subtype_val _

/-- Admit an authored structural coordinate against the retained generated schema. -/
def reference (owner : RpcOwner) {Request Response : Type}
    (witness : owner.Witness Request Response) (side : Side)
    (containing : String) (number : Nat) (source : Umpire.SourceLocation) :
    Except Error (Reference owner witness side containing) :=
  match (schemaFields (signature owner witness side)).attach.find?
      (fun item => item.val.1 == containing && item.val.2.number == number) with
  | none => .error ⟨source, containing ++ "." ++ toString number, "unknown containing schema or field"⟩
  | some item =>
    if equal : item.val.1 = containing then
      .ok ⟨item.val.2, equal ▸ item.property⟩
    else .error ⟨source, containing, "containing schema mismatch"⟩

/-- Descriptor availability is independent of the current concrete value. -/
inductive Availability where
  | available | optional | oneof (group : String)
  deriving BEq, DecidableEq, Repr

/-- Collections are always present; singular presence follows the descriptor. -/
def availability (field : ValueField) : Availability :=
  match field.cardinality, field.presence with
  | .singular, .optional => .optional
  | .singular, .oneof group => .oneof group
  | _, _ => .available

/-- Serializable structural access coordinates, independent of generated Lean field spelling. -/
inductive Step where
  | field (containing : String) (number : Nat)
  | establish
  | select (group : String)
  | index (index : Nat)
  | cardinality
  | key (key : Scalar)
  | present
  deriving BEq, DecidableEq, Repr

/-- Exact proper-list semantics, without a traversal ceiling or fabricated truncated suffix. -/
inductive Sequence : Raw → List Raw → Prop where
  | nil : Sequence .nil []
  | cons : Sequence tail rest → Sequence (.pair head tail) (head :: rest)

private theorem sequence_sound (bound : Nat) (raw : Raw) (items : List Raw)
    (h : readSequence bound raw = some items) : Sequence raw items := by
  induction bound generalizing raw items with
  | zero =>
    cases raw <;> simp [readSequence] at h
    subst items
    exact .nil
  | succ bound ih =>
    cases raw with
    | nil => simp [readSequence] at h; subst items; exact .nil
    | atom n => simp [readSequence] at h
    | pair head tail =>
      cases eq : readSequence bound tail with
      | none => simp [readSequence, eq] at h
      | some rest =>
        simp [readSequence, eq] at h
        subst items
        exact .cons (ih tail rest eq)

private def fieldValue (number : Nat) (items : List Raw) : Option Raw :=
  items.findSome? fun item => match item with
    | .pair (.atom n) value => if n == number then some value else none
    | _ => none

private def mapValue (key : Scalar) (items : List Raw) : Option Raw :=
  items.findSome? fun item => match item with
    | .pair rawKey value => if rawKey == literal key then some value else none
    | _ => none

/-- One concrete structural selection. Membership, order, exact keys, and presence are explicit. -/
inductive StepDenotes : Step → Option Raw → Option Raw → Prop where
  | field (identity : readText bound name = some containing) (list : Sequence data items) :
      StepDenotes (.field containing number)
        (some (.pair (.atom 6) (.pair name data))) (fieldValue number items)
  | establish : StepDenotes .establish (some value) (some value)
  | select : StepDenotes (.select group) (some value) (some value)
  | index (list : Sequence data items) (selected : items[index]? = some value) :
      StepDenotes (.index index) (some (.pair (.atom 7) data)) (some value)
  | cardinality (list : Sequence data items) :
      StepDenotes .cardinality (some (.pair (.atom 7) data)) (some (.atom items.length))
  | key (list : Sequence data items) :
      StepDenotes (.key key) (some (.pair (.atom 8) data)) (mapValue key items)
  | present : StepDenotes .present value (some (literal (.boolean value.isSome)))

/-- A path denotes its actual selection from the original admitted concrete value. -/
inductive Denotes (root : Raw) : List Step → Option Raw → Prop where
  | root : Denotes root [] (some root)
  | step : Denotes root path input → StepDenotes next input output →
      Denotes root (path ++ [next]) output

/-- Schema typing and availability for every structural path, independently of concrete denotation. -/
inductive TypedPath (schema : Schema) : List Step → Singular → Cardinality → Availability → Prop where
  | root : TypedPath schema [] (.message schema.root) .singular .available
  | field : TypedPath schema path (.message name) .singular .available →
      (name, field) ∈ schemaFields schema →
      TypedPath schema (path ++ [.field name field.number]) field.type field.cardinality (availability field)
  | establish : TypedPath schema path type card .optional →
      TypedPath schema (path ++ [.establish]) type card .available
  | select : TypedPath schema path type card (.oneof group) →
      TypedPath schema (path ++ [.select group]) type card .available
  | index : TypedPath schema path type .repeated .available →
      TypedPath schema (path ++ [.index index]) type .singular .available
  | key : TypedPath schema path type (.map keyType) .available →
      TypedPath schema (path ++ [.key key]) type .singular .optional
  | present : TypedPath schema path type card available → available ≠ .available →
      TypedPath schema (path ++ [.present]) .boolean .singular .available

/-- A typed selection from one immutable admitted payload. Construction is confined to checked access. -/
structure Cursor (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits)
    (type : Singular) (cardinality : Cardinality) (availability : Availability) where
  private mk ::
  origin : Checked owner reference side limits
  path : List Step
  datum : Option Raw
  typed : TypedPath (signature owner reference side) path type cardinality availability
  denotes : Denotes origin.value path datum
  ready : availability = .available → datum.isSome = true

/-- Check expected descriptor types with kernel equality; this cannot establish optional presence. -/
def Cursor.refine (cursor : Cursor owner witness side limits type cardinality available)
    (expectedType : Singular) (expectedCardinality : Cardinality) (expectedAvailability : Availability)
    (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits expectedType expectedCardinality expectedAvailability) :=
  if ht : type = expectedType then
    if hc : cardinality = expectedCardinality then
      if ha : available = expectedAvailability then .ok (ht ▸ hc ▸ ha ▸ cursor)
      else .error ⟨source, reprStr cursor.path, "availability mismatch"⟩
    else .error ⟨source, reprStr cursor.path, "cardinality mismatch"⟩
  else .error ⟨source, reprStr cursor.path, "field type mismatch"⟩

/-- Start traversal at the exact task2-admitted root under the same generated authority. -/
def root (value : Checked owner witness side limits) :
    Cursor owner witness side limits (.message (signature owner witness side).root)
      .singular .available :=
  ⟨value, [], some value.value, .root, .root, by simp⟩

private def pathName (path : List Step) : String := reprStr path

private def reject (source : Umpire.SourceLocation) (path : List Step) (reason : String) :
    Except Error α := .error ⟨source, pathName path, reason⟩

private def supported (schema : Schema) (type : Singular) : Option String :=
  match type with
  | .floating _ => some "unsupported floating-point evaluation"
  | .unsupported reason => some ("unsupported " ++ reason)
  | .message name => match schema.nodes.find? (·.name == name) with
    | some node => match node.valueShape with
      | some (.message _ reason) => reason.map ("unsupported " ++ ·)
      | _ => some "message schema unavailable"
    | none => some "unknown message schema"
  | _ => none

/-- Read a schema member. Presence-bearing results remain unavailable for dependent consumption. -/
def Cursor.field (cursor : Cursor owner witness side limits (.message containing) .singular .available)
    (ref : Reference owner witness side containing) (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits ref.field.type ref.field.cardinality
      (availability ref.field)) := do
  let path := cursor.path ++ [.field containing ref.field.number]
  if let some reason := supported (signature owner witness side) ref.field.type then
    reject source path reason
  match input : cursor.datum with
  | some (.pair (.atom 6) (.pair name data)) =>
    if identity : readText limits.bytes name = some containing then
      match parsed : readSequence limits.collection data with
      | none => reject source path "malformed fields or collection limit"
      | some items =>
        if !items.all (fun item => match item with | .pair (.atom _) _ => true | _ => false) then
          reject source path "malformed field entry"
        let value := fieldValue ref.field.number items
        if ready : availability ref.field = .available → value.isSome = true then
          pure ⟨cursor.origin, path, value, .field cursor.typed ref.member,
            .step cursor.denotes (input ▸ .field identity (sequence_sound _ _ _ parsed)), ready⟩
        else reject source path "missing implicit, required, or collection field"
    else reject source path "message identity mismatch"
  | _ => reject source path "message type mismatch"

/-- Establish optional presence (including a keyed lookup) before consuming dependent data. -/
def Cursor.establish (cursor : Cursor owner witness side limits type cardinality .optional)
    (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits type cardinality .available) :=
  match input : cursor.datum with
  | none => reject source cursor.path "required presence is absent"
  | some value => .ok ⟨cursor.origin, cursor.path ++ [.establish], some value, .establish cursor.typed,
      .step cursor.denotes (input ▸ .establish), by simp⟩

/-- Establish the exact oneof group/member, rejecting an unselected branch even at its default. -/
def Cursor.select (cursor : Cursor owner witness side limits type cardinality (.oneof group))
    (expected : String) (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits type cardinality .available) :=
  if expected != group then reject source cursor.path "oneof group mismatch"
  else match input : cursor.datum with
  | none => reject source cursor.path "oneof member is not selected"
  | some value => .ok ⟨cursor.origin, cursor.path ++ [.select group], some value, .select cursor.typed,
      .step cursor.denotes (by rw [input]; exact .select), by simp⟩

/-- Presence can be tested only for descriptor presence or a keyed lookup, never an implicit scalar. -/
def Cursor.present (cursor : Cursor owner witness side limits type cardinality available)
    (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits .boolean .singular .available) :=
  if h : available = .available then reject source cursor.path "field has no optional presence"
  else .ok ⟨cursor.origin, cursor.path ++ [.present], some (literal (.boolean cursor.datum.isSome)),
    .present cursor.typed h, .step cursor.denotes .present, by simp⟩

/-- Ordered repeated selection checks the declared ceiling and actual length separately. -/
def Cursor.index (cursor : Cursor owner witness side limits type .repeated .available)
    (index : Nat) (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits type .singular .available) := do
  let path := cursor.path ++ [.index index]
  if index ≥ limits.collection then reject source path "index exceeds collection ceiling"
  match input : cursor.datum with
  | some (.pair (.atom 7) data) =>
    match parsed : readSequence limits.collection data with
    | none => reject source path "malformed repeated value or collection limit"
    | some items => match selected : items[index]? with
      | none => reject source path "repeated index out of range"
      | some value => pure ⟨cursor.origin, path, some value, .index cursor.typed,
          .step cursor.denotes (input ▸ .index (sequence_sound _ _ _ parsed) selected), by simp⟩
  | _ => reject source path "repeated type mismatch"

/-- An exact natural cardinality with its concrete path derivation. -/
structure CardinalityValue (cursor : Cursor owner witness side limits type .repeated .available) where
  private mk ::
  value : Nat
  denotes : Denotes cursor.origin.value (cursor.path ++ [.cardinality]) (some (.atom value))

/-- Cardinality is exact and bounded; it is a natural, not a protobuf integer. -/
def Cursor.cardinality (cursor : Cursor owner witness side limits type .repeated .available)
    (source : Umpire.SourceLocation) : Except Error (CardinalityValue cursor) := do
  match input : cursor.datum with
  | some (.pair (.atom 7) data) =>
    match parsed : readSequence limits.collection data with
    | none => reject source cursor.path "malformed repeated value or collection limit"
    | some items => pure ⟨items.length,
        .step cursor.denotes (input ▸ .cardinality (sequence_sound _ _ _ parsed))⟩
  | _ => reject source cursor.path "repeated type mismatch"

/-- Read the exact natural cardinality through the checked selection. -/
def Cursor.length (cursor : Cursor owner witness side limits type .repeated .available)
    (source : Umpire.SourceLocation) : Except Error Nat :=
  (cursor.cardinality source).map (·.value)

/-- Every successfully returned length is the denoted concrete repeated cardinality. -/
theorem Cursor.length_correspondence
    (cursor : Cursor owner witness side limits type .repeated .available)
    (h : cursor.length source = .ok count) :
    Denotes cursor.origin.value (cursor.path ++ [.cardinality]) (some (.atom count)) := by
  unfold Cursor.length at h
  cases result : cursor.cardinality source with
  | error error => simp [result, Except.map] at h
  | ok selected =>
    simp [result, Except.map] at h
    subst count
    exact selected.denotes

private def keyTypeMatches (type : Singular) (key : Scalar) : Bool :=
  match type, key with
  | .boolean, .boolean _ | .text, .text _ => true
  | .integer expected, .integer actual value =>
    let low : Int := if expected.signed then -(Int.ofNat (2^(expected.bits - 1))) else 0
    let high : Int := Int.ofNat (2^(expected.bits - if expected.signed then 1 else 0)) - 1
    expected == actual && low ≤ value && value ≤ high
  | _, _ => false

/-- Exact typed lookup distinguishes a missing key from malformed keys and exhausted bounds. -/
def Cursor.lookup (cursor : Cursor owner witness side limits type (.map keyType) .available)
    (key : Scalar) (source : Umpire.SourceLocation) :
    Except Error (Cursor owner witness side limits type .singular .optional) := do
  let path := cursor.path ++ [.key key]
  if !keyTypeMatches keyType key then reject source path "map key type or range mismatch"
  if (literal key).nodes > limits.work || (Encoding.encode (literal key)).length > limits.bytes then
    reject source path "map key work or byte limit"
  match input : cursor.datum with
  | some (.pair (.atom 8) data) =>
    match parsed : readSequence limits.collection data with
    | none => reject source path "malformed map or collection limit"
    | some items =>
      if data.nodes + items.length * (literal key).nodes > limits.work then
        reject source path "map lookup work limit"
      if !items.all (fun item => match item with | .pair _ _ => true | _ => false) then
        reject source path "malformed map entry"
      pure ⟨cursor.origin, path, mapValue key items, .key cursor.typed,
        .step cursor.denotes (input ▸ .key (sequence_sound _ _ _ parsed)), by intro h; cases h⟩
  | _ => reject source path "map type mismatch"

/-- Exact structural scalar type, retaining enum identity and integer kind. -/
def scalarType : Scalar → Singular
  | .boolean _ => .boolean
  | .text _ => .text
  | .bytes _ => .bytes
  | .integer kind _ => .integer kind
  | .enumeration name _ => .enumeration name
  | .floating double _ => .floating double

/-- An exact scalar parsed from the denoted concrete field, never from descriptive metadata. -/
structure ScalarValue (cursor : Cursor owner witness side limits type .singular .available) where
  private mk ::
  value : Scalar
  raw : Raw
  denotes : Denotes cursor.origin.value cursor.path (some raw)
  parsed : readLiteral limits.bytes raw = some value
  typed : scalarType value = type

/-- Scalar consumption requires availability and retains the exact descriptor kind in the cursor. -/
def Cursor.scalarValue (cursor : Cursor owner witness side limits type .singular .available)
    (source : Umpire.SourceLocation) : Except Error (ScalarValue cursor) := do
  if let some reason := supported (signature owner witness side) type then
    reject source cursor.path reason
  match input : cursor.datum with
  | none => reject source cursor.path "scalar is absent"
  | some raw => match parsed : readLiteral limits.bytes raw with
    | some value =>
      if typed : scalarType value = type then
        pure ⟨value, raw, input ▸ cursor.denotes, parsed, typed⟩
      else reject source cursor.path "scalar descriptor type mismatch"
    | none => reject source cursor.path "not a supported scalar"

/-- Read the exact scalar through its checked denotation. -/
def Cursor.scalar (cursor : Cursor owner witness side limits type .singular .available)
    (source : Umpire.SourceLocation) : Except Error Scalar :=
  (cursor.scalarValue source).map (·.value)

/-- Every returned scalar is parsed from the actual denoted field of the admitted payload. -/
theorem Cursor.scalar_correspondence
    (cursor : Cursor owner witness side limits type .singular .available)
    (h : cursor.scalar source = .ok value) :
    ∃ raw, Denotes cursor.origin.value cursor.path (some raw) ∧
      readLiteral limits.bytes raw = some value ∧ scalarType value = type := by
  unfold Cursor.scalar at h
  cases result : cursor.scalarValue source with
  | error error => simp [result, Except.map] at h
  | ok selected =>
    simp [result, Except.map] at h
    subst value
    exact ⟨selected.raw, selected.denotes, selected.parsed, selected.typed⟩

/-- Every returned selection denotes its structural path in the actual task2-admitted payload. -/
theorem Cursor.correspondence (cursor : Cursor owner witness side limits type card available) :
    Denotes cursor.origin.value cursor.path cursor.datum := cursor.denotes

end Umpire.Value.Field
