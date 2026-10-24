---
satisfies: [R6, R7]
---
# fn-251-one-probe-engine-and-one-catalog-build.6 Delete the temporal/annotate facade crates

## Description
Implement **Facade crates (B4)**: delete `temporal/annotate` and `temporal/annotate/macros`; add a crate-root `CatalogContext` alias and api re-export; switch the four importers; move the view tests to `catalog/tests`.

## Acceptance
- [ ] Both crates are gone from the workspace; importers build.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
