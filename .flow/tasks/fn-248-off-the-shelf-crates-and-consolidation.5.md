---
satisfies: [R1, R5]
---
# fn-248-off-the-shelf-crates-and-consolidation.5 thiserror and prost-types in runtime, catalog and DSL; remove enum-iterator

## Description
Library-level crate swaps in the runtime, catalog and DSL.

- `thiserror` for hand-written `Display` + `Error` impls: umpire-catalog `gap.rs` (`GapError`), `lib.rs` (`UnknownEnum`), `value.rs` (`GenerationError`), `probe.rs` (`ProbeCoverageError`, `ProbeError`); umpire-dsl `catalog.rs` (`CatalogError`), `machine.rs` (`StepError`), `path.rs` (`PatternError`). Leave `AdmissionErrors` and `FillError` (per-item line loops) and non-error types.
- `prost-types` for the protobuf well-known types: drop `compile_well_known_types()` in `model/umpire/protocol/runtime/build.rs`; `src/any.rs` keeps a thin wrapper over `Any::from_msg`/`to_msg` that preserves the Malformed-versus-Unsupported refusal split; update the `extern_path` mappings in the Kettle and `temporal/api` build scripts to `::prost_types`.
- Remove `enum-iterator` from `model/umpire/author/dsl/Cargo.toml`, `[workspace.dependencies]`, and the `deny.toml` duplicate-version comment.

Both crates reach wasm guests: every guest still builds for `wasm32-wasip1`, and the guest dependency closure inventory is updated. Update affected Go generated-type mappings, `extern_path` resolution and in-tree readers as part of the prost-types change. Regenerate guests and current fixtures together; no adapter for old generated types, old guests or old wire bytes is required. Retain typed decoding and the Malformed-versus-Unsupported refusal contract.

## Acceptance
- [ ] The listed error types derive `thiserror::Error`; their messages are unchanged in meaning.
- [ ] Well-known types come from `prost-types`; `any.rs` still distinguishes Malformed from Unsupported.
- [ ] `enum-iterator` appears in no manifest or `deny.toml`.
- [ ] All guests build for `wasm32-wasip1`; `make model-check` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

