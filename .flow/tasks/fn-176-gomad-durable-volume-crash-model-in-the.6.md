---
satisfies: [R1, R4]
---
# fn-176-gomad-durable-volume-crash-model-in-the.6 Broker split ownership: helper-owned namespace and volume under the Go broker

## Description
Make Broker runs use the Rust namespace and volume while scheduling, runtime decisions and the shared clock stay with the Go Broker (R1, R4). The committed-workflow crash/reopen gate and the SQLite durable cells run under a Broker (`CrashOnOutput`, `Restart`), and the Broker disables the helper local model (process.go:324), so the Rust volume cannot reach those gates without this split mode. fn-178 later moves the Broker in-process; this mode stays as its Go reference path.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasi/process.go`, `tools/gomad/internal/backends/wasm/wasi/helper_model.go`, `tools/gomad/internal/backends/wasm/wasi/broker_process.go`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`, `tools/gomad/internal/backends/wasm/wasi/process_test.go`, `tools/gomad/internal/backends/wasm/wasi/broker_test.go`
**Touches:** [tools/gomad/internal/backends/wasm/wasi/process.go, tools/gomad/internal/backends/wasm/wasi/helper_model.go, tools/gomad/internal/backends/wasm/wasi/broker_process.go, tools/gomad/internal/backends/wasm/wasi/broker.go, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md, tools/gomad/internal/backends/wasm/wasi/process_test.go, tools/gomad/internal/backends/wasm/wasi/broker_test.go]

### Approach
- New ownership mode (`filesystem-owned`, decision):
  - The helper owns namespace, volume and the WASI transcript.
  - `runtime_*`, `clock_time_get`, `sched_yield` and `poll_oneoff` clock parts stay remote to Go, and the Broker keeps the choice/runtime session and clock.
  - Remote replies carry `local_state` (`now`, `advances`) and canonical transcript bytes exactly as helper-owned runs do (protocol.md). The volume clock therefore follows Broker time.
- fd 1/2 writes stay remote in Broker runs, so `CrashOnOutput`, `StopOnOutput` and `CrashAfterOutputs` (broker.go:306-352, ~560) still observe output in order. Everything else in the namespace is local.
- Revocation-after-reply invariant: the durable effects at a Broker crash/stop must equal the Go-owned path's.
  - Today a revocation detected in `Broker.Delivered` after the reply interrupts the next call (process.go:534 `revokedAfterReply`, 703-742). With local storage, the guest could perform local durable effects before its next remote call.
  - Approach: Go marks a reply `revocable` when Delivered has not yet been evaluated. Before the guest's next local effect (any namespace/volume mutation or read), the helper sends a lightweight `sync` frame and waits for `continue` or the typed `interrupt`.
  - Only remote replies trigger this, so the steady state stays callback-free.
- On a Broker `crash`/`stop` interrupt, the helper flushes storage evidence and the image as fn-176.5 does for `storage_crash`. A pending storage crash and a Broker revocation are ordered by which one the guest reaches first; add a test for each order.
- Call accounting: local calls consume the global call IDs in the helper. Broker `Limits.Calls` and `nextCall` adopt helper state from `local_state` on remote frames.
- Parity: run Broker fixtures with the Go-owned path (`remoteModel` extended to Broker runs) and the split mode, and require identical transcripts, storage evidence and images:
  - `TestBrokerCrashRecoveryFreshGuest`
  - `TestBrokerCrashRetainsInterruptedRuntimePrefix`
  - a CrashOnOutput SQLite fixture
- Extend `TestTransportRevocationAfterReplyInterruptsTheCrossingCall` (process_test.go:607) to the split mode, including a local storage write between reply and next remote call.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/process.go:300-560,600-808`
- `tools/gomad/internal/backends/wasm/wasi/broker_process.go`, `tools/gomad/internal/backends/wasm/wasi/broker.go:239-360,488-640`
- `tools/gomad/internal/backends/wasm/wasi/process_test.go:259,607-664`
- `tools/gomad/internal/backends/wasm/wasi/broker_test.go:87-200`

## Acceptance
- [ ] Broker storage runs answer every non-fd-1/2 namespace and volume op in the helper (Go fs-served counter equals the number of fd 1/2 writes).
- [ ] Go-owned vs split-mode parity passes on the Broker fixtures above, including images and recovered history.
- [ ] Revocation-after-reply tests pass in split mode for crash and stop, with and without an intervening local storage effect, under 12 busy loops (as fn-156.8 measured).
- [ ] protocol.md documents the mode, the `revocable` reply flag and the `sync`/`continue` frames.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
