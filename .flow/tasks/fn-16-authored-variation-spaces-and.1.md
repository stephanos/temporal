---
satisfies: [R1, R2, R3, R8]
---
# fn-16-authored-variation-spaces-and.1 Define and check the authored experiment-space language

## Description
Define the closed reusable Space declarations, checked values, fixed bounds, and canonical errors for R1-R3/R8. Extend shared declaration vocabulary for space metadata while leaving Property, Behavior, Query, and target semantics unchanged.

**Size:** M
**Files:** `model/Umpire/Core.lean`, `model/Umpire/Space/Language.lean`, `model/Umpire/Space/Tests/Fixtures.lean`, `model/Umpire/Space/Tests/Validation.lean`
**Touches:** [model/Umpire/Core.lean, model/Umpire/Space/Language.lean, model/Umpire/Space/Tests/Fixtures.lean, model/Umpire/Space/Tests/Validation.lean]

### Approach
- Add the five canonical declaration kinds and update exhaustive names/tests without broadening existing role or Property acceptance.
- Define authored/checked spaces, axes, choices, baseline/effect rules, fault intents with symmetric incompatibilities, coverage subjects/goals, limits, and structured `SpaceError`.
- Resolve every reference against the base checked Query/Behavior/target/property closure and validate role/setup/capability/occurrence compatibility.
- Canonicalize before duplicate/conflict reporting and detect product overflow without materializing assignments.
- Preserve all existing comments in touched core files.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Core.lean:28-69` — declaration vocabulary and metadata
- `model/Umpire/Behavior/Language.lean:51-165` — roles, bindings, named occurrences, and checked Behavior
- `model/Umpire/Query/Language.lean:41-120` — Query forms and checked context
- `model/Umpire/Behavior/Language.lean:816-900` — authored-to-checked validation pattern
- `model/Umpire/Property/Language.lean:192-307` — checked declaration and typed error conventions

### Acceptance
- [ ] Exact v1 bounds and overflow checks reject invalid spaces before assignment materialization.
- [ ] Canonical tests cover empty/singleton axes, baseline/effect rules, duplicate/case-colliding IDs, role conflicts, stale references, fault occurrence/capability/incompatibility, and coverage feasibility.
- [ ] Reordered equivalent declarations produce equal checked values, errors, and digests.
- [ ] Properties and target-owned outcomes are neither copied nor writable through Space declarations.
- [ ] Existing comments remain intact.

## Acceptance
- [ ] The checked Space language implements the exact parent contracts and bounds.
- [ ] Invalid references/effects/conflicts fail with canonical typed errors and no partial value.
- [ ] Reordering is deterministic and existing semantic APIs remain unchanged.

## Done summary
Defined and checked the authored experiment-space language, including exact v1 limits, canonical typed failures, role/value/fault/coverage validation, deterministic checked values, and reusable Switch fixtures with focused validation coverage. Verification passed the implemented `.1` target, Switch example, aggregate suites, and regression; the cumulative Compilation, Metadata, and Temporal variation-space targets remain expected pre-feature baseline failures assigned to tasks `.4`, `.3`, and `.5`, respectively.

The review's canonical-error ordering finding was fixed by validating declaration identities before Cartesian bounds, with a reordered-duplicate regression test. Review then returned SHIP; non-blocking memory capture was skipped because flow memory is not initialized.

stage: impl-review - ran [2026-08-28T00:20:38.988563Z..2026-08-28T00:22:40.193742Z]
## Evidence
- Commits: 14362b9dc15a9b4e786e4a91aa007a4aa9399735, 514502d565d74dcac04fa4683ce3c5c9625eb587
- Tests: cd model && mise exec -- lake build Umpire.Space.Tests.Validation, cd model && mise exec -- lake build Umpire.Examples.SwitchTests, cd model && mise exec -- lake build UmpireTests TemporalModelTests, make umpire-check-regression, BASELINE_EXPECTED_FAILURE:cd model && mise exec -- lake build Umpire.Space.Tests.Compilation - cumulative target assigned to fn-16-authored-variation-spaces-and.4, BASELINE_EXPECTED_FAILURE:cd model && mise exec -- lake build Umpire.Space.Tests.Metadata - cumulative target assigned to fn-16-authored-variation-spaces-and.3, BASELINE_EXPECTED_FAILURE:cd model && mise exec -- lake build Temporal.Feature.Nexus.Examples.VariationSpaceTests - cumulative target assigned to fn-16-authored-variation-spaces-and.5
- PRs: