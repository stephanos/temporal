# WASM helper local host-call fast path

## Conversation Evidence

> user (turn 1): "are there more levers for us that would bring those numbers for wasm execution down (ignore building wasm; pure execution)"
> user (turn 2): "yes, let's profile"
> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 16): "or fn-153"

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

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

- Execute request: the local model configuration carries argv, environment and the 32-byte entropy key, so the helper answers those imports locally.
- Transcript and digest format change for every profile (stock, cooperative, network-v2, broker): Go's environment and the helper write the same versioned binary record stream, and Go gains a strict decoder for it. Remote callback replies carry binary records in place of canonical JSON lines; the 64 KiB `remote_transcript` chunking and 8 MiB aggregate bound stay.
- Retained transcript capacity is counted in binary bytes, with fixed per-record worst-case reservations defined once per language from the shared record specification.

## Edge Cases & Constraints

- Guest memory ranges, iovec and buffer limits, and output limits keep their current bounds and error classifications.
- Digest-only and retained-transcript modes both keep working.
- Formats may change; byte-compatibility with earlier transcripts is not required, but replay of new evidence must be exact.
- fn-153 moves the Go readers of helper frames and the helper-model protocol to a shared strict JSON decoder. A binary transcript keeps the same strictness guarantees in its own decoder: trailing data, truncation and unknown records are refused with the existing refusal classes. It must not reintroduce hand-written scanners that fn-153 removes.

## Acceptance Criteria

- **R1:** Every import the helper supports is linked as a typed host function, with no per-call lookup by import name. Errors: an unknown import or mismatched signature is still refused before `started` with the existing `unsupported`/`invalid` classification.
- **R2:** The local model path constructs no JSON values, base64 payloads or JSON replies per call. The transcript is recorded and digested in a binary form that Go consumers can decode. Errors: malformed guest buffers keep their current errno and capacity outcomes; the binary decoder refuses truncation, trailing data, unknown record tags and oversized counts with the existing refusal classes.
- **R3:** Clock reads are recorded without one transcript entry per read, and the digest still binds every returned timestamp: consecutive reads with the same clock id and precision fold into one run record, timestamps follow from the identity-bound clock configuration, and a run closes before any other record, remote splice, interrupt or terminal flush. Errors: a replay whose clock behaviour differs (count, clock id, precision or model position) → evidence digest mismatch, as today; the frozen clock (zero read step) still records the read count.
- **R4:** `random_get`, `args_*` and `environ_*` are answered in the helper whenever it owns the model, with guest-visible bytes identical to the Go adapter's answers for the same request inputs. Errors: guest range errors keep the helper ABI's current errno; an oversize request or entropy offset overflow keeps the `entropy-bytes` capacity outcome; `args_get`/`environ_get` before their size call keep today's classification.
- **R5:** On the Temporal activity selection, the diagnostic profile's encoding plus transcript-hash nested time falls by ≥ 80%, and the runtime-cost fixture still produces identical evidence for the Go-owner and helper-owner model paths. No error surface beyond R1–R4.

## Boundaries

- Filesystem, volume, network and multi-instance work is out of scope (sibling specs). [paraphrase]
- `wasmtime-go` is out of scope. [paraphrase]
- No change to guest-side runtime hooks.
- No human-readable transcript renderer in the CLI; the Go decoder serves tests and inspection code. [planning decision]

## Decision Context

Part of the user's choice to "double down on Rust" and accept Go→Rust duplication for performance. A spike measured native Rust host calls at 6 ns (no-op) and 80 ns (with an 8-byte guest-memory write). The JSON layering on the local path therefore dominates host-call cost, not Wasmtime. [paraphrase]

### Planning decisions (2026-10-13, autonomous planning pass)

- **One transcript format for all profiles.** Go's environment emits the binary records too, so the Go-owner differential reference and the helper stay byte-identical and stock, network-v2 and broker runs need no second format. Byte-compatibility with earlier evidence is not required; old artifacts are refused by the changed identity, not by a spurious replay divergence.
- **Derived payloads are not repeated.** `random_get` records length and entropy offset; `args_*`/`environ_*` record the operation only. Their content follows from inputs already bound by the environment identity. When the helper owns the model, no remote path consumes entropy, so the entropy offset never crosses the protocol.
- **R3 means the digest binds every timestamp,** not that the transcript stores each one (the clock model is deterministic from `now`, epoch and read step).
- **Shared record specification, one encoder per language.** Each language has one deep encoder module that every producer calls (Go environment, Go remote-reply builder, helper local path), so they cannot drift. The Go codec file joins the WASI implementation identity source list.
- **Sequencing.** This spec depends on fn-153 because fn-153.20 rewrites the same Go helper-frame readers and identity list. The R5 measurement uses the `diagnostic-profile` feature, which fn-166 R8 now keeps (owner decision, 2026-10-13).
- **Rotation.** The transcript format change rotates the WASI implementation identity and the helper digest. It joins the fn-153/fn-154/fn-166 rotation candidate when timing allows, otherwise it follows it as one requalification together with fn-173.

Sibling specs: WASM helper per-run preparation fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process; Guest-side cost experiments and baselines.

## Early proof point

Task fn-174-gomad-wasm-helper-local-host-call-fast.2 validates the core approach: Go and the helper write one binary record stream and the Go-owner/helper-owner differential tests stay byte-identical. If the two owners cannot share one encoder specification without drift, re-evaluate whether Go keeps JSON lines and only the helper's digest input changes before continuing with fn-174-gomad-wasm-helper-local-host-call-fast.3+.

## Quick commands

```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad3/backends/wasm helper
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestHelperModel|TestHelperBridge|TestCaptured|TestEnvironment|TestTransport' ./backends/wasm/wasi)
```


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Every import the helper supports is linked as a typed host function, with no per-call lookup by import name. Errors: an unknown import or mismatched signature is still refused before `started` with the existing `unsupported`/`invalid` classification. | fn-174-gomad-wasm-helper-local-host-call-fast.1 | — |
| R2 | The local model path constructs no JSON values, base64 payloads or JSON replies per call. The transcript is recorded and digested in a binary form that Go consumers can decode. Errors: malformed guest buffers keep their current errno and capacity outcomes; the binary decoder refuses truncation, trailing data, unknown record tags and oversized counts with the existing refusal classes. | fn-174-gomad-wasm-helper-local-host-call-fast.2, fn-174-gomad-wasm-helper-local-host-call-fast.3 | — |
| R3 | Clock reads are recorded without one transcript entry per read, and the digest still binds every returned timestamp: consecutive reads with the same clock id and precision fold into one run record, timestamps follow from the identity-bound clock configuration, and a run closes before any other record, remote splice, interrupt or terminal flush. Errors: a replay whose clock behaviour differs (count, clock id, precision or model position) → evidence digest mismatch, as today; the frozen clock (zero read step) still records the read count. | fn-174-gomad-wasm-helper-local-host-call-fast.4 | — |
| R4 | `random_get`, `args_*` and `environ_*` are answered in the helper whenever it owns the model, with guest-visible bytes identical to the Go adapter's answers for the same request inputs. Errors: guest range errors keep the helper ABI's current errno; an oversize request or entropy offset overflow keeps the `entropy-bytes` capacity outcome; `args_get`/`environ_get` before their size call keep today's classification. | fn-174-gomad-wasm-helper-local-host-call-fast.5 | — |
| R5 | On the Temporal activity selection, the diagnostic profile's encoding plus transcript-hash nested time falls by ≥ 80%, and the runtime-cost fixture still produces identical evidence for the Go-owner and helper-owner model paths. No error surface beyond R1–R4. | fn-174-gomad-wasm-helper-local-host-call-fast.6 | — |
