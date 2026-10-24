---
satisfies: [R4]
---
# fn-247-gomad-protobuf-host-side-evidence.4 Move network evidence (GMNETEV1) to protobuf

## Description
Replace the `GMNETEV1` encoder (`wasmhost/src/cluster/network/evidence.rs`) and decoder (`evidence/decode.rs`) with `prost` messages, keeping the `transcript_bytes` bound and the capacity outcome when exceeded. Update the Go inspection check in `internal/model/cluster/wasm_inspection.go` and the Rust test helpers (`tests/common/network_inspection.rs`, `closed_cluster_guards.rs`). Delete their varint/flag helpers.

Complexity amendment (owner, 2026-10-24): fn-249 retires the test-only Go reference models first, so this task generates and updates no Go protobuf code. Conformance against the Go model is replaced by the Rust corpus tests fn-249 captures.
## Acceptance
- [ ] No hand-written varint/frame codec remains for network evidence; bound and refusal tests pass on real evidence.
- [ ] No Go network-evidence check remains.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
