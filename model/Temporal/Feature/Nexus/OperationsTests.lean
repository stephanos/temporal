import Temporal.Feature.Nexus.Operations
import Temporal.Feature.Nexus.Operations.AsyncStartTests
import Temporal.Feature.Nexus.Operations.CancellationTests
import Temporal.Feature.Nexus.Operations.PlanningTests
import Temporal.Feature.Nexus.Operations.SuccessfulCompletionTests

namespace Temporal.Feature.Nexus.OperationsTests

open Umpire
open Temporal.Feature.Nexus.Lifecycle
open Temporal.Feature.Nexus.Operations

#check (Temporal.Feature.Nexus.Operations.source : SourceLocation)
#check (Temporal.Feature.Nexus.Operations.AsyncStart.property : CheckedProperty)
#check (Temporal.Feature.Nexus.Operations.AsyncStart.behavior : CheckedBehavior)
#check (Temporal.Feature.Nexus.Operations.AsyncStart.query : CheckedQuery LawStatement)
#check (Temporal.Feature.Nexus.Operations.Cancellation.property : CheckedProperty)
#check (Temporal.Feature.Nexus.Operations.Cancellation.behavior : CheckedBehavior)
#check (Temporal.Feature.Nexus.Operations.Cancellation.query : CheckedQuery LawStatement)
#check (Temporal.Feature.Nexus.Operations.SuccessfulCompletion.property : CheckedProperty)
#check (Temporal.Feature.Nexus.Operations.SuccessfulCompletion.behavior : CheckedBehavior)
#check (Temporal.Feature.Nexus.Operations.SuccessfulCompletion.query : CheckedQuery LawStatement)

end Temporal.Feature.Nexus.OperationsTests
