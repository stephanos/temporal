---
satisfies: [R2, R4]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.2 framework-checker: BFS explore with budget, invariants per state, postconditions per edge, verify_all, refuted

## Description
Implement `Checker::explore::<M>(budget)`: BFS from `starts()` with a `HashSet<M>` visited set and a parent map for paths; evaluate invariants on each new state and postconditions on each emitted edge with `before`, `after` and the action; `Result` with verdict, counts (starts, reached, emitted, depth, cut); `Result::verify_all() -> Report` with per-claim results and vacuity warnings (action never fired, `after` claim never triggered); `Checker::refuted(claim)` for negative controls. Hit the Scala note's Kettle counts: 81 starts, 283 states, depth 62.

**Files:** model2/framework/checker/src/lib.rs (+ explore.rs, report.rs), model2/framework/samples/tests/kettle.rs.

**Quick commands:** `mise exec -- cargo test -p framework-checker -p samples`

## Acceptance
- Kettle fixture: 283 states, depth 62, both invariant/postcondition claims verified, no vacuity.
- Budget < 1 refuses; a small budget yields unknown with counts.
- `!beeping` refuted with a path from a start.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
