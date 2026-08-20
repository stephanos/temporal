import Temporal.System.UpdateTasks
import Umpire3.Refinement

namespace Umpire3.Temporal.System.UpdateTasks

def Projects (systemState : State) (productState : Temporal.Product.Update.State) : Prop :=
  systemState.visible = productState

theorem stepSimulates (systemState : State) (productState : Temporal.Product.Update.State)
    (action : Action) (nextSystemState : State) (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    ∃ nextProductState,
      Temporal.Product.Update.product.StepStar productState nextProductState ∧
      Projects nextSystemState nextProductState := by
  rcases systemStep with ⟨result, _, rfl⟩
  subst productState
  exact ⟨result.nextState.visible, ⟨result.productActions, result.productRun⟩, rfl⟩

def updateTasksRefinesProduct : Refinement system Temporal.Product.Update.product where
  Relates := Projects
  initial := by
    intro state initialState
    subst state
    exact ⟨Temporal.Product.Update.initial, rfl, rfl⟩
  step := stepSimulates

end Umpire3.Temporal.System.UpdateTasks
