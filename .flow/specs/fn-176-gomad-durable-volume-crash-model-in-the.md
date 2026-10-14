# Durable volume crash model in the Rust helper

## Conversation Evidence

> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 16): "or fn-153"

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

<!-- Goal & Context: 25% [paraphrase], 75% [inferred] -->

Storage-configured runs (durability, crash/reopen and SQLite crash selections) use the Go volume model: crash points, faults, snapshots, crash enumeration, `StorageImage` production, and ephemeral SQLite lock databases. Every storage operation crosses to Go. The volume model is used only by the WASM backend (adapter, simulation, qualification), so it can move to Rust rather than be duplicated. Go keeps only what it needs to read the resulting evidence.

The goal is for storage-configured runs to make no storage callbacks to Go, and to keep their crash semantics exactly.

## Architecture & Data Models

- The helper hosts the volume model on top of the Rust namespace from the filesystem spec: volume configuration and limits, scheduled storage faults, operation sequencing, crash-point enumeration and selection, durable versus volatile state, recovery images, and the SQLite lock-database exception.
- The helper produces storage evidence (operations, snapshots, crash enumeration, the recovery image). Go consumers decode it for replay, retained artifacts, qualification and recovered runs.
- A recovered run starts from a recovery image supplied in the execute request.
- The Go volume package is a generated projection of the native overlay's volume (drift-tested). It stays in place as the differential oracle and as the source of decoder types. Production helper-owned and broker runs never execute it. The Go namespace-plus-volume execution path remains only as the test-only Go-owner reference and for stock-profile runs. This refines "Go keeps only a decoder": R2 needs a Go oracle, and a generated copy costs nothing to maintain.
- Two ownership modes:
  - **Helper-owned runs** (no broker): the helper owns namespace, volume, model and transcript.
  - **Broker runs** (split mode): the helper owns namespace, volume and transcript, while the Go broker keeps scheduling, runtime decisions and the shared clock. fd 1/2 writes stay remote in broker runs so output-triggered crash/stop boundaries still observe them in order.
  - The crash/reopen gate and the SQLite durable cells run under a broker, so R4 needs split mode. The clusters spec later moves the broker in-process and keeps split mode as its Go reference path.
- The volume clock is the helper model's virtual clock. In broker runs it follows broker time through the `local_state` adopted from remote replies.

## API Contracts

- Execute request:
  - storage model configuration, plus a Go-computed storage-model identity
  - crash selection and crash-enumeration limits
  - a flag requesting the recovery image
  - an optional recovery image descriptor (bytes, SHA-256), followed by chunked image frames
  - unchanged boot-identity inputs
- Results:
  - storage event lines, rendered byte-identical to the Go event JSON lines
  - a versioned binary storage-state envelope (snapshots, operations, per-volume enumerations, image identity) with a length-and-SHA-256 trailer
  - the recovery image blob, when requested
  - Go decodes all of these strictly into the existing volume types. Format changes are allowed.
- Recovery image: a versioned binary blob carrying everything recovery uses (durable tree, persisted state, history, next operation and inode counters, fault occurrence counters, storage-model identity). Its identity is the SHA-256 of the blob. The previous identity, a hash of the JSON of snapshots and occurrences, is retired because it omitted recovery-relevant state.
- Crash-state identities, volume identities and pending digests stay byte-identical to the Go projection, so they remain comparable with native crash catalogues.
- `storage_crash` becomes helper-originated in helper-owned runs: the helper stops before returning the syscall, flushes evidence, then reports the terminal. Broker `crash`/`stop` interrupts flush storage evidence the same way.
- Broker split mode adds a `revocable` reply flag and a `sync`/`continue` exchange. Before its next local storage effect after such a reply, the helper confirms that no revocation crossed the reply. This keeps the durable effects at a broker crash identical to the Go-owned path.

## Edge Cases & Constraints

- Every fault action and crash boundary behaves as in the Go model, including partial writes, sync semantics, rename/unlink durability and lock-database exceptions.
- Capacity limits (volume bytes, files, events) keep their classifications.
- A recovery image that is inconsistent with the volume configuration is refused.
- Storage evidence formats are ported in their post-fn-153 form (strict decoding, retired canonical encoder), not the formats fn-153 replaces.
- A short- or torn-write fault that does not shorten the matched write is a runtime `invalid`, as in Go, with the same transcript state.
- Storage transcript capacity is counted on the Go event JSON line length.
- Namespace filesystem-bytes and volume capacity remain distinct classifications.
- Recovered runs continue fault occurrence counts from the image.
- Lock-database paths never enter an image, and an image containing them is refused.
- Recovery images are bounded by the sum of volume capacities plus metadata. Larger images are a capacity outcome.
- The `crash-after` fault action is rejected by storage-model validation but referenced in the simulation provider's crash-bound logic. The Rust port mirrors the rejection, and the dead provider branch is deleted.

## Acceptance Criteria

