import Umpire3.Execution
import Umpire3.FiniteView
import Shared.TraceReplay

namespace Umpire3

namespace TraceReplay

def followNamed {model : Behavior World} {world : World}
    (view : FiniteView model world) :
    List (model.State world) → List String → List (model.State world) :=
  Shared.TraceReplay.followNamed view.successors view.actionName

def check {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (identifiers : List String) : Bool :=
  Shared.TraceReplay.check view.successors view.actionName view.initials property identifiers

theorem followNamed_sound {model : Behavior World} {world : World}
    (view : FiniteView model world) {states : List (model.State world)}
    {identifiers : List String} {final : model.State world}
    (member : final ∈ followNamed view states identifiers) :
    ∃ initial actions, initial ∈ states ∧ Runs (model.at world) initial actions final := by
  induction identifiers generalizing states with
  | nil =>
      exact ⟨final, [], member, Runs.nil (model := model.at world) final⟩
  | cons identifier identifiers inductionHypothesis =>
      change final ∈ followNamed view
        (states.flatMap fun state =>
          (view.successors state).filterMap fun successor =>
            if view.actionName successor.1 = identifier then some successor.2 else none)
        identifiers at member
      rcases inductionHypothesis member with ⟨nextState, actions, nextMember, tail⟩
      rcases List.mem_flatMap.mp nextMember with ⟨state, stateMember, filteredMember⟩
      rcases List.mem_filterMap.mp filteredMember with
        ⟨successor, successorMember, selected⟩
      split at selected
      · have nextEquality : successor.2 = nextState := by simpa using selected
        subst nextState
        have step := (view.executable.successors_exact world state successor.1 successor.2).mp
          successorMember
        exact ⟨state, successor.1 :: actions, stateMember, Runs.cons step tail⟩
      · simp at selected

theorem checked {model : Behavior World} {world : World}
    (view : FiniteView model world) (property : model.State world → Bool)
    (identifiers : List String) (accepted : check view property identifiers = true) :
    ∃ state, model.Reachable world state ∧ property state = false := by
  simp only [check, Shared.TraceReplay.check, List.any_eq_true,
    Bool.not_eq_true'] at accepted
  rcases accepted with ⟨state, member, violated⟩
  rcases followNamed_sound view member with ⟨initial, actions, initialMember, run⟩
  exact ⟨state, ⟨initial, actions,
    (view.executable.initials_exact world initial).mp initialMember, run⟩, violated⟩

end TraceReplay

end Umpire3
