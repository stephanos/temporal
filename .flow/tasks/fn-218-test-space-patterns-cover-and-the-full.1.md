---
satisfies: [R1]
---
# fn-218-test-space-patterns-cover-and-the-full.1 Full path! grammar: actions, inputs, outcomes, alternatives, repetition, permutations; alternative enumeration

## Description
Extend `Path` and `path!` (umpire-dsl) with the test-space waypoints: action, action with input patterns (ranges, sets, exact, `_`/`..`), actor wildcard `Actor::*`, action with outcome `a => (o | o)`, alternatives `(a | b)`, repetition `[x; n]` / `[x; a..=b]`, `unordered![..]`, gaps `..`, `..=n`, `..!(x, y)`; an alternative enumerator computing the finite alternative set and its count before search; trybuild `.stderr` fixtures for each malformed form and for any of these in a `#[progress]` claim (`a progress claim is two state predicates`). After fn-215.1. **Touches:** `model/umpire/dsl/**`, `dsl/macros/src/machine.rs`, `dsl/tests/ui/`. **Feedback loop (R30):** paste `make model-time`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
