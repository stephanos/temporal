---
satisfies: [R2]
---
# fn-178-gomad-multi-instance-clusters-in-one.2 World and clock conformance harness with Go drift guard

## Description
Extend the fn-175 conformance harness with `internal/model/world` and `clock` models, using the Go `world.Model` and the simulation clock arbiter as references (R2). This is the early first-class gate: the Rust ports in fn-178.3 and fn-178.5 land against it. It also guards Go drift, because Go stays authoritative for native and the runner.

**Size:** M
**Files:** `tools/gomad/internal/backends/wasm/cluster/world_conformance_test.go` (new), `tools/gomad/internal/backends/wasm/cluster/clock_conformance_test.go` (new), `tools/gomad/internal/backends/wasm/cluster/testdata/world-conformance/`, `tools/gomad/internal/backends/wasm/cluster/testdata/clock-conformance/`, `tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs`, `tools/gomad/internal/backends/wasm/wasmhost/protocol.md` (Conformance driver section)
**Touches:** [tools/gomad/internal/backends/wasm/cluster/world_conformance_test.go, tools/gomad/internal/backends/wasm/cluster/clock_conformance_test.go, tools/gomad/internal/backends/wasm/cluster/testdata/**, tools/gomad/internal/backends/wasm/wasmhost/src/conformance.rs, tools/gomad/internal/backends/wasm/wasmhost/protocol.md]

### Approach
- World records: `config{seed, limits}`, then `register`, `ready`, `cancel` and `quiesce` steps, plus `snapshot`. After every step, compare:
  - the result or error class (capacity is checked atomically before mutation)
  - the delivery batch (order per `lessEvent`, queue.go:38-65, including `choiceRank` for equivalence classes)
  - the `Quiescence` kind
  - `TranscriptDigest` and `StateDigest`
  - at the end, the expanded snapshot wire bytes, once fn-178.4 exists
- Clock records: participants join and leave, `SimulationTimeRequest{Generation, Current, Deadline, Arrivals}`, progress-credit events (accepted, delivered, forwarded, runnable, removed), external participants and restart holds. After each step, compare responses (Advance, Retry, Deadlock, External) and the arbiter state.
- Corpora:
  - scripted cases from `internal/model/world/world_test.go`, `internal/model/world/capacity_test.go` and `internal/model/world/replay_test.go`, and from `internal/model/clock/simulation_time_test.go` (858 lines of cases)
  - seeded generated sequences, including equal-time equivalence classes, cancels racing ready, and crash/restart-hold interleavings
  - recorded broker event logs captured from `TestTemporalIndependentClustersPartitionHealWholeClusterCrash` and the broker fixtures
- Drift guard: the corpus header pins `world.ImplementationSHA256()` and the clock implementation hash (`internal/model/clock` `BrokerImplementationSHA256`). A Go test fails when either hash changes without a corpus refresh, which forces the Rust differential to be rerun. Wire `test-conformance` (fn-175.1) to include these models.

### Key context
- fn-177.1 pins the Go network owner with a checked-in JSON corpus and an implementation-hash drift pin replayed by plain `cargo test`. Keep World/clock corpora and pins in the same style where it fits, so the three Rust duplicates are guarded alike.

### Investigation targets
**Required:**
- `tools/gomad/internal/model/world/world.go`, `tools/gomad/internal/model/world/queue.go`, `tools/gomad/internal/model/world/choice.go`, `tools/gomad/internal/model/world/snapshot.go:24-335`
- `tools/gomad/internal/model/clock/simulation_time.go`, `tools/gomad/internal/model/clock/simulation_progress.go:1-440`
- `tools/gomad/internal/model/clock/simulation_time_test.go`
- fn-175.1's `conformance.rs` and the Go differential test
## Acceptance
- [ ] `go test -tags test_dep -run 'TestWorldConformance|TestClockConformance' ./internal/backends/wasm/cluster` (from `tools/gomad`) runs scripted, generated and recorded corpora against Go and the conformance helper, and reports the first divergence with seed and step.
- [ ] Drift tests fail on a Go World or clock implementation-hash change until the corpus is refreshed.
- [ ] Until fn-178.3 and fn-178.5 land, the Rust side answers `unsupported` and the Rust corpus tests are ignored with a reason.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
