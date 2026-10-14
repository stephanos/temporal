# Network models in the Rust helper

## Conversation Evidence

> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 15): "does that impact an existing spec like fn-154"
> user (turn 17, selected): "targeted-deps"
> owner decision (2026-10-13): re-scope this spec as a building block for fn-178; the helper glue, Go plumbing, guest-level differential, gates and speedup move to fn-178.

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

<!-- Goal & Context: 25% [paraphrase], 75% [inferred] -->

The network model is not WASM-only. fn-154 (simpler virtual network with stalling partitions) is rewriting it as one local connection owner shared by both simulation backends. That rewrite covers byte-bounded FIFO queues, held bytes across partitions, virtual-time stall expiry and timeout reset, and payload-free history. A Rust copy must therefore duplicate the post-fn-154 semantics; it does not replace the Go owner.

`Config.NetworkV2` is set only by the multi-node simulation provider (`internal/backends/wasm/cluster/provider.go`), which always attaches a Go broker, and the installed CLI refuses network and storage configurations (`internal/backends/wasm/installation/installation.go`). The helper disables its local model whenever a broker is present (`wasi/process.go`). A single-instance helper path for v2 networks would therefore only speed up direct test-harness runs. Real campaigns benefit only once fn-178 hosts the whole cluster, broker included, in one helper.

The goal of this spec is the building block fn-178 needs: a Rust network owner with the post-fn-154 semantics, proven equal to the Go owner by a Go-generated operation corpus. Wiring it into the helper (the `net_stream` ABI, idle, fault application, evidence) and the Go side happen in fn-178, for the in-process cluster.

## Architecture & Data Models

- The helper crate gains a Rust duplicate of the network owner as fn-154 leaves it: the per-pair byte-bounded FIFO owner, held bytes across disabled links, effective-heal release, stall intervals with strict expiry and timeout reset, pending-dial stall, and the history and reservation rules. Topology faults (partition, heal, delay, reset) are owner operations; v2 has no seed or delay bounds, so delay comes only from the `delay` action.
- The Rust owner lives in its own helper module (`wasmhost/src/network/`) with no knowledge of the helper `Model`, the `net_stream` buffer format or the broker. It is multi-domain from the start: registered domains with run, node, incarnation and address, revocation, links keyed by node, and request IDs with a per-slot base (`wasi/broker_network.go`), so fn-178 can attach every cluster instance to one owner without a refactor.
- The Go connection owner from fn-154 stays authoritative for the native backend, for single-instance and broker runs until fn-178 lands, and as the reference implementation.
- A versioned corpus of operation scripts and expected per-step results is generated from the Go owner and replayed by plain `cargo test`. A Go currency test regenerates the corpus in memory and fails on any difference.

## API Contracts

- The Rust owner's API mirrors the Go `Model` API (`internal/model/network/adapter.go`): register/revoke domains, dispatch of ops 0-13, cancel, partition/heal/delay/reset, advance(now) → ready ids, next deadline, and the final Record (transitions plus snapshot structure). fn-178 is its only consumer.
- The Rust source pins the Go projection's network implementation digest it was last checked against; a Go test fails when the projection changes without a Rust update.

## Edge Cases & Constraints

- Same-time actions follow serialized model order, including fn-154's equal-time heal-versus-expiry orders and simultaneous pair expiry.
- Reservation checks are atomic: insufficient history capacity changes nothing. Overflow in stall arithmetic rejects before mutation.
- The corpus includes multi-node, broker-shaped scenarios (several node pairs, strict topology, revocation, per-slot request bases), because the in-process cluster is the owner's only consumer.

## Acceptance Criteria

- ~~**R1:**~~ ~~Runs configured with the v2 network model use the helper's local scheduling and clock model, and make no network or runtime-decision callbacks to Go (v1 already runs in the helper, and fn-166 R8 deletes it). Errors: invalid network configuration → `invalid` outcome before execution; oversize or malformed `net_stream` buffers (nonzero reserved fields, bad version) → existing capacity or errno classification; evidence over its bound → `network-v2-evidence-bytes` capacity.~~ Moved to fn-178 R6 (owner decision 2026-10-13).
- **R2:** A differential harness runs identical seeds, fault plans and operation sequences through fn-154's Go connection owner and the Rust owner, and requires identical network history, held/timeout outcomes and results. It covers fn-154's equal-time heal-versus-expiry orders and history-reservation exhaustion cases. It runs at model level: a Go-generated operation-script corpus replayed by the Rust owner, with a currency test against the Go owner. ~~and at guest level (Go-owner and helper-owner runs of a v2 fault fixture compared byte for byte)~~ The guest-level comparison moved to fn-178 R7 (owner decision 2026-10-13). No error surface beyond the harness.
- ~~**R3:**~~ ~~The Temporal named network reset/retry exact-replay gate and the prototype network-fault workflow pass with the Rust network models, and seeds still produce distinct schedules. If the v1 prototype gate has been removed, the prototype network-fault workflow runs with an equivalent v2 fault plan. No error surface beyond R1–R2.~~ Moved to fn-178 R8 (owner decision 2026-10-13).
- ~~**R4:**~~ ~~Authorized execution of a network-v2 Temporal selection drops against the pre-change baseline at the same seed. The baseline comes from the guest-side experiments spec (fn-172 R3), recorded on the post-fn-154 candidate; the comparison includes the parent Go process cost. No error surface beyond R1–R3.~~ Moved to fn-178 R8 (owner decision 2026-10-13).

## Boundaries

