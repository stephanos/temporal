---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.3 Rust World model: ordering, seeded equivalence ranking, capacity and digests

## Description
Port the World model to Rust (R2): registration, readiness, cancellation, quiescence batching, deterministic ordering with seeded equivalence ranking, capacity limits, transitions and digests.

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/world/mod.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/world/order.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/world/digest.rs` (new), `tools/gomad3/backends/wasm/wasmhost/src/main.rs`, `tools/gomad3/backends/wasm/wasmhost/src/conformance.rs`
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/world/**, tools/gomad3/backends/wasm/wasmhost/src/main.rs, tools/gomad3/backends/wasm/wasmhost/src/conformance.rs]

### Approach
- Port these from Go:
  - `world.go`: `Register`, `Ready`, `Cancel`, `Quiesce` (world.go:237-300; batch of every queued event at the minimum `At`, sorted, removed, `now = At`, transition appended)
  - `queue.go`: `lessEvent` order (At, Priority, Resource adapter/kind/key, Request kind, readiness kind, EquivalenceClass, then `choiceRank` only for non-empty classes, then RequestID)
  - `choice.go`: `choiceRank`, HMAC-SHA256 keyed from the seed with domain `gomad3/world/seed/v1`; implement HMAC over the pinned `sha2`, with no new crate (fn-178.1)
  - `types.go`: `Limits`, `Quiescence`, `InitialTime`
  - `errors.go`: error classes
  - `snapshot.go`: `stateDigest`, the transcript digest and the `appendX` framing
- Capacity checks run before mutation and are atomic, as in `capacity_test.go`.
- Use ordered containers throughout.
- Expose a typed API that the broker scheduler (fn-178.7) calls, plus a snapshot builder that fn-178.4's encoder consumes.

### Investigation targets
**Required:**
- `tools/gomad3/world/world.go`, `tools/gomad3/world/queue.go`, `tools/gomad3/world/choice.go`, `tools/gomad3/world/types.go`, `tools/gomad3/world/errors.go`
- `tools/gomad3/world/snapshot.go:24-346`
- `tools/gomad3/world/world_test.go:50-120` (ordering tests), `tools/gomad3/world/capacity_test.go`

## Acceptance
- [ ] World conformance corpus (scripted, generated, recorded) passes with identical batches, quiescence, errors and digests.
- [ ] HMAC unit vectors match Go's `choiceRank` for a set of seeds and classes (`cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked world`).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
