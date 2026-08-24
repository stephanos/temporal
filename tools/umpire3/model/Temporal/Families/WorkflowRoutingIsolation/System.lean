import Temporal.Families.WorkflowRoutingIsolation.LineageSystem
import Temporal.Families.WorkflowRoutingIsolation.RoutingSystem

namespace Umpire3.Temporal.System.RoutingIsolation

structure ExecutableViews where
  lineage : ExecutableView WorkflowLineage.behavior
  routing : ExecutableView WorkflowRouting.behavior

def executableViews : ExecutableViews where
  lineage := WorkflowLineage.executable
  routing := WorkflowRouting.executable

end Umpire3.Temporal.System.RoutingIsolation