- Cross-instance (broker) networking is out of scope. [paraphrase]
- Native-backend network semantics are unchanged; network semantics changes belong to fn-154, not here. [paraphrase]
- No helper integration: the `net_stream` ABI, local-model glue, idle, fault-plan application, network evidence, Go plumbing and run-level gates belong to fn-178. The release helper serves no network traffic from this spec and gains no test-only entry point.
- No single-instance (broker-less) helper path for v2 networks; those runs keep the Go owner.

## Decision Context

This is part of the user's choice to move hot paths into Rust and accept duplication. Because fn-154 makes the network model a shared owner for both backends, this spec duplicates rather than moves, and it depends on fn-154. Porting the pre-fn-154 model would port semantics that are about to be replaced. [paraphrase]

### Planning decisions (2026-10-13, autonomous planning pass)

- **v2 only.** fn-166 R8 deletes network v1, and v1 already runs in the helper.
- **Golden corpus, not a hidden helper subcommand.** The release helper gains no test-only entry point. Go generates operation scripts and expected results from the Go owner, the Rust tests replay them, and a Go currency test regenerates and compares the corpus.

### Re-scope (owner decision, 2026-10-13)

The planning pass asked whether direct-run scope was intended, because Runner/simulation runs always use a broker. The owner answered by re-scoping: this spec keeps the Go-owner operation corpus (fn-177.1) and the Rust owner (fn-177.2, fn-177.3) and becomes a building block for fn-178.

- Moved to fn-178: the `net_stream` ABI and helper glue, idle integration and binary network evidence (former fn-177.4, now fn-178.12 and fn-178.7), the Go configuration plumbing and evidence decoding (former fn-177.5, now fn-178.13), and the guest-level differential, network gates and speedup (former fn-177.6, now folded into fn-178.9 and fn-178.11). R1 moved to fn-178 R6, the guest-level half of R2 to fn-178 R7, and R3 and R4 to fn-178 R8.
- Dropped as meaningful only for broker-less single-instance runs: enabling the helper's local model for v2 runs without a broker (the `process.go` gate change), the Rust readiness/idle owner replacing the Go `internal/model/world` package for single instances, the fixed 127.0.0.1/127.0.0.2 single-instance domains and per-run fault plan with reset by port, removing the `internal/model/world` entries from the single-instance evidence document, the named reset/retry Temporal gate and the prototype network-fault workflow on the Rust model (both broker-less), and the single-instance network-v2 speedup against fn-172's network-v2 baseline.
- Dependencies: fn-174 (typed imports and binary framing) and fn-172 (network-v2 baseline) were needed only by the moved parts and were removed. fn-178 reaches fn-174 through fn-176 → fn-175 → fn-174 and depends on fn-172 directly. fn-154 stays.

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Multi-instance clusters in one helper process (depends on this spec); Guest-side cost experiments and baselines.

## Early proof point

Task fn-177-gomad-network-models-in-the-rust-helper.2 validates the core approach (a hand-written Rust owner replays the Go owner's operation corpus exactly). If it fails, re-evaluate generating the Rust owner from the same native sources before continuing with fn-177-gomad-network-models-in-the-rust-helper.3.

## Quick commands

```bash
cargo +1.94.1 test --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked network
(cd tools/gomad && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestNetworkCorpus|TestModel|TestNetworkV2' ./internal/model/network/... ./internal/backends/wasm/wasi)
```


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | ~~Runs configured with the v2 network model use the helper's local scheduling and clock model, and make no network or runtime-decision callbacks to Go (v1 already runs in the helper, and fn-166 R8 deletes it). Errors: invalid network configuration → `invalid` outcome before execution; oversize or malformed `net_stream` buffers (nonzero reserved fields, bad version) → existing capacity or errno classification; evidence over its bound → `network-v2-evidence-bytes` capacity.~~ Moved to fn-178 R6 (owner decision 2026-10-13). | — | Moved to fn-178 R6 (owner decision 2026-10-13); no fn-177 task. |
| R2 | A differential harness runs identical seeds, fault plans and operation sequences through fn-154's Go connection owner and the Rust owner, and requires identical network history, held/timeout outcomes and results. It covers fn-154's equal-time heal-versus-expiry orders and history-reservation exhaustion cases. It runs at model level: a Go-generated operation-script corpus replayed by the Rust owner, with a currency test against the Go owner. ~~and at guest level (Go-owner and helper-owner runs of a v2 fault fixture compared byte for byte)~~ The guest-level comparison moved to fn-178 R7 (owner decision 2026-10-13). No error surface beyond the harness. | fn-177-gomad-network-models-in-the-rust-helper.1, fn-177-gomad-network-models-in-the-rust-helper.2, fn-177-gomad-network-models-in-the-rust-helper.3 | — |
| R3 | ~~The Temporal named network reset/retry exact-replay gate and the prototype network-fault workflow pass with the Rust network models, and seeds still produce distinct schedules. If the v1 prototype gate has been removed, the prototype network-fault workflow runs with an equivalent v2 fault plan. No error surface beyond R1–R2.~~ Moved to fn-178 R8 (owner decision 2026-10-13). | — | Moved to fn-178 R8 (owner decision 2026-10-13); no fn-177 task. |
| R4 | ~~Authorized execution of a network-v2 Temporal selection drops against the pre-change baseline at the same seed. The baseline comes from the guest-side experiments spec (fn-172 R3), recorded on the post-fn-154 candidate; the comparison includes the parent Go process cost. No error surface beyond R1–R3.~~ Moved to fn-178 R8 (owner decision 2026-10-13). | — | Moved to fn-178 R8 (owner decision 2026-10-13); no fn-177 task. |