- **R1:** Storage-configured runs answer every storage operation in the helper, including faults, crash enumeration, crash selection and recovery-image production, with zero storage callbacks to Go. Errors: invalid storage configuration, crash selection or recovery image → `invalid` outcome before execution; capacity exhaustion → existing capacity classification. [paraphrase]
- **R2:** A differential harness runs identical operation and fault sequences through the Go and Rust volume models and requires identical replies, crash enumerations and recovery images. No error surface beyond the harness.
- **R3:** Go consumers (replay, retained artifacts, recovered runs, qualification) decode the helper-produced storage evidence, and existing retained-artifact replay paths pass with newly produced evidence. Errors: undecodable or truncated storage evidence → classified refusal, never a silent pass.
- **R4:** The Temporal committed-workflow crash/reopen gate and the SQLite durable qualification cells (the `qualify-workloads` success and canary cells) pass with the Rust volume model. No error surface beyond R1–R3.

## Early proof point

Task fn-176-gomad-durable-volume-crash-model-in-the.1 validates the core approach: the fn-175 conformance driver can carry stateful volume, fault, enumeration and recovery sequences with byte-identical identities. If it fails, re-evaluate porting against the Go projection (for example by generating Rust from the overlay sources) before continuing with fn-176.2 and later tasks.

## Quick commands

```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked volume
make -C tools/gomad3/backends/wasm test-conformance
cd tools/gomad3 && go test -tags test_dep -count=1 -run 'TestVolumeConformance|TestStorage|TestSQLiteStorage' ./backends/wasm/wasi
```

## Boundaries

- Native-backend storage semantics are unchanged. [inferred]
- Network and multi-instance work are out of scope. [paraphrase] The broker split mode exists only so durable gates run under today's Go broker; moving the broker is the clusters spec.
- Native-catalogue-dependent SQLite selection cells stay native-owned (fn-128, fn-149) and are not part of R4.
- A direct Rust-versus-native comparison is out of scope. Native parity is transitive through the drift-tested Go projection.

## Decision Context

The user accepted Go→Rust duplication for performance. Here the volume model is WASM-only, so it moves rather than duplicates, and Go keeps only a decoder. This spec depends on the Filesystem namespace spec. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process (depends on this spec); Guest-side cost experiments and baselines.

Planning decisions (2026-10-13, made without the user and open to override):
- Keep the generated Go projection as the R2 oracle (see Architecture).
  - Rejected: deleting it. That would leave R2 without a reference, and the projection is drift-tested against the native overlay anyway.
- Add broker split ownership in this spec rather than waiting for the clusters spec. Otherwise R4 is unreachable, because the clusters spec depends on this one.
- Redefine the recovery-image identity as a hash of a self-contained binary image.
  - Rejected: replicating Go `json.Marshal` byte order in Rust, which is fragile and incomplete.
- Ship the image blob only on request. Always ship its identity.
- Render storage event lines byte-identical to Go, so storage transcripts stay comparable across the Go-owner reference and the helper. Carry structured evidence in a binary envelope.
- Paths in task files are current as of planning. fn-166 (.7, .11) may move `tools/gomad3/volume` under `internal/model` and rename the tree.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Storage-configured runs answer every storage operation in the helper, including faults, crash enumeration, crash selection and recovery-image production, with zero storage callbacks to Go. Errors: invalid storage configuration, crash selection or recovery image → `invalid` outcome before execution; capacity exhaustion → existing capacity classification. | fn-176-gomad-durable-volume-crash-model-in-the.2, fn-176-gomad-durable-volume-crash-model-in-the.3, fn-176-gomad-durable-volume-crash-model-in-the.4, fn-176-gomad-durable-volume-crash-model-in-the.5, fn-176-gomad-durable-volume-crash-model-in-the.6 | — |
| R2 | A differential harness runs identical operation and fault sequences through the Go and Rust volume models and requires identical replies, crash enumerations and recovery images. No error surface beyond the harness. | fn-176-gomad-durable-volume-crash-model-in-the.1, fn-176-gomad-durable-volume-crash-model-in-the.2, fn-176-gomad-durable-volume-crash-model-in-the.3, fn-176-gomad-durable-volume-crash-model-in-the.4 | — |
| R3 | Go consumers (replay, retained artifacts, recovered runs, qualification) decode the helper-produced storage evidence, and existing retained-artifact replay paths pass with newly produced evidence. Errors: undecodable or truncated storage evidence → classified refusal, never a silent pass. | fn-176-gomad-durable-volume-crash-model-in-the.5, fn-176-gomad-durable-volume-crash-model-in-the.7 | — |
| R4 | The Temporal committed-workflow crash/reopen gate and the SQLite durable qualification cells (the `qualify-workloads` success and canary cells) pass with the Rust volume model. No error surface beyond R1–R3. | fn-176-gomad-durable-volume-crash-model-in-the.6, fn-176-gomad-durable-volume-crash-model-in-the.7 | — |

