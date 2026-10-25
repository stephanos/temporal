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
One operator table, leaf renderer and Div/Rem helper (`backends/tla/src/lower.rs`) now serve both TLA+ lowerings (`emit.rs` and `safety/value.rs`); the two lowerings stay separate. Emitted TLA+ is byte-identical (no golden changed; new Rem/Div pins pass on the old sources too). The review found a pre-existing bug, fixed here: the safety lowering emitted `* -5` for a negative literal divisor, which SANY rejects; operands are now parenthesised. Fresh review NEEDS_WORK → fixes 1c57aed017. Covered by the joined fn-248.7 `UMPIRE_APALACHE=require make model-validate` at 9b3e26654a (exit 0; .flow/artifacts/fn-248-off-the-shelf-crates-and-consolidation/task7/validation.md).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 767f7c71d5, 1c57aed017
- Tests: cargo test -p umpire-tla -p umpire-fixtures -p umpire-testing -p umpire-apalache -p umpire-tlc -p umpire-temporal-tests, UMPIRE_APALACHE=require make model-validate (9b3e26654a, exit 0)
- PRs: