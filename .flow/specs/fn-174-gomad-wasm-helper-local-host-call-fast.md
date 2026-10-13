# WASM helper local host-call fast path

## Conversation Evidence

> user (turn 1): "are there more levers for us that would bring those numbers for wasm execution down (ignore building wasm; pure execution)"
> user (turn 2): "yes, let's profile"
> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 16): "or fn-153"

## Goal & Context

<!-- Goal & Context: 25% [paraphrase], 75% [inferred] (profiling evidence) -->

fn156 moved scheduling, clock and poll decisions into the helper, but the local path still encodes every callback as JSON. Diagnostic profiling of the Temporal activity selection (≈ 210k host calls, 163k of them `clock_time_get`) showed, per run: encoding ≈ 90 ms over ≈ 437k spans, transcript hashing ≈ 145 ms over ≈ 333k spans, and clock calls ≈ 150 ms. Every callback builds a JSON value, re-serializes it, formats call/reply transcript lines, base64-encodes runtime payloads it then decodes, and re-parses its own output. Imports are linked dynamically, with a string match on the import name for every call. `random_get`, `args_*` and `environ_*` still go to the Go process.

The goal is to make local host calls cost close to the raw ABI work, so the guest dominates. The user prefers performance in Rust over a single implementation.

## Architecture & Data Models

- Imports are bound as typed host functions per known WASI and cooperative import, replacing dynamic dispatch on name strings.
- The local model exchanges typed values with the ABI layer; no JSON value, base64 payload or JSON reply is constructed on the local path.
- The WASI transcript is recorded in a compact binary form and digested over that form. Rendering to a human-readable form, if needed, happens only at flush or in consumers.
- Clock reads are recorded compactly (e.g. as counted steps against the model clock) rather than one call/reply pair per read, while replay still reproduces every returned timestamp.
- `random_get`, `args_sizes_get`/`args_get` and `environ_sizes_get`/`environ_get` are answered in the helper from execution-request inputs (entropy key, argv, environment), producing the same bytes the Go adapter produces today.

## API Contracts

- Execute request: carries what the helper needs to answer argv, environment and entropy locally. [inferred]
- Transcript and digest format change; Go consumers that read transcripts gain a decoder for the new form. [inferred]

## Edge Cases & Constraints

- Guest memory ranges, iovec and buffer limits, and output limits keep their current bounds and error classifications.
- Digest-only and retained-transcript modes both keep working.
- Formats may change; byte-compatibility with earlier transcripts is not required, but replay of new evidence must be exact.
- fn-153 moves the Go readers of helper frames and the helper-model protocol to a shared strict JSON decoder. A binary transcript keeps the same strictness guarantees in its own decoder: trailing data, truncation and unknown records are refused with the existing refusal classes. It must not reintroduce hand-written scanners that fn-153 removes.

## Acceptance Criteria

- **R1:** Every import the helper supports is linked as a typed host function, with no per-call lookup by import name. Errors: an unknown import is still refused at link time with the existing classification. [inferred]
- **R2:** The local model path constructs no JSON values, base64 payloads or JSON replies per call. The transcript is recorded and digested in a binary form that Go consumers can decode. Errors: malformed guest buffers keep their current errno and capacity outcomes. [inferred]
- **R3:** Clock reads are recorded without one transcript entry per read, and replaying a recorded run reproduces every returned timestamp. Errors: replay against a tape whose clock record disagrees → divergence outcome, as today. [inferred]
- **R4:** `random_get`, `args_*` and `environ_*` are answered in the helper, with bytes identical to the Go adapter's answers for the same request inputs. Errors: buffer-capacity and range errors match the Go adapter's errno. [inferred]
- **R5:** On the Temporal activity selection, the diagnostic profile's encoding plus transcript-hash nested time falls by ≥ 80%, and the runtime-cost fixture still produces identical evidence for the Go-owner and helper-owner model paths. No error surface beyond R1–R4. [inferred]

## Boundaries

- Filesystem, volume, network and multi-instance work is out of scope (sibling specs). [paraphrase]
- `wasmtime-go` is out of scope. [paraphrase]
- No change to guest-side runtime hooks. [inferred]

## Decision Context

Part of the user's choice to "double down on Rust" and accept Go→Rust duplication for performance. A spike measured native Rust host calls at 6 ns (no-op) and 80 ns (with an 8-byte guest-memory write). The JSON layering on the local path therefore dominates host-call cost, not Wasmtime. [paraphrase]

Sibling specs: WASM helper per-run preparation fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process; Guest-side cost experiments and baselines.
