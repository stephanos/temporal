import Temporal.System.WorkflowProgress
import Umpire3.Refinement

namespace Umpire3.Temporal.System.WorkflowProgress

def Projects (systemState : State)
    (productState : Umpire3.Temporal.Product.WorkflowProgress.State) : Prop :=
  systemState.visible = productState

theorem initialProjects (systemState : State) (isInitial : system.Initial systemState) :
    ∃ productState, Umpire3.Temporal.Product.WorkflowProgress.model.Initial productState ∧
      Projects systemState productState := by
  subst systemState
  exact ⟨Umpire3.Temporal.Product.WorkflowProgress.initial, rfl, rfl⟩

theorem stepSimulates (systemState : State)
    (productState : Umpire3.Temporal.Product.WorkflowProgress.State)
    (action : Action) (nextSystemState : State)
    (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    ∃ nextProductState, Umpire3.Temporal.Product.WorkflowProgress.model.StepStar
      productState nextProductState ∧ Projects nextSystemState nextProductState := by
  rcases systemStep with ⟨result, _, rfl⟩
  subst productState
  exact ⟨result.nextState.visible, ⟨result.productActions, result.productRun⟩, rfl⟩

def workflowProgressRefinesProduct :
    Refinement system Umpire3.Temporal.Product.WorkflowProgress.model where
  Relates := Projects
  initial := initialProjects
  step := stepSimulates

end Umpire3.Temporal.System.WorkflowProgress
