import Temporal.Feature.Nexus.Operations.Internal

/-!
# Shared deterministic Nexus operation planning

This lower seam adapts the checked lifecycle target to the incremental planner. Operation-specific
runs stay beside their Property, Behavior, and Query in the three walkthrough modules.
-/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

private def finiteEvidence? : Option (FiniteCompletenessEvidence LawStatement target) :=
  (CheckedQueryTarget.ofTarget target).completeness

private def incrementalKernel? : Option (IncrementalPlannerKernel target) :=
  match evidenceEq : finiteEvidence? with
  | none => none
  | some evidence =>
      some <| IncrementalPlannerKernel.ofFinite evidence {
        action := by
          simp [finiteEvidence?, CheckedQueryTarget.ofTarget, target, checkedTarget,
            targetAuthoring, AuthoredTarget.make, targetDefinition] at evidenceEq
          cases Option.some.inj evidenceEq
          simp [finitePlanning, actionDomain]
          decide
        initial := by
          intro setup
          simp [target, checkedTarget, targetAuthoring, AuthoredTarget.make, targetDefinition,
            transitionKernel, initialStates]
        step := by
          intro state action
          simp [target, checkedTarget, targetAuthoring, AuthoredTarget.make, targetDefinition,
            transitionKernel, stepResults]
      }

private theorem incrementalKernel?_isSome : incrementalKernel?.isSome = true := by
  rfl

def incrementalKernel : IncrementalPlannerKernel target :=
  incrementalKernel?.get incrementalKernel?_isSome

end Temporal.Feature.Nexus.Operations
