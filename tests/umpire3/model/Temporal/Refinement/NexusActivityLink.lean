import Temporal.System.NexusActivityLink
import Umpire3.Refinement

namespace Umpire3.Temporal.System.NexusActivityLink

def Projects (systemState : State)
    (productState : Umpire3.Temporal.Product.NexusActivityLink.State) : Prop :=
  systemState.visible = productState

theorem initialProjects (systemState : State) (isInitial : system.Initial systemState) :
    ∃ productState, Umpire3.Temporal.Product.NexusActivityLink.model.Initial productState ∧
      Projects systemState productState := by
  subst systemState
  exact ⟨Umpire3.Temporal.Product.NexusActivityLink.initial, rfl, rfl⟩

theorem stepSimulates (systemState : State)
    (productState : Umpire3.Temporal.Product.NexusActivityLink.State)
    (action : Action) (nextSystemState : State)
    (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    ∃ nextProductState, Umpire3.Temporal.Product.NexusActivityLink.model.StepStar
      productState nextProductState ∧ Projects nextSystemState nextProductState := by
  rcases systemStep with ⟨result, _, rfl⟩
  subst productState
  exact ⟨result.nextState.visible, ⟨result.productActions, result.productRun⟩, rfl⟩

def nexusActivityLinkRefinesProduct :
    Refinement system Umpire3.Temporal.Product.NexusActivityLink.model where
  Relates := Projects
  initial := initialProjects
  step := stepSimulates

end Umpire3.Temporal.System.NexusActivityLink
