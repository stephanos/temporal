---
satisfies: [R4]
---
# fn-247-gomad-protobuf-host-side-evidence.5 Move volume image and storage evidence to protobuf

## Description
Replace `GMVOLIMG` (`wasmhost/src/volume/image.rs`, Go `internal/model/volume/recovery_image.go`, `internal/model/wasi/storage_image.go`) and the storage evidence streams (`storage_transport`, footer/section layout) with `prost`/Go protobuf messages, unless the task 1 inventory classifies one as guest-boundary. Keep every existing limit and refusal.

Complexity amendment (owner, 2026-10-24): fn-249 retires the test-only Go reference models first, so this task generates and updates no Go protobuf code. Conformance against the Go model is replaced by the Rust corpus tests fn-249 captures.
## Acceptance
- [ ] Migrated formats have no hand-written codec; limit and refusal tests pass on real evidence; guest-boundary exceptions cite the inventory row.
- [ ] No Go volume/storage image codec remains.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
