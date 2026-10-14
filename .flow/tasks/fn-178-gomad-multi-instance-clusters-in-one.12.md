---
satisfies: [R6]
---
# fn-178-gomad-multi-instance-clusters-in-one.12 Rust net_stream ABI, cluster network glue and binary network evidence

## Description
Serve every cluster instance's `net_stream` in the helper from one shared Rust network owner (fn-177), apply the broker's network configuration (addresses, links, strict topology, central fault schedule, live link changes, revocation), and emit each instance's binary network evidence stream on every termination (R6, helper side). The scheduler, clock and World integration of network readiness follow in fn-178.7. Go decoding follows in fn-178.13. Moved here from former fn-177.4 and adapted to the in-process cluster: the broker is always present, so there is no single-instance glue.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/network/stream_abi.rs` (new: buffer codec and errno table), `tools/gomad/internal/backends/wasm/wasmhost/src/network/cluster.rs` (new: broker-shaped glue over the fn-177 owner), `tools/gomad/internal/backends/wasm/wasmhost/src/cluster.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/cluster.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/network/stream_abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/network/cluster.rs, tools/gomad/internal/backends/wasm/wasmhost/src/network/mod.rs, tools/gomad/internal/backends/wasm/wasmhost/src/cluster.rs, tools/gomad/internal/backends/wasm/wasmhost/src/model.rs, tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs, tools/gomad/internal/backends/wasm/wasmhost/src/protocol.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/cluster.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md]

### Approach
- Buffer codec: port `wasi/network_v2.go:129-226`. That covers version 2, ops 0-13, the request ID rule (nonzero, < 2^48), reserved-zero input checks at `[40:48]`, `[52:56]` and `[60:64]`, query returning 2 without a network, payload bounds of 64..65,600 bytes, and the errno table `networkV2Errno` (`:228`, stall timeout → ETIMEDOUT).
- Glue: mirror `brokerNetwork` (`wasi/broker_network.go`).
  - Validate as `newBrokerNetwork` does, before `started`: v2 limits, no per-node fault schedule, IPv4 addresses, link endpoints and duplicates, fault names, order and actions (partition, heal, delay).
  - Build nodes `<id>.server`/`<id>.client` with the default 10.0.x.y and 10.1.x.y addresses, and links in sorted node order under the `StrictTopology` rule.
  - Register both domains per incarnation, set the request base to `slot << 48` (`attach`, `:140-154`), and revoke both domains on stop or crash with the Go broker's `graceful` flag.
  - Apply scheduled faults and live link changes (`ChangeLink`, `:224`) to all four client/server pairs of the two nodes.
  - Expose `deadline()` (owner deadline folded with the next fault) and `advance(now)` for fn-178.7.
- Per-instance World registration of stream requests and the quiescence entries in the operations log (`network_v2.go:180-220`, `:315-345`) are wired by fn-178.7 on the Rust World model. This task leaves the hook and the evidence slot.
- Evidence: a binary per-instance `network` stream, framed like fn-174's binary transcript and tagged with the instance id (fn-178.6 evidence queues). It carries the operations log (request/result and fault records) and the shared owner's Record structure at the instance's termination, as `NetworkV2Evidence` reads it in broker mode (`network_v2.go:370-391`). It is capped by the instance's `transcript_bytes` with capacity class `network-v2-evidence-bytes`, also in digest-only mode, and flushed on exit, trap, fuel, capacity, storage interrupt and modeled crash or stop.
- Single-instance runs keep `runtime_net_stream` remote forwarding (`model.rs:373`, `abi.rs:700`) and the Go owner; `forwards_bounded_v2_stream_import_to_remote_model` (`tests/engine.rs:208`) stays.
- protocol.md: the cluster network configuration shape and the per-instance network evidence stream, with result counts.

### Investigation targets
**Required:**
- `tools/gomad/internal/backends/wasm/wasi/network_v2.go` (whole file)
- `tools/gomad/internal/backends/wasm/wasi/broker_network.go` (whole file)
- `tools/gomad/internal/backends/wasm/wasmhost/src/model.rs:234-275`, `:367-463`
- `tools/gomad/internal/backends/wasm/wasmhost/src/abi.rs:650-744`
- fn-177's owner API in `wasmhost/src/network/`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
```

## Acceptance
- [ ] A cluster instance's `net_stream` is served by the shared Rust owner with no callback to Go (a cluster test where one instance dials its own node's server and exchanges bytes without idling). Malformed buffers keep their errno or capacity classification (tests per reserved-field and bound case).
- [ ] Invalid cluster network configuration (per-node fault schedule, bad address, duplicate or unknown link endpoint, unordered fault or unknown action) is `invalid` before `started` (tests mirroring `newBrokerNetwork`).
- [ ] Faults and live link changes hit all four client/server pairs, and crash or stop revokes both domains (unit tests with two node pairs).
- [ ] Per-instance network evidence is emitted on exit, trap, capacity and storage interrupt, is capped with `network-v2-evidence-bytes`, and does not affect other instances' evidence (tests).
- [ ] protocol.md documents the cluster network configuration and evidence stream.


## Done summary
`wasmhost/src/network/{stream_abi.rs,cluster.rs,cluster/evidence.rs}` plus cluster/engine/abi wiring: the cluster `network` object (Go BrokerConfig network fields) and per-instance `network_v2` are validated before `started` with Go's `newBrokerNetwork`/`validateNetworkModelV2` refusals; one shared Rust owner gets `<id>.server`/`<id>.client` nodes with Go's default addresses, strict-topology links in sorted node order, both domains per incarnation, `slot << 48` request bases, central faults and live link changes on all four client/server pairs (`Shared::change_link`, `deadline()`, `advance(now)` exposed for fn-178.7), and graceful revocation after a finished incarnation's evidence. Cluster `net_stream` is served in the helper (no Go callback) with Go's buffer layout, reserved-field/version/payload checks, errno table and per-instance network World (register on pending, readiness quiescence, cancel). Each instance streams `GMNETEV1` binary evidence at every terminal (operations log with exact Go-JSON accounting for `network-v2-transcript-bytes`, shared Record, expanded network World), refused as `network-v2-evidence-bytes` capacity when over transcript bytes; neighbors' evidence unaffected. Single executions keep remote `runtime_net_stream` forwarding. Checked-in Rust fixture `tests/testdata/cluster_network/` is decoded by fn-178.13 into documents byte-identical to the Go broker path. Open for fn-178.7/.8: broker virtual time (fault application after t=1, attach-and-advance on every call, idle), live link control frames, evidence on storage interrupts and modeled crash/stop.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4f83f4e13d, 86f47634fb
- Tests: make -C tools/gomad/internal/backends/wasm test-helper (exit 0; cluster.rs 10, network::cluster unit 8), cargo clippy -D warnings default and guest-profile,diagnostic-fuel-off (exit 0), cargo fmt --check (exit 0), flowctl claude impl-review fn-178.12 --spec claude:claude-opus-5-5:high: SHIP (5 P3 follow-ups applied)
- PRs: