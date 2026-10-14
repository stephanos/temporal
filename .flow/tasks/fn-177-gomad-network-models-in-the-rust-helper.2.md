---
satisfies: [R2]
---
# fn-177-gomad-network-models-in-the-rust-helper.2 Rust network owner: connections, FIFO queues, topology and history

## Description
Add the helper's Rust network owner module covering domains, listeners, dial/accept, per-direction byte-bounded FIFO queues, read/write/half-close/close/deadlines/poll/cancel, limits, topology faults with held bytes and effective-heal release, and history records. Stall expiry and reservations follow in task 3. The module knows nothing about the helper `Model`.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/network/mod.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/network/connection.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/network/history.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs` (module), `tools/gomad/internal/backends/wasm/wasmhost/tests/network_corpus.rs` (new)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/network/**, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/network_corpus.rs]

### Approach
- Mirror the post-fn-154 Go owner (`internal/model/network/generated.go`, generated from native `gomadio/network_connection.go` and `simulation_network.go`): its operation precedence, errno vocabulary (`adapter.go` error mapping, "stale file handle"), limits (`listeners`, `connections`, `bytes`, `connection_bytes`, `transitions`) and history shapes.
- Multi-domain API: register a domain with run, node, incarnation and address, and revoke it. Links are keyed by node, and request IDs carry a per-slot base (see `wasi/broker_network.go:131-160`), so fn-178 can attach several instances without a refactor.
- Corpus runner test: deserialize scripts with `serde_json` (test only) and compare every step. In this task, require the non-stall subset to pass; mark the stall, dial-stall and reservation scripts as expected-pending, to be enabled in task 3.

### Investigation targets
**Required:**
- `tools/gomad/internal/model/network/generated.go` (connection, queue, topology and history sections)
- `tools/gomad/internal/model/network/adapter.go`
- `tools/gomad/internal/backends/native/toolchain/runtime/overlay/src/internal/gomadio/network_connection.go`
- `tools/gomad/internal/backends/wasm/wasi/broker_network.go:120-230`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked network
```

## Acceptance
- [ ] The non-stall corpus subset replays exactly (step outputs and final Record structure).
- [ ] The owner API supports multiple registered domains and per-slot request bases (unit test with two node pairs).
- [ ] `cargo test` and `clippy` pass; no dependency added.

## Done summary
Rust network owner (`wasmhost/src/network/{mod,connection,history}.rs`, `mod network` in main.rs, dead-code allowed until fn-178) ports the post-fn-154 Go connection owner: multi-domain register/revoke with run/node/incarnation/address, per-slot u64 request IDs, listeners, dial/accept, per-direction byte-bounded FIFO queues, read/write/half-close/close/deadlines/poll/cancel, limits, partition/heal/delay/reset with held bytes and effective-heal release, binary history identity and the final Record. Integrated onto current `gomad` (e0645443a9, post gomad3->gomad rename) as 2792c94070. The corpus runner replays 97 of 114 Go-generated scripts exactly (90 connection/random + 7 stall-classified scripts the reservation bookkeeping already satisfies); the remaining 15 stall and 2 broker scripts are enforced expected-pending (the test fails if one starts passing) for fn-177.3. Two-pair/slot-base unit tests cover multi-domain use. No dependency added. Review fixes: precomputed replay lanes (O(1) lookup like Go), missing link dials take the disabled-link stall path instead of recording `refused`, auto-port exhaustion records port 0, link-key/zero dedup.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2792c9407071b26cde225131be0e0ad7cc69ec77
- Tests: cargo +1.94.1 fmt --check (exit 0), CARGO_TARGET_DIR=/home/agent/wt/fn177-tmp/target2 cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked (exit 0; 33+48+29 passed, 2 pre-existing ignored) at 2792c94070, cargo +1.94.1 clippy --release --locked --all-targets -- -D warnings (exit 0) at 2792c94070, go test -tags test_dep -count=1 -run 'TestNetworkCorpus|TestModel|TestNetworkV2' ./internal/model/network/... ./internal/backends/wasm/wasi (exit 0; TestNetworkCorpusIsCurrent, TestNetworkV2RustOwnerPinIsCurrent pass), flowctl claude impl-review fn-177.2 --base e0645443a9 --spec claude:claude-opus-5-5:high: round1 NEEDS_WORK (4 actionable fixed), round2 SHIP session per receipt .flow/tmp/impl-review-fn1772-integrate-20261014.json
- PRs: