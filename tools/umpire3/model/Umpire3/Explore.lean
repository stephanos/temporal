namespace Umpire3

structure FiniteHole (α : Type) where
  identifier : String
  values : List α

def assignments : List (FiniteHole α) → List (List α)
  | [] => [[]]
  | hole :: rest => hole.values.flatMap fun value =>
      (assignments rest).map (value :: ·)

def canonicalSymmetry [Ord α] (assignment : List α) : Bool :=
  assignment.Pairwise (fun left right => compare left right != .gt)

def reduceSymmetry [Ord α] (candidates : List (List α)) : List (List α) :=
  candidates.filter canonicalSymmetry

def independentPairs (independent : α → α → Bool) (trace : List α) : Nat :=
  (trace.zip (trace.drop 1)).countP fun pair => independent pair.1 pair.2

theorem assignments_empty : assignments ([] : List (FiniteHole Nat)) = [[]] := by rfl

theorem reduced_candidates_come_from_input [Ord α] {candidate : List α} {candidates : List (List α)}
    (member : candidate ∈ reduceSymmetry candidates) : candidate ∈ candidates := by
  simpa [reduceSymmetry] using (List.mem_filter.mp member).1

end Umpire3
