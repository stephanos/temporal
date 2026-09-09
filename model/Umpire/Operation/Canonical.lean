import Umpire.Value

/-!
Version 2 structural identity for parameterized operations. Tags and field positions below are
format semantics; schema descriptors and value shapes are both retained, including defaults.
The alphabet is compatible with existing finite catalog keys. No display printer supplies identity.

Identity names the selected operation directly — its fully qualified method name, both payload
signature roots, and its interaction shape — and reaches the descriptor closure those names select
through `closure`, a 128-bit structural fold. Version 1 rendered that closure as exact tree bytes,
which is O(schema bytes) in both the key and the time to build it: 27,718,530 characters at about
five seconds for one real `Temporal.API` method. Folding keeps every component meaning-bearing,
because every one of them still reaches the fold, while the key itself stays bounded.

`closure` folds through `String.hash`, a deterministic compiled primitive rather than a kernel
reducible one, so identity is observed by evaluation — `#guard`, `#eval`, `native_decide` — and no
longer reduces under `decide`. Version 1 lost that too for any real schema, whose closure needs
millions of kernel steps; the pinned toolchain keeps the fold reproducible across runs.

Migration `parameterized-operation-identity-v2` supersedes version 1. The version prefix changes
with the meaning, so a version 1 key never reads as a version 2 key.
-/
namespace Umpire.Operation.Canonical
open Value Value.Encoding

private def boolean (b : Bool) : Tree := .atom (if b then 1 else 0)

/-- A 128-bit fold of exact structural content, carried as two independently mixed lanes. -/
structure Digest where
  low : UInt64
  high : UInt64
  deriving BEq, DecidableEq, Repr

namespace Digest

private def start : Digest := ⟨14695981039346656037, 11400714819323198485⟩

/-- Absorb one 64-bit word into both lanes. Lane constants differ, so the lanes stay independent. -/
private def step (d : Digest) (word : UInt64) : Digest :=
  ⟨(d.low ^^^ word) * 1099511628211, (d.high ^^^ (word + d.low)) * 11400714819323198485⟩

private def tag (d : Digest) (n : Nat) : Digest := d.step n.toUInt64

/-- Absorb a string by its content hash and its exact byte length, so equal hashes still separate
strings of different size. -/
private def text (d : Digest) (value : String) : Digest :=
  (d.step value.hash).tag value.utf8ByteSize

private def integer (d : Digest) (value : Int) : Digest := d.tag (integerData value)

private def option (absorb : Digest → α → Digest) (d : Digest) : Option α → Digest
  | none => d.tag 0
  | some value => absorb (d.tag 1) value

private def scalar (d : Digest) : Scalar → Digest
  | .boolean value => (d.tag 0).tag (if value then 1 else 0)
  | .text value => (d.tag 1).text value
  | .bytes value => value.foldl (fun acc b => acc.tag b.toNat) (d.tag 2)
  | .integer kind value => ((d.tag 3).tag (integerKinds.idxOf kind)).integer value
  | .enumeration name number => ((d.tag 4).text name).integer number
  | .floating double bits => ((d.tag 5).tag (if double then 1 else 0)).tag bits

private def singular (d : Digest) : Singular → Digest
  | .boolean => d.tag 0
  | .text => d.tag 1
  | .bytes => d.tag 2
  | .integer k => (d.tag 3).tag (integerKinds.idxOf k)
  | .enumeration n => (d.tag 4).text n
  | .message n => (d.tag 5).text n
  | .floating double => (d.tag 6).tag (if double then 1 else 0)
  | .unsupported r => (d.tag 7).text r

private def cardinality (d : Digest) : Cardinality → Digest
  | .singular => d.tag 0
  | .repeated => d.tag 1
  | .map key => (d.tag 2).singular key

private def presence (d : Digest) : Presence → Digest
  | .implicit value => (d.tag 0).scalar value
  | .optional => d.tag 1
  | .required => d.tag 2
  | .oneof name => (d.tag 3).text name

private def field (d : Digest) (f : ValueField) : Digest :=
  option scalar
    (((((d.tag f.number).text f.name).singular f.type).cardinality f.cardinality).presence f.presence)
    f.defaultValue

