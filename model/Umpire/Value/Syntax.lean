import Umpire.Operation
import Umpire.Value.Encoding

/-!
Exact structural value syntax for schema admission. This is inert data, not a Property language.
Fields use numeric descriptor identities; explicit presence is represented by field membership.
Scalar tags retain signed integer kinds and enum names. The checked owner validates all shapes.
-/
namespace Umpire.Value
open Operation Encoding

/-- Unchecked, exact structural syntax. Only `Checked` values cross the semantic boundary. -/
abbrev Raw := Tree

/-- A proper list in the closed binary encoding. -/
def sequence : List Tree → Tree := List.foldr Tree.pair .nil

/-- Exact Unicode scalar sequence, with no locale or normalization policy. -/
def textData (value : String) : Tree := sequence (value.toList.map fun c => .atom c.toNat)

/-- A nonnegative encoding of an integer that preserves its sign without narrowing. -/
def integerData : Int → Nat
  | .ofNat n => 2 * n
  | .negSucc n => 2 * n + 1

/-- Inverse of the sign-preserving integer representation. -/
def dataInteger (n : Nat) : Int :=
  if n % 2 = 0 then .ofNat (n / 2) else .negSucc (n / 2)

/-- Stable tags for each integer descriptor kind. -/
def integerKinds : List IntegerKind :=
  [.int32, .int64, .uint32, .uint64, .sint32, .sint64, .fixed32, .fixed64, .sfixed32, .sfixed64]

/-- Exact literal construction does not perform admission or clamp numbers. -/
def literal : Scalar → Raw
  | .boolean value => .pair (.atom 0) (.atom (if value then 1 else 0))
  | .text value => .pair (.atom 1) (textData value)
  | .bytes value => .pair (.atom 2) (sequence (value.map fun b => .atom b.toNat))
  | .integer kind value => .pair (.atom 3)
      (.pair (.atom (integerKinds.idxOf kind)) (.atom (integerData value)))
  | .enumeration name number => .pair (.atom 4) (.pair (textData name) (.atom (integerData number)))
  | .floating double bits => .pair (.atom 5) (.pair (.atom (if double then 1 else 0)) (.atom bits))

/-- Exact nested message construction; omitted fields and present defaults remain distinct. -/
def message (name : String) (fields : List (Nat × Raw)) : Raw :=
  .pair (.atom 6) (.pair (textData name)
    (sequence (fields.map fun (number, value) => .pair (.atom number) value)))

/-- Repeated values retain both order and duplicate elements. -/
def repeated (values : List Raw) : Raw := .pair (.atom 7) (sequence values)

/-- Raw map entries retain duplicates until the explicitly selected codec normalization boundary. -/
def map (entries : List (Scalar × Raw)) : Raw :=
  .pair (.atom 8) (sequence (entries.map fun (key, value) => .pair (literal key) value))

/-- Read a proper list with a bound checked before each additional element. -/
def readSequence : Nat → Tree → Option (List Tree)
  | _, .nil => some []
  | 0, _ => none
  | n + 1, .pair a b => (readSequence n b).map (a :: ·)
  | _, _ => none

/-- Read exact text; malformed Unicode atoms are rejected rather than replaced. -/
def readText (bound : Nat) (tree : Tree) : Option String := do
  let chars ← (← readSequence bound tree).mapM fun t => do
    let .atom n := t | none
    let c := Char.ofNat n
    if c.toNat = n then some c else none
  pure (String.ofList chars)

/-- Parse a literal while retaining integer and enum identity and exact bytes. -/
def readLiteral (bound : Nat) : Raw → Option Scalar
  | .pair (.atom 0) (.atom 0) => some (.boolean false)
  | .pair (.atom 0) (.atom 1) => some (.boolean true)
  | .pair (.atom 1) data => (readText bound data).map .text
  | .pair (.atom 2) data => do
    let bytes ← (← readSequence bound data).mapM fun t => do
      let .atom n := t | none
      if n < 256 then some (UInt8.ofNat n) else none
    pure (.bytes bytes)
  | .pair (.atom 3) (.pair (.atom kind) (.atom n)) =>
    (integerKinds[kind]?).map fun k => .integer k (dataInteger n)
  | .pair (.atom 4) (.pair name (.atom n)) =>
    (readText bound name).map fun s => .enumeration s (dataInteger n)
  | .pair (.atom 5) (.pair (.atom width) (.atom bits)) =>
    if width ≤ 1 then some (.floating (width == 1) bits) else none
  | _ => none

end Umpire.Value
