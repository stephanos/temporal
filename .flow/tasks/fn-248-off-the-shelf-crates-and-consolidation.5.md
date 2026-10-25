---
satisfies: [R1]
---
# fn-248-off-the-shelf-crates-and-consolidation.5 thiserror and prost-types in runtime, catalog and DSL; remove enum-iterator

## Description

**Size:** M
**Files:** model/umpire/protocol/runtime/**, model/umpire/catalog/**, model/umpire/author/dsl/**, model/umpire/reference/kettle/**, model/temporal/api/**, model/temporal/annotate/syntax/**, model/temporal/realization/**, model/temporal/guest/**, model/temporal/hosts/go/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml
**Touches:** [model/umpire/protocol/runtime/**, model/umpire/catalog/**, model/umpire/author/dsl/**, model/umpire/reference/kettle/**, model/temporal/api/**, model/temporal/annotate/syntax/**, model/temporal/realization/**, model/temporal/guest/**, model/temporal/hosts/go/**, model/Cargo.toml, model/Cargo.lock, model/deny.toml, model/temporal/annotate/src/lib.rs, model/temporal/annotate/tests/**, model/temporal/testing/realization_control.rs, model/temporal/feature/activity/temporal.rs, model/umpire/author/umpire/tests/consumer/src/lib.rs]

Isolated source lane after fn247 closure; no whole-task dependency on another fn248 source lane. Coordinate shared interfaces, accept central merge conflicts and retain private source/index/build targets. The final integration task owns snapshot reacceptance, current guest publication and the one R5 full pipeline. No local lane result claims joined validation.
Library-level crate swaps in the runtime, catalog and DSL.

- `thiserror` for hand-written `Display` + `Error` impls: umpire-catalog `gap.rs` (`GapError`), `lib.rs` (`UnknownEnum`), `value.rs` (`GenerationError`), `probe.rs` (`ProbeCoverageError`, `ProbeError`); umpire-dsl `catalog.rs` (`CatalogError`), `machine.rs` (`StepError`), `path.rs` (`PatternError`). Leave `AdmissionErrors` and `FillError` (per-item line loops) and non-error types.
- Use prost-build's consistent default well-known-type mappings across runtime, Kettle and shared Temporal codegen: remove `compile_well_known_types()` and the custom package-wide google.protobuf extern mapping from `model/umpire/protocol/runtime/build.rs`, `model/umpire/reference/kettle/build.rs` and `model/umpire/catalog/build/src/codegen.rs` (called by `temporal/api/codegen.rs`). Nonprimitive WKTs use `prost-types`; `Empty` maps to `()`, scalar wrappers map to their Rust primitives (`DoubleValue` to `f64`, etc.), and string/bytes wrappers use the corresponding Rust string/byte types. Do not replace the package prefix alone. `src/any.rs` keeps a thin native wrapper over `Any::from_msg`/`to_msg` that preserves the Malformed-versus-Unsupported refusal split.
- Remove `enum-iterator` from `model/umpire/author/dsl/Cargo.toml`, `[workspace.dependencies]`, and the `deny.toml` duplicate-version comment.

After joining, both crate changes must be accounted for in wasm guest closures: every guest still builds for `wasm32-wasip1`, and the guest dependency closure inventory is updated. Update affected Go generated-type mappings, `extern_path` resolution and in-tree readers as part of the prost-types change. The integration task regenerates necessary guests and current fixtures together after shared codegen/type paths are joined; no adapter for old generated types, old guests or old wire bytes is required. Retain typed decoding and the Malformed-versus-Unsupported refusal contract.

### Native type and refusal boundary

`prost_types::Any` is a foreign type, so the existing inherent impl cannot simply move onto it. Use a narrow local trait/free-function wrapper (or a justified local wrapper) and update all callers coherently. Keep explicit missing/canonical type-URL admission before calling `to_msg`: prost-types compares parsed full type names, so blindly delegating would accept a different URL prefix and merge failure classes. Test missing URL, wrong type/prefix, malformed payload and valid payload with the same Malformed-versus-Unsupported semantic split; old method names/aliases are unnecessary.

Reanchor all generated-type owners, not only the two build.rs mappings: catalog/build `wire_schema.rs` and Temporal annotate/syntax `realization.rs` also emit `::umpire_runtime::google::protobuf` paths. Reconcile these with .4's isolated codegen edits; verify generated Rust compiles with one consistent WKT mapping, including descriptor-derived primitive paths rather than nonexistent `prost_types::Empty`/`DoubleValue` structs. Add focused generation/consumer controls for the current Temporal fields `PendingActivityInfo.use_workflow_build_id` (Empty/unit) and `TaskQueueMetadata.max_tasks_per_second` (DoubleValue/f64), retaining protobuf presence and wire semantics. Do not alter schemas or Go protocol meaning merely to make the crate swap easier. Keep fn248's approved in-tree producer/reader updates and current integrity; no old-guest adapters.

## Acceptance
- [ ] The listed error types derive `thiserror::Error`; their messages are unchanged in meaning.
- [ ] Nonprimitive well-known types come from `prost-types`; Empty/scalar/string/bytes wrappers follow prost-build's default primitive mappings consistently in every generator and descriptor-derived reader. Both current Temporal Empty/DoubleValue field controls pass; `any.rs` still distinguishes Malformed from Unsupported.
- [ ] `enum-iterator` appears in no manifest or `deny.toml`.
- [ ] All guests build for `wasm32-wasip1`; Focused checks below pass; joined full validation belongs to the final integration task.


## Focused verification and integration handoff

Run from the repository root, with a lane-owned target; commands containing `cd model` enter the Rust workspace. These are mandatory affected-scope checks, not another full Model gate:

```bash
cd model && mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --test dispatch --locked
cd model && mise exec -- cargo test -p umpire-catalog -p umpire-dsl --lib --locked
cd model && mise exec -- cargo test -p umpire-temporal-api --test codegen --locked
cd model && mise exec -- cargo build --target wasm32-wasip1 -p umpire-runtime --locked
```

Lane-local candidate snapshots may be generated/reviewed to exercise focused controls in the isolated copy; final canonical reacceptance still occurs once after joining. Report actual failures and candidate snapshot diffs. Add controls specified by this task and run the new controls by their actual selectors. After central conflict resolution, rerun affected focused checks; final integration owns reviewed snapshot acceptance/current artifacts and R5. No historical output-byte comparison is required.

## Done summary
Replaced the eight named library error implementations with thiserror derives, adopted prost-build's default native WKT mappings throughout runtime/Kettle/Temporal generation and readers, and removed unused enum-iterator. Native Any retains exact canonical URL admission and the MalformedRequest/Unsupported split through one local extension trait.

baseline: green (all four task Quick commands before edits). The catalog/DSL library targets collected zero tests; the relevant integration suites subsequently passed.

Verification is recorded with exact commands, logs, exits and source hashes in verification.json, successful-commands.json and the stage-specific results files. This includes runtime wire/transaction controls, catalog/DSL integration controls, API generation/descriptors/mutations, Kettle and all Temporal realization tests, annotation/facade consumers, both affected-package all-targets Clippy commands with -D warnings, formatting, the targeted Go WKT presence test, and wasm32-wasip1 builds of runtime and all six guests. The wasm-to-final differences are test-only import/return cleanup, listed in source-differences.json; no compiled guest input changed afterward.

The new API codegen controls cover actual PendingActivityInfo Empty/unit and TaskQueueMetadata DoubleValue/f64 presence and wire round trips; wire_schema additionally mutates the wrapper's descriptor `.value` into the native scalar. Runtime controls cover missing payload/URL, wrong type/prefix, malformed bytes and valid payload. Removing the canonical URL guard fails its negative control. Primitive named-field mirrors fail explicitly instead of generating nonexistent native fields; their refusal control was observed assertion-red before restoration. Original failing and inconclusive observations remain in observations.json and their logs.

Both direct dependencies were already admitted and resolved; dependency-admission.json records current primary version/release evidence. guest-closure-inventory.json records the actual wasm target closures, including host proc/build separation. No snapshots changed, no canonical guest was published, and no whole-Model or all-Go pipeline ran. Task7 owns joined snapshot acceptance/publication and R5. The task4/task5 source join points and additional authorized mechanical caller paths are in join-notes.md.

stage: impl-review - ran (codex:gpt-6.1-sol:high; fresh three-draw review, same model family; authentic SHIP receipt under task5/review/receipt.json)
Tier: session (jev-unavailable(no_key))

Conductor integration used the actual premerge base ee1c1abbb05bf4a694e5879b2d6940df886e1bd6 and merge 3a0b9cc7f38a462edd701f5f9af751af3caad3eb. The sole conflict preserves native WKT primitive resolution before the reviewed shared prost naming fallback. All 1,715 join source hashes match correctedB e0eec761ce7c29595a42efc9271241e4d79d8763 exactly. The joined affected syntax/API command passed 26 tests with exit0; eight prior join commands and Clippy/format remain source-bound green in join45/review-fix. Original worker wasm builds are scoped source evidence, not current joined publication or full R5 acceptance. The final integration task remains unstarted and owns these deferred obligations.

The authentic pinned codex:gpt-6.1-sol:high review is same-family, readonly and fresh; its actual draw outputs and final receipt are durable in task5/review. Native Any strict admission and Empty/DoubleValue primitive presence controls remain. No schema, generated Go protocol meaning, snapshot acceptance or compatibility adapter was added during this integration.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8d7f14273476a5af9050673a2fc77c0f015b774d, 3cf2a8464aebb0706b5a2dd42f09210f3e3f6a01, e0eec761ce7c29595a42efc9271241e4d79d8763, 3a0b9cc7f38a462edd701f5f9af751af3caad3eb
- Tests: cd model && mise exec -- cargo test -p umpire-runtime --test wire --test core_protocol --test proposal_wire --test dispatch --locked, cd model && mise exec -- cargo test -p umpire-catalog -p umpire-dsl --lib --locked, cd model && mise exec -- cargo test -p umpire-temporal-api --test codegen --test wire_schema --locked, cd model && mise exec -- cargo test -p umpire-catalog --test generation --test probe --test black_box --locked, cd model && mise exec -- cargo test -p umpire-dsl --test path_domains --test path_language_patterns --test outcome --test catalog --locked, cd model && mise exec -- cargo test -p umpire-catalog-build --test neutral --locked, cd model && mise exec -- cargo test -p umpire-temporal-api --test descriptor --locked, cd model && mise exec -- cargo test -p umpire-temporal-annotate-syntax --test admission --test realize --test expansion --locked, cd model && mise exec -- cargo test -p umpire-reference-kettle --test realization --test annotations --test descriptor --locked, cd model && mise exec -- cargo test -p umpire-temporal --tests --locked, cd model && mise exec -- cargo test -p umpire-temporal-api --test codegen --test wire_schema --test descriptor --locked, cd model && mise exec -- cargo test -p umpire-runtime --test stateful --test proposal --test selected_start --locked, cd model && mise exec -- cargo test -p umpire-temporal --test admission --locked, cd model && mise exec -- cargo test -p umpire-temporal-annotate-syntax --lib --locked, cd model && mise exec -- cargo build --target wasm32-wasip1 -p umpire-runtime --locked, cd model && mise exec -- cargo build --target wasm32-wasip1 -p umpire-stock-guest -p umpire-control-guest -p umpire-caught-panic-guest -p umpire-overflow-guest -p umpire-kettle-guest -p umpire-temporal-guest --locked, cd model && mise exec -- cargo clippy -p umpire-runtime -p umpire-catalog -p umpire-dsl -p umpire-catalog-build -p umpire-catalog-syntax -p umpire-reference-kettle -p umpire-temporal-api -p umpire-temporal-annotate-syntax -p umpire-temporal --all-targets --locked -- -D warnings, cd model && mise exec -- cargo test -p umpire-author-consumer --lib --locked, cd model && mise exec -- cargo test -p umpire-temporal-annotate --lib --test derive_catalog --test response_checks --locked, cd model && mise exec -- cargo clippy -p umpire-author-consumer -p umpire-temporal-annotate -p umpire-temporal-tests --all-targets --locked -- -D warnings, cd model/temporal/hosts/go && CGO_ENABLED=0 mise exec -- go test -tags test_dep -run '^TestNativeWellKnownPresence$' ., mise exec -- cargo fmt --manifest-path model/Cargo.toml --all -- --check, git diff --check, cd model && BASH_ENV=/dev/null CARGO_TARGET_DIR=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2485-target MISE_AUTO_INSTALL=false MISE_EXEC_AUTO_INSTALL=false mise exec -- cargo test -p umpire-temporal-annotate-syntax -p umpire-temporal-api --lib --test codegen --test wire_schema --test descriptor --test services --locked
- PRs: