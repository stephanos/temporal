import Temporal.System.NexusTasks
import Umpire3.Refinement

namespace Umpire3.Temporal.System.NexusTasks

def Projects (systemState : State) (productState : Temporal.Product.Nexus.State) : Prop :=
  systemState.visible = productState

theorem initialProjects (systemState : State) (isInitial : system.Initial systemState) :
    ∃ productState, Temporal.Product.Nexus.product.Initial productState ∧
      Projects systemState productState := by
  subst systemState
  exact ⟨Temporal.Product.Nexus.initial, rfl, rfl⟩

theorem stepSimulates (systemState : State)
    (productState : Temporal.Product.Nexus.State) (action : Action) (nextSystemState : State)
    (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    ∃ nextProductState,
      Temporal.Product.Nexus.product.StepStar productState nextProductState ∧
      Projects nextSystemState nextProductState := by
  rcases systemStep with ⟨result, _, rfl⟩
  subst productState
  exact ⟨result.nextState.visible, ⟨result.productActions, result.productRun⟩, rfl⟩

def nexusTasksRefinesProduct : Refinement system Temporal.Product.Nexus.product where
  Relates := Projects
  initial := initialProjects
  step := stepSimulates

theorem classifiedStepStutters {systemState nextSystemState : State}
    {productState : Temporal.Product.Nexus.State} {action : Action}
    (classification : StutteringAction action)
    (related : Projects systemState productState)
    (systemStep : system.Step systemState action nextSystemState) :
    nexusTasksRefinesProduct.Stutters systemState productState action nextSystemState := by
  rcases systemStep with ⟨result, resultMember, rfl⟩
  have emptyActions := classification systemState result resultMember
  have sameVisible : systemState.visible = result.nextState.visible := by
    have productRun := result.productRun
    rw [emptyActions] at productRun
    exact Runs.empty productRun
  exact ⟨related, ⟨result, resultMember, rfl⟩, sameVisible.symm.trans related⟩

end Umpire3.Temporal.System.NexusTasks
