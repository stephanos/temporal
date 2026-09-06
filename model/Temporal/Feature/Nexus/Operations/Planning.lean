import Temporal.Feature.Nexus.Operations.Internal

/-!
# Shared deterministic Nexus operation planning imports

Operation-specific modules derive their planner kernels directly from their checked Queries through
`IncrementalPlannerKernel.ofCheckedQuery`. This module retains the established lower import seam.
-/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

namespace Internal

theorem lifecycleBehaviorDomainComplete :
    ∃ domain, finiteMachine.kernel.behaviorDomain = .complete domain := by
  simp [FiniteMachine.kernel]

theorem lifecycleActionDomainCanonical :
    actionDomain.mergeSort (fun left right =>
      decide (modelValueOrderKey left ≤ modelValueOrderKey right)) = actionDomain := by
  apply List.mergeSort_of_pairwise
  decide

theorem lifecycleInitialStatesCanonical (setup : List RoleBinding) :
    (initialState? setup).toList.mergeSort (fun left right =>
      decide (modelValueOrderKey left ≤ modelValueOrderKey right)) =
      (initialState? setup).toList := by
  cases initialState? setup <;> simp only [Option.toList, List.mergeSort_nil,
    List.mergeSort_singleton]

theorem lifecycleStepResultsCanonical (state action : ModelValue) :
    (stepResult? state action).toList.mergeSort (fun left right =>
      decide (transitionResultOrderKey left ≤ transitionResultOrderKey right)) =
      (stepResult? state action).toList := by
  cases stepResult? state action <;> simp only [Option.toList, List.mergeSort_nil,
    List.mergeSort_singleton]

end Internal

end Temporal.Feature.Nexus.Operations
