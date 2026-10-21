---
satisfies: [R1, R2, R3, R4]
---
# fn-227-gomad-close-wasm-map-order-determinism.1 Sort live-node shutdown and invalid-volume validation with behavioral regressions and qualification

## Description
**Touches:** tools/gomad/internal/backends/wasm/determinism/**, tools/gomad/internal/backends/wasm/wasi/*_test.go, tools/gomad/testdata/parity/**, tools/gomad/internal/backends/wasm/{README.md,QUALIFICATION.md}. Runtime implementation is received from fn-232.8, not duplicated here.

Own the transferable regressions and inventory reconciliation for the three map-order sites captured by fn-188. Use current Go sorting/tests as semantic references; implement final Rust behavior in fn-232.8. Preserve shutdown semantics and refusal classification. Record actual identities and receive fn-232.13 qualification without a second Go-host cycle.

**Re-homing (owner decision 2026-10-20, Rust host rewrite):** complete only the parts that carry into the Rust host: the behavioral regression tests with varied insertion order, the determinism inventory rows naming them, and the qualification ledger entry. The Go `ClusterSession.Shutdown` and storage-crash validation sorting fixes are not completed here; R1 and R2 of fn-227 are owned by fn-232.8 (simulation exploration in Rust), where the cluster session ordering is implemented once in Rust. The tests written here become golden corpus cases there.

## Acceptance
Vary insertion order with multiple live cluster nodes and multiple invalid volumes in transferable regression cases; World/network evidence and refusal messages must match. Reconcile inventory rows against the actual current source and name passing behavioral tests; do not claim Rust coverage from Go tests. Inventory/architecture/source checks pass. Completion also requires linked fn-232.8 Rust ordering evidence and fn-232.13 frozen-candidate runtime/workload/replay evidence with actual identity changes. Until those exist, retain open status and distinguish prepared cases from completed acceptance.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
