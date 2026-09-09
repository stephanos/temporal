import Umpire.Value

/-!
Version 2 structural identity for parameterized operations. Tags and field positions below are
format semantics; schema descriptors and value shapes are both retained, including defaults.
The alphabet is compatible with existing finite catalog keys. No display printer supplies identity.

Identity names the selected operation directly — its fully qualified method name, both payload
signature roots, and its interaction shape — and reaches the descriptor closure those names select
through `closure`, a 256-bit structural fold. Version 1 rendered that closure as exact tree bytes,
which is O(schema bytes) in both the key and the time to build it: 27,718,530 characters at about
five seconds for one real generated WorkflowService method.

What the fold guarantees, and what it does not. Every meaning-bearing component still reaches
identity: node order, syntax, descriptor, file context, references, and value shapes with their
defaults all absorb into the digest, so a schema revision changes the key even when method and
message names do not. It is a digest and not an injection, and no bounded key can be one — the
schema space is unbounded and the key is not. The model already rests identity on that trade at
its artifact boundary, where a Behavior Fingerprint digests unbounded canonical content.

Two properties keep the trade contained. Admission never consults the digest: `checkRpc` compares
the whole `RpcSchema` structurally, so a collision cannot admit a wrong schema, only merge two
identities. And the digest is derived from the closure rather than supplied beside it, so it cannot
disagree with the schema it names — a generator-emitted field would be independent data, which is
the provenance `checkRpc` exists to refuse.

`closure` folds through `String.hash`, a deterministic compiled primitive rather than a kernel
reducible one, so identity is observed by evaluation — `#guard`, `#eval`, `native_decide` — and no
longer reduces under `decide`. Version 1 lost that too for any real schema, whose closure needs
millions of kernel steps; the pinned toolchain keeps the fold reproducible across runs. A pure-Lean
SHA-256 over the same closure measures 3.95 s/MB, four orders of magnitude past the bound this
identity has to hold.

Migration `parameterized-operation-identity-v2` supersedes version 1. The version prefix changes
with the meaning, so a version 1 key never reads as a version 2 key.
-/
namespace Umpire.Operation.Canonical
open Value Value.Encoding

private def boolean (b : Bool) : Tree := .atom (if b then 1 else 0)

/-- A 256-bit fold of exact structural content, carried as four independently mixed lanes. -/
structure Digest where
  lane0 : UInt64
  lane1 : UInt64
  lane2 : UInt64
  lane3 : UInt64
  deriving BEq, DecidableEq, Repr

namespace Digest

private def start : Digest :=
  ⟨14695981039346656037, 11400714819323198485, 14029467366897019727, 1609587929392839161⟩

/-- Absorb one 64-bit word into every lane. Each lane mixes the preceding lane under its own odd
multiplier, so the lanes stay independent and the fold stays order sensitive. -/
private def step (d : Digest) (word : UInt64) : Digest :=
  ⟨(d.lane0 ^^^ word) * 1099511628211,
   (d.lane1 ^^^ (word + d.lane0)) * 11400714819323198485,
   (d.lane2 ^^^ (word + d.lane1)) * 14029467366897019727,
   (d.lane3 ^^^ (word + d.lane2)) * 1609587929392839161⟩

private def tag (d : Digest) (n : Nat) : Digest := d.step n.toUInt64

/-- Absorb a string as its content hash, the hashes of its two halves, and its exact byte length.
Reading content through `String.hash` is what keeps the fold affordable on a megabyte closure;
halving it means a difference has to collide in the half that contains it, at the same size. -/
private def text (d : Digest) (value : String) : Digest :=
  let half := value.length / 2
  (((d.step value.hash).step (value.take half).hash).step (value.drop half).hash).tag
    value.utf8ByteSize

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
    .atom digest.lane0.toNat, .atom digest.lane1.toNat,
    .atom digest.lane2.toNat, .atom digest.lane3.toNat,
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
  obtain ⟨name, request, response, lane0, lane1, lane2, lane3, client, server⟩ := components
  refine ⟨textData_inj name, textData_inj request, textData_inj response, ?_,
    boolean_inj client, boolean_inj server⟩
  have lane0 : (closure s).lane0 = (closure t).lane0 := UInt64.toNat_inj.mp lane0
  have lane1 : (closure s).lane1 = (closure t).lane1 := UInt64.toNat_inj.mp lane1
  have lane2 : (closure s).lane2 = (closure t).lane2 := UInt64.toNat_inj.mp lane2
  have lane3 : (closure s).lane3 = (closure t).lane3 := UInt64.toNat_inj.mp lane3
  cases hs : closure s
  cases ht : closure t
  simp_all

end Umpire.Operation.Canonical
