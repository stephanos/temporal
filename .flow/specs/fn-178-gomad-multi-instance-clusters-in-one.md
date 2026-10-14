# Multi-instance clusters in one helper process

## Conversation Evidence

> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 15): "does that impact an existing spec like fn-154"
> user (turn 16): "or fn-153"
> user (turn 17, selected): "targeted-deps"

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

<!-- Goal & Context: 20% [paraphrase], 80% [inferred] -->

Multi-instance Temporal clusters run one helper process per instance. A Go broker serializes their continuations, owns the shared simulation clock and applies World's externally visible delivery order. With a broker configured, the helper's local model is disabled, so every runtime decision and every cross-instance message crosses process boundaries, and instances hand off control between processes.

The goal is to host all instances of a cluster as separate Wasmtime stores in one helper process, with clock, World ordering and broker scheduling in Rust, so cluster runs make no per-decision or per-message callbacks to Go.

## Architecture & Data Models

- One helper process hosts N instance stores sharing one engine and compiled module, each with its own namespace, volume, network endpoint, model state and evidence streams.
- Broker scheduling, the shared simulation clock and World delivery order run in Rust. World and the simulation clock are shared with the native backend and the runner in Go, so the Rust versions are duplicates kept equivalent by differential tests.
- Incarnation lifecycle (crash, stop, revocation, restart from a recovery image) is modeled per instance inside the process.
- Go receives per-instance and World evidence at the end of the run, and verifies or replays it.
- World evidence uses the single compressed snapshot profile with a binary envelope that fn-153 introduces. Multi-node connections use fn-154's shared connection semantics, via fn-177's Rust network owner.
- Cluster networking: one Rust network owner (fn-177) is shared by all instances in the helper. The helper serves every instance's `net_stream`, applies the broker's addressing, link topology, central fault schedule and live link changes, and folds network deadlines into the in-process clock. Each instance emits a binary network evidence stream that Go decodes into the document the Go-broker path produces, recomputing identities with the generated Go codec. The Go owner stays for single-instance runs, the native backend and the Go-broker reference path.
- Rust emits the expanded World snapshot wire through a writer generated from the same checked-in schema as the Go codec. Go wraps it in the compressed envelope with its existing encoder, then validates it by restoring the snapshot. No compression or HMAC crate is added: World's seeded equivalence ranking uses HMAC-SHA256 built on the pinned SHA-256 crate.
- Scheduling is strictly serialized, as today: one active instance, ordered handoff at World-visible operations. The scheduler sits behind a seam so the parallelism decision can be evaluated later.
- The per-process Go broker path stays selectable as the end-to-end reference, using split ownership from the durable volume spec. The differential harness guards Go World and clock drift by pinning their implementation hashes.

## API Contracts

- Execute request: cluster configuration (instances, per-instance inputs, World and clock configuration, network configuration with limits, addresses, links, strict topology and the central fault schedule, declared crash/stop boundaries). Go stays the configuration authority and refuses invalid network configuration before the helper starts. One `started` and one `authorize` cover the cluster.
- Control frames: incarnation restart, and live lifecycle commands from the Go scenario. Boundaries declared up front are applied deterministically in Rust. The helper blocks for host commands only at deterministic pause points. Unbound live commands are polled at scheduler handoffs; they are as nondeterministic as today and remain non-replayable.
- Results: per-incarnation results stream as each incarnation terminates, including the instance's binary network evidence. World evidence arrives at finish. Both are decodable by the existing Go consumers, and format changes are allowed.

## Edge Cases & Constraints

- A trap, fuel exhaustion or capacity failure in one instance must not corrupt other instances' state or evidence. Modeled crashes must stay distinguishable from host failures.
- Watchdog timeouts still bound the whole run.
- Memory accounting per instance is no longer a process measurement; budgets need an in-process equivalent.
- The revocation-after-reply race class fixed in fn156 must not reappear in the in-process scheduler. Invariant: a revocation caused by an instance's host call takes effect before that instance performs any further effect.
- A store is never reused after a trap or fuel exhaustion; a restart builds a fresh store.
- A panic in shared scheduler, World or clock state is a whole-run infrastructure outcome. An epoch-based watchdog trap is never reported as a modeled outcome.
- Uncompressed World evidence crossing the pipe is bounded by the snapshot profile's expanded limit and is chunked.
- Network fault timing is in virtual time and must interleave with scheduler and clock decisions exactly as in the Go broker. Buffer limits for `net_stream` payloads keep their bounds.
- Per-instance network evidence is emitted on every termination (exit, trap, fuel, capacity, storage interrupt, modeled crash or stop), capped by the instance's transcript byte limit with the `network-v2-evidence-bytes` capacity class.

