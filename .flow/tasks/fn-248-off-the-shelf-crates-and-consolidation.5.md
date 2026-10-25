---
satisfies: [R1]
---
# fn-248-off-the-shelf-crates-and-consolidation.5 thiserror and prost-types in runtime, catalog and DSL; remove enum-iterator

## Description

**Size:** M
**Files:** model/umpire/protocol/runtime/**, model/umpire/catalog/**, model/umpire/author/dsl/**, model/umpire/reference/kettle/**, model/temporal/api/**, model/temporal/annotate/syntax/**, model/temporal/realization/**, model/temporal/guest/**, model/temporal/hosts/go/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml
**Touches:** [model/umpire/protocol/runtime/**, model/umpire/catalog/**, model/umpire/author/dsl/**, model/umpire/reference/kettle/**, model/temporal/api/**, model/temporal/annotate/syntax/**, model/temporal/realization/**, model/temporal/guest/**, model/temporal/hosts/go/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Library-level crate swaps in the runtime, catalog and DSL.

- `thiserror` for hand-written `Display` + `Error` impls: umpire-catalog `gap.rs` (`GapError`), `lib.rs` (`UnknownEnum`), `value.rs` (`GenerationError`), `probe.rs` (`ProbeCoverageError`, `ProbeError`); umpire-dsl `catalog.rs` (`CatalogError`), `machine.rs` (`StepError`), `path.rs` (`PatternError`). Leave `AdmissionErrors` and `FillError` (per-item line loops) and non-error types.
- `prost-types` for the protobuf well-known types: drop `compile_well_known_types()` in `model/umpire/protocol/runtime/build.rs`; `src/any.rs` keeps a thin wrapper over `Any::from_msg`/`to_msg` that preserves the Malformed-versus-Unsupported refusal split; update the `extern_path` mappings in the Kettle and `temporal/api` build scripts to `::prost_types`.
- Remove `enum-iterator` from `model/umpire/author/dsl/Cargo.toml`, `[workspace.dependencies]`, and the `deny.toml` duplicate-version comment.

After joining, both crate changes must be accounted for in wasm guest closures: every guest still builds for `wasm32-wasip1`, and the guest dependency closure inventory is updated. Update affected Go generated-type mappings, `extern_path` resolution and in-tree readers as part of the prost-types change. The integration task regenerates necessary guests and current fixtures together after shared codegen/type paths are joined; no adapter for old generated types, old guests or old wire bytes is required. Retain typed decoding and the Malformed-versus-Unsupported refusal contract.

### Native type and refusal boundary

`prost_types::Any` is a foreign type, so the existing inherent impl cannot simply move onto it. Use a narrow local trait/free-function wrapper (or a justified local wrapper) and update all callers coherently. Keep explicit missing/canonical type-URL admission before calling `to_msg`: prost-types compares parsed full type names, so blindly delegating would accept a different URL prefix and merge failure classes. Test missing URL, wrong type/prefix, malformed payload and valid payload with the same Malformed-versus-Unsupported semantic split; old method names/aliases are unnecessary.

Reanchor all generated-type owners, not only the two build.rs mappings: catalog/build `wire_schema.rs` and Temporal annotate/syntax `realization.rs` also emit `::umpire_runtime::google::protobuf` paths. Reconcile these with .4's isolated codegen edits; verify generated Rust compiles with one WKT type identity. Do not alter schemas or Go protocol meaning merely to make the crate swap easier. Keep fn248's approved in-tree producer/reader updates and current integrity; no old-guest adapters.

## Acceptance
- [ ] The listed error types derive `thiserror::Error`; their messages are unchanged in meaning.
- [ ] Well-known types come from `prost-types`; `any.rs` still distinguishes Malformed from Unsupported.
- [ ] `enum-iterator` appears in no manifest or `deny.toml`.
- [ ] All guests build for `wasm32-wasip1`; Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --test dispatch --locked
cd model && mise exec -- cargo test -p umpire-catalog -p umpire-dsl --lib --locked
cd model && mise exec -- cargo build --target wasm32-wasip1 -p umpire-runtime --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

