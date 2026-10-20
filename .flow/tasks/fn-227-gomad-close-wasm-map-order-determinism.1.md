---
satisfies: [R1, R2, R3, R4]
---
# fn-227-gomad-close-wasm-map-order-determinism.1 Sort live-node shutdown and invalid-volume validation with behavioral regressions and qualification

## Description
WASM wasi package owner: close the three map-order escapes captured by fn-188. Sort ClusterSession.Shutdown by node ID and incarnation before stop frames; sort both storage crash maps by volume before validating. Preserve shutdown semantics and refusal classification. Replace escape rows/caveats only with behavioral regression coverage and retain changed model identity qualification.

## Acceptance
Vary insertion order with multiple live cluster nodes and multiple invalid volumes; World/network evidence and refusal messages must match. Inventory rows name behavioral tests and become controlled. Inventory/architecture/source checks pass. Retain frozen-candidate WASM runtime/workload/replay qualification after model identity rotation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
