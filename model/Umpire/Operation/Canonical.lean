import Umpire.Value

/-!
Version 1 structural identity for parameterized operations. Tags and field positions below are
format semantics; schema descriptors and value shapes are both retained, including defaults.
The alphabet is compatible with existing finite catalog keys. No display printer supplies identity.
-/
namespace Umpire.Operation.Canonical
open Value Value.Encoding

private def option (f : α → Tree) : Option α → Tree
  | none => .nil
  | some value => .pair (.atom 1) (f value)

private def boolean (b : Bool) : Tree := .atom (if b then 1 else 0)

private def singular : Singular → Tree
  | .boolean => .atom 0
  | .text => .atom 1
  | .bytes => .atom 2
  | .integer k => .pair (.atom 3) (.atom (integerKinds.idxOf k))
  | .enumeration n => .pair (.atom 4) (textData n)
  | .message n => .pair (.atom 5) (textData n)
  | .floating d => .pair (.atom 6) (boolean d)
  | .unsupported r => .pair (.atom 7) (textData r)

private def cardinality : Cardinality → Tree
  | .singular => .atom 0
  | .repeated => .atom 1
  | .map key => .pair (.atom 2) (singular key)

private def presence : Presence → Tree
  | .implicit value => .pair (.atom 0) (literal value)
  | .optional => .atom 1
  | .required => .atom 2
  | .oneof name => .pair (.atom 3) (textData name)

private def field (f : ValueField) : Tree := sequence [
  .atom f.number, textData f.name, singular f.type, cardinality f.cardinality,
  presence f.presence, option literal f.defaultValue]

private def shape : ValueShape → Tree
  | .message fields unsupported => sequence [
      .atom 0, sequence (fields.map field), option textData unsupported]
  | .enumeration closed numbers => sequence [
      .atom 1, boolean closed, sequence (numbers.map fun n => .atom (integerData n))]

private def node (n : SchemaNode) : Tree := sequence [
  textData n.name, textData n.protoSyntax, textData n.descriptor, textData n.fileContext,
  sequence (n.references.map textData), option shape n.valueShape]

private def schema (s : Schema) : Tree :=
  .pair (textData s.root) (sequence (s.nodes.map node))

/-- Every meaning-bearing component of the selected generated operation is encoded exactly. -/
def rpcSchema (s : RpcSchema) : Tree := sequence [
  textData s.fullName, schema s.request, schema s.response,
  sequence (s.schemaInputs.map node), boolean s.clientStreaming, boolean s.serverStreaming]

/-- Exact versioned tree bytes rendered in the existing catalog's identifier alphabet. -/
def key (data : Tree) : String :=
  "parameterized-v1-" ++ String.intercalate "_" ((Encoding.encode data).map fun b => toString b.toNat)

end Umpire.Operation.Canonical
