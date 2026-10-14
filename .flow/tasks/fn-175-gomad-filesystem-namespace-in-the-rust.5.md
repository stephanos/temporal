---
satisfies: [R1]
---
# fn-175-gomad-filesystem-namespace-in-the-rust.5 Route filesystem imports to the Rust namespace and return stdout/stderr from the helper

## Description
Route every filesystem import to the Rust namespace in gated runs, and return stdout/stderr from the helper (R1). This is the switch-over task. It builds on fn-174's typed host functions and binary transcript, so filesystem calls pay neither the JSON nor the pipe cost.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs` (routing, stdout/stderr streams), `tools/gomad/internal/backends/wasm/wasi/helper_model.go` (evidence streams), `tools/gomad/internal/backends/wasm/wasi/process.go` (Result stdout/stderr, output accounting, mirror), `tools/gomad/internal/backends/wasm/wasi/integration_test.go` (parity cases), `tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/integration_test.go, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs]

### Approach
- Assumed fn-174 interface: imports are bound as typed host functions per WASI op. The local path records each call/reply into fn-174's binary transcript recorder, which digests it and charges `transcript_bytes`, keeps the global call-ID sequence and reports `local_state`. Go has a strict decoder for that transcript.
  - If a different shape lands, adapt to it; do not build a second recorder.
- Go keeps its namespace only as validator and identity source. Any fs callback that reaches Go in a gated run is a protocol violation (`invalid`).
- In gated runs, each `fd_*`/`path_*` typed import calls `namespace.rs` directly. For `poll_oneoff`, the local poll handles any mix of clock and fd subscriptions.
- Reproduce Go's per-call admission in the same order before mutating: call-limit, the frame and reply reservations (`1024 + length*4/3 + 4`, plus `4*len` for poll) and transcript-bytes capacity. Capacity errors must keep their dimensions (`buffer-bytes`, `reply-frame`, `callback-frame`, `transcript-bytes`).
- Unsupported replies terminate as `unsupported` with Go's message, and the transcript entry is retained, as the Go path does today.
- stdout/stderr: add `stdout` and `stderr` evidence streams beside `internal/model/choice`, `diagnostic`, `transcript` and `internal/model/network` (model.rs:247 flush; Go `receiveHelperEvidence` in helper_model.go). Go fills `Result.Stdout`/`Stderr` from them instead of `environment.Output()` (process.go:369).
- Live mirror (decision): flush the stdout stream when 64 KiB accumulate, before any remote callback, and at termination, and feed `Request.StdoutMirror` as chunks arrive. Mirror timeliness is best-effort.
- Output accounting: after fn-175.3, define the terminal `output_bytes` as the namespace's offered-output count in gated runs, and update the terminal check (process.go:791) to compare against the helper-reported value for gated runs.
- Prove R1 without diagnostic builds, so the gate holds on release helpers: add a Go-side per-op counter of filesystem ops served by the Go namespace to `Result` (test-visible map op→count), required to be empty in gated runs. Per-op counts keep one leaked op visible.
- Classify local namespace failures through a typed boundary error (kind + dimension) mapped onto the existing `invalid`/`capacity`/`unsupported` terminations, rather than adding more `contains("capacity:")` matching. Messages keep Go's `WASI <kind>: <dimension>` text.
- Abnormal ends: on interrupt, divergence or capacity the helper flushes stdout/stderr like other streams. On watchdog kill or helper crash, `Result.Stdout` holds the chunks received so far (best effort), which is documented in protocol.md. A run never flips ownership mid-run.
- End-to-end parity, extending the `TestHelperModelPreservesGoOwnerEvidence` pattern (integration_test.go:1313): run filesystem-heavy fixtures (captured inputs, SQLite guest without storage) with `remoteModel=true` (Go namespace reference) and the default (Rust namespace), and require `sameEvidence` (integration_test.go:268) plus equal `Calls`, `TimeAdvances` and `EnvironmentIdentity`.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:247-470` (flush, transcript, remote adoption, handle_input local predicate)
- `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs:196-620` (invoke decoding and reply writes, fd 1/2 output reservation)
- `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs:147-270` (callback_input)
- `tools/gomad/internal/backends/wasm/wasi/integration_test.go:268,1313,1405` (sameEvidence, Go-owner parity, small parity module)
- fn-174.1 (typed host functions), fn-174.2 (shared binary WASI transcript records in Go and the helper) and fn-174.3 (JSON-free typed local model path): the APIs this task plugs into

### Key context
- This task's spec depends on fn-174 (set at plan time). Do not start before fn-174's typed-import and binary-transcript tasks are done.
## Acceptance
- [ ] Gated runs answer every `fd_*`/`path_*` op and every `poll_oneoff` (clock, fd or mixed) in the helper; the Go fs-served counter is 0 on the parity fixtures.
- [ ] Go-owner vs helper-owner parity passes on the filesystem-heavy fixtures, with identical transcript digest, stdout, stderr, choice bytes, Calls and TimeAdvances.
- [ ] Capacity and unsupported outcomes keep their classifications and dimensions (`wasmhost/tests/engine.rs` cases for write-buffer, filesystem-bytes, descriptors, files, transcript-bytes, unsupported fdflags).
- [ ] `make -C tools/gomad/internal/backends/wasm test-helper` and `test-model` pass, and the focused wasi helper tests pass: `go test -tags test_dep -count=1 -run 'TestHelper|TestNamespaceConformance' ./internal/backends/wasm/wasi` from `tools/gomad`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
