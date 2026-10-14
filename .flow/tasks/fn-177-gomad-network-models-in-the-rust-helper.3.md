---
satisfies: [R2]
---
# fn-177-gomad-network-models-in-the-rust-helper.3 Rust network owner: stall expiry, dial stall and history reservations

## Description
Complete the Rust owner with continuous stall intervals, strict expiry (`start + limit + 1ns`), timeout reset of the pair, pending-dial stall, terminal history reservations, epoch/incarnation invalidation and advance/next-deadline ordering, so the full corpus replays exactly (R2).

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/network/stall.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/network/mod.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/network/connection.rs`, `tools/gomad/internal/backends/wasm/wasmhost/src/network/history.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/network_corpus.rs`
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/network/**, tools/gomad/internal/backends/wasm/wasmhost/tests/network_corpus.rs]

### Approach
- Follow the fn-154 spec's Ordering section and the Go owner's stall and reservation code (`internal/model/network/generated.go`: held/reservation logic around `:439-471`, timeout-reset transition around `:1461`; line numbers shift once fn-154 lands, so locate them by name).
- Same-time actions follow serialized model order. A heal committed before due expiry invalidates that epoch; an expiry committed first resets permanently. At exactly `start + limit` the expiry is not yet due.
- Check reservations atomically: a held write admits its record and any new reservation before bytes enter; a partition admits its topology record and all new pair reservations before any mutation. Insufficient capacity changes nothing.
- Enable every expected-pending corpus script from task 2.

### Investigation targets
**Required:**
- `.flow/specs/fn-154-gomad-simpler-virtual-network-with.md` (Ordering, replay and identities)
- `tools/gomad/internal/model/network/generated.go`
- `tools/gomad/internal/model/network/stall_test.go`

### Quick commands
```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked network
```

## Acceptance
- [ ] The full corpus replays exactly, including both equal-time heal/expiry orders, simultaneous pair expiry, reservation exhaustion and pending-dial stall.
- [ ] Overflow in stall arithmetic rejects before mutation (corpus or unit test).
- [ ] No expected-pending scripts remain.

## Done summary
Completed the Rust owner (`wasmhost/src/network/stall.rs` plus mod/connection/history/tests changes, commit d6d30e2a98 on `fn177-integrate`): continuous per-direction stall intervals with strict expiry at `start + limit + 1ns` (not due at exactly `start + limit`), pair timeout reset, pending-dial stall over missing or disabled links with caller-deadline settlement, atomic history reservations for held writes, partitions and dials (insufficient capacity changes nothing), stall-arithmetic overflow rejected before mutation, epoch/incarnation invalidation, dial-owner retirement on every completion path, and advance/next-deadline ordering. All 114 Go-generated corpus scripts replay exactly with no carve-outs (both equal-time heal/expiry orders, simultaneous pair expiry, reservation exhaustion, pending-dial stall, both broker-strict-topology scripts); no expected-pending scripts remain. No Go source, generation input or default WASM runtime behaviour changed (the module is dead code in the release helper until fn-178). Open P3 tidy-ups from review (non-blocking): unused `Dial.epoch`, two inline copies of the stall-deadline arithmetic, repeated dial-retire check, manual reservation release in reset/revoke/close.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d6d30e2a98
- Tests: cargo +1.94.1 fmt --check (exit 0) at d6d30e2a98, CARGO_TARGET_DIR=/home/agent/wt/fn177-tmp/target cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked (exit 0; 40+48+48 passed incl. network_full_corpus_requires_all_114_scripts_every_step_and_final_record, 2 pre-existing ignored), cargo +1.94.1 clippy --release --locked --all-targets -- -D warnings (exit 0), go test -tags test_dep -count=1 -run 'TestNetworkCorpus|TestModel|TestNetworkV2' ./internal/model/network/... ./internal/backends/wasm/wasi (exit 0), go test -tags test_dep -count=1 ./internal/archcheck ./internal/model/network/... (exit 0), make -C tools/gomad wasm-helper && make -C tools/gomad test-wasm GOMAD3_STOCK_GO=<go1.27.1> (exit 0 at tip; base e0645443a9 also exit 0), flowctl claude impl-review fn-177.3 --base 2792c94070 --spec claude:claude-opus-5-5:high: SHIP (3 P3 non-blocking), receipt .flow/tmp/impl-review-fn1773-integrate-20261014.json
- PRs: