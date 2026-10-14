---
satisfies: [R3]
---
# fn-175-gomad-filesystem-namespace-in-the-rust.4 Namespace image in the execute request: Go flattening, chunked frames, Rust validation

## Description
Carry the namespace configuration from Go to the helper and build the Rust namespace before guest start (R3). Go keeps captured-input decoding and validation, which it already does. The helper receives a flattened, pre-validated namespace image, re-validates its structure and limits, and refuses bad input as `invalid` before instantiation.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/helper_model.go` (namespace image builder, sender), `tools/gomad/internal/backends/wasm/wasi/process.go` (gate and frame sequence), `tools/gomad/internal/backends/wasm/wasi/namespace.go` (export the flattened image from the constructed Go namespace), `tools/gomad/internal/backends/wasm/wasi/identity.go` (embed list if new Go files), `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs` (read image frames after tape), `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/namespace.go, tools/gomad/internal/backends/wasm/wasi/identity.go, tools/gomad/internal/backends/wasm/wasi/helper_model_test.go, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs]

### Approach
- Gate (decision): the helper owns the namespace when it owns the cooperative model (process.go:324: cooperative, `!remoteModel`, no `NetworkV2`, no `Broker`) and `Config.Storage == nil`. All other runs (stock profile, storage, broker, network v2) keep the Go namespace. Network-v2 runs always attach a broker, so they get the Rust namespace through fn-176's broker split mode and fn-178, not through this gate.
- Image (decision): Go builds its namespace as today (`newNamespace`, namespace.go:107), which keeps all current validation and `invalid`/`capacity` classes. It then exports a flattened image from it:
  - the preopen list in order
  - directories and read-only files with bytes, in ino order so inode numbers match
  - writable dirs, cwd and stdin
  - the namespace limits (descriptors, files, filesystem bytes, pending events) and the clock epoch
- Transport: the execute frame gains a `namespace` descriptor with `chunks`, `files`, `bytes` and an image SHA-256. After authorization and after tape chunks (`sendHelperTape`, process.go:600, is the precedent), Go sends exactly `chunks` binary-safe image frames, each ≤ 8 MiB.
- Rust validation, all `invalid` before instantiation:
  - aggregate bytes ≤ `FilesystemBytes`
  - node count ≤ `Files`
  - absolute clean paths, no duplicates, parents present
  - read-only files only under captured roots
  - the image digest matches
- Format: a small binary record format with magic and version, decoded strictly (no trailing data, unknown record kinds or truncation). Do not add a JSON scanner; any JSON fields in the execute frame use the existing strict serde path (`deny_unknown_fields`), consistent with fn-153.20.
- Rust builds `namespace.rs` (fn-175.2) from the image and holds it in `State`. Nothing routes to it yet (fn-175.5).
- Fs calls still reach Go in this task. fn-175.5 adds the guard that an fs callback reaching Go in a gated run is a protocol violation (`invalid`).
- New Go files must be added to the `identity.go` embed list and inventory (identity.go:17,33).

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/process.go:184-340` (execute frame, gate, frame sequence)
- `tools/gomad/internal/backends/wasm/wasi/helper_model.go:68-150` (helperConfig, tape sender, evidence receiver)
- `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs:286-340` (Execute validation)
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:164-190` (`read_tape` chunk precedent)
- `tools/gomad/internal/backends/wasm/wasi/identity.go`

### Key context
- fn-174.5 also extends the execute request (argv, environment, entropy key). Add the `namespace` field beside those fields; do not reshape them.
## Acceptance
- [ ] Gated runs deliver the namespace image and the helper builds an identical namespace: the fn-175.1 digest of the Go-built and the Rust-built namespace is equal for the activity-selection config and for the captured-input fixtures (`TestCaptured*`).
- [ ] Malformed, truncated, oversized (bytes or files over limit), duplicate-path and digest-mismatched images each terminate as `invalid` before `_start`, covered by `wasmhost/tests/engine.rs` cases.
- [ ] Writes into a captured read-only input return the read-only errno (69) on the Rust side (conformance script) and in a guest fixture.
- [ ] Non-gated runs (storage, broker, network v2, stock, `remoteModel`) send no image and behave as before (`make -C tools/gomad/internal/backends/wasm test-model` and helper bridge tests pass).


## Done summary
Go exports a flattened, strictly versioned GMWN namespace image (declared chunks/files/bytes/SHA-256) after authorization and tape; the helper re-validates and builds its namespace before instantiation, refusing malformed/truncated/oversized/duplicate/digest-mismatched images as `invalid`. Live conformance sessions now carry the production image and require identical canonical state; activity-selection and captured-input configurations are covered. Writes into captured inputs return ROFS (69). Impl review: NEEDS_WORK -> fixed (8e14d75969, d0372e7a20) -> SHIP.

Native execution remains deferred (fn-128/fn-149). Performance proof belongs to the .6 task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fast-batch tip 08f92e58ee (frozen qualification candidate d0372e7a20)
- Tests: frozen qualification /home/agent/wt/fast-qual-20261014T1345: run-qualification.sh + continuations (stages.log ends PASS frozen-qualification module=gomad), make -C tools/gomad/internal/backends/wasm test-conformance, cargo +1.94.1 test/clippy -D warnings (default, diagnostic-profile, conformance)
- PRs: