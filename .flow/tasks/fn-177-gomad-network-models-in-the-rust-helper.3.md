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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
