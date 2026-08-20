import Umpire3.Transition

namespace Umpire3

structure Refinement (system product : TransitionSystem) where
  Relates : system.State → product.State → Prop
  initial : ∀ systemState,
    system.Initial systemState →
    ∃ productState, product.Initial productState ∧ Relates systemState productState
  step : ∀ systemState productState action nextSystemState,
    Relates systemState productState →
    system.Step systemState action nextSystemState →
    ∃ nextProductState,
      product.StepStar productState nextProductState ∧
      Relates nextSystemState nextProductState

def Refinement.Stutters (refinement : Refinement system product)
    (systemState : system.State) (productState : product.State)
    (action : system.Action) (nextSystemState : system.State) : Prop :=
  refinement.Relates systemState productState ∧
    system.Step systemState action nextSystemState ∧
    refinement.Relates nextSystemState productState

def Refinement.identity (model : TransitionSystem) : Refinement model model where
  Relates := Eq
  initial := by
    intro state initial
    exact ⟨state, initial, rfl⟩
  step := by
    intro state productState action nextState related step
    subst productState
    exact ⟨nextState, model.stepStarSingle step, rfl⟩

end Umpire3