## Acceptance Criteria

- **R1:** A cluster run executes all instances in one helper process, with broker scheduling, the shared clock and World delivery order in Rust, and makes no runtime-decision or cross-instance-message callbacks to Go. Errors: invalid cluster configuration → `invalid` outcome before execution; host failure in the helper → whole-run infrastructure outcome. [paraphrase]
- **R2:** A differential harness drives identical event sequences through the Go and Rust World and clock implementations and requires identical delivery order and World evidence. No error surface beyond the harness.
- **R3:** Modeled incarnation crash, stop, revocation and restart keep their current semantics, and a trap or capacity failure in one instance is reported for that instance without corrupting the others' evidence. Errors: a crash during a pending cross-instance delivery follows the Go broker's current classification.
- **R4:** The independent-clusters partition/heal/whole-cluster-crash gate and the multi-instance qualification success and canary cells pass, and cluster authorized execution drops against the pre-change baseline from the guest-side experiments spec. No error surface beyond R1–R3.
- **R5:** Per-instance peak memory is reported and enforced against the existing multi-instance budget. Errors: an instance exceeding its memory limit → that instance's capacity classification. [inferred] Resolution (confirmed by fn-178.1, 2026-10-14, with the amendment below):
  - Each instance's guest linear-memory limit and peak are enforced and reported per store.
  - The single helper's peak RSS is checked against the existing conservative resident bound, replacing per-incarnation helper RSS samples.
  - Amendment (fn-178.1): the cluster helper's peak RSS is one measured sample (the helper's own `getrusage` high-water mark in `cluster_result.metrics`), the actual simultaneous peak of every resident instance. In `EvaluateBudget` (fn-178.11) a cluster sample is marked as such and:
    - covers every resident guest at once: the per-resident warm-measurement rule becomes one warm cluster sample whose instance list contains every resident, instead of one sample keyed per incarnation prefix;
    - is warm or cold by its compile-cache hit, as today;
    - is compared with the existing resident limit (`ResidentHelperPeakLimitBytes`, 8 GiB) in place of the summed per-resident warm peaks (`ConservativeResidentHelperPeakBytes` is the cluster sample itself);
    - is not held to the per-helper limits (4 GiB warm `HelperPeakLimitBytes`, 6 GiB cold), which bound one guest's helper; single-instance helper samples keep them. Sum of instance `memory_bytes` is not an admission bound; guest memory is reserved virtually per store.
