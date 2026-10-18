---
satisfies: [R3]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.3 Progress: whenever/eventually within a scope, weak fairness on internal steps, lasso counterexamples

## Description
**Deps 2026-10-18 (R18):** build the scoped graph as a `petgraph` `DiGraph` over `StateId`s and use `tarjan_scc`; a fair cycle is a non-trivial SCC inside the scope where every weakly-fair internal step enabled throughout is taken; the lasso is a BFS prefix to the SCC plus a cycle within it.

**Recut 2026-10-18 (R8, golden files):** progress results land in `report.txt` and lasso counterexamples in `paths/<claim>.txt`; the no-heater control Model is a case of its own. Extend the oracle explorer with the same progress check.

Implement progress checking over the explored graph restricted to the scope's action set: from every state where `whenever` holds, every fair path reaches a state where `eventually` holds; weak fairness on internal steps; a refutation is a lasso (prefix plus cycle) or a dead end. Verify Kettle's `boils` within `without(DepressButton)`; refute it on a control Model whose heater never fires.

**Files:** model2/umpire/checker/src/progress.rs, tests.

**Quick commands:** `mise exec -- cargo test -p umpire-checker -p samples`
## Acceptance
- Kettle's `report.txt` lists `boils` verified; the control case's golden holds the lasso (prefix and cycle marked).
- The oracle test covers progress.
- An unknown scope action refuses with its name (trybuild golden).
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
