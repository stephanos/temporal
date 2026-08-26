---
satisfies: [R3, R4, R6, R7, R8, R10, R11]
---
# fn-17-bounded-semantic-exploration-and.6 Prove Nexus fault-matrix exploration and the pure protocol

## Description
Apply the reusable pure engine to fn-16's exact Nexus fault-matrix space, construct one Temporal-owned proof-bearing exact-catalog certificate, and prove the versioned protocol without adding a command surface.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/Exploration.lean`, `model/Temporal/Feature/Nexus/Examples/ExplorationTests.lean`, `model/Temporal/Feature/Nexus/Examples/ExactCatalogExploration.lean`, `model/Temporal/Feature/Nexus/Examples/ExactCatalogExplorationTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Examples/Exploration.lean, model/Temporal/Feature/Nexus/Examples/ExplorationTests.lean, model/Temporal/Feature/Nexus/Examples/ExactCatalogExploration.lean, model/Temporal/Feature/Nexus/Examples/ExactCatalogExplorationTests.lean, model/TemporalModelTests.lean]

### Approach
- Build a Temporal-owned `space` binding around the exact fault-matrix checked space and base kernel; do not place Nexus identities under `Umpire`.
- Pin exhaustive, pairwise, t-wise, seeded, and coverage-guided outputs; four goal credits; request-only fault coordinates; target-owned model coordinates; and reordered/seeded determinism.
- Construct `ExactCatalogArtifactCertificate/v1` for existing fn-5 subject `workflow-nexus.query.exact-action-caller-closure`. Tie its stable projection binding to the whole canonical ExperimentSpec, checked Query/model trace/property context, compilation equality proof, and recomputed coverage signature without reading the fixture path.
- Prove one-member exact exhaustive selection preserves the ExperimentSpec bytes and identity. Mutate catalog subject, projection binding, Query/trace/property context, compilation witness, bytes, identity, and signature one at a time and reject before selection.
- Exercise the exact protocol equations on both source kinds: initialize fixes selection; `nextBatch(state)` accepts no size and creates exactly `min(8, remaining)` outstanding candidates in selection order; observe requires that exact batch and records identity-sorted opaque admission bindings without changing coverage.
- Keep all binding lookup semantic and in-memory. Do not add `temporal-model-explore`, `umpire-explore`, Make wiring, runtime I/O, leasing, persistence, or `umpire-fuzz`; fn-33 owns the operational binding and CLI.

### Investigation targets
**Required** (read before coding):
- fn-5 checked catalog, stable-regression projection binding, and exact caller-closure entry
- `model/Temporal/Feature/Nexus/Examples/VariationSpace.lean` — fn-16 exact space after dependency lands
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean` — checked base semantics/kernel
- `model/Temporal/Feature/Nexus/CallerClosure.lean` — checked Query/model/property context
- `model/Umpire/Artifact.lean` — canonical ExperimentSpec identity/content rules

## Acceptance
- [ ] The exact four-point space produces stable strategy selections and truthful semantic reports without authored runtime outcomes.
- [ ] The Temporal-owned exact certificate preserves the existing caller-closure ExperimentSpec bytes/identity and proves its catalog, projection, checked semantic context, compilation, and coverage binding entirely in memory.
- [ ] One-at-a-time certificate mutations reject atomically; no fixture path is read and no second registry is introduced.
- [ ] Initialize/nextBatch/observe tests pin cursor, deterministic 1/8/9-member grouping, one outstanding batch, complete observation, identity-sorted admission map, protocol status, final byte stability, and every awaiting/drained/missing/extra/duplicate/crossed rejection.
- [ ] Reusable Umpire modules contain no Temporal or Nexus identity/string/import.
- [ ] `TemporalModelTests` passes without a new executable, Make target, runtime effect, or persistence surface.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
