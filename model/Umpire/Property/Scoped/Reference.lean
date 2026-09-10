import Umpire.Property.Scoped.Kernel

/-! Positional correspondence between countdown windows and bounded temporal witnesses. -/

namespace Umpire.Property.Scoped

private theorem window_witness (bound : Nat) (coordinates : List Coordinate) :
    ((coordinates.take (bound + 1)).any (·.response) = true) ↔
      ∃ j, ∃ h : j < coordinates.length, j ≤ bound ∧ coordinates[j].response = true := by
  constructor
  · intro found
    obtain ⟨point, member, response⟩ := List.any_eq_true.mp found
    obtain ⟨j, index, value⟩ := List.mem_take_iff_getElem.mp member
    exact ⟨j, by omega, by omega, by simpa [value] using response⟩
  · rintro ⟨j, index, bounded, response⟩
    exact List.any_eq_true.mpr ⟨coordinates[j],
      List.mem_take_iff_getElem.mpr ⟨j, by omega, rfl⟩, response⟩

/-- Independent positional meaning: every trigger has an inclusive later response. -/
def PositionalMeaning (bound : Nat) (coordinates : List Coordinate) : Prop :=
  ∀ i, ∀ hi : i < coordinates.length, coordinates[i].trigger = true →
    ∃ j, ∃ hj : j < coordinates.length,
      i ≤ j ∧ j - i ≤ bound ∧ coordinates[j].response = true

/-- The suffix reference is exactly the usual quantified bounded-position formula. -/
theorem closedReference_positions (bound : Nat) (coordinates : List Coordinate) :
    closedReference bound coordinates = true ↔ PositionalMeaning bound coordinates := by
  induction coordinates with
  | nil => simp [Shared.ScopedObligation.closedReference, PositionalMeaning]
  | cons point rest ih =>
      change Shared.ScopedObligation.closedReference bound (point :: rest) = true ↔ _
      rw [Shared.ScopedObligation.closedReference, Bool.and_eq_true]
      constructor
      · rintro ⟨head, tail⟩ i hi trigger
        cases i with
        | zero =>
            have found : (((point :: rest).take (bound + 1)).any (·.response)) = true := by
              simpa [show point.trigger = true from trigger] using head
            obtain ⟨j, hj, bounded, response⟩ := (window_witness bound _).mp found
            exact ⟨j, hj, Nat.zero_le _, by simpa using bounded, response⟩
        | succ i =>
            obtain ⟨j, hj, ordered, bounded, response⟩ := ih.mp tail i (by simpa using hi) trigger
            exact ⟨j + 1, by simpa using hj, by omega, by simpa using bounded, response⟩
      · intro meaning
        constructor
        · cases triggered : point.trigger with
          | false => simp
          | true =>
              obtain ⟨j, hj, _, bounded, response⟩ := meaning 0 (by simp) triggered
              have found := (window_witness bound (point :: rest)).mpr
                ⟨j, hj, by simpa using bounded, response⟩
              simpa using found
        · apply ih.mpr
          intro i hi trigger
          obtain ⟨j, hj, ordered, bounded, response⟩ :=
            meaning (i + 1) (by simpa using hi) trigger
          cases j with
          | zero => omega
          | succ j => exact ⟨j, by simpa using hj, by omega, by simpa using bounded, response⟩

/-- The executable independent-obligation kernel has quantified bounded-position meaning. -/
theorem consumeMany_positions (bound : Nat) (coordinates : List Coordinate) :
    (consumeMany bound [] coordinates).all (fun obligation => decide (obligation = .satisfied)) = true ↔
      PositionalMeaning bound coordinates := by
  rw [closed_agrees]
  exact closedReference_positions bound coordinates

private theorem mem_positions (start : Nat) (bits : List Bool) (position : Nat) :
    position ∈ positions start bits ↔
      ∃ i, ∃ hi : i < bits.length, position = start + i ∧ bits[i] = true := by
  induction bits generalizing start with
  | nil => simp [positions]
  | cons hit rest ih =>
      simp only [positions, List.mem_append]
      constructor
      · intro member
        rcases member with head | tail
        · cases hit with
          | false => simp at head
          | true =>
              have same : position = start := by simpa using head
              exact ⟨0, by simp, by simpa using same, rfl⟩
        · obtain ⟨i, hi, same, found⟩ := (ih (start + 1)).mp tail
          exact ⟨i + 1, by simpa using hi, by omega, found⟩
      · rintro ⟨i, hi, same, found⟩
        cases i with
        | zero =>
            left
            simp only [List.getElem_cons_zero] at found
            simp [found, same]
        | succ i =>
            right
            exact (ih (start + 1)).mpr ⟨i, by simpa using hi, by omega, found⟩

private theorem positions_meaning (start bound : Nat) (coordinates : List Coordinate) :
    ((positions start (coordinates.map (·.trigger))).all fun first =>
      (positions start (coordinates.map (·.response))).any fun second =>
        first ≤ second && second - first ≤ bound) = true ↔ PositionalMeaning bound coordinates := by
  rw [List.all_eq_true]
  constructor
  · intro meaning i hi trigger
    have first := (mem_positions start (coordinates.map (·.trigger)) (start + i)).mpr
      ⟨i, by simpa using hi, rfl, by simpa using trigger⟩
    obtain ⟨second, member, answer⟩ := List.any_eq_true.mp (meaning _ first)
    obtain ⟨j, hj, same, response⟩ := (mem_positions start _ second).mp member
    simp only [Bool.and_eq_true, decide_eq_true_eq] at answer
    exact ⟨j, by simpa using hj, by omega, by omega, by simpa using response⟩
  · intro meaning first member
    obtain ⟨i, hi, same, trigger⟩ := (mem_positions start _ first).mp member
    obtain ⟨j, hj, ordered, bounded, response⟩ := meaning i (by simpa using hi) (by simpa using trigger)
    apply List.any_eq_true.mpr
    refine ⟨start + j, ?_, ?_⟩
    · exact (mem_positions start _ _).mpr ⟨j, by simpa using hj, rfl, by simpa using response⟩
    · simp only [Bool.and_eq_true, decide_eq_true_eq]
      exact ⟨by omega, by omega⟩

/-- Kernel countdowns agree with the existing checked `eventuallyWithin` evaluator on its
actual admitted action/outcome coordinates. The existing evaluator remains the reference authority. -/
theorem checked_eventuallyWithin_agrees
    (property : CheckedProperty) (input : CheckedPropertyEvaluationInput property)
    (clause : { clause // clause ∈ property.clauses })
    (id : DefinitionId) (trigger response : PropertyPattern) (bound : Nat)
    (shape : clause.val = .eventuallyWithin id trigger response ⟨bound, .semanticTransitions⟩)
    (triggerAligned : trigger.field = .selectedAction)
    (responseAligned : response.field = .outcome ∨ response.field = .resultingState ∨
      response.field = .observation) :
    (consumeMany bound [] ((input.scopedCoordinates trigger response).map
      (fun point => Coordinate.mk point.1 point.2))).all
        (fun obligation => decide (obligation = .satisfied)) =
      evaluatePropertyClause property input clause := by
  apply Bool.eq_iff_iff.mpr
  rw [consumeMany_positions, evaluatePropertyClause_scoped_positions property input clause
    id trigger response bound shape triggerAligned responseAligned]
  have agreement := positions_meaning 1 bound ((input.scopedCoordinates trigger response).map
    (fun point => Coordinate.mk point.1 point.2))
  simpa [List.map_map, Function.comp_def] using agreement.symm

end Umpire.Property.Scoped
