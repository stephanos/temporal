import Temporal.Feature.Nexus.Lifecycle
import Temporal.Feature.Nexus.Observation
import Temporal.Feature.Nexus.Operations

/-!
# Ordinary Nexus model

This is the single ordinary Nexus entry import. Read the model in this order:

1. `Temporal.Feature.Nexus.Lifecycle.Semantics` for states, events, and transitions.
2. `Temporal.Feature.Nexus.Lifecycle.Target` for the proof-carrying finite Target, explicit
   composition, and `checkTarget` boundary.
3. `Temporal.Feature.Nexus.Operations.AsyncStart` for its checked Property, Behavior, Query, named
   Limits, and deterministic plan.
4. `Temporal.Feature.Nexus.Operations.Cancellation` and
   `Temporal.Feature.Nexus.Operations.SuccessfulCompletion` for the other Target-owned outcomes.
5. `Temporal.Feature.Nexus.Observation` for typed profile/rule/mapping authoring and offline Evidence
   evaluation.
6. `Temporal.System.Nexus.Core` and `Temporal.System.Nexus.ImplementationLink` for the independently
   authored System mechanism and its correspondence with this Feature model.

The established operation Queries default to no authored Known Gaps. Authors may attach a checked
set; planning composes it with phase-owned gaps before traversal or artifact publication without
changing behavior fingerprints or outcomes. Gaps state limitations and never establish success.

The Lifecycle and Operations facades document their focused modules. `Temporal.Feature.NexusTests`
is the compiled facade-only walkthrough and negative specimen. Advanced AutoClose material remains
opt-in through an explicit
`Temporal.Feature.Nexus.Experimental.AutoClose` import; this ordinary facade does not import
Experimental modules.
-/
