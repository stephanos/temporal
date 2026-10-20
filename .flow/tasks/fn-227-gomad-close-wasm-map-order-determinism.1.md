---
satisfies: [R1, R2, R3, R4]
---
# fn-227-gomad-close-wasm-map-order-determinism.1 Sort live-node shutdown and invalid-volume validation with behavioral regressions and qualification

## Description
WASM wasi package owner: close the three map-order escapes captured by fn-188. Sort ClusterSession.Shutdown by node ID and incarnation before stop frames; sort both storage crash maps by volume before validating. Preserve shutdown semantics and refusal classification. Replace escape rows/caveats only with behavioral regression coverage and retain changed model identity qualification.

**Re-homing (owner decision 2026-10-20, Rust host rewrite):** complete only the parts that carry into the Rust host: the behavioral regression tests with varied insertion order, the determinism inventory rows naming them, and the qualification ledger entry. The Go `ClusterSession.Shutdown` and storage-crash validation sorting fixes are not completed here; R1 and R2 of fn-227 are owned by fn-232.8 (simulation exploration in Rust), where the cluster session ordering is implemented once in Rust. The tests written here become golden corpus cases there.

## Acceptance
Vary insertion order with multiple live cluster nodes and multiple invalid volumes; World/network evidence and refusal messages must match. Inventory rows name behavioral tests and become controlled. Inventory/architecture/source checks pass. Retain frozen-candidate WASM runtime/workload/replay qualification after model identity rotation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
