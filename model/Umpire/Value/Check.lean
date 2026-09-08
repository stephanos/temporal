import Umpire.Value.Syntax

/-!
Schema-driven admission of exact operation values. The owner and its payload-indexed witness are
retained by the public checked carrier. Concrete depth, work, payload bytes, and collection ceilings
are independent; failures return no partial value. Decoded maps use last-value normalization, while
model literals reject duplicate keys. Floating-point evaluation and groups/extensions are unsupported.
-/
namespace Umpire.Value
open Operation Encoding

/-- Independent concrete admission ceilings. Depth counts singular values, including the root. -/
structure Limits where
  depth : Nat
  work : Nat
  bytes : Nat
  collection : Nat
  deriving BEq, DecidableEq, Repr

/-- A source-owned failure, without a partial admitted subtree. -/
structure Error where
  path : String
  reason : String
  deriving BEq, DecidableEq, Repr

/-- Duplicate raw map entries normalize only at the decoded codec boundary. -/
inductive InputMode where
  | literal | decoded
  deriving BEq, DecidableEq, Repr

private structure Budget where
  work : Nat := 0
  bytes : Nat := 0

private abbrev CheckM := StateT Budget (Except Error)

private def fail (path reason : String) : CheckM α := throw ⟨path, reason⟩

private def charge (limits : Limits) (path : String) (work bytes : Nat) : CheckM Unit := do
  let used ← get
  if used.work + work > limits.work then fail path "work limit"
  if used.bytes + bytes > limits.bytes then fail path "byte limit"
  set (Budget.mk (used.work + work) (used.bytes + bytes))

private def entries (limits : Limits) (path : String) (tree : Tree) : CheckM (List Tree) := do
  match readSequence limits.collection tree with
  | some result =>
    charge limits path (result.length + 1) 0
    pure result
  | none => fail path "collection limit or malformed collection"

private def shape (schema : Schema) (limits : Limits) (path name : String) : CheckM ValueShape := do
  charge limits path (schema.nodes.length + 1) 0
  match schema.nodes.find? (·.name == name) with
  | some node =>
    match node.valueShape with
    | some shape => pure shape
    | none => fail path "concrete schema unavailable"
  | none => fail path "unknown schema reference"

private def scalarCost : Scalar → Nat
  | .text s => s.utf8ByteSize
  | .bytes b => b.length
  | .enumeration name _ => name.utf8ByteSize + 4
  | .boolean _ => 1
  | .integer kind _ => kind.bits / 8
  | .floating double _ => if double then 8 else 4

private def checkScalar (schema : Schema) (limits : Limits) (path : String)
    (typ : Singular) (raw : Raw) : CheckM Scalar := do
  if let .floating _ := typ then fail path "unsupported floating-point evaluation"
  if let .unsupported reason := typ then fail path ("unsupported " ++ reason)
  if let .pair (.atom 3) (.pair _ (.atom number)) := raw then
    if number ≥ 2^65 then fail path "integer range"
  if let .pair (.atom 4) (.pair _ (.atom number)) := raw then
    if number ≥ 2^32 then fail path "enum int32 range"
  let some value := readLiteral limits.bytes raw | fail path "malformed scalar or byte limit"
  charge limits path (1 + scalarCost value) (scalarCost value)
  match typ, value with
  | .boolean, .boolean _ | .text, .text _ | .bytes, .bytes _ => pure value
  | .integer expected, .integer actual n =>
    if expected != actual then fail path "integer kind mismatch"
    let low : Int := if expected.signed then -(Int.ofNat (2^(expected.bits - 1))) else 0
    let high : Int := Int.ofNat (2^(expected.bits - if expected.signed then 1 else 0)) - 1
    if n < low || n > high then fail path "integer range"
    pure value
  | .enumeration expected, .enumeration actual n =>
    if expected != actual then fail path "enum identity mismatch"
    if n < -2147483648 || n > 2147483647 then fail path "enum int32 range"
    let .enumeration closed numbers ← shape schema limits path expected
      | fail path "enum schema mismatch"
    charge limits path numbers.length 0
    if closed && !numbers.contains n then fail path "unknown closed enum number"
    pure value
  | _, _ => fail path "scalar type mismatch"

private def keyLess (a b : Scalar) : Bool :=
  match a, b with
  | .boolean x, .boolean y => !x && y
  | .text x, .text y => x < y
  | .integer _ x, .integer _ y => x < y
  | _, _ => false

