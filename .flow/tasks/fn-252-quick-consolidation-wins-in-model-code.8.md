---
satisfies: [R8, R9]
---
# fn-252-quick-consolidation-wins-in-model-code.8 Move the stock test fixture out of umpire_runtime

## Description
Implement **Runtime fixture (B12)**: move the `#[path]` Kettle fixture from `umpire/protocol/runtime/src/stock.rs` into the stock guest crate.

## Acceptance
- [ ] `umpire_runtime::Model` is no longer exported; the stock guest builds for wasm32-wasip1.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
