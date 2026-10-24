---
satisfies: [R2]
---
# fn-247-gomad-protobuf-host-side-evidence.2 Replace the Go deflate port and hand-written gzip with flate2

## Description
Swap `wasmhost/src/world/gzip.rs` to `flate2` (`rust_backend`, pinned `=`) with the same bounds (384 MiB expanded, 64 MiB compressed, first member only, consumed-byte return). Replace the heap profile's hand-written stored-block gzip in `linked_inspection.rs` with the same helper. Delete `world/deflate/` (port, Go oracle, README, GO-LICENSE) and its rows in the determinism inventory, `internal/archcheck/inventory.go` exclusions and `rust/driver/static-ignored.json` if present. Go conformance already gunzips with stock `compress/gzip`.

## Acceptance
- [ ] No hand-written deflate/gzip codec remains; decode refuses truncation, corruption, trailing data and over-limit expansion with tests on real envelopes.
- [ ] `cargo test --workspace` and `make -C tools/gomad test-rust-engine validate-wasm-determinism` green.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
