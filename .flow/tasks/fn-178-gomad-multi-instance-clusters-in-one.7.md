---
satisfies: [R1, R6]
---
# fn-178-gomad-multi-instance-clusters-in-one.7 In-process broker scheduler: continuations, clock, World ordering and cross-instance network

## Description
Add the in-process broker (R1, R6): a serialized scheduler of instance continuations, the Rust clock for idle and time advance, World ordering of externally visible operations, and cross-instance networking on the Rust network model with fn-154's shared connection semantics. fn-178.12 provides the shared owner's `net_stream` serving, broker glue and evidence stream; this task folds network readiness, deadlines and faults into scheduling (moved here from former fn-177.4's idle integration).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/broker.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/cluster.rs`, `tools/gomad3/backends/wasm/wasmhost/src/model.rs` (broker-mode routing of `runtime_idle`, `runtime_net_stream`), `tools/gomad3/backends/wasm/wasmhost/src/network/cluster.rs` (fn-178.12 glue: readiness and World hooks), `tools/gomad3/backends/wasm/wasmhost/tests/cluster.rs`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/broker.rs, tools/gomad3/backends/wasm/wasmhost/src/cluster.rs, tools/gomad3/backends/wasm/wasmhost/src/model.rs, tools/gomad3/backends/wasm/wasmhost/src/network*, tools/gomad3/backends/wasm/wasmhost/tests/cluster.rs]

### Approach
- Mirror the Go broker (`wasi/broker.go`, `broker_process.go`):
  - one `active` instance
  - a `runnable` instance is chosen in sorted id order (broker.go:136)
  - when every instance is idle with a resume or error, advance the network to the current time, mark all runnable and reschedule
- World-visible operations are only `runtime_idle` and `runtime_net_stream` (broker.go:269). Each registers a World request (`Resource{wasm-broker, incarnation, "node/inc"}`, SHA-256 payload), becomes ready at the clock's current time, and must quiesce to exactly one delivery. Otherwise the outcome is `invalid` ("broker delivery order"), as at broker.go:239-252.
- Idle: build `{Generation, Current, Deadline, Arrivals}`, ask the Rust clock (fn-178.5) for a quiesce, and resume with the returned time (broker_process.go:85-100).
- Network: use fn-177's multi-domain Rust network owner through fn-178.12's glue.
  - Fold the network deadline (owner deadline and next scheduled fault) and the owner's ready set into the idle quiesce, and advance the network to the current time before rescheduling, as the Go broker does (broker.go:215-216, :408-412, :485-495).
  - Apply `ChangeLink` and scheduled faults at the virtual times they name, interleaved with scheduler and clock decisions exactly as in the Go broker.
  - Register each instance's stream requests in its World and append the quiescence entries to its network evidence, as `networkV2State` does (network_v2.go:180-220, :315-345), on the Rust World model (fn-178.3).
  - If fn-177's owner lacks an operation the broker needs, extend it behind the same owner rather than adding a second transport.
- Per-instance runtime decisions and choice tapes stay with each instance's local model; the broker only orders continuations.
- World evidence: at Finish, build the snapshot (fn-178.3) and stream the expanded wire (fn-178.4) in 64 KiB chunks.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/broker.go:49-360`
- `tools/gomad3/backends/wasm/wasi/broker_process.go`
- `tools/gomad3/backends/wasm/wasi/broker_network.go` (network adapter shape to replace)
- fn-177.2 and fn-177.3 (Rust network owner in `wasmhost/src/network/`) and fn-178.12 (`net_stream` serving, broker glue and evidence), the API this task schedules
- `.flow/specs/fn-154-gomad-simpler-virtual-network-with.md` (R9 shared connection owner)
## Acceptance
- [ ] Two- and three-instance broker fixtures (`TestBrokerCrashRecoveryFreshGuest`-style guests without crashes, and a network ping-pong across instances with partition and heal) run in one helper with no runtime-decision or cross-instance-message callbacks to Go.
- [ ] The global broker event order and World evidence equal the Go broker's for the same fixtures and seeds; the full Go-reference comparison lands in fn-178.9, and here it is checked with recorded Go orders.
- [ ] Equal-time ordering and heal-versus-expiry cases from fn-154 behave identically (recorded Go cases).
- [ ] Network deadlines, scheduled faults and stall expiry inside a partition interleave with idle and clock decisions as in the Go broker's recorded orders, with no network callbacks to Go.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
