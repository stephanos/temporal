# Implementation verification boundary

Umpire3 verifies Temporal through explicit executable seams rather than claiming whole-program Go
proof. Lean proves model safety, executable equivalence, refinements, monitor equivalence, module
composition, and bounded exploration properties. Go tests validate strict transport, deterministic
compilation, descriptor conformance, evidence qualification, fault cleanup, campaigns, profile
authority, hard worker termination, and release qualification.

Real-cluster integration exercises the Nexus task protocol and Workflow Task acknowledgement through
public Temporal APIs. The independent root Umpire3 copies additionally run generated SDK participant
programs for Workflow, Activity, Update, child, callback, timeout, and Nexus behavior and
qualify target-specific public-history predicates. These copies no longer choose checkpoint success
from test input. Cancellation/retry injects its selected drop at the external Nexus cancellation
handler and requires positive occurrence evidence before the fault interval can close. Some retained
contracts still reduce the original probe's generated request, learned footprint, lifecycle outcome,
or exploration dimensions. The candidate therefore distinguishes real mechanism coverage from exact
behavioral parity. Synthetic adapters and controlled canary workers are negative-control tools, not
real-deployment qualification.

The root adapters now drive dedicated public mechanisms for ordinary Nexus completion, completion
before the start response, cancellation retry, start-to-close timeout, callback completion after the
caller closes, a shared callback handler, bidirectional Nexus/Activity links, continuation, reset,
Task Queue routing, and Workflow Task ownership fencing. Public history and a separately normalized
server-history client provide independent evidence sources. Descriptor-derived invalid values,
runtime-learned call footprints, the Recovered/Degraded/Flagged taxonomy, and full model-derived
lifecycle exploration remain candidate gaps rather than implied passes.

The seeded cross-layer gate is executable local evidence for discovery mechanics: it requires typed
mutation selection, exact qualified-violation preservation, multi-axis minimization, strict redacted
bundle decoding, replay reproduction, and compilable ordinary regression promotion. It does not
substitute for discovering an unknown production defect.

Gobra was not adopted because the relevant decisions remain embedded across stateful Temporal
services rather than a stable production function with explicit concurrency and persistence
assumptions. Reconsider only after such a function exists and the proposed contract identifies the
exact Lean theorem it discharges. Grove/Goose is likewise deferred until a concrete crash-consistency
seam justifies its proof and maintenance cost.

The checked `umpire3/1.2` manifest is a candidate. Remote, gRPC-only, and production canary results
must pass `umpire3 qualify` (or the compatibility `cmd/umpire3-qualify` entry point) and be bound to
the exact release, experiment, build/configuration, evidence, fault realization, and cleanup digests
before a qualified release can be emitted.

Candidate truth is machine-readable. Parity ledger v2 marks only the TaskAck target exact and
equivalent; the other 19 retained assurance rows are partial and not yet implemented. Migration
ledger v3 records 14 exact, 4 semantic-equivalent, and 10 partial live root behaviors, and the
assurance composition obligation remains pending. Candidate validation permits these explicit gaps;
qualified validation rejects them, along with evidence below profile-qualified level.

The combined root target preserves and runs both implementations sequentially. At the current audit
revision, retained Umpire2 exploration/generated-completion probes do not receive their expected
CHASM transition telemetry even when run alone. That baseline failure is reported independently and
must not be hidden by changing the preserved Umpire2 assertions or by attributing it to Umpire3.
