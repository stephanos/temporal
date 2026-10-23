---
satisfies: [R3, R5]
---
# fn-240-gomad-context-sized-rust-crates-with.3 Extract wasm-sim crate (clock, network, volume)

## Description
Move wasmhost/src/{clock,network,volume} (+event_order, stress_jitter if survey confirms no wasmtime/engine edge) into a wasm-sim crate with no wasmtime dependency. namespace stays out (imports wasmtime). Existing corpus/conformance tests become tests/ boundary tests. Go helper build and artifact path unchanged; shared cargo target dir convention preserved. Ordering: after fn-232 chain lands (largest open wasmhost diff); conductor checks open branches for overlap before claiming.

## Acceptance
grep wasmtime under wasm-sim is empty; Go helper build unchanged; cargo test --workspace green; contract doc present; measurements in .flow/artifacts/fn-240/wasm-sim.md.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
