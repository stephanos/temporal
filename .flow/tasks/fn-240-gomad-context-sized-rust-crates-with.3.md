---
satisfies: [R3, R5, R11, R12]
---
# fn-240-gomad-context-sized-rust-crates-with.3 Extract wasm-sim crate (clock, network, volume)

## Description
Move wasmhost/src/{clock,network,volume} (+event_order, stress_jitter if survey confirms no wasmtime/engine edge) into a wasm-sim crate with no wasmtime dependency. namespace stays out (imports wasmtime). Existing corpus/conformance tests become tests/ boundary tests. Go helper build and artifact path unchanged; shared cargo target dir convention preserved. Ordering: after fn-232 chain lands (largest open wasmhost diff); conductor checks open branches for overlap before claiming. R11: the extracted crate's public functions return its own error enum (no Result<_, String> in pub items; map_err to_string chains become From impls); pinned messages stay byte-identical. R12: remove the #[allow(too_many_lines|cognitive_complexity)] on any function this task rewrites and update .flow/artifacts/fn-240/lint-allows.md.

## Acceptance
grep wasmtime under wasm-sim is empty; Go helper build unchanged; cargo test --workspace green; contract doc present; measurements in .flow/artifacts/fn-240/wasm-sim.md. No pub Result<_, String> in the crate; allow-list did not grow.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
