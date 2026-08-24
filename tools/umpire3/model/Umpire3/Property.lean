import Umpire3.Transition

namespace Umpire3

def Safety (model : TransitionSystem) (property : model.State → Prop) : Prop :=
  ∀ state, model.Reachable state → property state

theorem Safety.initial (safety : Safety model property)
    (initial : model.Initial state) : property state := by
  apply safety state
  exact ⟨state, [], initial, Runs.nil state⟩

end Umpire3
