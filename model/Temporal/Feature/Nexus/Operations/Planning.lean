import Temporal.Feature.Nexus.Operations.Internal

/-!
# Shared deterministic Nexus operation planning evidence

Operation-specific modules derive their planner kernels directly from their checked Queries through
`IncrementalPlannerKernel.ofCheckedQuery`. This module retains the established lower import seam and
shares the Lifecycle domain-completeness and canonical-order evidence used to prove admission.
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
      decide (stepOrderKey left ≤ stepOrderKey right)) =
      (stepResult? state action).toList := by
  cases stepResult? state action <;> simp only [Option.toList, List.mergeSort_nil,
    List.mergeSort_singleton]

end Internal

/-- A checked Lifecycle Query with complete canonical action evidence admits the existing planner. -/
theorem lifecycleIncrementalKernelResult_isSome
    (query : CheckedQuery LawStatement)
    (queryTarget : query.target = target)
    (evidence : FiniteCompletenessEvidence LawStatement query.target)
    (queryCompleteness : query.completeness = some evidence)
    (evidenceActions : evidence.actions = actionDomain) :
    (IncrementalPlannerKernel.ofCheckedQuery target.id query).toOption.isSome = true := by
  have targetBneSelf : (target.id != target.id) = false := by
    cases target.id with
    | mk value =>
      change (value != value) = false
      exact bne_self_eq_false value
  apply IncrementalPlannerKernel.ofCheckedQuery_isSome target.id query evidence
  · simpa [queryTarget] using targetBneSelf
  · exact queryCompleteness
  · rw [queryTarget]
    exact Internal.lifecycleBehaviorDomainComplete
  · rw [evidenceActions]
    exact Internal.lifecycleActionDomainCanonical
  · intro setup
    rw [queryTarget]
    exact Internal.lifecycleInitialStatesCanonical setup
  · intro state action
    rw [queryTarget]
    exact Internal.lifecycleStepResultsCanonical state action

end Temporal.Feature.Nexus.Operations
