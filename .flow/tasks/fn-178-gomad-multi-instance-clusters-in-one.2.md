---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.2 World and clock conformance harness with Go drift guard

## Description
Pin the Go `internal/model/world` owner and the shared simulation clock with checked-in conformance corpora that the Rust ports (fn-178.3, fn-178.5) replay with plain `cargo test` (R2). This is the early first-class gate, in the style of fn-177.1's network corpus (design record item 9: fn-175.1's subprocess conformance driver is not on `gomad`). It also guards Go drift, because Go stays authoritative for native and the runner.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/cluster/world_conformance_test.go` (new), `tools/gomad/internal/backends/wasm/cluster/clock_conformance_test.go` (new), `tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/world_conformance/` and `.../clock_conformance/` (corpora and `PINNED_GO_*_SHA256` pins), `tools/gomad/internal/backends/wasm/wasmhost/tests/world_conformance.rs` and `tests/clock_conformance.rs` (new Rust replays, ignored until the ports land), `tools/gomad/internal/model/world/compressed_decode.go` (exported full-snapshot decode and expanded-wire wrap used by the recorded capture and fn-178.4)
**Touches:** [tools/gomad/internal/backends/wasm/cluster/world_conformance_test.go, tools/gomad/internal/backends/wasm/cluster/clock_conformance_test.go, tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/world_conformance/**, tools/gomad/internal/backends/wasm/wasmhost/tests/testdata/clock_conformance/**, tools/gomad/internal/backends/wasm/wasmhost/tests/world_conformance.rs, tools/gomad/internal/backends/wasm/wasmhost/tests/clock_conformance.rs, tools/gomad/internal/model/world/compressed_decode.go]

### Approach
- World corpus: `config{seed, limits}`, then `register`, `ready`, `cancel` and `quiesce` steps with the Go result per step:
  - the result or error class and exact message (capacity is checked atomically before mutation)
  - the delivery batch (order per `lessEvent`, queue.go:38-65, including `choiceRank` for equivalence classes) and `Quiescence` kind
  - the cardinality summary and `TranscriptDigest` every step; next identities and `StateDigest` at checkpoints and the last step
  - Go's final expanded `GMWSNAP1` wire and `GMWORLD2` envelope digest when the config fits the compressed profile (fn-178.4 input)
- Clock corpus, driven through the exported `clock.BrokerClock` API (broker and process clocks): participants register, activate and leave; `Quiesce{Generation, Current, Deadline, Arrivals}`; every progress-credit event; restart-hold participants and external participants. After each step: the step error, the responses resolved during it (Advance, Retry, Deadlock, External, or an error) and the observable arbiter state (current time and the waiting set).
- Corpora:
  - scripted cases from `internal/model/world/world_test.go`, `capacity_test.go` and `replay_test.go`, and from `internal/model/clock/simulation_time_test.go`
  - seeded generated sequences, including equal-time equivalence classes, cancels racing ready, and crash/restart-hold interleavings
  - recorded: a World corpus captured from the retained broker World of `TestTemporalIndependentClustersPartitionHealWholeClusterCrash` (its first 4096 transitions), re-derived from its checked-in inputs on every run; broker-shaped clock flows (the Go broker keeps no clock log)
- Drift guard: every corpus header and a pin file carry `world.ImplementationSHA256()` or `clock.BrokerImplementationSHA256()`. A Go test fails when either hash or any Go result changes until the corpus is regenerated, which forces the Rust replay to be rerun.

### Key context
- fn-177.1 pins the Go network owner with a checked-in JSON corpus and an implementation-hash drift pin replayed by plain `cargo test`; World and clock follow it so the three Rust duplicates are guarded alike.

### Investigation targets
**Required:**
- `tools/gomad/internal/model/world/world.go`, `queue.go`, `choice.go`, `snapshot.go:24-335`
- `tools/gomad/internal/model/clock/simulation_time.go`, `simulation_progress.go`, `simulation_broker.go`, `simulation_time_test.go`
- `tools/gomad/internal/model/network/corpus_test.go` (currency-test pattern)
## Acceptance
- [ ] `go test -tags test_dep -run 'TestWorldConformance|TestClockConformance' ./internal/backends/wasm/cluster` (from `tools/gomad`) regenerates scripted, generated and recorded corpora against Go, requires the checked-in bytes, and reports the first divergent step with its seed or origin.
- [ ] Drift tests fail on a Go World or clock implementation-hash change until the corpus is refreshed.
- [ ] Until fn-178.3 and fn-178.5 land, the Rust corpus replay tests are ignored with a reason; a non-ignored Rust test checks corpus headers against the pins.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
