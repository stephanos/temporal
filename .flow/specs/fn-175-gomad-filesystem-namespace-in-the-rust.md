# Filesystem namespace in the Rust helper

## Conversation Evidence

> user (turn 3): "> Move the virtual filesystem's write path into the helper, does that mean from Go to Rust?"
> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 11): "how far sould we go with the rust code? can you do a breakdown of what's in Go and whats in Rust"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

<!-- Goal & Context: 30% [paraphrase], 70% [inferred] (profiling evidence) -->

The in-memory WASI filesystem (fd table, rights, files and directories, all `fd_*` and `path_*` operations) lives in the Go adapter. Every filesystem call is therefore a round trip from the helper to Go. On the Temporal activity selection, ≈ 7.5k remote callbacks (6.7k `fd_pwrite`, ≈ 390 `fd_pread`, from SQLite) averaged ≈ 95 µs each. That is ≈ 0.7 s, about 30% of authorized execution, with ≈ 75% of each call spent waiting on Go's reply.

The goal is to answer every plain-namespace filesystem operation inside the helper, so that runs without a durable storage model make no filesystem callbacks to Go.

## Architecture & Data Models

- The helper owns the namespace for runs without a storage model: the descriptor table with rights and flags, preopens, regular files and directories, read-only captured inputs, writable directories, offsets and append mode, and stdout/stderr routing with output limits.
- Captured read-only inputs and writable-directory configuration arrive in the execute request.
- `poll_oneoff` subscriptions on file descriptors are answered locally.
- The Go namespace stays as the reference implementation. It also remains the fallback while runs that still use the remote model (storage, network v2, broker) exist.
- A differential conformance harness drives identical operation sequences through both implementations.

- Ownership gate: the helper owns the namespace exactly when it already owns the cooperative model (cooperative profile, no network v2, no broker, not the test-only Go-owner reference) and no storage model is configured. Stock-profile, storage, broker and network-v2 runs keep the Go namespace until their own specs move them. When the network spec enables the local model for network-v2 runs, those runs inherit the gate.
- Go keeps captured-input decoding and namespace validation. It builds its namespace as today, then exports a flattened namespace image (preopens in order, directories and read-only files in inode order, writable directories, working directory, stdin, limits). The helper re-validates the image's structure and limits and builds its own namespace from it. In gated runs, the Go namespace serves no calls.
- The conformance driver is a test-only helper build behind a non-default Cargo feature in a separate target directory. It speaks a model-agnostic record protocol, so the volume (fn-176) and World/clock (fn-178) ports reuse it.

## API Contracts

- Execute request: gains a namespace image descriptor (chunk count, bytes, files, SHA-256). Exactly that many binary image frames follow authorization and the tape chunks, each bounded by the frame limit. The image format is versioned and decoded strictly: truncation, trailing data, unknown record kinds and digest mismatch are refused.
- Results: new `stdout` and `stderr` evidence streams replace Go's accumulation from `fd_write` callbacks. The stdout stream is flushed at 64 KiB, before any remote callback and at termination, and feeds the live stdout mirror.
- Evidence: local filesystem calls record transcript entries through the local host-call fast path's recorder (fn-174), and keep semantic parity with the Go namespace's entries. No final namespace state is retained as evidence; the namespace digest is a test-only harness artifact.
- Test-visible result field: a per-operation count of filesystem calls the Go namespace served. Gated runs must report none.

## Edge Cases & Constraints

- Errno parity with the Go namespace for: bad descriptor, missing rights, directory vs file, read-only, nonexistent path, path traversal outside preopens, offset overflow, write-buffer and filesystem-byte capacity, file and descriptor count limits.
- Output-byte accounting for fd 1/2, including positional writes. The Go namespace is the reference: offered bytes are charged before the rights check, so rejected writes still count. The terminal output check compares against the helper-reported offered output in gated runs.
- Admission checks run in Go's order before any mutation: call limit, frame and reply reservations, transcript capacity. Capacity dimensions keep their names.
- Unsupported operations and flags terminate as `unsupported` with Go's message, and the transcript entry is retained, as on the Go path.
- Ordered containers wherever iteration order reaches replies, digests or evidence. No host clock: file times come from the model's virtual clock.
- Byte-compatibility with earlier evidence is not required.
- Watchdog kill or helper crash: retained stdout/stderr are the chunks received so far.

## Acceptance Criteria

- **R1:** For runs without a storage model, every `fd_*` and `path_*` operation and fd-subscription `poll_oneoff` is answered in the helper, and the Temporal activity selection completes with zero filesystem callbacks to Go. Errors: each operation's error cases return the same errno as the Go namespace (see R2). [paraphrase]
- **R2:** A differential harness runs generated and recorded operation sequences, including every error case listed under Edge Cases, through the Go and Rust namespaces and requires identical replies, errnos and final namespace digests. No error surface beyond the harness itself.
- **R3:** Captured read-only inputs and writable directories are configured from the execute request, and writes into read-only inputs fail with the read-only errno. Errors: invalid or oversized namespace configuration → `invalid` outcome before execution.
- **R4:** Authorized execution of the Temporal activity selection on linux/arm64 drops by ≥ 25% against the pre-change baseline at the same seed, with identical choice traces. No error surface beyond R1–R3.

