import Temporal.Feature.Nexus.Operations.AsyncStart
import Temporal.Feature.Nexus.Operations.Cancellation
import Temporal.Feature.Nexus.Operations.SuccessfulCompletion

/-! Checked Nexus operation walkthroughs for starting, canceling, and successfully completing. -/

/-!
# Reading the ordinary Nexus operation walkthroughs

Read `Temporal.Feature.Nexus.Operations.AsyncStart`, then
`Temporal.Feature.Nexus.Operations.Cancellation`, and finally
`Temporal.Feature.Nexus.Operations.SuccessfulCompletion`. Each child keeps its complete Property,
Behavior, Query, and deterministic planning run together. The lower
`Temporal.Feature.Nexus.Operations.Search` module owns only their shared planning import seam.

This facade preserves the existing `Temporal.Feature.Nexus.Operations` import and declarations.
-/

namespace Temporal.Feature.Nexus.Operations

open Umpire
open Temporal.Feature.Nexus.Lifecycle

def incrementalKernel : SearchView target :=
  Eq.mpr (congrArg SearchView AsyncStart.query_target) AsyncStart.incrementalKernel

end Temporal.Feature.Nexus.Operations
