import Testpilot.Correlated
import Umpire.Property.Correlated

/-!
Finite-row compiler certificates connect actual portable histories to checked Property inputs.
Certificates are checked once over the complete transition table. The history theorem constructs
its reference input by the same checked-input append operation as task 6; it never assumes an
unrelated input with an equal verdict and adds no runtime replay or oracle comparison.
-/
namespace Umpire.Case.CorrelatedProofs
open Property.Correlated
open Shared.CorrelatedObligation

abbrev Row := Shared.CorrelatedObligation.Transition

private def trace (row : Row) : ModelTrace ModelValue ModelValue ModelValue ModelValue :=
  ⟨row.1, [.result row.2.1 row.2.2]⟩

/-- The applicable source predicates projected from one checked reference input. -/
def coordinates (binding : PortableReference) (input : CheckedPropertyEvaluationInput binding.reference) :
    List Shared.CorrelatedObligation.Match :=
  (input.correlatedCoordinates binding.original.triggerPattern binding.original.responsePattern).map
    (fun point => ⟨point.1, point.2⟩)

private def append (binding : PortableReference)
    (first second : CheckedPropertyEvaluationInput binding.reference) :
    CheckedPropertyEvaluationInput binding.reference :=
  first.appendCorrelated second binding.original.declaration.id binding.original.triggerPattern
    binding.original.responsePattern binding.original.declaration.bound binding.shape

private theorem append_coordinates (binding : PortableReference)
    (first second : CheckedPropertyEvaluationInput binding.reference) :
    coordinates binding (append binding first second) =
      coordinates binding first ++ coordinates binding second := by
  unfold coordinates append
  rw [CheckedPropertyEvaluationInput.correlatedCoordinates_append first second binding.original.declaration.id
    binding.original.triggerPattern binding.original.responsePattern binding.original.declaration.bound binding.shape]
  exact List.map_append

/-- Every certified row is checked against its exact prior/action/result, including all facts. -/
structure Rows (binding : PortableReference) (table : List Row) : Prop where
  valid : ∀ row ∈ table, ∃ input : CheckedPropertyEvaluationInput binding.reference,
    checkPropertyEvaluationInput binding.reference (trace row) = .ok input ∧
    coordinates binding input = [binding.portable.coordinate row.2.1 row.2.2]

private def checkRows (binding : PortableReference) :
    (table : List Row) → Except String (PLift (Rows binding table))
  | [] => pure ⟨⟨by simp⟩⟩
  | row :: rest => do
      match checked : checkPropertyEvaluationInput binding.reference (trace row) with
      | .error _ => throw "unsupported checked transition input"
      | .ok input =>
          if aligned : coordinates binding input = [binding.portable.coordinate row.2.1 row.2.2] then
            let tail ← checkRows binding rest
            pure ⟨⟨by
              intro candidate member
              rcases List.mem_cons.mp member with same | member
              · subst candidate
                exact ⟨input, checked, aligned⟩
              · exact tail.down.valid candidate member⟩⟩
          else throw "portable predicate disagrees with checked Property projection"

/-- A compiler certificate starts with the actual admitted initial value and covers every table row. -/
structure Certificate (binding : PortableReference) (table : List Row) (initial : ModelValue) : Prop where
  initial : ∃ input : CheckedPropertyEvaluationInput binding.reference,
    checkPropertyEvaluationInput binding.reference ⟨initial, []⟩ = .ok input ∧ coordinates binding input = []
  rows : Rows binding table

/-- Certify the complete finite table once; unsupported source projections reject compilation. -/
def certify (binding : PortableReference) (table : List Row) (initial : ModelValue) :
    Except String (PLift (Certificate binding table initial)) := do
  let rows ← checkRows binding table
  match checked : checkPropertyEvaluationInput binding.reference ⟨initial, []⟩ with
  | .error _ => throw "unsupported initial Property input"
  | .ok input =>
      if empty : coordinates binding input = [] then
        pure ⟨⟨⟨input, checked, empty⟩, rows.down⟩⟩
      else throw "initial state produced a semantic coordinate"

/-- Certify every compiled clause against the full authorized table. -/
def certifyAll (bindings : List PortableReference) (table : List Row) (initial : ModelValue) :
    Except String (PLift (∀ binding ∈ bindings, Certificate binding table initial)) := do
  match equation : bindings with
  | [] => pure ⟨by simp [equation]⟩
  | binding :: rest =>
      let head ← certify binding table initial
      let tail ← certifyAll rest table initial
      pure ⟨by
        intro candidate member
        rw [equation] at member
        rcases List.mem_cons.mp member with same | member
        · subst candidate; exact head.down
        · exact tail.down candidate member⟩

