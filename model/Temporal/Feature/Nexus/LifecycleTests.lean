import Temporal.Feature.Nexus.Lifecycle
import Temporal.Feature.Nexus.Lifecycle.SemanticsTests
import Temporal.Feature.Nexus.Lifecycle.TargetTests

namespace Temporal.Feature.Nexus.LifecycleTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle

#check (Temporal.Feature.Nexus.Lifecycle.OperationState : Type)
#check (Temporal.Feature.Nexus.Lifecycle.OperationEvent : Type)
#check (Temporal.Feature.Nexus.Lifecycle.step : OperationState → OperationEvent → Option OperationState)
#check (Temporal.Feature.Nexus.Lifecycle.source : SourceLocation)
#check (Temporal.Feature.Nexus.Lifecycle.definitions : List DefinitionMetadata)
#check (Temporal.Feature.Nexus.Lifecycle.finiteMachine : FiniteMachine
  (List RoleBinding) ModelValue ModelValue ModelValue ModelValue)
#check (Temporal.Feature.Nexus.Lifecycle.authoritativeInitial :
  List RoleBinding → ModelValue → Prop)
#check (Temporal.Feature.Nexus.Lifecycle.authoritativeStep : ModelValue → ModelValue →
  TransitionResult ModelValue ModelValue ModelValue → Prop)
#check (Temporal.Feature.Nexus.Lifecycle.target : QueryTarget LawStatement)

end Temporal.Feature.Nexus.LifecycleTests
