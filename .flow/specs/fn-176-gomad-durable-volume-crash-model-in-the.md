# Durable volume crash model in the Rust helper

## Conversation Evidence

> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 16): "or fn-153"

## Goal & Context

<!-- Goal & Context: 25% [paraphrase], 75% [inferred] -->

Storage-configured runs (durability, crash/reopen and SQLite crash selections) use the Go volume model: crash points, faults, snapshots, crash enumeration, `StorageImage` production, and ephemeral SQLite lock databases. Every storage operation crosses to Go. The volume model is used only by the WASM backend (adapter, simulation, qualification), so it can move to Rust rather than be duplicated. Go keeps only what it needs to read the resulting evidence.

The goal is for storage-configured runs to make no storage callbacks to Go, and to keep their crash semantics exactly.

## Architecture & Data Models

- The helper hosts the volume model on top of the Rust namespace from the filesystem spec: volume configuration and limits, scheduled storage faults, operation sequencing, crash-point enumeration and selection, durable versus volatile state, recovery images, and the SQLite lock-database exception.
- The helper produces storage evidence (operations, snapshots, crash enumeration, the recovery image). Go consumers decode it for replay, retained artifacts, qualification and recovered runs.
- A recovered run starts from a recovery image supplied in the execute request.

## API Contracts

- Execute request: storage model configuration, crash selection and limits, and an optional recovery image. [inferred]
- Results: storage operations, snapshots, crash enumeration and the recovery image, in a format Go can decode. Format changes are allowed. [inferred]

## Edge Cases & Constraints

- Every fault action and crash boundary behaves as in the Go model, including partial writes, sync semantics, rename/unlink durability and lock-database exceptions.
- Capacity limits (volume bytes, files, events) keep their classifications.
- A recovery image that is inconsistent with the volume configuration is refused.
- Storage evidence formats are ported in their post-fn-153 form (strict decoding, retired canonical encoder), not the formats fn-153 replaces.

## Acceptance Criteria

- **R1:** Storage-configured runs answer every storage operation in the helper, including faults, crash enumeration, crash selection and recovery-image production, with zero storage callbacks to Go. Errors: invalid storage configuration, crash selection or recovery image → `invalid` outcome before execution; capacity exhaustion → existing capacity classification. [paraphrase]
- **R2:** A differential harness runs identical operation and fault sequences through the Go and Rust volume models and requires identical replies, crash enumerations and recovery images. No error surface beyond the harness. [inferred]
- **R3:** Go consumers (replay, retained artifacts, recovered runs, qualification) decode the helper-produced storage evidence, and existing retained-artifact replay paths pass with newly produced evidence. Errors: undecodable or truncated storage evidence → classified refusal, never a silent pass. [inferred]
- **R4:** The Temporal committed-workflow crash/reopen gate and the SQLite durable qualification cells pass with the Rust volume model. No error surface beyond R1–R3. [inferred]

## Boundaries

- Native-backend storage semantics are unchanged. [inferred]
- Network and multi-instance work are out of scope. [paraphrase]

## Decision Context

The user accepted Go→Rust duplication for performance. Here the volume model is WASM-only, so it moves rather than duplicates, and Go keeps only a decoder. This spec depends on the Filesystem namespace spec. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process (depends on this spec); Guest-side cost experiments and baselines.
