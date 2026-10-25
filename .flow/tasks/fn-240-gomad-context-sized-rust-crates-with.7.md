---
satisfies: [R5, R6, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.7 Extract wasm-model and the wasm-conformance bin

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Description
Move protocol, transcript, abi message types, model, local_model, namespace, storage_transport, storage_evidence_bounds, metrics, module_metadata, profile_inventory into crates/wasm-model, depending on wasm-sim and wasm-hooks only. Invert the survey's upward edges: abi->engine (1), protocol->guest_profile (1), model->linked (2), storage_evidence_bounds->linked (1), storage_transport->linked_simulation (resolved by .3). Turn conformance.rs into bins/wasm-conformance linking wasm-model only and delete the conformance feature and all cfg(not(feature = conformance)) lines from lib.rs. Makefile conformance recipe builds the bin by package name. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); typed error kinds and documented refusal behavior remain tested; message text and goldens may change. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

Throughput amendment (owner, 2026-10-24): convert this crate's message-prefix dispatch (`runner_failure:`, `refused:`, `capacity:`) to the fn-240.5 kind enum in the same pass as R11's error enums.

Scope note (owner, 2026-10-24): fn-249 deletes the `conformance` binary and feature before this task. Extract `wasm-model` only; there is no `wasm-conformance` bin to create.
## Acceptance
cargo tree -p wasm-model -i wasmtime empty; cargo tree -p wasm-conformance -i wasmtime empty; conformance corpus (make validate-replay-plan and the Go-parity conformance lane) passes with the new bin; no cfg(feature = conformance) remains; contract doc and tests/ present; measurements in .flow/artifacts/fn-240/wasm-model.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
