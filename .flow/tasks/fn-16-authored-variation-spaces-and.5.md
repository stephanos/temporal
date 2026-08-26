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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
