---
satisfies: [R1, R3, R5]
---
# fn-178-gomad-multi-instance-clusters-in-one.6 Multi-store cluster host: shared engine, isolated per-instance stores, limits and results

## Description
Turn the single-shot helper into a cluster host (R1, R3, R5). The helper compiles once and instantiates N stores with isolated per-instance state, runs them under the fn-178.1 threading model, and reports per-instance results. This task has no broker semantics yet: instances do not interact, and each uses its own local model (fn-174/fn-175/fn-176 paths). Network endpoints join in fn-178.12.

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/src/engine.rs`, `tools/gomad3/backends/wasm/wasmhost/src/cluster.rs` (new: instance table, runner), `tools/gomad3/backends/wasm/wasmhost/src/protocol.rs` (cluster execute and per-instance result frames), `tools/gomad3/backends/wasm/wasmhost/src/metrics.rs` (cluster-level process metrics), `tools/gomad3/backends/wasm/wasmhost/tests/cluster.rs` (new)
**Touches:** [tools/gomad3/backends/wasm/wasmhost/src/engine.rs, tools/gomad3/backends/wasm/wasmhost/src/cluster.rs, tools/gomad3/backends/wasm/wasmhost/src/protocol.rs, tools/gomad3/backends/wasm/wasmhost/src/metrics.rs, tools/gomad3/backends/wasm/wasmhost/tests/cluster.rs]

### Approach
- Cluster execute frame per the fn-178.1 draft: module identity once, then an instance list with per-instance boot config (memory, fuel, namespace image, storage config or recovery image, network endpoint config, local model config). Exactly one `started` and one `authorize` cover the whole cluster. Invalid cluster config (duplicate ids, unknown references, limits) is `invalid` before any instantiation.
- Shared `Engine`, `Module` and one `InstancePre` from `Linker::instantiate_pre`. Per instance:
  - `Store<State>` with its own `Limits` (`ResourceLimiter`: `memory_bytes` and the per-instance peak; engine.rs:20-58)
  - fuel set per store
  - its own namespace, volume and model
  - its own evidence queues, tagged with the instance id in evidence frames
- Isolation:
  - A trap, fuel exhaustion or capacity failure ends only that instance. Its store is dropped and its evidence is flushed and terminated, with a per-instance result frame (`termination`, `peak_memory_bytes`, `fuel_remaining`, `output_bytes`).
  - A panic or engine error follows the fn-178.1 taxonomy (instance-level or whole-run `infrastructure`).
- Watchdog per fn-178.1: an epoch backstop, and the Go timeout bounds the run.
- Metrics: process CPU and peak RSS are reported once per cluster; per-instance peak linear memory comes from each limiter (R5).
- Tests (`wasmhost/tests/cluster.rs`, spawning `CARGO_BIN_EXE_gomad3-wasmhost` like `tests/engine.rs`):
  - three WAT instances: one exits, one traps, one exceeds memory; the survivors' evidence is unaffected
  - an invalid cluster config is refused before `started`
  - per-instance peak memory is reported

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasmhost/src/engine.rs:20-120,273-620` (Limits, State, execute, prepare, failure classification)
- `tools/gomad3/backends/wasm/wasmhost/src/protocol.rs:248-451`
- `tools/gomad3/backends/wasm/wasmhost/tests/engine.rs` (protocol-driving test harness)
- fn-178.1's draft protocol section in `wasmhost/protocol.md`

### Key context
- Never reuse a store after a trap or fuel exhaustion; build a fresh store for a restart (fn-178.8).
- Wasmtime keeps instances alive until their store is dropped, so use one store per incarnation.

## Acceptance
- [ ] A cluster of N independent guest instances runs in one helper process with one compilation; each instance's evidence equals its single-instance run (a Go test comparing the cluster run with N single runs of the same fixtures).
- [ ] Trap, fuel and memory-limit failures are reported for the failing instance only; the others' evidence is byte-identical to the failure-free run.
- [ ] Per-instance `peak_memory_bytes` is reported and enforced against each instance's `memory_bytes` (instance-level `capacity`).
- [ ] `make -C tools/gomad3/backends/wasm test-helper` passes, including `tests/cluster.rs`.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