- **R6:** (moved from fn-177 R1, owner decision 2026-10-13) In a cluster run, every instance's `net_stream` is served in the helper by one Rust network owner shared by all instances, with the broker's addressing, link topology, central fault schedule and live link changes applied at the virtual times the Go broker applies them, and network deadlines folded into the in-process clock. Cluster runs make no network callbacks to Go. Errors: invalid cluster network configuration (per-node fault schedule, bad address, duplicate or unknown link endpoint, unordered fault or unknown action) → `invalid` outcome before execution; oversize or malformed `net_stream` buffers (nonzero reserved fields, bad version) → existing capacity or errno classification; an instance's network evidence over its bound → that instance's `network-v2-evidence-bytes` capacity.
- **R7:** (moved from fn-177 R2's guest level, owner decision 2026-10-13) A guest-level network differential runs network fault fixtures through the Go-broker reference path (Go owner) and the in-process cluster (Rust owner), over several seeds and with replay, and requires byte-identical per-instance network evidence and broker network Record. The fixtures cover partition, heal, delay, stall expiry, a live link change, an incarnation crash with open connections, and one storage-interrupt combination. Go decodes the helper's network evidence strictly and recomputes its identities with the generated Go codec. Errors: malformed or truncated network evidence → `invalid` refusal for the run.
- **R8:** (moved from fn-177 R3 and R4, adapted to clusters, owner decision 2026-10-13) The cluster network gates (the independent-clusters partition/heal/whole-cluster-crash gate and the broker network-stream replay fixture) pass on the Rust network owner. Seeds still produce distinct schedules, and each gate report names the network owner that ran. The R4 speedup comparison includes the parent Go process cost. No error surface beyond R6–R7.

## Early proof point

Task fn-178-gomad-multi-instance-clusters-in-one.1 validates the core approach: a measured two-store handoff spike and recorded decisions on control protocol, threading, terminal taxonomy, watchdog and memory budgets. If the spike shows handoff cost or isolation that cannot meet R1/R3, re-evaluate a hybrid (one helper per instance with the Rust broker in a coordinator helper) before continuing with fn-178.6 and later tasks.

## Quick commands

```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked
cd tools/gomad && go test -tags test_dep -count=1 -run 'TestWorldConformance|TestClockConformance|TestBroker' ./internal/backends/wasm/cluster ./internal/backends/wasm/wasi
```

## Boundaries

- The Go World and simulation clock remain authoritative for the native backend and the runner; they are not removed. [paraphrase]
- `wasmtime-go` and in-operator embedding are out of scope; the cluster stays in a helper subprocess. [paraphrase]
- Making unbound live crash/stop/link commands deterministic or replayable is out of scope. The retained-replay contract already requires declared boundaries.
- Implementing parallel instance execution is out of scope unless the parallelism decision task recommends it; that would be a follow-up task.
- The single-helper cluster claims model fidelity only, not hard isolation.

## Decision Context

The user accepted duplicating Go logic in Rust when it buys performance. World and the simulation clock are shared with native, so this is the one sibling that duplicates rather than moves. It is also the largest change, with real design risk around in-process isolation, memory budgets and incarnation semantics, which is why a design review is recommended before work starts. This spec depends on the Durable volume and Network models specs, on fn-154 (shared multi-node connection owner) and on fn-153 (World snapshot format), so it ports final formats and semantics rather than ones those specs are replacing. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Guest-side cost experiments and baselines.

Planning decisions (2026-10-13, made without the user and open to override):
- Parked unknowns were turned into scheduled work.
  - Per-instance memory budgets are decided in the design task fn-178.1, with the default recommended under R5.
  - Parallel versus serialized execution is measured and decided in fn-178.10, after the harness, the serialized scheduler and the Go client exist. Serialized is the working assumption.
- Threading: the default recommendation is one OS thread per store with a baton, which keeps synchronous typed imports from the host-call fast path. The alternative is Wasmtime async fibers. fn-178.1's spike measures both before the cluster host is built.
- World snapshot: the Rust writer is generated from the shared schema. Go compresses and validates.
  - Rejected: Rust gzip, which risks byte divergence from Go's encoder and adds a crate.
  - Rejected: a hand-written Rust codec, which drifts.
- Interfaces assumed from siblings:
  - fn-177 provides only the Rust network owner with fn-154 semantics: multiple registered domains, link topology changes, revocation and per-slot request bases, proven against a Go-generated corpus that includes broker-shaped scenarios. It has no helper integration; fn-178.12 adds the `net_stream` ABI, broker glue and evidence, and fn-178.7 extends the owner if the broker needs an operation it lacks.
  - fn-176 provides Rust-held recovery images and the broker split mode that serves as the Go reference path.
  - fn-175 provides the conformance driver.
- R4 baseline: fn-172's retained multi-instance baseline when present. Otherwise fn-178.11 measures the pre-change per-process broker build at the same seed and records the substitution.
- Paths in task files are current as of planning. fn-166 (.7, .10, .11) moves `internal/model/world`, `simulation/*` and `internal/model/cluster` and renames the tree.

### Design record (fn-178.1, 2026-10-14)

Replaces the planner defaults above where they differ. Spike numbers: `.flow/artifacts/fn178-store-handoff-spike-2026-10-14.md`. Frames and taxonomy: `wasmhost/protocol.md`, "Cluster execution (draft, fn-178)".

1. **Threading: one OS thread per store with a baton (a).** Measured on the owner's Linux/arm64 host while another worker ran WASM gates (load average 3–17): thread baton handoff 12.5–13.6 µs per switch (Mutex+Condvar and park/unpark alike), async fibers 157 ns per switch; a typed sync import costs 2.0 ns per call from a sync or an async store, an async import that never yields 22 ns. Fibers are about 80x cheaper per switch, but they need the wasmtime `async` feature (four new `Cargo.lock` packages: `wasmtime-internal-component-macro`, `wasmtime-internal-wit-bindgen`, `wit-parser`, `id-arena`), async variants of every handoff-capable import and `call_async` for every cluster store. The Go broker hands off only in `prepareIdle`, so handoffs are bounded by guest idles, and each one already costs at least a pipe round trip plus a process wake today. Decision: (a), with the baton behind the scheduler seam so fn-178.10 can swap in fibers if handoff count times 13 µs is a material share of authorized execution. Stores are created on their own thread with an 8 MiB host stack (the single helper's main-thread size); `max_wasm_stack` stays at the engine default.
2. **Scheduling:** strictly serialized behind a `Scheduler` seam (`next(view) -> instance`). fn-178.6 ships run-to-completion in id order; fn-178.7 ports `Broker.schedule` (handoff at `runtime_idle`, first runnable in sorted id order, network advance before resuming a round of idlers).
3. **Control protocol:** one `cluster_execute` frame, one `started`, one `authorize`; instance-tagged frames are the single-execution frames with the prefix `{"instance":"<id>",`, so stripping the prefix gives the single-run bytes. Pause points and control frames as in item 3 of the task, reserved for fn-178.7/.8. Per-instance results stream at each terminal; World evidence at finish.
4. **Terminal taxonomy:** per instance: exit, trap, capacity, unsupported, invalid (that instance's ABI, model or correctly tagged reply validation), divergence, storage_crash, crash, stop; the store is dropped and never reused. Whole run: unattributable Go frames or EOF (invalid), any `infrastructure` classification in any store, and every panic. Deviation from the planner default: a panic is never instance-level, because shared transport, scheduler, World or clock state may be torn and proving otherwise per call site is not worth the risk.
5. **Watchdog:** the Go process timeout bounds the whole run, including pause-point waits. Deviation: no epoch ticker in the default build. Epoch interruption changes generated code and the configuration revision (and compile-cache identity) of every single-instance run, and a stuck serialized guest blocks the cluster either way; a watchdog kill is whole-run `infrastructure`, never modeled. Revisit only with a revision bump.
6. **Memory budget (R5):** confirmed, with the amendment recorded under R5.
7. **Dependencies:** no new crates (the threading decision keeps the lock unchanged). HMAC-SHA256 for `choiceRank` over the pinned `sha2`. Rust writes the expanded `GMWSNAP1` wire from a generated writer; Go wraps it in `GMWORLD2` with its encoder and validates via `world.Restore`. Cluster-scope `world` evidence crosses the pipe in 64 KiB chunks, bounded by the 384 MiB expanded limit.
8. **Go reference:** the per-process Go broker path stays selectable end to end (fn-176.6 split mode).
9. **Conformance harness (fn-178.2):** fn-175.1's subprocess conformance driver is not on `gomad` yet. World and clock follow fn-177.1's pattern instead: Go generates checked-in JSON corpora under `wasmhost/tests/testdata/{world,clock}_conformance` with the Go implementation-hash pins, a Go test fails when the corpus is stale or a pin changes, and plain `cargo test` replays every step against the Rust ports. Clock steps are driven through the exported `clock.BrokerClock` API; the compared arbiter state is the observable one (current time and the set of waiting participants).
10. **Volume attachment:** fn-178.6 instance entries reserve `storage`/`recovery_image` for fn-176's helper-owned volume; remote filesystem callbacks stay instance-tagged until then.
11. **Diagnostic builds:** `diagnostic-profile` and `guest-profile` keep per-thread attribution state that store threads cannot reach, so diagnostic builds refuse `cluster_execute` as `unsupported`. Cluster attribution is out of scope until fn-178.10 needs it (then: per-store sessions merged at finish).
12. **Early proof point:** the spike supports R1 and R3 without the hybrid. Isolation comes from separate stores with separate limiters, fuel and state on separate threads; the measured handoff cost is bounded by guest idles. fn-178.6 and later continue in one helper. The helper still registers untyped `func_new` imports (`engine.rs:394`) while the spike measured typed `func_wrap`; threads leave import registration unchanged either way, so fn-174's typed imports apply as they land.

### Network work moved from fn-177 (owner decision, 2026-10-13)

`Config.NetworkV2` is set only by the multi-node simulation provider (`internal/backends/wasm/cluster/provider.go`), which always attaches a Go broker, and the installed CLI refuses network and storage configurations (`internal/backends/wasm/installation/installation.go`). The helper disables its local model whenever a broker is present (`wasi/process.go`). fn-177's single-instance local model for v2 runs would only have helped direct test-harness runs, so the owner re-scoped fn-177 to the corpus and the Rust owner, and moved its helper and Go integration here.

- Former fn-177.4 (`net_stream` ABI, glue, idle, evidence) became fn-178.12 (ABI, broker-shaped glue, per-instance evidence) plus the idle and fault interleaving in fn-178.7.
- Former fn-177.5 (Go plumbing, evidence decoding) became fn-178.13.
- Former fn-177.6 (guest-level differential, network gates, speedup) is folded into fn-178.9 (network differential in the Go-broker parity test) and fn-178.11 (network gates, owner naming in reports, parent Go cost in the speedup record).
- fn-177 R1 became R6, R2's guest level became R7, and R3 and R4 became R8.
- Dropped, because they only matter for broker-less single-instance runs:
  - enabling the helper's local model for v2 runs without a broker (the `process.go` gate change), and the callback-count test for such runs
  - the Rust readiness and idle owner replacing the Go `internal/model/world` package for single instances; the Rust World and clock (fn-178.3, fn-178.5) take that role in the cluster
  - the fixed 127.0.0.1/127.0.0.2 single-instance domains and the per-run fault plan, including reset by port in handle order (the broker refuses local fault schedules and has no reset action)
  - removing the `internal/model/world` entries from the single-instance evidence document; broker-mode evidence stays unchanged and is the byte reference
  - the named reset/retry Temporal gate (`TestTemporalNetworkNamedResetRetriesAndExactReplay`) and the prototype network-fault workflow on the Rust model; both are broker-less single-instance runs
  - the single-instance network-v2 speedup against fn-172's network-v2 baseline; R4 and R8 use the cluster baseline

## Resolved via Research
<!-- provenance: plan Step 1 (docs-scout, practice-scout, docs-gap-scout, memory-scout) on 2026-10-13 -->

### docs-scout
- **Wasmtime 47.0.3**: `Store<T>` is `Send` iff `T: Send`. `Engine`, `Module` and `InstancePre` are `Send + Sync`. Items used with a foreign store panic, and instances are freed only when their store drops. Source: https://docs.rs/wasmtime/47.0.3/wasmtime/struct.Store.html
- **ResourceLimiter**: `memory_growing(current, desired, maximum)` is per store and also gates the initial allocation. It does not track peaks; record the peak inside the limiter. Source: https://docs.rs/wasmtime/47.0.3/wasmtime/trait.ResourceLimiter.html
- **Fuel and epochs**: fuel is per store, and a new store starts at 0. `Engine::increment_epoch` is engine-wide, with a per-store deadline. Epochs are not deterministic and serve only as a backstop. Bulk-memory operations consume fuel proportional to size since Wasmtime 46. Source: https://docs.rs/wasmtime/47.0.3/wasmtime/struct.Config.html
- **Async**: `Config::async_support` was removed in 42. A store requires async once async host functions or async yielding are configured, and then sync `call` errors. Source: https://github.com/bytecodealliance/wasmtime/blob/main/RELEASES.md
- **InstancePre**: one `Linker::instantiate_pre` shared across stores. Copy-on-write memory init on non-Linux Unix needs `Module::deserialize_file`, so measure RSS on macOS. Source: https://docs.wasmtime.dev/examples-fast-instantiation.html

### practice-scout
- **Gotcha:** never reuse a store after a trap, fuel exhaustion or epoch cancellation (CVE-2026-104855 class, fixed in 47.0.3). Treat a trap as the end of the incarnation. Source: https://www.wiz.io/vulnerability-database/cve/cve-2026-104855
- **Gotcha:** use ordered containers wherever iteration order reaches delivery order or evidence bytes, and compare raw bytes across languages rather than decoded JSON. Source: https://casa.rub.de/forschung/publikationen/detail/cross-language-differential-testing-of-json-parsers
- **Practice:** serialize guest execution with a single-active scheduler (the deterministic-simulation-testing model), and give every guest its own store, limiter and seed. Source: https://github.com/tokio-rs/turmoil/issues/19

### docs-gap-scout
- Docs to update: protocol.md (single-execution statement, frame and callback limits, getrusage fields, store admission), QUALIFICATION.md (multi-instance owner, per-process RSS budgets, "broker's single exact clock"), CLI.md (multi-instance replay, cost and budget text), README (broker and Go host model statements), ARCHITECTURE.md and SPEC.md (Rust duplicate of World and clock; model fidelity only), the tools/gomad3 README clock-facade sentence. Folded into fn-178.11.

### memory-scout
- No memory entries cover brokers, World, the clock or differential testing. "Freeze crash-test state before exact prefix comparison" applies to helper kill and resume of a multi-instance process tree.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | A cluster run executes all instances in one helper process, with broker scheduling, the shared clock and World delivery order in Rust, and makes no runtime-decision or cross-instance-message callbacks to Go. Errors: invalid cluster configuration → `invalid` outcome before execution; host failure in the helper → whole-run infrastructure outcome. | fn-178-gomad-multi-instance-clusters-in-one.6, fn-178-gomad-multi-instance-clusters-in-one.7, fn-178-gomad-multi-instance-clusters-in-one.9 | — |
| R2 | A differential harness drives identical event sequences through the Go and Rust World and clock implementations and requires identical delivery order and World evidence. No error surface beyond the harness. | fn-178-gomad-multi-instance-clusters-in-one.2, fn-178-gomad-multi-instance-clusters-in-one.3, fn-178-gomad-multi-instance-clusters-in-one.4, fn-178-gomad-multi-instance-clusters-in-one.5 | — |
| R3 | Modeled incarnation crash, stop, revocation and restart keep their current semantics, and a trap or capacity failure in one instance is reported for that instance without corrupting the others' evidence. Errors: a crash during a pending cross-instance delivery follows the Go broker's current classification. | fn-178-gomad-multi-instance-clusters-in-one.6, fn-178-gomad-multi-instance-clusters-in-one.8, fn-178-gomad-multi-instance-clusters-in-one.9 | — |
| R4 | The independent-clusters partition/heal/whole-cluster-crash gate and the multi-instance qualification success and canary cells pass, and cluster authorized execution drops against the pre-change baseline from the guest-side experiments spec. No error surface beyond R1–R3. | fn-178-gomad-multi-instance-clusters-in-one.11 | — |
| R5 | Per-instance peak memory is reported and enforced against the existing multi-instance budget. Errors: an instance exceeding its memory limit → that instance's capacity classification. [inferred] Resolution confirmed by fn-178.1 (see R5). | fn-178-gomad-multi-instance-clusters-in-one.1, fn-178-gomad-multi-instance-clusters-in-one.11, fn-178-gomad-multi-instance-clusters-in-one.6 | — |
| R6 | (moved from fn-177 R1, owner decision 2026-10-13) In a cluster run, every instance's `net_stream` is served in the helper by one Rust network owner shared by all instances, with the broker's addressing, link topology, central fault schedule and live link changes applied at the virtual times the Go broker applies them, and network deadlines folded into the in-process clock. Cluster runs make no network callbacks to Go. Errors: invalid cluster network configuration (per-node fault schedule, bad address, duplicate or unknown link endpoint, unordered fault or unknown action) → `invalid` outcome before execution; oversize or malformed `net_stream` buffers (nonzero reserved fields, bad version) → existing capacity or errno classification; an instance's network evidence over its bound → that instance's `network-v2-evidence-bytes` capacity. | fn-178-gomad-multi-instance-clusters-in-one.12, fn-178-gomad-multi-instance-clusters-in-one.13, fn-178-gomad-multi-instance-clusters-in-one.7 | — |
| R7 | (moved from fn-177 R2's guest level, owner decision 2026-10-13) A guest-level network differential runs network fault fixtures through the Go-broker reference path (Go owner) and the in-process cluster (Rust owner), over several seeds and with replay, and requires byte-identical per-instance network evidence and broker network Record. The fixtures cover partition, heal, delay, stall expiry, a live link change, an incarnation crash with open connections, and one storage-interrupt combination. Go decodes the helper's network evidence strictly and recomputes its identities with the generated Go codec. Errors: malformed or truncated network evidence → `invalid` refusal for the run. | fn-178-gomad-multi-instance-clusters-in-one.13, fn-178-gomad-multi-instance-clusters-in-one.9 | — |
| R8 | (moved from fn-177 R3 and R4, adapted to clusters, owner decision 2026-10-13) The cluster network gates (the independent-clusters partition/heal/whole-cluster-crash gate and the broker network-stream replay fixture) pass on the Rust network owner. Seeds still produce distinct schedules, and each gate report names the network owner that ran. The R4 speedup comparison includes the parent Go process cost. No error surface beyond R6–R7. | fn-178-gomad-multi-instance-clusters-in-one.11 | — |
