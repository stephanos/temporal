---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.3 Rust World model: ordering, seeded equivalence ranking, capacity and digests

## Description
Port the World model to Rust (R2): registration, readiness, cancellation, quiescence batching, deterministic ordering with seeded equivalence ranking, capacity limits, transitions and digests.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/world/mod.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/world/order.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/world/digest.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/world_conformance.rs` (un-ignore the World replay)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/world/**, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/world_conformance.rs]

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
- `tools/gomad/internal/model/world/world.go`, `tools/gomad/internal/model/world/queue.go`, `tools/gomad/internal/model/world/choice.go`, `tools/gomad/internal/model/world/types.go`, `tools/gomad/internal/model/world/errors.go`
- `tools/gomad/internal/model/world/snapshot.go:24-346`
- `tools/gomad/internal/model/world/world_test.go:50-120` (ordering tests), `tools/gomad/internal/model/world/capacity_test.go`

## Acceptance
- [ ] World conformance corpus (scripted, generated, recorded) passes with identical batches, quiescence, errors and digests.
- [ ] HMAC unit vectors match Go's `choiceRank` for a set of seeds and classes (`cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked world`).


## Done summary
Rust World model in `wasmhost/src/world/{mod,order,digest}.rs` (`mod world` in main.rs, dead code allowed until fn-178.7): register/ready/cancel/quiesce with atomic pre-mutation capacity checks and Go's exact error text, BTreeSet queue ordered exactly like Go `lessEvent` (empty classes carry a zero rank), `choiceRank` as HMAC-SHA256 over the pinned sha2 (no new crate), transcript digests and the streamed state digest. Exposes accessors (cardinality, next ids, requests/events/transitions, state digest) for the fn-178.4 writer and fn-178.7 scheduler. `tests/world_conformance.rs` replays all 62 Go World corpora step for step (results, errors, deliveries, cardinality, digests) - exact on first run. Unit tests: RFC 4231 HMAC vectors, Go `choiceRank` vectors, ID/sequence exhaustion without mutation. Replay plans and recordings are not ported (the broker uses neither).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f2ffb223dd, a69f36796c
- Tests: cargo +1.94.1 test --release --locked world (exit 0; world_corpus_replays_exactly over 62 corpora), cargo +1.94.1 test --release --locked (full helper suite, exit 0), cargo +1.94.1 clippy --release --locked --all-targets -- -D warnings (exit 0), cargo +1.94.1 fmt --check (exit 0), flowctl claude impl-review fn-178.3 --spec claude:claude-opus-5-5:high: SHIP (4 P3 follow-ups applied in a69f36796c)
- PRs: