---
satisfies: [R3, R4, R6, R7, R8]
---
# fn-17-bounded-semantic-exploration-and.6 Prove Nexus fault-matrix exploration and the pure protocol

## Description
### Umpire4 reconciliation (normative)

This task proves the pure Lean `initialize` / `nextBatch` / `observe` protocol and the Nexus semantic fixture only. It must not add `temporal-model-explore`, Make command wiring, runtime I/O, leasing, persistence, or `umpire-fuzz`; fn-33 owns the downstream Go campaign and CLI.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Apply the reusable engine to fn-16's exact Temporal space and expose R3/R4/R6/R7/R8 through one effect-thin model command.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Examples/Exploration.lean`, `model/Temporal/Feature/Nexus/Examples/ExplorationTests.lean`, `model/Temporal/Tool/Explore.lean`, `model/Temporal/Tool/ExploreTests.lean`, `model/TemporalModelTests.lean`, `model/lakefile.toml`
**Touches:** [model/Temporal/Feature/Nexus/Examples/Exploration.lean, model/Temporal/Feature/Nexus/Examples/ExplorationTests.lean, model/Temporal/Tool/Explore.lean, model/Temporal/Tool/ExploreTests.lean, model/TemporalModelTests.lean, model/lakefile.toml]

### Approach
- Build a Temporal-owned binding around the exact fault-matrix checked space and base kernel; do not place Nexus names under `Umpire`.
- Pin expected exhaustive/pairwise/coverage outputs, four goal credits, request-only fault coordinates, target-owned model coordinates, and reordered/seeded determinism.
- Implement the exact five strategy strings and option rules from the parent API contract plus a canonical success/error JSON envelope over compiled bindings; reject every alias, irrelevant option, and missing required option.
- Keep binding lookup separate from fn-5's metadata catalog and validate the exact shared space identity/digest so it cannot become a second semantic registry.
- Register only the `temporal-model-explore` Lake executable and tests.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Feature/Nexus/Examples/VariationSpace.lean` — fn-16 exact space after dependency lands
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean` — checked base semantics/kernel
- `model/Temporal/Tool/Inspect.lean:17-88` — canonical effect-thin command pattern
- `model/Temporal/Tool/InspectTests.lean:10-71` — exact stdout/stderr/status tests
- `model/lakefile.toml:1-20` — current target registration

### Acceptance
- [ ] The exact four-point space produces stable strategy selections and truthful semantic reports without authored outcomes.
- [ ] Invalid space/strategy/budget/strength/seed paths return status 1, empty stdout, and one canonical stderr document.
- [ ] Direct command output is byte-identical across repeated runs and equivalent authoring order.
- [ ] Reusable modules contain no Temporal or Nexus identity/string/import.
## Acceptance
- [ ] The Temporal example and command prove R3/R4/R6/R7/R8.
- [ ] `TemporalModelTests` and `temporal-model-explore` build.
- [ ] The binding is an adapter, not a second catalog authority.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
