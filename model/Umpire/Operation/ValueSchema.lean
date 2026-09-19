/-!
Descriptor-derived value shapes. Integer wire kinds remain distinct even when their mathematical
ranges agree. Recursive messages refer to named nodes in the complete operation schema graph.
Defaults are exact scalar data; floats retain their bit patterns as metadata but are not evaluable.
-/
set_option backward.match.sparseCases false

namespace Umpire.Operation

/-- The ten Protobuf integer kinds, without narrowing at authoring time. -/
inductive IntegerKind where
  | int32 | int64 | uint32 | uint64 | sint32 | sint64 | fixed32 | fixed64 | sfixed32 | sfixed64
  deriving BEq, DecidableEq, Repr

/-- Whether the descriptor's integer kind is signed. -/
def IntegerKind.signed : IntegerKind → Bool
  | .uint32 | .uint64 | .fixed32 | .fixed64 => false
  | _ => true

/-- Width of the descriptor's integer kind. -/
def IntegerKind.bits : IntegerKind → Nat
  | .int32 | .uint32 | .sint32 | .fixed32 | .sfixed32 => 32
  | _ => 64

/-- Exact scalar data; numeric enum identity includes its declaring type. -/
inductive Scalar where
  | boolean (value : Bool)
  | text (value : String)
  | bytes (value : List UInt8)
  | integer (kind : IntegerKind) (value : Int)
  | enumeration (name : String) (number : Int)
  | floating (double : Bool) (bits : Nat)
  deriving BEq, DecidableEq, Repr

/-- Singular field type; special descriptor forms remain visible and explicitly unsupported. -/
inductive Singular where
  | boolean | text | bytes
  | integer (kind : IntegerKind)
  | enumeration (name : String)
  | message (name : String)
  | floating (double : Bool)
  | unsupported (reason : String)
  deriving BEq, DecidableEq, Repr

/-- Descriptor cardinality, including the typed key of a map. -/
inductive Cardinality where
  | singular | repeated | map (key : Singular)
  deriving BEq, DecidableEq, Repr

/-- Presence and defaults follow the descriptor, independently of concrete field omission. -/
inductive Presence where
  | implicit (defaultValue : Scalar)
  | optional
  | required
  | oneof (name : String)
  deriving BEq, DecidableEq, Repr

/-- A field is identified by its Protobuf number; its name owns diagnostics. -/
structure ValueField where
  number : Nat
  name : String
  type : Singular
  cardinality : Cardinality
  presence : Presence
  defaultValue : Option Scalar := none
  deriving BEq, DecidableEq, Repr

/-- Complete local schema data, including closed-enum policy and special message forms. -/
inductive ValueShape where
  | message (fields : List ValueField) (unsupported : Option String := none)
  | enumeration (closed : Bool) (numbers : List Int)
  deriving BEq, DecidableEq, Repr

end Umpire.Operation
