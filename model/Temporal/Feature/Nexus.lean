import Temporal.Feature.Nexus.Lifecycle
import Temporal.Feature.Nexus.Observation
import Temporal.Feature.Nexus.Operations

/-!
# Ordinary Nexus model

This is the single ordinary Nexus entry import. Read the model in this order:

1. `Temporal.Feature.Nexus.Lifecycle.Semantics` for states, events, and transitions.
2. `Temporal.Feature.Nexus.Operations.AsyncStart` for starting an operation.
3. `Temporal.Feature.Nexus.Operations.Cancellation` for canceling a started operation.
4. `Temporal.Feature.Nexus.Operations.SuccessfulCompletion` for completing a started operation.
5. `Temporal.Feature.Nexus.Observation` for the offline evidence boundary.
6. `Temporal.System.Nexus.Core` and `Temporal.System.Nexus.ImplementationLink` for the independently
   authored System mechanism and its correspondence with this Feature model.

The Lifecycle and Operations facades document their focused implementation modules. Advanced
AutoClose and caller-closure material remains opt-in through explicit
`Temporal.Feature.Nexus.Experimental.AutoClose` and
`Temporal.Feature.Nexus.Experimental.CallerClosure` imports; this ordinary facade does not import
Experimental modules.
-/
