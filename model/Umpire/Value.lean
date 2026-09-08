import Umpire.Value.Check

/-!
Exact bounded operation values. `check` consumes a generated owner's typed witness and keeps both
indices through the carrier and codec. `Encoded` separates structural schema metadata from concrete
payload bytes; it is version 1 and is not the Testpilot wire protocol. `decodeCanonical` checks the
schema, version, side, bounds, and canonical bytes. `decodeRaw` is the explicit last-value map
normalization boundary. The binary codec's universal proof, rather than a codec equality certificate
stored by admission, establishes round-trip correspondence for every admitted value.

Payload bytes and encoded bytes must each fit the byte ceiling. Each admission traversal is work
bounded, and encoded bytes and tree nodes must also fit the work ceiling before decoding. This
conservative format is intended for bounded model values; portable protobuf lowering remains a
separate owner. Floats remain discoverable but their concrete evaluation is explicitly unsupported.
-/
namespace Umpire.Value
open Operation Encoding

/-- Which generated payload signature a value inhabits. -/
inductive Side where
  | request | response
  deriving BEq, DecidableEq, Repr

/-- Select a payload schema without losing the generated owner or witness indices. -/
def signature (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) : Schema :=
  match side with
  | .request => (owner.schema reference).request
  | .response => (owner.schema reference).response

/-- Canonical schema-admitted data; the proof fields state admission and resource invariants only. -/
structure Checked (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits) where
  private mk ::
  value : Raw
  admitted : (normalize (signature owner reference side) limits .literal value).toOption = some value
  bounded : value.nodes ≤ limits.work ∧ (Encoding.encode value).length ≤ limits.work ∧
    (Encoding.encode value).length ≤ limits.bytes

private def admitCanonical (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits) (value : Raw) :
    Except Error (Checked owner reference side limits) :=
  match result : normalize (signature owner reference side) limits .literal value with
  | .error error => .error error
  | .ok normalized =>
    if same : normalized = value then
      if bounded : value.nodes ≤ limits.work ∧ (Encoding.encode value).length ≤ limits.work ∧
          (Encoding.encode value).length ≤ limits.bytes then
        .ok ⟨value, by simp [result, same, Except.toOption], bounded⟩
      else .error ⟨(signature owner reference side).root, "encoded work or byte limit"⟩
    else .error ⟨(signature owner reference side).root, "noncanonical value"⟩

/-- Admit exact modeled values, with explicit raw-codec normalization when selected by the caller. -/
def check (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits) (value : Raw)
    (mode : InputMode := .literal) : Except Error (Checked owner reference side limits) := do
  let normalized ← normalize (signature owner reference side) limits mode value
  admitCanonical owner reference side limits normalized

/-- A versioned payload with exact schema metadata outside its concrete binary value. -/
structure Encoded where
  version : Nat
  schema : RpcSchema
  side : Side
  payload : List UInt8
  deriving BEq, DecidableEq, Repr

/-- Canonical encoding retains the selected method/schema and serializes the actual value tree. -/
def encode (value : Checked owner reference side limits) : Encoded :=
  ⟨1, owner.schema reference, side, Encoding.encode value.value⟩

private def decodeTree (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits) (input : Encoded) :
    Except Error Raw := do
  let path := (signature owner reference side).root
  if input.version != 1 then throw ⟨path, "unsupported value encoding version"⟩
  if input.schema ≠ owner.schema reference then throw ⟨path, "incompatible operation schema"⟩
  if input.side ≠ side then throw ⟨path, "payload side mismatch"⟩
  if input.payload.length > limits.bytes then throw ⟨path, "encoded byte limit"⟩
  if input.payload.length > limits.work then throw ⟨path, "encoded work limit"⟩
  match Encoding.decode limits.work input.payload with
  | some (value, []) => pure value
  | _ => throw ⟨path, "malformed encoding or structural limit"⟩

/-- Decode raw structural entries, applying Protobuf last-value semantics only to map duplicates. -/
def decodeRaw (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits) (input : Encoded) :
    Except Error (Checked owner reference side limits) := do
  check owner reference side limits (← decodeTree owner reference side limits input) .decoded

/-- Canonical decoding rejects overlong numbers, trailing bytes, reordered fields and duplicate maps. -/
def decodeCanonical (owner : RpcOwner) {Request Response : Type}
    (reference : owner.Witness Request Response) (side : Side) (limits : Limits) (input : Encoded) :
    Except Error (Checked owner reference side limits) := do
  let value ← decodeTree owner reference side limits input
  if Encoding.encode value != input.payload then
    throw ⟨(signature owner reference side).root, "noncanonical binary encoding"⟩
  admitCanonical owner reference side limits value

private theorem admitCanonical_checked (value : Checked owner reference side limits) :
    admitCanonical owner reference side limits value.value = .ok value := by
  rcases value with ⟨value, admitted, bounded⟩
  unfold admitCanonical
  split
  next error result => simp [result, Except.toOption] at admitted
  next normalized result =>
    have same : normalized = value := by simpa [result, Except.toOption] using admitted
    subst normalized
    simp [bounded]

/-- The actual canonical decoder reverses the actual encoder for every admitted bounded value,
under the same generated owner, payload-indexed witness, schema, side, and resource scope. -/
theorem decode_encode (value : Checked owner reference side limits) :
    decodeCanonical owner reference side limits (encode value) = .ok value := by
  have parse := Encoding.decode_encode value.value limits.work [] value.bounded.1
  simp only [List.append_nil] at parse
  simp [decodeCanonical, decodeTree, encode, parse, Nat.not_lt.mpr value.bounded.2.1,
    Nat.not_lt.mpr value.bounded.2.2, admitCanonical_checked]

end Umpire.Value
