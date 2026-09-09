import Umpire.Value

/-!
Version 2 structural identity for parameterized operations. Tags and field positions below are
format semantics. The alphabet is compatible with existing finite catalog keys. No display printer
supplies identity.

Identity names the operation a payload belongs to: its fully qualified method name, the roots of
both payload signatures, and its interaction shape. It does not re-encode the descriptor closure
those names select. Every schema reaching a key is `owner.schema reference` for a `CheckedRpc`
binding, whose `agrees` field proves that selection at admission, and a generated catalog names a
method once; the closure is therefore a consequence of the selection the key already records
rather than an independent component of it.

Migration `parameterized-operation-identity-v2` supersedes version 1, which rendered the exact
encoded closure and produced a 27,718,530-character key for one real `Temporal.API` method. The
version prefix changes with the meaning, so a version 1 key never reads as a version 2 key.
-/
namespace Umpire.Operation.Canonical
open Value Value.Encoding

private def boolean (b : Bool) : Tree := .atom (if b then 1 else 0)

/-- The selected generated operation: its fully qualified method name, the roots of its request and
response signatures, and its client and server streaming shape. -/
def rpcSchema (s : RpcSchema) : Tree := sequence [
  textData s.fullName, textData s.request.root, textData s.response.root,
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

/-- Identity retains exactly the five components it names: two operations with the same identity
have the same method name, the same request and response signature roots, and the same client and
server streaming flags. Operations differing in any of them receive different keys. -/
theorem rpcSchema_inj {s t : RpcSchema} (same : rpcSchema s = rpcSchema t) :
    s.fullName = t.fullName ∧ s.request.root = t.request.root ∧
      s.response.root = t.response.root ∧
      s.clientStreaming = t.clientStreaming ∧ s.serverStreaming = t.serverStreaming := by
  have components := sequence_inj _ _ same
  simp only [List.cons.injEq, and_true] at components
  obtain ⟨name, request, response, client, server⟩ := components
  exact ⟨textData_inj name, textData_inj request, textData_inj response,
    boolean_inj client, boolean_inj server⟩

end Umpire.Operation.Canonical
