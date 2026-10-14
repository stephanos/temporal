---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.5 Rust simulation clock: arbiter, progress credits and restart holds

## Description
Port the shared simulation clock (arbiter and progress credits) to Rust (R2), as used by the broker for time advance, deadlock detection and restart holds.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/clock.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/world_clock_conformance.rs` (un-ignore the clock replay)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/clock.rs, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/world_clock_conformance.rs]

### Approach
- Port `internal/model/clock/simulation_time.go` (`simulationTimeArbiter`: current, forward, participants, quiesce) and `simulation_progress.go` (the progress-credit state machine; `settleLocked` at :393).
  - Generation must be exactly the previous value + 1.
  - Time becomes `max(current, min(deadlines))` when every participant is active, none is handling or delivered, and every non-external participant is waiting.
  - If every deadline is `MaxInt64`, the result is Deadlock.
- Include restart-hold participants (`broker.go:488` registers `restart/node/inc`) and the external participant semantics.
- Single-threaded API; the broker scheduler (fn-178.7) is the only caller. No locks are needed, because scheduling is serialized (fn-178.1).

### Investigation targets
**Required:**
- `tools/gomad/internal/model/clock/simulation_time.go`, `tools/gomad/internal/model/clock/simulation_progress.go`, `tools/gomad/internal/model/clock/simulation_broker.go`
- `tools/gomad/internal/backends/wasm/wasi/broker_process.go:85-100` (idle → prepareIdle → Quiesce)

## Acceptance
- [ ] Clock conformance corpus (scripted from `simulation_time_test.go`, generated, recorded) passes with identical responses and state.
- [ ] `cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked clock` passes.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
