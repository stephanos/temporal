---
satisfies: [R1, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.1 One shortest-path search in the suite

## Description
Implement **One search (A3)**: `space.rs:254` and `run.rs:123` use `pattern_search::Graph::shortest` over `Path::alternatives`; delete `search.rs`'s `Along`/`Through`/`arrive`/`trace`/`settled`. Reconcile the cut-frontier rule and `Unbridged.waypoint` mapping; measure speed (keep the old path if the default gate slows by more than 25 percent).

## Acceptance
- [ ] `search.rs`'s parallel search is gone; witnesses and Unbridged reports unchanged.
- [ ] Timing before/after recorded.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
