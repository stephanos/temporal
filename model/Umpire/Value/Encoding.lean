import Std

/-!
A closed, prefix-coded binary tree format for exact structural values. Tags 0/1/2 mean empty,
natural atom, and pair. Naturals use little-endian binary digits 1/2 followed by 0, so even large
integer boundaries occupy bounded space. Decoding consumes a caller-supplied node budget.
The suffix law proves correspondence for every tree, independently of value admission.
-/
namespace Umpire.Value.Encoding

/-- Inert serialization data, with no schema or semantic authority. -/
inductive Tree where
  | nil
  | atom (value : Nat)
  | pair (left right : Tree)
  deriving BEq, DecidableEq, Repr

/-- Number of serialized tree nodes consumed by decoding. -/
def Tree.nodes : Tree → Nat
  | .nil | .atom _ => 1
  | .pair a b => 1 + a.nodes + b.nodes

/-- Prefix encoding of an arbitrary natural without machine-integer narrowing. -/
def encodeNat (n : Nat) : List UInt8 :=
  if n = 0 then [0]
  else (if n % 2 = 0 then 1 else 2) :: encodeNat (n / 2)
termination_by n

/-- Parse one natural and retain the unconsumed suffix. -/
def decodeNat : List UInt8 → Option (Nat × List UInt8)
  | [] => none
  | x :: xs =>
    if x = 0 then some (0, xs)
    else if x = 1 then (decodeNat xs).map fun (n, rest) => (2 * n, rest)
    else if x = 2 then (decodeNat xs).map fun (n, rest) => (2 * n + 1, rest)
    else none

/-- Encode an entire exact tree. -/
def encode : Tree → List UInt8
  | .nil => [0]
  | .atom n => 1 :: encodeNat n
  | .pair a b => 2 :: (encode a ++ encode b)

/-- Decode atomically, returning the unconsumed bytes under a structural depth budget. -/
def decode : Nat → List UInt8 → Option (Tree × List UInt8)
  | 0, _ => none
  | fuel + 1, input =>
    match input with
    | [] => none
    | tag :: rest =>
      if tag = 0 then some (.nil, rest)
      else if tag = 1 then (decodeNat rest).map fun (n, tail) => (.atom n, tail)
      else if tag = 2 then do
        let (a, tail) ← decode fuel rest
        let (b, tail) ← decode fuel tail
        pure (.pair a b, tail)
      else none

/-- Natural decoding reverses the actual byte encoder, for every suffix. -/
theorem decodeNat_encodeNat (n : Nat) (rest : List UInt8) :
    decodeNat (encodeNat n ++ rest) = some (n, rest) := by
  induction n using Nat.strongRecOn with
  | ind n ih =>
    by_cases zero : n = 0
    · subst n
      simp [encodeNat, decodeNat]
    · have smaller : n / 2 < n := by omega
      have prior :=  ih (n / 2) smaller
      by_cases even : n % 2 = 0
      · have value : 2 * (n / 2) = n := by omega
        rw [encodeNat]
        simp only [if_neg zero, if_pos even, List.cons_append]
        simp [decodeNat, prior, value]
      · have value : 2 * (n / 2) + 1 = n := by omega
        rw [encodeNat]
        simp only [if_neg zero, if_neg even, List.cons_append]
        simp [decodeNat, prior, value]

/-- Every tree is recovered exactly by the byte decoder under a sufficient structural budget. -/
theorem decode_encode (tree : Tree) (fuel : Nat) (rest : List UInt8)
    (enough : tree.nodes ≤ fuel) : decode fuel (encode tree ++ rest) = some (tree, rest) := by
  induction tree generalizing fuel rest with
  | nil =>
    cases fuel with
    | zero => simp [Tree.nodes] at enough
    | succ fuel => simp [encode, decode]
  | atom n =>
    cases fuel with
    | zero => simp [Tree.nodes] at enough
    | succ fuel => simp [encode, decode, decodeNat_encodeNat]
  | pair a b left right =>
    cases fuel with
    | zero => simp [Tree.nodes] at enough
    | succ fuel =>
      have ha : a.nodes ≤ fuel := by simp [Tree.nodes] at enough; omega
      have hb : b.nodes ≤ fuel := by simp [Tree.nodes] at enough; omega
      simp [encode, decode, List.append_assoc, left fuel (encode b ++ rest) ha,
        right fuel rest hb]

end Umpire.Value.Encoding
