---
satisfies: [R4, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.4 Shared operator table and leaf rendering for the two TLA+ lowerings

## Description
Implement **TLA+ lowering table (A8)**: `tla/src/emit.rs` `render` and `tla/src/safety/value.rs` share one `Op`→symbol table, leaf rendering and Div/Rem handling. Do not unify the lowerings.

## Acceptance
- [ ] One operator table and leaf renderer used by both lowerings.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
