---
satisfies: [R2]
---
# fn-183-gomad-wasm-post-migration-cleanup.2 Remove discarded recovery reconstruction without weakening validation

## Description
Remove RecoveryImage::decode namespace/accounting state that recover immediately rebuilds, preserving R2 semantics and all independent limits.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasmhost/src/volume/image.rs, namespace/volume helpers if needed, wasi/volume_test.go, storage evidence tests
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/volume/**, tools/gomad/internal/backends/wasm/wasmhost/src/namespace/**, tools/gomad/internal/backends/wasm/wasi/volume_test.go, tools/gomad/internal/backends/wasm/wasi/*storage*test.go]

### Approach
- Trace decode's fs.nodes/used_bytes construction and recover's clone_durable before changing representation. Keep durable volumes, counters and occurrences needed by recovery; construct the final namespace once.
- Separate validation from discarded construction. Keep schema, digest, parent/inode uniqueness, counters, ephemeral-path rejection and image limits before accepting an image. Use malformed images with recomputed valid digests so checksum rejection cannot hide lost structural validation.
- Preserve complete-state evidence, refused-history and recovery-image bounds independently, including saturating arithmetic and header admission. Do not derive evidence capacity from image size.
- Behavior pin: run identical Go/Rust recovery scripts before/after and compare image bytes/identities, replies, snapshots, fault counts and post-restart effects. Keep all cross-directory rename sync combinations.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/wasmhost/src/volume/image.rs:275
- tools/gomad/internal/backends/wasm/wasmhost/src/volume/state.rs
- tools/gomad/internal/backends/wasm/wasi/volume_test.go
- tools/gomad/internal/backends/wasm/wasi/storage_evidence.go
- tools/gomad/internal/backends/wasm/wasi/testdata/volume-conformance/README.md

### Quick commands
`cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked volume`; run volume conformance and storage-evidence controls from the backend's documented test-conformance target.

## Acceptance
- [ ] Redundant decode reconstruction removed; final recovered state and current image bytes/identities unchanged
- [ ] Authenticated malformed images with repeated inode references, duplicate objects, invalid counters/parents and forbidden paths remain refused
- [ ] Truncation, exact-limit/+1 and independent image/state/refused-history capacity tests preserve classifications
- [ ] Recovery differential and cross-directory rename sync cases preserve durable state, operation and fault occurrence counters

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