private def shape (d : Digest) : ValueShape → Digest
  | .message fields unsupported => option text (fields.foldl field (d.tag 0)) unsupported
  | .enumeration closed numbers =>
      numbers.foldl integer ((d.tag 1).tag (if closed then 1 else 0))

private def node (d : Digest) (n : SchemaNode) : Digest :=
  option shape
    (n.references.foldl text ((((d.text n.name).text n.protoSyntax).text n.descriptor).text n.fileContext))
    n.valueShape

private def schema (d : Digest) (s : Schema) : Digest := s.nodes.foldl node (d.text s.root)

end Digest

/-- Exact structural content of the operation's complete descriptor closure: the method name, both
payload closures in node order, the shared input closure, and every node's syntax, descriptor, file
context, references and value shape, defaults included. Changing any of them changes the digest. -/
def closure (s : RpcSchema) : Digest :=
  s.schemaInputs.foldl Digest.node
    (Digest.schema (Digest.schema (Digest.text Digest.start s.fullName) s.request) s.response)

/-- The selected generated operation: its fully qualified method name, the roots of its request and
response signatures, the digest of its descriptor closure, and its client and server streaming
shape. -/
def rpcSchema (s : RpcSchema) : Tree :=
  let digest := closure s
  sequence [textData s.fullName, textData s.request.root, textData s.response.root,
    .atom digest.low.toNat, .atom digest.high.toNat,
    boolean s.clientStreaming, boolean s.serverStreaming]

/-- Exact versioned tree bytes rendered in the existing catalog's identifier alphabet. -/
def key (data : Tree) : String :=
  "parameterized-v2-" ++ String.intercalate "_" ((Encoding.encode data).map fun b => toString b.toNat)

private theorem sequence_inj : ∀ (a b : List Tree), sequence a = sequence b → a = b
  | [], [], _ => rfl
  | [], _ :: _, same => by simp [sequence] at same
  | _ :: _, [], same => by simp [sequence] at same
  | _ :: as, _ :: bs, same => by
    simp only [sequence, List.foldr_cons, Tree.pair.injEq] at same
    rw [same.1, sequence_inj as bs same.2]

private theorem atoms_inj : ∀ (a b : List Char),
    a.map (fun c => Tree.atom c.toNat) = b.map (fun c => Tree.atom c.toNat) → a = b
  | [], [], _ => rfl
  | [], _ :: _, same => by simp at same
  | _ :: _, [], same => by simp at same
  | _ :: as, _ :: bs, same => by
    simp only [List.map_cons, List.cons.injEq, Tree.atom.injEq] at same
    rw [Char.toNat_inj.mp same.1, atoms_inj as bs same.2]

private theorem textData_inj {a b : String} (same : textData a = textData b) : a = b :=
  String.toList_inj.mp (atoms_inj a.toList b.toList (sequence_inj _ _ same))

private theorem boolean_inj {a b : Bool} (same : boolean a = boolean b) : a = b := by
  cases a <;> cases b <;> first | rfl | simp [boolean] at same

/-- Identity retains every component it names: two operations with the same identity have the same
method name, the same request and response signature roots, the same descriptor closure digest, and
the same client and server streaming flags. Closure content differs in the key exactly when it
differs in `closure`. -/
theorem rpcSchema_inj {s t : RpcSchema} (same : rpcSchema s = rpcSchema t) :
    s.fullName = t.fullName ∧ s.request.root = t.request.root ∧
      s.response.root = t.response.root ∧ closure s = closure t ∧
      s.clientStreaming = t.clientStreaming ∧ s.serverStreaming = t.serverStreaming := by
  have components := sequence_inj _ _ same
  simp only [List.cons.injEq, Tree.atom.injEq, and_true] at components
  obtain ⟨name, request, response, low, high, client, server⟩ := components
  refine ⟨textData_inj name, textData_inj request, textData_inj response, ?_,
    boolean_inj client, boolean_inj server⟩
  have low : (closure s).low = (closure t).low := UInt64.toNat_inj.mp low
  have high : (closure s).high = (closure t).high := UInt64.toNat_inj.mp high
  cases hs : closure s
  cases ht : closure t
  simp_all

end Umpire.Operation.Canonical