private def validKey : Singular → Bool
  | .boolean | .text | .integer _ => true
  | _ => false

private def normalizeValue (schema : Schema) (limits : Limits) (mode : InputMode) :
    Nat → String → Singular → Raw → CheckM Raw
  | 0, path, _, _ => fail path "depth limit"
  | depth + 1, path, typ, raw => do
    charge limits path 1 0
    match typ with
    | .message expected =>
      let .pair (.atom 6) (.pair name data) := raw | fail path "message type mismatch"
      let some actual := readText limits.bytes name | fail path "malformed message identity"
      charge limits path (actual.utf8ByteSize + 1) 0
      if actual != expected then fail path "message identity mismatch"
      let .message fields unsupported ← shape schema limits path expected
        | fail path "message schema mismatch"
      if let some reason := unsupported then fail path ("unsupported " ++ reason)
      let input ← entries limits path data
      let mut supplied : List (Nat × Raw) := []
      let mut selected : List String := []
      for entry in input do
        let .pair (.atom number) value := entry | fail path "malformed field"
        charge limits path (fields.length + supplied.length + 1) 0
        let some field := fields.find? (·.number == number) | fail path ("unknown field " ++ toString number)
        let fieldPath := path ++ "." ++ field.name
        if supplied.any (·.1 == number) then fail fieldPath "duplicate field"
        if let .oneof name := field.presence then
          if selected.contains name then fail fieldPath "oneof conflict"
          selected := name :: selected
        supplied := supplied ++ [(number, value)]
      let mut output : List (Nat × Raw) := []
      for field in fields.mergeSort (fun a b => a.number ≤ b.number) do
        let fieldPath := path ++ "." ++ field.name
        charge limits fieldPath (supplied.length + fields.length + 1) 0
        let value? := (supplied.find? (·.1 == field.number)).map Prod.snd
        let value? ← match value? with
          | some value => pure (some value)
          | none => match field.cardinality, field.presence with
            | .repeated, _ => pure (some (repeated []))
            | .map _, _ => pure (some (map []))
            | .singular, .implicit defaultValue => pure (some (literal defaultValue))
            | .singular, .required => fail fieldPath "missing required field"
            | _, _ => pure none
        if let some value := value? then
          let normalized ← match field.cardinality with
            | .singular => normalizeValue schema limits mode depth fieldPath field.type value
            | .repeated => do
              let .pair (.atom 7) data := value | fail fieldPath "repeated type mismatch"
              let input ← entries limits fieldPath data
              let repeatedOutput ← input.zipIdx |>.mapM fun (item, index) =>
                normalizeValue schema limits mode depth
                  (fieldPath ++ "[" ++ toString index ++ "]") field.type item
              pure (repeated repeatedOutput)
            | .map keyType => do
              if !validKey keyType then fail fieldPath "unsupported map key type"
              let .pair (.atom 8) data := value | fail fieldPath "map type mismatch"
              let input ← entries limits fieldPath data
              let mut mapOutput : List (Scalar × Raw) := []
              for (item, index) in input.zipIdx do
                let itemPath := fieldPath ++ "[" ++ toString index ++ "]"
                let .pair rawKey rawValue := item | fail itemPath "malformed map entry"
                let key ← checkScalar schema limits (itemPath ++ ".key") keyType rawKey
                charge limits itemPath (mapOutput.length * (1 + scalarCost key)) 0
                if mode == .literal && mapOutput.any (·.1 == key) then
                  fail itemPath "duplicate map literal key"
                let value ← normalizeValue schema limits mode depth (itemPath ++ ".value") field.type rawValue
                mapOutput := (mapOutput.filter (·.1 != key)) ++ [(key, value)]
              charge limits fieldPath (mapOutput.length * mapOutput.length) 0
              pure (map (mapOutput.mergeSort fun a b => !keyLess b.1 a.1))
          output := output ++ [(field.number, normalized)]
      pure (message expected output)
    | _ => pure (literal (← checkScalar schema limits path typ raw))

/-- Normalize concrete values according to the retained complete descriptor graph.
All implicit defaults and empty collections are explicit in the canonical carrier. -/
def normalize (schema : Schema) (limits : Limits) (mode : InputMode) (value : Raw) :
    Except Error Raw := do
  let (value, _) ← normalizeValue schema limits mode limits.depth schema.root (.message schema.root)
    value |>.run {}
  pure value

end Umpire.Value
