---
satisfies: [R2]
---
# fn-175-gomad-filesystem-namespace-in-the-rust.1 Namespace conformance harness: Rust driver, Go differential and state digest

## Description
Build the Go-vs-Rust differential conformance harness first (R2), so every later fn-175 task lands against it. It is also the harness base that fn-176 (volume) and fn-178 (World, clock) extend, so keep the driver protocol model-agnostic.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml` (new non-default `conformance` feature), `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs` (new: driver loop), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs` (feature-gated entry), `tools/gomad/internal/backends/wasm/wasi/namespace_conformance_test.go` (new), `tools/gomad/internal/backends/wasm/wasi/namespace.go` (state digest), `tools/gomad/internal/backends/wasm/wasi/testdata/namespace-conformance/` (golden corpus), `tools/gomad/internal/backends/wasm/Makefile` (`test-conformance` target)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasi/namespace.go, tools/gomad/internal/backends/wasm/wasi/namespace_conformance_test.go, tools/gomad/internal/backends/wasm/wasi/testdata/namespace-conformance/**, tools/gomad/internal/backends/wasm/Makefile]

### Approach
- Driver protocol: one model selector per session (`namespace` now; leave room for `internal/model/volume`, `internal/model/world` and `clock`). Records are framed as length-prefixed or LF-delimited records of `{op, input}` using the existing protocol call/reply input shapes (protocol.md table), so recorded transcripts can be replayed unchanged. Output is `{errno, output, unsupported, message}` or a boundary error (`invalid:<dim>` or `capacity:<dim>`), plus a `digest` record on request.
- Build the feature into a separate target directory (`wasmhost/target-conformance`), following the `diagnostic-profile` rule in protocol.md "Separate diagnostic builds". It is never installed and never qualifies.
- Go side: the Go test drives `Environment.Handle` (environment.go:259) directly as the reference, and spawns the conformance helper (`GOMAD_WASM_CONFORMANCE_HELPER`, or the Makefile-built path; skip with a clear message when absent, like `helperPath`, process_test.go:280).
- Namespace state digest (new, harness-only, not runtime evidence): SHA-256 over a canonical sorted tree walk (path, kind, readonly, ino, size, data digest, modified), the descriptor table (fd, node ino, offset, rights, inheriting, flags, preopen), `nextFD`, `nextInode`, files and bytes, and offered output. Define it once in a short format note in the test file and implement the same walk on both sides.
- Sequences:
  - (a) Hand-written scripts covering every spec Edge Case errno: bad fd, missing rights, dir vs file, read-only, ENOENT, `..` escape/absolute/NUL/>4096 path, offset overflow, write-buffer, filesystem-bytes, files and descriptors capacity.
  - (b) Seeded generated sequences over a small path alphabet, with a printed seed for repro.
  - (c) Recorded sequences extracted from retained guest transcripts (fs ops only).
- Persist (a) plus a small (c) sample as the checked-in golden corpus with expected replies and digests. A plain `cargo test` replays it against the Rust namespace once it exists; until fn-175.2 lands, the Rust side reports `unsupported` and the corpus test is `#[ignore]`d with a reason.
- Regenerating the corpus from the Go reference is a Go test flag (`-update-conformance`); a drift test fails when Go output differs from the checked-in corpus.
- Drift guard: wire `test-conformance` into `make -C tools/gomad test-wasm`, so a change to the Go namespace reruns the live differential. The checked-in corpus drift test also runs in plain `go test`.
- Generator bounds: path depth ≤ 4, file size ≤ 64 KiB, fds up to `Descriptors`, with deliberate exhaustion runs for files/descriptors/filesystem-bytes limits.

### Investigation targets
**Required** (read before coding):
- `tools/gomad/internal/backends/wasm/wasi/environment.go:259-445` (handle, reservation and dispatch order the driver must mirror)
- `tools/gomad/internal/backends/wasm/wasi/environment_test.go` (`testConfig`, `newTestEnvironment`, `call`, `openPath` helpers to reuse)
- `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs`, `tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml` (`diagnostic-profile` feature pattern)
- `tools/gomad/internal/backends/wasm/wasi/process_test.go:280` (helper discovery and skip pattern)

**Optional:**
- `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs` (spawning `CARGO_BIN_EXE_gomad3-wasmhost`)

### Key context
- Assumes no fn-174 interface: the driver talks to a pure Rust namespace module, not to the ABI or transcript layer.
- Compare raw reply bytes and digests, not decoded JSON, to avoid cross-language JSON parsing differences.
- Do not build the harness on the `diagnostic-profile` feature; the harness must run on ordinary test builds.

## Acceptance
- [ ] `go test -tags test_dep -run TestNamespaceConformance ./internal/backends/wasm/wasi` (from `tools/gomad`) runs the scripted, generated and recorded corpora through the Go namespace and the conformance helper, and reports the first diverging op with seed and index.
- [ ] Every spec Edge Case errno has at least one scripted sequence in the checked-in corpus; the Go drift test fails if Go output changes without `-update-conformance`.
- [ ] `make -C tools/gomad/internal/backends/wasm test-conformance` builds the feature helper in its separate target dir and runs the Go differential; the default `make helper` binary does not contain the driver (feature off).
- [ ] Driver protocol and digest are documented in one short section of `wasmhost/protocol.md` ("Conformance driver"), marked as test-only.


## Done summary
Model-agnostic Go-vs-Rust conformance harness: test-only `conformance` feature helper in `wasmhost/target-conformance`, checked-in corpus with every Edge Case errno, drift test with `-update-conformance`, harness-only state digest, 100 seeded sessions x 128 ops over a depth-4 alphabet with exhaustion seeds for files/descriptors/filesystem and output bytes. After review the driver admits, records and answers typed filesystem ops, clock reads and polls through the production `namespace_call`, `transcript` and new `local_model` code; `test-wasm` runs the differential once. Impl review: NEEDS_WORK -> fixed (aa6783e923, 0f3fc73141) -> SHIP.

Native execution remains deferred (fn-128/fn-149). Performance proof belongs to the .6 task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fast-batch tip 08f92e58ee (frozen qualification candidate d0372e7a20)
- Tests: frozen qualification /home/agent/wt/fast-qual-20261014T1345: run-qualification.sh + continuations (stages.log ends PASS frozen-qualification module=gomad), make -C tools/gomad/internal/backends/wasm test-conformance, cargo +1.94.1 test/clippy -D warnings (default, diagnostic-profile, conformance)
- PRs: