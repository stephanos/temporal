---
satisfies: [R3]
---
# fn-254-gomad-deeper-rust-host-testing-mutation.3 Fuzz retained-evidence and wire decoders

## Description
R3: fuzz every decoder of retained evidence or wire data that the survey lists.

1. Pin cargo-fuzz (nightly, separate target directory) or bolero; the survey decides. bolero also yields proptest-style properties from the same harness.
2. Seed each target's corpus from the golden fixtures.
3. Each target checks four things: no panic, a bounded time per input, bounded allocation (limits at least as strict as the production read limits, e.g. the authenticated artifact limits), and refusal of malformed input with the decoder's typed error.
4. Every crash or hang input is minimised and kept as a regression fixture that the normal test suite runs, together with the fix.
5. `mise run deep:fuzz` runs all targets for a configured time budget.

## Acceptance
- Each surveyed decoder has a fuzz target and a seeded corpus; `deep:fuzz` runs them all within budget.
- Every crash or hang found during adoption is fixed, minimised and kept as a fixture in the normal suite.
- A seeded decoder bug (an unchecked length) is found by its target within the budget.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
