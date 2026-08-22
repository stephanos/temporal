import Temporal.Families.WorkflowRoutingIsolation.LineageFeature
import Temporal.Families.WorkflowRoutingIsolation.RoutingFeature
import Umpire3.Behavior

namespace Umpire3.Temporal.Feature.RoutingIsolation

structure Behaviors where
  lineage : Behavior Unit
  routing : Behavior Unit

def behaviors : Behaviors where
  lineage := Behavior.ofTransitionSystem WorkflowLineage.model
  routing := Behavior.ofTransitionSystem WorkflowRouting.model

end Umpire3.Temporal.Feature.RoutingIsolation
