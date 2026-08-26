---
satisfies: [R1, R3, R5, R6]
---
# fn-33-run-resumable-semantic-exploration.6 Bind one runnable caller-closure exploration through the catalog

## Description
Add one checked runnable-exploration binding that closes the current model-to-runner gap without creating another catalog or widening the runner.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/RuntimeExploration.lean`, `model/Temporal/Feature/Nexus/Examples/RuntimeExplorationTests.lean`, `model/Temporal/System/Nexus/Exploration.lean`, `model/Temporal/System/Nexus/ExplorationTests.lean`, `model/Temporal/Tool/Exploration.lean`, `model/Temporal/Tool/ExplorationTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Examples/RuntimeExploration.lean, model/Temporal/Feature/Nexus/Examples/RuntimeExplorationTests.lean, model/Temporal/System/Nexus/Exploration.lean, model/Temporal/System/Nexus/ExplorationTests.lean, model/Temporal/Tool/Exploration.lean, model/Temporal/Tool/ExplorationTests.lean, model/TemporalModelTests.lean]

### Approach

- Define `temporal.nexus.caller-closure.runtime-smoke` as a checked one-point exhaustive exploration over the exact complete current caller-closure ExperimentSpec already admitted by fn-19/fn-20; selection budget one, seed zero, no faults, and no pins.
- Under `Temporal.System.Nexus`, bind that semantic exploration to the exact ephemeral-local RuntimeConfiguration/input-set identity, runner and conformance identities, campaign time range 1 second through 5 minutes, parallelism exactly one, bridge batch exactly one, seed set `{0}`, and empty pinned-set digest.
- Project the checked `RunnableExplorationBinding` through fn-5's existing catalog/list/explain authority. The projection is keyed by the same exploration identity and cannot act as a second registry or invent bounds.
- Prove every ExperimentSpec, configuration, profile, runner/checker, bound, seed, pin, or catalog-identity drift rejects before bridge or runtime I/O.
- Keep semantic exploration under Feature, operational admissibility under System, and effect-thin discovery under Tool.

### Investigation targets

**Required** (read before coding):
- fn-5 checked catalog/list/explain API
- fn-17 checked Exploration protocol and semantic identity
- fn-19 exact caller-closure ExperimentSpec/RuntimeConfiguration input set
- fn-20 exact conformance checker binding

## Acceptance
- [ ] Exactly one checked runnable binding exists for `temporal.nexus.caller-closure.runtime-smoke`, and its one candidate is byte-/identity-equal to the ExperimentSpec accepted by fn-19/fn-20.
- [ ] The binding fixes the ephemeral-local input set/profile, runner/conformance identities, exhaustive budget one, seed zero, no faults/pins, parallelism one, bridge batch one, and 1-second-to-5-minute campaign range.
- [ ] Fn-5-backed `list` and `explain` return the binding and bounds without a second catalog; unknown or semantic-only explorations are honestly listed as non-runnable or rejected for run.
- [ ] One-at-a-time spec/config/profile/checker/bounds/seed/pin/catalog mutations reject before process or environment I/O.
- [ ] Feature imports no System; the focused System binding consumes Feature and runner metadata without redefining product meaning; Tool remains effect-thin.
- [ ] Focused Lean fixtures and `TemporalModelTests` pass with existing comments preserved.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
