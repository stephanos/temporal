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

<!-- Goal & Context: 20% [paraphrase], 80% [inferred] -->

Multi-instance Temporal clusters run one helper process per instance. A Go broker serializes their continuations, owns the shared simulation clock and applies World's externally visible delivery order. With a broker configured, the helper's local model is disabled, so every runtime decision and every cross-instance message crosses process boundaries, and instances hand off control between processes.

The goal is to host all instances of a cluster as separate Wasmtime stores in one helper process, with clock, World ordering and broker scheduling in Rust, so cluster runs make no per-decision or per-message callbacks to Go.

## Architecture & Data Models

- One helper process hosts N instance stores sharing one engine and compiled module, each with its own namespace, volume, network endpoint, model state and evidence streams.
- Broker scheduling, the shared simulation clock and World delivery order run in Rust. World and the simulation clock are shared with the native backend and the runner in Go, so the Rust versions are duplicates kept equivalent by differential tests.
- Incarnation lifecycle (crash, stop, revocation, restart from a recovery image) is modeled per instance inside the process.
- Go receives per-instance and World evidence at the end of the run, and verifies or replays it.
- World evidence uses the single compressed snapshot profile with a binary envelope that fn-153 introduces. Multi-node connections use fn-154's shared connection semantics, via the Rust network model.

## API Contracts

- Execute request: cluster configuration (instances, per-instance inputs, World and clock configuration, fault plan). [inferred]
- Results: per-instance results plus World evidence, decodable by the existing Go consumers. Format changes are allowed. [inferred]

## Edge Cases & Constraints

- A trap, fuel exhaustion or capacity failure in one instance must not corrupt other instances' state or evidence. Modeled crashes must stay distinguishable from host failures.
- Watchdog timeouts still bound the whole run.
- Memory accounting per instance is no longer a process measurement; budgets need an in-process equivalent.
- The revocation-after-reply race class fixed in fn156 must not reappear in the in-process scheduler.

## Acceptance Criteria

- **R1:** A cluster run executes all instances in one helper process, with broker scheduling, the shared clock and World delivery order in Rust, and makes no runtime-decision or cross-instance-message callbacks to Go. Errors: invalid cluster configuration → `invalid` outcome before execution; host failure in the helper → whole-run infrastructure outcome. [paraphrase]
- **R2:** A differential harness drives identical event sequences through the Go and Rust World and clock implementations and requires identical delivery order and World evidence. No error surface beyond the harness. [inferred]
- **R3:** Modeled incarnation crash, stop, revocation and restart keep their current semantics, and a trap or capacity failure in one instance is reported for that instance without corrupting the others' evidence. Errors: a crash during a pending cross-instance delivery follows the Go broker's current classification. [inferred]
- **R4:** The independent-clusters partition/heal/whole-cluster-crash gate and the multi-instance qualification success and canary cells pass, and cluster authorized execution drops against the pre-change baseline from the guest-side experiments spec. No error surface beyond R1–R3. [inferred]
- **R5:** Per-instance peak memory is reported and enforced against the existing multi-instance budget. Errors: an instance exceeding its memory limit → that instance's capacity classification. [inferred]

## Boundaries

- The Go World and simulation clock remain authoritative for the native backend and the runner; they are not removed. [paraphrase]
- `wasmtime-go` and in-operator embedding are out of scope; the cluster stays in a helper subprocess. [paraphrase]

## Decision Context

The user accepted duplicating Go logic in Rust when it buys performance. World and the simulation clock are shared with native, so this is the one sibling that duplicates rather than moves. It is also the largest change, with real design risk around in-process isolation, memory budgets and incarnation semantics, which is why a design review is recommended before work starts. This spec depends on the Durable volume and Network models specs, on fn-154 (shared multi-node connection owner) and on fn-153 (World snapshot format), so it ports final formats and semantics rather than ones those specs are replacing. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Guest-side cost experiments and baselines.

## Parked unknowns

- How per-instance memory budgets are measured inside one process (guest linear-memory size versus the process high-water mark). Resolve by a design decision during plan review.
- Whether instances should run truly in parallel on host threads or stay strictly serialized as today. Serialization is simpler for determinism; resolve after the R2 harness exists.

## Requirement coverage

| R-ID | Task |
|---|---|
| R1 | fn-N.M (TBD - populate via /flow-next:plan) |
| R2 | fn-N.M (TBD - populate via /flow-next:plan) |
| R3 | fn-N.M (TBD - populate via /flow-next:plan) |
| R4 | fn-N.M (TBD - populate via /flow-next:plan) |
| R5 | fn-N.M (TBD - populate via /flow-next:plan) |
