import Temporal.System.NexusClosure
import Umpire3.Refinement

namespace Umpire3.Temporal.System.NexusClosure

def Projects (systemState : State)
    (productState : Umpire3.Temporal.Product.NexusClosure.State) : Prop :=
  systemState.visible = productState

theorem initialProjects (systemState : State) (isInitial : system.Initial systemState) :
    ∃ productState, Umpire3.Temporal.Product.NexusClosure.model.Initial productState ∧
      Projects systemState productState := by
  subst systemState
  exact ⟨Umpire3.Temporal.Product.NexusClosure.initial, rfl, rfl⟩

theorem stepSimulates (systemState : State)
    (productState : Umpire3.Temporal.Product.NexusClosure.State)
    (action : Action) (nextSystemState : State)
    (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    ∃ nextProductState, Umpire3.Temporal.Product.NexusClosure.model.StepStar
      productState nextProductState ∧
      Projects nextSystemState nextProductState := by
  rcases systemStep with ⟨result, _, rfl⟩
  subst productState
  exact ⟨result.nextState.visible, ⟨result.productActions, result.productRun⟩, rfl⟩

def nexusClosureRefinesProduct :
    Refinement system Umpire3.Temporal.Product.NexusClosure.model where
  Relates := Projects
  initial := initialProjects
  step := stepSimulates

end Umpire3.Temporal.System.NexusClosure
