# fn232 exact determinism receiving source checkpoint

Owners: fn-232.13/R12 and fn-227.1. Base: `312087e126e5016272a6bdc7db6f5456d3e15ab5`. This checkpoint changes two existing receiving tests; it creates no IDs, waiver, task completion or final qualification claim. The protected inventory prefix, accepted counts and remaining 23 original references are unchanged.

Original source is sealed at `1d695061b6382009d0659238c652a5a765b1d5f5:tools/gomad/internal/backends/wasm/wasi/determinism_inventory_test.go`.

- Line 168, `TestDeterminismInventoryRejectedImportsAreRefused`: the original iterates inventory-rejected WASI imports, requires a probe for every row, checks the count and asserts unsupported plus exact namespace refusal message. The existing Rust counterpart `rejected_stock_import_inventory_has_an_actual_ordinary_refusal_for_every_site` retains its inventory/probe set equality and four ordinary guest executions, unsupported/no-exit/empty-output checks. Its message assertion now requires exact full native outcome equality. Native transport prefixes the operation (`path_readlink: `); the following inner text is exactly the original `operation outside stock WASI profile: path_readlink`, analogously for all four operations. No substring allowance remains.
- Line 203, `TestDeterminismInventoryCooperativeImportsRefuseUnlisted`: the original derives the accepted cooperative set from inventory and probes all listed names, three unknown names, a malformed duplicate `decision`, missing `idle` and cooperative imports in stock. The existing Rust counterpart `cooperative_profile_preserves_the_original_complete_import_boundary` now derives all `import:gomad_*` rows from the reviewed inventory instead of seven literals. It sorts the set and prepends the malformed duplicate decision by name, preserving the original duplicate shape despite the changed sorting. The complete existing matrix and test ID remain.

The cooperative test is a direct validator matrix. It does **not** independently prove caller ordering or execute a guest. Existing production preparation parses actual module metadata before calling the validator; retained replay also parses actual module bytes before validation and linked preparation. Independent behavioral error-precedence/no-side-effect probes of both callers remain required before claiming the entire original pre-launch predicate is received. Source call order does not substitute for those probes.

Validation in this source-only worktree: `rustfmt +1.99.1 --edition 2024` on both changed Rust files (exit 0); `git diff --check` (exit 0); pure Python resolution/parsing of the exact `include_str!` target (exit 0, seven unique cooperative rows, four rejected stock rows). No Cargo, Go or guest build/run was performed, and no target directory was allocated. Runtime assertions remain unexecuted for this candidate.

Root must run these focused commands serially with its assigned exclusive existing `CARGO_TARGET_DIR`, from the integrated frozen repository root:

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad-wasmhost --test core_runtime_preservation rejected_stock_import_inventory_has_an_actual_ordinary_refusal_for_every_site -- --exact
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad --lib replay::preflight::profile_import_tests::cooperative_profile_preserves_the_original_complete_import_boundary -- --exact
```

Run without helper diagnostic/conformance/determinism fixture features so the ordinary integration test is enabled. These focused tests supplement the owner's integrated lint/package/qualification gates; they do not replace final frozen qualification or close the caller-ordering gap.

Worktree `/home/agent/wt/fn232-determinism-exact` and branch `fn232-determinism-exact` remain for root integration; no other scratch checkout or build output was created. `/home/agent/determinism-required-docs.txt` is a disposable documentation read capture. Approximately 12 GiB free was observed before handover.

Fresh independent source review: `gpt-6.1-sol` at high, same family as the writer, fresh context `/root/determinism_exact_receiving/exact_review`. Verdict: no actionable findings; accepted the narrowed two-test source scope. Reviewer independently checked the inventory include path, original negative matrix, native `engine.rs` operation prefix and both exact test filters. Caller ordering and runtime qualification remain unresolved; the reviewer performed no builds or execution.