/-- This relation records exact checked construction from the retained operation history. -/
inductive HistoryInput (binding : PortableReference) (initial : ModelValue) :
    List Row → CheckedPropertyEvaluationInput binding.reference → Prop
  | initial (input) (checked : checkPropertyEvaluationInput binding.reference ⟨initial, []⟩ = .ok input) :
      HistoryInput binding initial [] input
  | extend (history input row next)
      (before : HistoryInput binding initial history input)
      (checked : checkPropertyEvaluationInput binding.reference (trace row) = .ok next) :
      HistoryInput binding initial (history ++ [row]) (append binding input next)

/-- Exact admitted rows construct a checked reference input with the actual runtime coordinate sequence. -/
theorem Certificate.history {binding : PortableReference} {table : List Row} {initial : ModelValue}
    (certificate : Certificate binding table initial)
    (history : List (Admitted table)) :
    ∃ input : CheckedPropertyEvaluationInput binding.reference,
      HistoryInput binding initial (history.map (·.value)) input ∧
      coordinates binding input = history.map (fun row => binding.portable.coordinate row.value.2.1 row.value.2.2) := by
  have build : ∀ reversed : List (Admitted table),
      ∃ input : CheckedPropertyEvaluationInput binding.reference,
        HistoryInput binding initial (reversed.reverse.map (·.value)) input ∧
        coordinates binding input = reversed.reverse.map
          (fun row => binding.portable.coordinate row.value.2.1 row.value.2.2) := by
    intro reversed
    induction reversed with
    | nil =>
        obtain ⟨input, checked, empty⟩ := certificate.initial
        exact ⟨input, .initial input checked, empty⟩
    | cons row rest ih =>
        obtain ⟨input, before, projected⟩ := ih
        obtain ⟨next, checked, aligned⟩ := certificate.rows.valid row.value row.member
        refine ⟨append binding input next, ?_, ?_⟩
        · simpa using HistoryInput.extend (binding := binding) (initial := initial)
            (rest.reverse.map (·.value)) input row.value next before checked
        · simp [append_coordinates, projected, aligned]
  simpa using build history.reverse

/-- Actual retained portable countdowns agree with the existing checked `eventuallyWithin` authority
on an input constructed from exactly this operation's admitted history. -/
theorem Certificate.closed_property {binding : PortableReference} {table : List Row} {initial : ModelValue}
    {clauses : List Clause} {history : List (Admitted table)} (certificate : Certificate binding table initial)
    (window : Window clauses history) (same : window.clause.val = binding.portable) :
    ∃ input : CheckedPropertyEvaluationInput binding.reference,
      HistoryInput binding initial (history.map (·.value)) input ∧
      window.obligations.all (fun obligation => decide (obligation = .satisfied)) =
        evaluatePropertyClause binding.reference input binding.clause := by
  obtain ⟨input, admitted, projected⟩ := certificate.history history
  refine ⟨input, admitted, ?_⟩
  rw [window.consistent, same, ← projected]
  exact checked_eventuallyWithin_agrees binding.reference input binding.clause
    binding.original.declaration.id binding.original.triggerPattern binding.original.responsePattern
    binding.original.declaration.bound rfl binding.triggerAligned binding.responseAligned

private def endpointAnswer : Shared.CorrelatedObligation.Verdict → PropertyEndpointAnswer
  | .satisfied => .satisfied | .violated => .violated | .unresolved => .unresolved

/-- The actual portable endpoint decision agrees with task 6 for both close modes and incomplete
inputs, including preservation of any violation already established before evidence failed. -/
theorem endpoint_agrees (obligations : List Shared.CorrelatedObligation.Obligation)
    (closed incomplete : Bool) (ending : TraceEnding) :
    endpointAnswer (Shared.CorrelatedObligation.answer obligations closed incomplete
      (ending == .final)) =
    let selected := if incomplete || !closed then TraceEnding.«partial» else ending
    let result := Property.Correlated.close selected obligations
    if incomplete && result != .violated then .unresolved else result := by
  cases violated : obligations.contains .violated <;>
    cases satisfied : obligations.all (fun obligation => decide (obligation = .satisfied)) <;>
    cases closed <;> cases incomplete <;> cases ending <;>
    simp only [Shared.CorrelatedObligation.answer, Property.Correlated.close, violated, satisfied,
      Bool.false_eq_true, if_false, if_true] <;> rfl

end Umpire.Case.CorrelatedProofs
