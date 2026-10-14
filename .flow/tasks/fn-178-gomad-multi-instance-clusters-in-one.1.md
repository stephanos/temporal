---
satisfies: [R5]
---
# fn-178-gomad-multi-instance-clusters-in-one.1 Design record and handoff spike: control protocol, threading, terminal taxonomy, watchdog, memory budget

## Description
A design-decision and spike task that settles the open design points before any cluster code exists. It resolves the parked memory-budget unknown (R5) and fixes the control protocol, terminal taxonomy, threading model, watchdog and evidence policy that fn-178.6 and later tasks build on (R1, R3). The spec's Decision Context recommends a design review before work starts; this task is that review's input and output.

Performance measurements follow the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

**Size:** M
**Files:** `tools/gomad3/backends/wasm/wasmhost/protocol.md` (draft "Cluster execution" section, marked draft), `tools/gomad3/backends/wasm/wasmhost/examples/store_handoff.rs` (throwaway spike, deleted before done or kept as an ignored bench), the fn-178 spec (Decision Context, via `flowctl spec set-plan`), `.flow/artifacts/` (spike numbers)
**Touches:** [tools/gomad3/backends/wasm/wasmhost/protocol.md, tools/gomad3/backends/wasm/wasmhost/examples/**, .flow/specs/fn-178-gomad-multi-instance-clusters-in-one.md]

### Approach
Decide and record each item with its rationale. Each item below lists the planner's recommended default; deviate only with measured or code-cited reasons.

1. **Threading model.** Spike two stores of a small cooperative guest that hand off at every host call, using
   - (a) one OS thread per store with a baton (mutex/condvar or channel), versus
   - (b) Wasmtime async fibers (`call_async`, async host functions; `Config::async_support` was removed in 42, and a store "requires async" once async host functions are registered).

   Measure handoff cost and the impact on fn-174's typed sync imports.
   - Default: (a). Imports stay sync, it is serialized by construction, `catch_unwind` per store step is simple, and the guest stack is set via `max_wasm_stack` on adequately sized threads.
2. **Scheduling:** strictly serialized, one active instance at a time, matching the Go broker's `active`/`schedule()` (broker.go:136). Keep a seam (a scheduler trait) so the later parallelism spike (fn-178.10) can be evaluated without restructuring.
3. **Control protocol.** One helper per cluster; the cluster execute frame carries instances, per-instance boot config (fuel, memory, namespace/volume/network config, recovery image references), World/clock config, the fault plan and declared boundaries. Defaults:
   - **Declared boundaries** (`CrashOnOutput`, `StopOnOutput`, `CrashAfterOutputs`, storage crash, scheduled link faults) are validated by Go and re-validated by Rust, and applied in Rust deterministically.
   - **Host commands** (restart of incarnation N+1, live `Crash`/`Stop`/`Link` from the Go scenario through `gomad3sim.WASMExecution`, `Finish`, `Shutdown`) arrive as control frames.
   - **Pause points:** the helper blocks for commands only at deterministic pause points (an instance terminal, a declared boundary firing, a restart hold, cluster deadlock or idle). Unbound live commands are polled at scheduler handoffs, which is as nondeterministic as today's `broker.Crash` under `b.mu`. Such runs remain non-replayable; `simulation/replay.go:19-56` already rejects them.
   - **Per-instance results** stream as each incarnation terminates, so a later helper failure does not lose finished instances. World evidence comes at Finish.
4. **Terminal taxonomy.**
   - Modeled crash/stop/storage_crash is a per-instance modeled outcome.
   - Trap, fuel or capacity is a per-instance outcome. That store is dropped and never reused after a trap (practice-scout: CVE-2026-104855 class).
   - Host or engine error, a panic in shared scheduler/World/clock state, or a helper exit is a whole-run `infrastructure` outcome.
   - Panic policy: catch a panic per store step. If shared state may be torn, it is whole-run infrastructure; otherwise it is that instance's infrastructure outcome.
5. **Watchdog.** The Go process timeout still bounds the whole run (`hostexec` Timeout). In Rust, an engine-wide epoch ticker serves only as a backstop for a stuck guest. Epochs are not deterministic, so an epoch trap is classified `infrastructure`, never a modeled outcome. A restart hold waiting on Go commands counts against the Go timeout.
6. **Memory budget (R5; resolves the parked unknown).** Default:
   - Each instance keeps its own `ResourceLimiter` (engine.rs:20-58 `Limits`) enforcing the per-instance `memory_bytes` and recording the per-instance linear-memory peak, as today. Exceeding it is that instance's `capacity`.
   - Whole-helper peak RSS becomes a cluster sample.
   - The qualification budget (`qualification/budget.go`, `cost.go`) compares the cluster RSS against the existing conservative bound (the sum of per-resident warm helper peaks, budget.go:76-82), instead of per-incarnation RSS.
7. **Dependencies.** No new crates. HMAC-SHA256 for World `choiceRank` is implemented over the pinned `sha2`. Rust emits the expanded `GMWSNAP1` wire; Go builds the `GMWORLD2` gzip envelope with its existing encoder and validates via `world.Restore`, so gzip byte stability stays Go's. Uncompressed World evidence crosses the pipe in 64 KiB chunks, bounded by the profile's 384 MiB expanded limit.
8. **Go reference.** The Go broker path, one helper per instance in split mode from fn-176.6, stays selectable for end-to-end parity, like `remoteModel`.

Record the decisions in the fn-178 spec's Decision Context, replacing the planner's defaults (the spec has no Parked unknowns section any more: memory is resolved here, parallelism in fn-178.10). Write the draft protocol section.

### Investigation targets
**Required:**
- `tools/gomad3/backends/wasm/wasi/broker.go` (topology, schedule, revoke, Restart, triggers)
- `tools/gomad3/backends/wasm/simulation/provider.go:272-900` (Open, Start, Crash, Stop, Link, Finish, Shutdown)
- `tools/gomad3/backends/wasm/simulation/replay.go:19-60` (retained lifecycle contract)
- `tools/gomad3/backends/wasm/wasmhost/src/engine.rs:20-120,273-430` (Limits, State, execute pipeline)
- `tools/gomad3/backends/wasm/qualification/budget.go`, `tools/gomad3/backends/wasm/qualification/cost.go`

### Key context
- Wasmtime 47.0.3: `Store<T>` is `Send` iff `T: Send`; `Engine`, `Module` and `InstancePre` are `Send + Sync`. `ResourceLimiter::memory_growing` is per store and also gates the initial allocation. `Engine::increment_epoch` is engine-wide. Fuel is per store and a new store starts at 0.
## Acceptance
- [ ] The spike numbers (handoff ns per switch for both models, with the guest and host recorded) are retained under `.flow/artifacts/`.
- [ ] The fn-178 spec's Decision Context records items 1–8 with rationale; the R5 'Planned resolution' note is confirmed or amended.
- [ ] `wasmhost/protocol.md` has a draft "Cluster execution" section listing frames, pause points, terminal taxonomy and evidence streaming.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
