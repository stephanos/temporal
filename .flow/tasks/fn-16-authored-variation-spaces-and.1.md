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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
