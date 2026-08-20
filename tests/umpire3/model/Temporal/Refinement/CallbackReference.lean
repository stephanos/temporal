import Temporal.System.CallbackReference
import Umpire3.Refinement

namespace Umpire3.Temporal.System.CallbackReference

def Projects (systemState : State)
    (productState : Umpire3.Temporal.Product.CallbackReference.State) : Prop :=
  systemState.visible = productState

theorem initialProjects (systemState : State) (isInitial : system.Initial systemState) :
    ∃ productState, Umpire3.Temporal.Product.CallbackReference.model.Initial productState ∧
      Projects systemState productState := by
  subst systemState
  exact ⟨Umpire3.Temporal.Product.CallbackReference.initial, rfl, rfl⟩

theorem stepSimulates (systemState : State)
    (productState : Umpire3.Temporal.Product.CallbackReference.State)
    (action : Action) (nextSystemState : State)
    (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    ∃ nextProductState, Umpire3.Temporal.Product.CallbackReference.model.StepStar
      productState nextProductState ∧ Projects nextSystemState nextProductState := by
  rcases systemStep with ⟨result, _, rfl⟩
  subst productState
  exact ⟨result.nextState.visible, ⟨result.productActions, result.productRun⟩, rfl⟩

def callbackReferenceRefinesProduct :
    Refinement system Umpire3.Temporal.Product.CallbackReference.model where
  Relates := Projects
  initial := initialProjects
  step := stepSimulates

end Umpire3.Temporal.System.CallbackReference
