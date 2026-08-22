import Temporal.Families.WorkflowRoutingIsolation.LineageRefinement
import Temporal.Families.WorkflowRoutingIsolation.RoutingRefinement

namespace Umpire3.Temporal.Refinement.RoutingIsolation

structure Simulations where
  lineage : SafetySimulation WorkflowLineage.System.behavior WorkflowLineage.Feature.behavior
  routing : SafetySimulation WorkflowRouting.System.behavior WorkflowRouting.Feature.behavior

def soundSimulations : Simulations where
  lineage := WorkflowLineage.soundSimulation
  routing := WorkflowRouting.soundSimulation

theorem mutationsBreakDeclaredSimulations :
    (¬StepSimulation WorkflowLineage.System.mutatedBehavior WorkflowLineage.Feature.behavior
      WorkflowLineage.Projects WorkflowLineage.actionMap) ∧
    (¬StepSimulation WorkflowRouting.System.mutatedBehavior WorkflowRouting.Feature.behavior
      WorkflowRouting.Projects WorkflowRouting.actionMap) :=
  ⟨WorkflowLineage.mutationBreaksDeclaredSimulation,
    WorkflowRouting.mutationBreaksDeclaredSimulation⟩

end Umpire3.Temporal.Refinement.RoutingIsolation
