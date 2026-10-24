---
satisfies: [R5, R10, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.8 Split wasm-engine from wasm-cluster; wasmhost helper bin

## Description
Residual wasmhost becomes crates/wasm-engine (engine, abi dispatch, cache, linked, linked_* helpers folded into src/linked/, execute half of linked_simulation, module_metadata) and crates/wasm-cluster (cluster/{handoff,execution,network}, linked_cluster). Invert engine->cluster (7 sites): the store's broker member and network endpoint become a slot trait implemented by wasm-cluster; 'cluster requested' frame dispatch moves to bins/wasmhost. bins/wasmhost holds main.rs with the reference-helper feature and the publish-executable.sh artifact path unchanged. Fallback recorded in Amendment 4: if the inversion exceeds about twenty sites, keep engine+cluster as one crate and record why. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); pinned messages stay byte-identical. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

## Acceptance
wasm-engine is the only crate whose public API names wasmtime types (grep over crates/*/src for 'wasmtime::' outside wasm-engine is empty); helper artifact path unchanged; helper corpus, engine, cluster, linked_* integration tests pass from their new crates; inversion site count recorded; measurements in .flow/artifacts/fn-240/wasm-engine.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
