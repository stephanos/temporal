---
satisfies: [R3]
---
# fn-174-gomad-wasm-helper-local-host-call-fast.4 Compact clock run records in both owners

## Description
Fold consecutive clock reads into one run record in Go and the helper, so the digest still binds every timestamp without one call/reply record per read (R3).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/transcript.go`, `tools/gomad/internal/backends/wasm/wasi/clock.go`, `tools/gomad/internal/backends/wasm/wasi/environment.go`, `tools/gomad/internal/backends/wasm/wasi/helper_model.go`, `tools/gomad/internal/backends/wasm/wasmhost/src/transcript.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/transcript*.go, tools/gomad/internal/backends/wasm/wasi/clock.go, tools/gomad/internal/backends/wasm/wasi/environment.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/wasmhost/src/transcript.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md]

### Approach
- Run key: (clock id, precision); record first call ID, starting model `now` and count. Unsupported clock ids (CPU clock) are their own non-folded records.
- Close the open run before any other record, before splicing remote bytes, and at interrupt and terminal flush, in both retained and digest-only modes. Define `local_state.transcript_bytes` (checked at `helper_model.go:79`) as bytes including the closed run, and close the run before sending a callback.
- Implement the run state once per language inside the encoder modules from task 2, not in callers (`clock.go:28-48` Go clock model; Rust `Model::clock_time_get`).
- Frozen clock (zero read step) still records the count.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/clock.go:28-48`
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:320-362`
- `tools/gomad/internal/backends/wasm/wasi/helper_model.go:77-106`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad/internal/backends/wasm helper
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestTranscript|TestHelperModel|TestCooperative' ./internal/backends/wasm/wasi)
```

## Acceptance
- [ ] A run of N reads produces one record in both owners; Go owner/helper parity holds for mixed clock/remote/interrupt sequences (new test covering a run closed by a remote splice, an interrupt and terminal).
- [ ] Changing one read's clock id, precision or the read count changes the digest (unit tests both languages); frozen-clock counts are bound.
- [ ] Replay of a recorded cooperative run reproduces identical evidence; a perturbed run fails with the existing evidence digest mismatch.

## Done summary
Consecutive clock reads with the same id/precision fold into one tag-33 run record in both owners; runs close before any other record, remote splice, interrupt and terminal flush, and the frozen clock still binds the count. The Rust encoder now refuses a read that does not extend its open run. Impl review: SHIP.

Native execution remains deferred (fn-128/fn-149). Performance proof belongs to the .6 task.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fast-batch tip 08f92e58ee (frozen qualification candidate d0372e7a20)
- Tests: frozen qualification /home/agent/wt/fast-qual-20261014T1345: run-qualification.sh + continuations (stages.log ends PASS frozen-qualification module=gomad), make -C tools/gomad/internal/backends/wasm test-conformance, cargo +1.94.1 test/clippy -D warnings (default, diagnostic-profile, conformance)
- PRs: