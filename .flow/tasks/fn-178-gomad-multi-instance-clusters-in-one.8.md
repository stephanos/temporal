---
satisfies: [R3]
---
# fn-178-gomad-multi-instance-clusters-in-one.8 In-process incarnation lifecycle: boundaries, revocation, restart holds and recovery-image restart

## Description
Model the incarnation lifecycle in-process (R3): declared crash/stop boundaries, storage crashes, revocation, restart holds and restart from the Rust-held recovery image. The revocation-after-reply race class fixed in fn156.8 must not reappear.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/broker.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/cluster.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs` (control frames: restart, live commands), `tools/gomad/internal/backends/wasm/wasmhost/tests/cluster.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/broker.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/cluster.rs]

### Approach
- Boundaries, validated as in Go (marker ≤ 4096 bytes, one `CrashAfterOutputs` group, members are incarnation 1 with ≥ 2 members):
  - `CrashOnOutput`/`StopOnOutput` evaluated on fd 1/2 output in `finishCall` order (broker.go:306-352)
  - the `CrashAfterOutputs` barrier (broker.go:~560)
  - storage `crash-before` from the instance's Rust volume (fn-176)
- Revoke (broker.go:488):
  - register a restart-hold participant in the clock
  - remove the instance's clock participant
  - revoke its network endpoint (stop is graceful, crash is not)
  - cancel its pending World request
  - set the instance state to revoked with a typed `crash`/`stop` boundary
  - flush its evidence, including storage evidence and image (fn-176.5/.6 semantics)
  - drop its store
- Pending cross-instance deliveries at a crash follow the Go broker's classification. Capture golden cases (graceful stop vs crash, in-flight vs queued, either end of a link) from the Go broker and assert them.
- Revocation-after-reply analogue (invariant): a revocation decided by the effects of an instance's host call takes effect before that instance performs any further effect. In-process, the broker evaluates boundaries before returning from the host call, so the "reply already sent" window does not exist. Port the cases of `TestTransportRevocationAfterReplyInterruptsTheCrossingCall` (wasi/process_test.go:607-664) as in-process tests, plus a crash landing between a durable write and the next call.
- Restart: a control frame `restart{node}` is accepted only when the incarnation is finished. It creates incarnation+1 with slot `old.slot + len(order)` (capacity-checked at `1<<16`), builds a fresh store from the Rust-held recovery image, re-registers with the network and clock, and releases the hold (broker.go:598). Go supplies only the new incarnation's boot config; the image never leaves Rust unless requested.
- Live unbound commands (`Crash`, `Stop`, `Link`) are polled at scheduler handoffs per fn-178.1. They stay non-replayable, as today.
- Slot, incarnation and marker limits keep their `capacity` versus `invalid` classes.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/broker.go:270-692`
- `tools/gomad/internal/backends/wasm/wasi/process.go:516-560,700-760` (modeledReapedCancellation, revokedAfterReply)
- `tools/gomad/internal/backends/wasm/wasi/process_test.go:259,607-664`, `tools/gomad/internal/backends/wasm/wasi/broker_test.go:87-200`
- `tools/gomad/internal/backends/wasm/cluster/provider.go:482-780` (Start/Restart/Crash/Stop)
- Restarted incarnations carry an `incarnation` field in their instance-tagged frames and in `cluster_result.instances` entries (reserved by the fn-178.1 protocol draft).

## Acceptance
- [ ] Instance frames and `cluster_result.instances` entries of restarted incarnations carry `incarnation`.
- [ ] In-process crash, stop, storage-crash and restart fixtures produce the same per-incarnation evidence, boundaries and recovered history as the Go broker reference (recorded cases until fn-178.9's live comparison).
- [ ] Revocation-after-reply analogue tests pass for crash and stop, with and without an intervening durable write, under 12 busy loops.
- [ ] A trap in one instance during another's pending delivery leaves the other's evidence intact and follows the Go classification for the pending delivery.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
