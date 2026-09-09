import Umpire.Property.Evaluation

/-!
Operation-local keyed field captures.

A scoped clause declares named captures (`PropertyScopedCapture`); this module is the retained
state one operation keeps for them. Each declared name numbers its occurrences from zero in
admission order, and a recorded ordinal is never rewritten: a later occurrence becomes a new
ordinal rather than an implicit latest-match replacement. Reading is therefore deterministic —
a correlation operand names an exact earlier occurrence, and an ordinal that has not occurred yet
simply has no evidence, so its admission fails rather than binding the nearest match.

The store holds admitted projections re-keyed to their occurrence, which is exactly the shape a
checked predicate's field operands resolve against, so no second evaluator reads captured values.
-/

namespace Umpire.Property.Scoped

/-- Why a declared capture could not retain this step's evidence. -/
inductive CaptureError where
  /-- The step supplied more than one projection at the capture's exact declared coordinates. -/
  | ambiguous (name : DefinitionId)
  /-- The operation has already retained every occurrence the declared lifetime allows. -/
  | exhausted (name : DefinitionId)
  deriving BEq, DecidableEq, Repr

/-- One operation's retained captures, in the order they were admitted. Its representation is
hidden so retained occurrences can only be appended, never replaced or reordered. -/
structure Captures where
  private mk ::
  private entries : List PropertyFieldEvidence

/-- An operation retains nothing before its first admitted step. -/
def Captures.empty : Captures := ⟨[]⟩

/-- Every retained occurrence, keyed so a correlation operand resolves to exactly its ordinal. -/
def Captures.evidence (captures : Captures) : List PropertyFieldEvidence := captures.entries

/-- How many occurrences of one declared capture this operation has already retained; the next
occurrence takes this ordinal. -/
def Captures.count (captures : Captures) (name : DefinitionId) : Nat :=
  (captures.entries.filter fun value => value.path.capture.any (·.name == name)).length

/-- Retain one declared capture from this step. A step that supplies no projection at the declared
coordinates simply records nothing; two projections at those coordinates are ambiguous and reject. -/
private def Captures.retain (evidence : List PropertyFieldEvidence)
    (state : Captures × Nat) (declaration : PropertyScopedCapture) :
    Except CaptureError (Captures × Nat) :=
  match evidence.filter fun value => value.path == declaration.path with
  | [] => .ok state
  | [value] =>
      if state.1.count declaration.name ≥ declaration.lifetime then
        .error (.exhausted declaration.name)
      else
        .ok (⟨state.1.entries ++
          [value.capturedAs ⟨declaration.name, state.1.count declaration.name⟩]⟩, state.2 + 1)
  | _ => .error (.ambiguous declaration.name)

private def Captures.retainAll (evidence : List PropertyFieldEvidence) :
    List PropertyScopedCapture → Captures × Nat → Except CaptureError (Captures × Nat)
  | [], state => .ok state
  | declaration :: rest, state =>
      match Captures.retain evidence state declaration with
      | .ok next => Captures.retainAll evidence rest next
      | .error error => .error error

/-- Retain this step's occurrence of every declared capture, reporting how many values were newly
retained so the caller can charge them against its declared budget. The whole record fails closed:
a rejected capture publishes no partial state. -/
def Captures.record (captures : Captures) (declarations : List PropertyScopedCapture)
    (evidence : List PropertyFieldEvidence) : Except CaptureError (Captures × Nat) :=
  Captures.retainAll evidence declarations (captures, 0)

private theorem Captures.retain_extends (evidence : List PropertyFieldEvidence)
    (state next : Captures × Nat) (declaration : PropertyScopedCapture)
    (retained : Captures.retain evidence state declaration = .ok next) :
    ∃ added, next.1.evidence = state.1.evidence ++ added := by
  unfold Captures.retain at retained
  split at retained
  · exact ⟨[], by rw [← Except.ok.inj retained]; simp⟩
  · split at retained
    · exact absurd retained (by simp)
    · exact ⟨_, by rw [← Except.ok.inj retained]; rfl⟩
  · exact absurd retained (by simp)

private theorem Captures.retainAll_extends (evidence : List PropertyFieldEvidence)
    (declarations : List PropertyScopedCapture) (state next : Captures × Nat)
    (retained : Captures.retainAll evidence declarations state = .ok next) :
    ∃ added, next.1.evidence = state.1.evidence ++ added := by
  induction declarations generalizing state with
  | nil => exact ⟨[], by rw [← Except.ok.inj retained]; simp⟩
  | cons declaration rest ih =>
      rw [Captures.retainAll] at retained
      split at retained
      · rename_i intermediate step
        obtain ⟨head, extended⟩ :=
          Captures.retain_extends evidence state intermediate declaration step
        obtain ⟨tail, remaining⟩ := ih intermediate retained
        exact ⟨head ++ tail, by rw [remaining, extended, List.append_assoc]⟩
      · exact absurd retained (by simp)

/-- Retaining later occurrences only extends the store: an ordinal already recorded keeps the exact
value it was admitted with, so repeated triggers read independent immutable captures. -/
theorem Captures.record_extends (captures : Captures) (declarations : List PropertyScopedCapture)
    (evidence : List PropertyFieldEvidence) (next : Captures) (charged : Nat)
    (recorded : captures.record declarations evidence = .ok (next, charged)) :
    ∃ added, next.evidence = captures.evidence ++ added :=
  Captures.retainAll_extends evidence declarations (captures, 0) (next, charged) recorded

end Umpire.Property.Scoped
