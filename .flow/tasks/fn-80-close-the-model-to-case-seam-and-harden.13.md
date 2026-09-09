---
satisfies: [R2]
---
# fn-80-close-the-model-to-case-seam-and-harden.13 Add the second-lifecycle syntax test and Nexus3 wording

## Description
Step (d) of task .5's recorded sequence, the small remainder. Depends on step (c).

**Size:** S
**Files:** model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean; model/Temporal/Feature/Nexus3/Nexus.md
**Touches:** [model/Temporal/Feature/Nexus3/RaceSyntaxTests.lean, model/Temporal/Feature/Nexus3/Nexus.md]

### Scope
Add `RaceSyntaxTests.lean` — a genuinely second enum-like finite lifecycle elaborated through the generalized syntax, which is the real proof that the whitelists are gone rather than widened — and update `Nexus.md` to describe the generalized form.

Check for a conflict with fn-67 before editing `Nexus.md`: fn-67 has an open documentation task on that file and on `Integration.md`.

## Acceptance
- [ ] `RaceSyntaxTests.lean` elaborates a second, structurally different lifecycle through the same syntax, wired into the test aggregate.
- [ ] `Nexus.md` describes the generalized form; any overlap with fn-67's open documentation task is named rather than silently overwritten.
- [ ] Full model build and `make umpire-check-case-runtime-conformance` green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