## Early proof point

Task fn-175-gomad-filesystem-namespace-in-the-rust.1 validates the core approach: a model-agnostic Go-vs-Rust conformance driver with a namespace digest that pins every errno case. If it fails, re-evaluate the harness shape (driver binary versus Go-generated golden vectors only) before continuing with fn-175.2 and later tasks. fn-176 and fn-178 reuse the same harness.

## Quick commands

```bash
# Rust namespace unit and corpus tests
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked namespace
# Go-vs-Rust differential (builds the conformance helper in its own target dir)
make -C tools/gomad3/backends/wasm test-conformance
# Go namespace reference tests
make -C tools/gomad3/backends/wasm test-model
```

## Boundaries

- The durable volume crash model (storage-configured runs) is out of scope here; it has its own spec that builds on this one. [paraphrase]
- Network and multi-instance runs are out of scope. [paraphrase]
- Reducing SQLite's write count from the test side (journal or sync pragmas) is not pursued here. [inferred]
- Stock-profile runs keep the Go namespace. They have no helper-owned transcript, and the exploratory profile makes no performance claim.
- Porting captured-manifest decoding to Rust is out of scope: Go decodes and flattens.
- Retiring the Go namespace is out of scope. It stays as the reference implementation and as the fallback for non-gated runs.

## Decision Context

The user asked whether moving the filesystem write path means going from Go to Rust. The alternatives considered were a binary protocol, write-behind batching, shared-memory rings and test-side pragma changes. The user then chose to "double down on Rust" and accepted duplicating Go logic in Rust when it buys performance. A native Rust host call costs 6–80 ns, against ≈ 95 µs per round trip today. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; WASM helper local host-call fast path; Durable volume crash model in the Rust helper (depends on this spec); Network models in the Rust helper; Multi-instance clusters in one helper process; Guest-side cost experiments and baselines.

Planning decisions (2026-10-13, made without the user and open to override):
- This spec depends on fn-174. The switch-over task binds filesystem imports as fn-174's typed host functions and records through its binary transcript, instead of building a second JSON path that fn-174 would then remove. The pure Rust namespace, the harness and the configuration transport have no fn-174 dependency, but the spec-level edge serializes them too. That is accepted, because model.rs, abi.rs and protocol.rs are shared and would otherwise conflict.
- Go-side image building and validation are retained (rejected alternative: porting `captured` decoding to Rust, as extra duplication with no performance benefit, since it runs once per run).
- R1 is proven with a Go-side per-operation served counter, not with diagnostic helper counts, so the gate holds on release helpers too.
- R4 baseline: remeasure the pre-change helper at this spec's base commit on the same real Linux/arm64 host, unless fn-172 has retained a matching same-seed activity baseline. Historical acceptance numbers and the unretained session profile do not establish this baseline.
- Rejected: a namespace digest as runtime evidence. It is overkill; parity is shown by the harness and by Go-owner-versus-helper-owner evidence equality.
- Paths in task files are current as of planning. fn-166 (.6/.7/.11) may move `tools/gomad3` to `tools/gomad`, so implementers map the paths.


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | For runs without a storage model, every `fd_*` and `path_*` operation and fd-subscription `poll_oneoff` is answered in the helper, and the Temporal activity selection completes with zero filesystem callbacks to Go. Errors: each operation's error cases return the same errno as the Go namespace (see R2). | fn-175-gomad-filesystem-namespace-in-the-rust.2, fn-175-gomad-filesystem-namespace-in-the-rust.3, fn-175-gomad-filesystem-namespace-in-the-rust.5, fn-175-gomad-filesystem-namespace-in-the-rust.6 | — |
| R2 | A differential harness runs generated and recorded operation sequences, including every error case listed under Edge Cases, through the Go and Rust namespaces and requires identical replies, errnos and final namespace digests. No error surface beyond the harness itself. | fn-175-gomad-filesystem-namespace-in-the-rust.1, fn-175-gomad-filesystem-namespace-in-the-rust.2, fn-175-gomad-filesystem-namespace-in-the-rust.3 | — |
| R3 | Captured read-only inputs and writable directories are configured from the execute request, and writes into read-only inputs fail with the read-only errno. Errors: invalid or oversized namespace configuration → `invalid` outcome before execution. | fn-175-gomad-filesystem-namespace-in-the-rust.4 | — |
| R4 | Authorized execution of the Temporal activity selection on linux/arm64 drops by ≥ 25% against the pre-change baseline at the same seed, with identical choice traces. No error surface beyond R1–R3. | fn-175-gomad-filesystem-namespace-in-the-rust.6 | — |

