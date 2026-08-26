---
satisfies: [R1, R3, R5, R6]
---
# fn-33-run-resumable-semantic-exploration.6 Bind one runnable caller-closure exploration through the catalog

## Description
Join one checked runnable-exploration binding to an existing fn-5 catalog subject, closing the current model-to-runner gap without creating another catalog or widening the runner.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/RuntimeExploration.lean`, `model/Temporal/Feature/Nexus/Examples/RuntimeExplorationTests.lean`, `model/Temporal/System/Nexus/Exploration.lean`, `model/Temporal/System/Nexus/ExplorationTests.lean`, `model/Temporal/Tool/Exploration.lean`, `model/Temporal/Tool/ExplorationTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Examples/RuntimeExploration.lean, model/Temporal/Feature/Nexus/Examples/RuntimeExplorationTests.lean, model/Temporal/System/Nexus/Exploration.lean, model/Temporal/System/Nexus/ExplorationTests.lean, model/Temporal/Tool/Exploration.lean, model/Temporal/Tool/ExplorationTests.lean, model/TemporalModelTests.lean]

### Approach
- Define internal semantic source `temporal.nexus.caller-closure.runtime-smoke` with fn-17 `exactCatalogArtifacts`, reusing the Temporal-owned `ExactCatalogArtifactCertificate/v1` that ties existing fn-5 subject `workflow-nexus.query.exact-action-caller-closure` and its stable projection binding to the whole checked ExperimentSpec/Query/model/property context; preserve bytes and identity with exhaustive budget one, seed zero, no faults, and no pins.
- Under `Temporal.System.Nexus`, bind that source to the exact ephemeral-local RuntimeConfiguration/input-set identity, runner and conformance identities, campaign time range 1 second through 5 minutes, parallelism exactly one, seed set `{0}`, and empty pinned-set digest. The one-member source naturally yields one item from fn-17's fixed `min(8, remaining)` batching; the binding does not author a protocol batch size.
- Make the runnable-binding collection an operational relation keyed by the existing catalog subject. `list` walks fn-5's checked catalog and left-joins bindings; `explain` augments the selected catalog entry. Neither path enumerates the binding collection as a semantic registry or adds an fn-5 catalog entry.
- Prove every ExperimentSpec, configuration, profile, runner/checker, bound, seed, pin, catalog-subject, or catalog-content drift rejects before bridge or runtime I/O.
- Keep semantic exploration under Feature, operational admissibility under System, and effect-thin discovery under Tool.

### Investigation targets

**Required** (read before coding):
- fn-5 checked catalog/list/explain API and the existing exact-action caller-closure entry
- fn-17 `ExactCatalogArtifactCertificate/v1`, `exactCatalogArtifacts` source, and checked Exploration protocol
- fn-19 exact caller-closure ExperimentSpec/RuntimeConfiguration input set
- fn-20 exact conformance checker binding
## Acceptance
- [ ] Exactly one checked runnable binding joins existing fn-5 subject `workflow-nexus.query.exact-action-caller-closure`; no new catalog entry or second semantic registry is introduced.
- [ ] Its fn-17 exact source consumes the Temporal-owned proof-bearing certificate and contains one candidate byte-/identity-equal to the ExperimentSpec accepted by fn-5/fn-19/fn-20 without reading a projection fixture path.
- [ ] The binding fixes the ephemeral-local input set/profile, runner/conformance identities, exhaustive budget one, seed zero, no faults/pins, parallelism one, and 1-second-to-5-minute campaign range; its one-item batch is derived from the one-member source, not a binding-authored protocol size.
- [ ] Fn-5-backed `list` and `explain` left-join runnable metadata; catalog entries without a binding are honestly non-runnable and unknown subjects reject.
- [ ] One-at-a-time spec/config/profile/checker/bounds/seed/pin/catalog-subject/content mutations reject before process or environment I/O.
- [ ] Feature imports no System; the focused System binding consumes Feature and runner metadata without redefining product meaning; Tool remains effect-thin.
- [ ] Focused Lean fixtures and `TemporalModelTests` pass with existing comments preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
