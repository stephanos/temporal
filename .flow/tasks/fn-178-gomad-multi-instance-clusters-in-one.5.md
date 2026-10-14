---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.5 Rust simulation clock: arbiter, progress credits and restart holds

## Description
Port the shared simulation clock (arbiter and progress credits) to Rust (R2), as used by the broker for time advance, deadlock detection and restart holds.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/wasmhost/src/clock.rs` (new), `tools/gomad/internal/backends/wasm/wasmhost/src/main.rs`, `tools/gomad/internal/backends/wasm/wasmhost/tests/clock_conformance.rs` (un-ignore the clock replay)
**Touches:** [tools/gomad/internal/backends/wasm/wasmhost/src/clock.rs, tools/gomad/internal/backends/wasm/wasmhost/src/main.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/clock_conformance.rs]

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
Rust simulation clock in `wasmhost/src/clock.rs` (`mod clock` in main.rs, dead code allowed until fn-178.7): broker (strict, initial+1, Register activates) and process (strict/forward) clocks, register/activate/activateAt, the generation+1/deadline/epoch quiesce validation, advance/retry/external early answers, all 17 progress-credit events of `simulation_progress.go` with Go's exact error text, restart-hold and external participants, removal answering a parked waiter, and `settleLocked` (advance to the earliest deadline, deadlock when none). Single-threaded: `quiesce` returns an answer or `Parked`; answers produced for parked participants queue until `take_answers`; `abandon` mirrors context cancellation. Participants are keyed by handle (Go pointer identity), so a removed handle stays inactive when its name is reused. `tests/clock_conformance.rs` replays all 55 Go clock corpora step for step (errors, values, sorted responses, current time, waiting set), exact on first run; a perturbed corpus is detected.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3dc48a8f59, 95b0f9e130
- Tests: cargo +1.94.1 test --release --locked clock (exit 0; clock_corpus_replays_exactly over 55 corpora), cargo +1.94.1 clippy --release --locked --all-targets -- -D warnings (exit 0), cargo +1.94.1 fmt --check (exit 0), flowctl claude impl-review fn-178.5 --spec claude:claude-opus-5-5:high: SHIP (P3 storage/quoting follow-ups applied)
- PRs: