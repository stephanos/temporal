---
satisfies: [R2, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.2 Path stores only a Pattern

## Description
Implement **Path stores a Pattern (A4)**: drop `first`/`leg`/`more` in `dsl/src/path.rs:686-870` and `IntoWorldPath<kind::Legacy>`; derive `claim()` from the pattern.

## Acceptance
- [ ] No panicking `Path` accessors; legacy representation gone.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
