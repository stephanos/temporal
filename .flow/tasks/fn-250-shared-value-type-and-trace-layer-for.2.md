---
satisfies: [R2, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.2 Schema keyed by ir::Type; remove the wire Type and dead Set/Record decoding

## Description
Implement **One type system (A1)**: delete the wire `Type` (`tla/src/values.rs`), key `Schema` by `ir::Type` with one `accepts`, add emitter-owned `Module::schema()`/`safety_schema()` replacing the five derivations (tlc `module.rs`, apalache `replay.rs`, three tla tests), serve ITF `varTypes` from `tla/src/snowcat.rs` (remove `apalache/src/decode.rs` `wire_type`), and remove wire Set/Record with their TLC/ITF decoders and `default_value`.

## Acceptance
- [ ] No wire `Type`; one Snowcat printer; no Set/Record decoding code.
- [ ] Values outside their declared type refuse decoding in TLC and Apalache.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
