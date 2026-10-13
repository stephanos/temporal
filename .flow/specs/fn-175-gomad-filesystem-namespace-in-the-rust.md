# Filesystem namespace in the Rust helper

## Conversation Evidence

> user (turn 3): "> Move the virtual filesystem's write path into the helper, does that mean from Go to Rust?"
> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"

## Goal & Context

<!-- Goal & Context: 30% [paraphrase], 70% [inferred] (profiling evidence) -->

The in-memory WASI filesystem (fd table, rights, files and directories, all `fd_*` and `path_*` operations) lives in the Go adapter. Every filesystem call is therefore a round trip from the helper to Go. On the Temporal activity selection, ≈ 7.5k remote callbacks (6.7k `fd_pwrite`, ≈ 390 `fd_pread`, from SQLite) averaged ≈ 95 µs each. That is ≈ 0.7 s, about 30% of authorized execution, with ≈ 75% of each call spent waiting on Go's reply.

The goal is to answer every plain-namespace filesystem operation inside the helper, so that runs without a durable storage model make no filesystem callbacks to Go.

## Architecture & Data Models

- The helper owns the namespace for runs without a storage model: the descriptor table with rights and flags, preopens, regular files and directories, read-only captured inputs, writable directories, offsets and append mode, and stdout/stderr routing with output limits.
- Captured read-only inputs and writable-directory configuration arrive in the execute request.
- `poll_oneoff` subscriptions on file descriptors are answered locally.
- The Go namespace stays as the reference implementation. It also remains the fallback while runs that still use the remote model (storage, network v2, broker) exist.
- A differential conformance harness drives identical operation sequences through both implementations.

## API Contracts

- Execute request: gains the namespace configuration (captured inputs, writable directories, limits) needed to build the namespace in the helper. [inferred]
- Evidence: the filesystem portion of results (transcript entries, final state if retained) keeps semantic parity with the Go namespace. [inferred]

## Edge Cases & Constraints

- Errno parity with the Go namespace for: bad descriptor, missing rights, directory vs file, read-only, nonexistent path, path traversal outside preopens, offset overflow, write-buffer and filesystem-byte capacity, file and descriptor count limits.
- Output-byte accounting for fd 1/2, including positional writes.
- Byte-compatibility with earlier evidence is not required.

## Acceptance Criteria

- **R1:** For runs without a storage model, every `fd_*` and `path_*` operation and fd-subscription `poll_oneoff` is answered in the helper, and the Temporal activity selection completes with zero filesystem callbacks to Go. Errors: each operation's error cases return the same errno as the Go namespace (see R2). [paraphrase]
- **R2:** A differential harness runs generated and recorded operation sequences, including every error case listed under Edge Cases, through the Go and Rust namespaces and requires identical replies, errnos and final namespace digests. No error surface beyond the harness itself. [inferred]
- **R3:** Captured read-only inputs and writable directories are configured from the execute request, and writes into read-only inputs fail with the read-only errno. Errors: invalid or oversized namespace configuration → `invalid` outcome before execution. [inferred]
- **R4:** Authorized execution of the Temporal activity selection on darwin/arm64 drops by ≥ 25% against the pre-change baseline at the same seed, with identical choice traces. No error surface beyond R1–R3. [inferred]

## Boundaries

- The durable volume crash model (storage-configured runs) is out of scope here; it has its own spec that builds on this one. [paraphrase]
- Network and multi-instance runs are out of scope. [paraphrase]
- Reducing SQLite's write count from the test side (journal or sync pragmas) is not pursued here. [inferred]

## Decision Context

The user asked whether moving the filesystem write path means going from Go to Rust. The alternatives considered were a binary protocol, write-behind batching, shared-memory rings and test-side pragma changes. The user then chose to "double down on Rust" and accepted duplicating Go logic in Rust when it buys performance. A native Rust host call costs 6–80 ns, against ≈ 95 µs per round trip today. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Durable volume crash model in the Rust helper (depends on this spec); Network models in the Rust helper; Multi-instance clusters in one helper process; Guest-side cost experiments and baselines.
