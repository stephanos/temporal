---
satisfies: [R2]
---
# fn-177-gomad-network-models-in-the-rust-helper.2 Rust network owner: connections, FIFO queues, topology and history

## Description
Add the helper's Rust network owner module covering domains, listeners, dial/accept, per-direction byte-bounded FIFO queues, read/write/half-close/close/deadlines/poll/cancel, limits, topology faults with held bytes and effective-heal release, and history records. Stall expiry and reservations follow in task 3. The module knows nothing about the helper `Model`.

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/network/mod.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/network/connection.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/network/history.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/main.rs` (module), `tools/gomad3/backends/wasm/wasmhost/tests/network_corpus.rs` (new)
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/network/**, tools/gomad3/backends/wasm/wasmhost/src/main.rs, tools/gomad3/backends/wasm/wasmhost/tests/network_corpus.rs]

### Approach
- Mirror the post-fn-154 Go owner (`simulation/network/generated.go`, generated from native `gomadio/network_connection.go` and `simulation_network.go`): its operation precedence, errno vocabulary (`adapter.go` error mapping, "stale file handle"), limits (`listeners`, `connections`, `bytes`, `connection_bytes`, `transitions`) and history shapes.
- Multi-domain API: register a domain with run, node, incarnation and address, and revoke it. Links are keyed by node, and request IDs carry a per-slot base (see `wasi/broker_network.go:131-160`), so fn-178 can attach several instances without a refactor.
- Corpus runner test: deserialize scripts with `serde_json` (test only) and compare every step. In this task, require the non-stall subset to pass; mark the stall, dial-stall and reservation scripts as expected-pending, to be enabled in task 3.

### Investigation targets
**Required:**
- `tools/gomad3/simulation/network/generated.go` (connection, queue, topology and history sections)
- `tools/gomad3/simulation/network/adapter.go`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go`
- `tools/gomad3/backends/wasm/wasi/broker_network.go:120-230`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked network
```

## Acceptance
- [ ] The non-stall corpus subset replays exactly (step outputs and final Record structure).
- [ ] The owner API supports multiple registered domains and per-slot request bases (unit test with two node pairs).
- [ ] `cargo test` and `clippy` pass; no dependency added.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
