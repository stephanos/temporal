---
satisfies: [R1, R2, R3, R4, R5, R6, R7, R8]
---
# fn-16-authored-variation-spaces-and.5 Prove two-by-two synthetic and Temporal variation spaces

## Description
Land the early proof using the reusable Switch fixture and one concise Temporal BasicLifecycle combinatorial declaration for R1-R8. The Temporal example models a two-action lifecycle Behavior and two request-only fault axes over named start/success occurrences.

**Size:** M
**Files:** `model/Umpire/Examples/Switch.lean`, `model/Umpire/Examples/SwitchTests.lean`, `model/Temporal/Feature/Nexus/Examples/VariationSpace.lean`, `model/Temporal/Feature/Nexus/Examples/VariationSpaceTests.lean`, `model/UmpireTests.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Examples/Switch.lean, model/Umpire/Examples/SwitchTests.lean, model/Temporal/Feature/Nexus/Examples/VariationSpace.lean, model/Temporal/Feature/Nexus/Examples/VariationSpaceTests.lean, model/UmpireTests.lean, model/TemporalModelTests.lean]

### Approach
- Extend the synthetic Switch example with two independent role/fault axes that exercise non-empty selected choices, variants, and faults without changing Switch target semantics.
- Compose the parent spec's exact named BasicLifecycle two-transition Behavior/Query from existing start/success declarations and target; add the exact two baseline-versus-fault axes, named occurrences, two faults, and four coverage goals.
- Pin exactly four point IDs, metadata rows/digest, artifact identities/order, fault capability union, and target-owned traces for each proof fixture.
- Add reorder and representative negative fixtures from the parent early-proof contract.
- Register focused suites in reusable and Temporal aggregate tests without crossing package ownership.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Examples/Switch.lean:307-471` — checked behaviors, queries, and planner kernels
- `model/Umpire/Examples/SwitchTests.lean` — reusable example assertions
- `model/Temporal/Feature/Nexus/Examples/BasicLifecycle.lean:11-317` — target, meanings, and bounds
- `model/Temporal/Feature/Nexus/Examples/BasicOperations.lean:45-253` — start/success properties and occurrences
- `model/Temporal/Feature/Nexus/Examples/BasicOperationsTests.lean` — Temporal example test style

### Acceptance
- [ ] Each two-by-two space yields exactly four canonically ordered specs and one deterministic checked metadata projection; the Temporal fixture pins every exact parent-spec identity and assignment.
- [ ] Across the proof set all three existing artifact intent arrays are non-empty and exact.
- [ ] Every model outcome/state/observation equals ordinary target-kernel output; no authored fault changes it.
- [ ] Bounds, duplicate effect, stale occurrence/capability, impossible goal, incompatible selection, and non-artifact point controls fail at the intended boundary.
- [ ] Reordering equivalent declarations changes no bytes, and existing example tests remain green.
- [ ] Existing comments remain intact.

## Acceptance
- [ ] Synthetic and Temporal two-by-two proofs each return four exact specs.
- [ ] Intent metadata is populated while outcomes remain target-owned.
- [ ] Positive, negative, and determinism fixtures pass in their owning aggregates.

## Done summary
Implemented the exact Temporal Nexus two-by-two fault matrix under the Experimental owner, with a checked two-action start/success query, canonical metadata and artifact identities, request-only faults, target-owned results, reorder determinism, and representative negative proofs. Reused and registered the synthetic Switch Space proof suites, including intent projection and every parent early-proof failure boundary.

Baseline was green except for the expected pre-feature stale `Temporal.Feature.Nexus.Examples.VariationSpaceTests` target. The removed `Examples/` path was not recreated; its current `Temporal.Feature.Nexus.Experimental.VariationSpaceTests` equivalent and all final Validation, Compilation, Metadata, Switch, aggregate, lint, and regression gates pass. Gate receipts were non-blockingly unavailable because the preserved unrelated `.plans/UMPIRE4_ORDER.md` diff keeps the worktree dirty. Review fixed Experimental aggregate ownership and missing intent-suite registration, then returned SHIP; memory capture was non-blockingly unavailable because memory is not initialized.

stage: impl-review - ran [2026-08-28T02:22:39Z..2026-08-28T02:30:29Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d16ce380bd0ab3e2462394f14dc38a8d3f8b859a, 1be837385038002e2f2d800c00d349b7866efbb4
- Tests: baseline: red (cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.VariationSpaceTests failed pre-edit: stale removed path; replaced by current Experimental owner), cd model && mise exec -- lake build Umpire.Space.Tests.Validation, cd model && mise exec -- lake build Umpire.Space.Tests.Compilation, cd model && mise exec -- lake build Umpire.Space.Tests.Metadata, cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.VariationSpaceTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, cd model && mise exec -- lake build Umpire.Examples.SwitchTests TemporalModelTests TemporalExperimentalTests, make lint-model, make umpire-check-regression
- PRs:
