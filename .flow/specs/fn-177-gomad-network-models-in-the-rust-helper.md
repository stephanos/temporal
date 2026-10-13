# Network models in the Rust helper

## Conversation Evidence

> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 15): "does that impact an existing spec like fn-154"
> user (turn 17, selected): "targeted-deps"

## Goal & Context

<!-- Goal & Context: 25% [paraphrase], 75% [inferred] -->

The Go adapter enables the helper's local scheduling and clock model only when a run has no v2 network and no broker. Runs with the v2 network therefore send **every** scheduler, select and clock decision back to Go, exactly as before fn156, and fn156's 3.7–4.6× execution speedup does not reach them. The network model is not WASM-only. fn-154 (simpler virtual network with stalling partitions) is rewriting it as one local connection owner shared by both simulation backends. That rewrite covers byte-bounded FIFO queues, held bytes across partitions, virtual-time stall expiry and timeout reset, and payload-free history. The Rust model must therefore duplicate the post-fn-154 semantics; it does not replace the Go owner.

The goal is to run the post-fn-154 network model inside the helper, so network runs use the local model and make no network callbacks to Go.

## Architecture & Data Models

- The helper hosts a Rust duplicate of the network model as fn-154 leaves it: the per-pair byte-bounded FIFO owner, held bytes across disabled links, effective-heal release, stall intervals with strict expiry and timeout reset, pending-dial stall, the history and reservation rules, seeded delays, the fault plan, and the `net_stream`/`net` imports.
- The Go connection owner from fn-154 stays authoritative for the native backend and remains the reference implementation.
- Network evidence (network events and the network transcript) is produced by the helper and decoded by Go.
- With the v2 network configured, the local model is enabled.

## API Contracts

- Execute request: the network model configuration (seed, delay bounds, fault plan, profile v1 or v2). [inferred]
- Results: network events and transcript in a Go-decodable form. Format changes are allowed. [inferred]

## Edge Cases & Constraints

- Fault timing is in virtual time and must interleave with scheduler and clock decisions exactly as the Go model does.
- Buffer limits for `net_stream` payloads keep their bounds.
- Multi-instance broker networking is out of scope; it belongs to the clusters spec.

## Acceptance Criteria

- **R1:** Runs configured with the v1 or v2 network model use the helper's local scheduling and clock model, and make no network or runtime-decision callbacks to Go. Errors: invalid network configuration → `invalid` outcome before execution; oversize `net_stream` payloads → existing capacity classification. [paraphrase]
- **R2:** A differential harness runs identical seeds, fault plans and guest traffic through fn-154's Go connection owner and the Rust model, and requires identical network history, held/timeout outcomes and transcripts. It covers fn-154's equal-time heal-versus-expiry orders and history-reservation exhaustion cases. No error surface beyond the harness. [inferred]
- **R3:** The Temporal named network reset/retry exact-replay gate and the prototype network-fault workflow pass with the Rust network models, and seeds still produce distinct schedules. No error surface beyond R1–R2. [inferred]
- **R4:** Authorized execution of a network-v2 Temporal selection drops against the pre-change baseline at the same seed. The baseline comes from the guest-side experiments spec. No error surface beyond R1–R3. [inferred]

## Boundaries

- Cross-instance (broker) networking is out of scope. [paraphrase]
- Native-backend network semantics are unchanged; network semantics changes belong to fn-154, not here. [paraphrase]

## Decision Context

This is part of the user's choice to move hot paths into Rust and accept duplication. Because fn-154 makes the network model a shared owner for both backends, this spec duplicates rather than moves, and it depends on fn-154. Porting the pre-fn-154 model would port semantics that are about to be replaced. The motivating finding is that v2-network runs currently get none of fn156's local-model speedup. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Multi-instance clusters in one helper process (depends on this spec); Guest-side cost experiments and baselines.
