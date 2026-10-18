---
satisfies: [R2, R4]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.2 framework-checker: BFS explore with budget, invariants per state, postconditions per edge, verify_all, refuted

## Description
**Recut 2026-10-18 (R11, R14, R16):** run each step under `catch_unwind` and refute on panic or overflow with path, action and message; keep order structural (catalog order, ordered starts, FIFO BFS, no hash iteration in output); put the visited set behind `Visited<S>` with a full-state implementation.

**Recut 2026-10-18 (R8, golden files):** deliver the behaviour runner and the oracle test, not assertions. Add `insta` (dev) and a `samples` test that runs every registered Model and snapshots `cases/<model>/expected/report.txt` and `paths/<claim>.txt`; give states a stable `Display` and sort whatever reaches a golden. Add `proptest` (dev) and a brute-force reference explorer over small random Models (2-3 small vars, 2-4 guarded actions) that must agree with `Checker::explore` on reached states and claim verdicts. Add `make model2-update`.

Implement `Checker::explore::<M>(budget)`: BFS from `starts()` with a `HashSet<M>` visited set and a parent map for paths; evaluate invariants on each new state and postconditions on each emitted edge with `before`, `after` and the action; `Result` with verdict, counts (starts, reached, emitted, depth, cut); `Result::verify_all() -> Report` with per-claim results and vacuity warnings (action never fired, `after` claim never triggered); `Checker::refuted(claim)` for negative controls. Hit the Scala note's Kettle counts: 81 starts, 283 states, depth 62.

**Files:** model2/framework/checker/src/lib.rs (+ explore.rs, report.rs), model2/framework/samples/tests/kettle.rs.

**Quick commands:** `mise exec -- cargo test -p framework-checker -p samples`
## Acceptance
- `cases/kettle/expected/report.txt` shows 81 starts, 283 states, depth 62, both claims verified, no vacuity; `paths/never_beeps.txt` holds the `!beeping` counterexample.
- Two consecutive runs produce byte-identical goldens.
- The oracle property test passes on 256 cases.
- A budget below one refuses; a small-budget case's report shows `cut: true` and unknown verdicts.
- A golden with no case fails the run.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
