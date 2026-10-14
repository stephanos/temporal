# WASM helper per-run preparation fast path

## Conversation Evidence

> user (turn 1): "are there more levers for us that would bring those numbers for wasm execution down (ignore building wasm; pure execution)"
> user (turn 4): "> 3. Per-run preparation (not execution, but most of the remaining wall time) why do that at all?"
> user (turn 6): "> Module hash why do we need this at all? what if we didn't hash at all?"
> user (turn 10): "actually no; we dont want wasmtime-go; let's double down on Rust where it makes sense. happy to duplicate work in Rust that we have in Go if it buys us performance"
> user (turn 12): "write a new flow spec for items 1-3"
> user (turn 13): "actually expand flow spec to the other recommended parts too"
> user (turn 14, selected): "split-as-proposed"
> user (turn 16): "or fn-153"

## Goal & Context

Current WASM performance acceptance follows the [owner's Linux/arm64 host decision](../../MILESTONES.md#wasm-performance-acceptance-host).

<!-- Goal & Context: 30% [paraphrase], 70% [inferred] (profiling evidence from this session) -->

Every warm WASM execution pays seconds of fixed preparation before the guest runs and, on macOS, tens of seconds of teardown after it finishes. Profiling the Temporal `./tests` module (338 MB) on darwin/arm64 with the diagnostic helper showed, per warm run:

- helper preparation ≈ 3.5 s, of which compiled-artifact envelope checksum ≈ 2.0 s, module SHA-256 ≈ 1.0 s, cache read 0.15–0.85 s, deserialize 0.3–0.9 s;
- after the terminal frame, the helper spent ≈ 50 s dropping the compiled module: Wasmtime deregisters native unwind info one function at a time through macOS libunwind (`__deregister_frame`). The Go adapter waits for helper exit, so whole-run wall was 51–74 s for selections whose authorized execution took 0.6–4.7 s;
- a `std::mem::forget` workaround on the success path (since committed) cut whole-run wall to 4.5–7 s;
- with `native_unwind_info(false)` (measured through the same Wasmtime release in a spike), compile dropped 2 min → 27 s, file-mapped deserialize 1.8 s → 0.11 s, module teardown ≈ 4 min → 20 ms; traps, fuel and epoch interruption kept working.

The goal is to make warm per-run preparation and teardown negligible relative to guest execution, without weakening evidence truthfulness. This is the first of seven sibling specs moving WASM hot paths into the Rust helper (see Decision Context).

## Architecture & Data Models

- **Engine configuration** gains disabled native unwind info. The configuration revision string that names the engine configuration rotates, so cache keys, the `started` frame, authorization and the Go-side engine identity all name the new revision.
- **Module identity on a cache hit:** the helper resolves the cache entry from the *claimed* module digest and never reads module bytes. On a cache hit, the module file does not influence execution, so a mismatched file cannot make evidence lie: what runs is the artifact compiled for the claimed digest.
- **Module identity on a cache miss:** the helper reads the module, verifies its digest against the claim, compiles, and publishes. This is the only per-run module hash.
- **Go adapter:** stops hashing the module on every run. Module identity is established once when the module is prepared.
- **Compiled-artifact cache:** the envelope moves to a new format version whose integrity check is a fast hash suited to corruption detection (the documented trust model: checksums detect corruption, not forgery by the trusted same-UID owner). Earlier-format entries are ignored, as legacy entries are today. The integrity check is CRC-32 through the hardware-accelerated `crc32fast` crate, already in the helper's lockfile as a transitive dependency; SHA-256 through `sha2` runs in software on arm64 (≈ 340 MB/s measured) and cannot meet the throughput bound without adding crates.

## API Contracts

- Helper protocol: the configuration revision value in `started`, `authorize` and the engine identity changes to a new revision; no other frame shape changes.
- Cache envelope: new magic/version; manifest keeps binding guest digest, helper executable digest, engine version, configuration revision, host OS/arch and engine compatibility hash, under a new magic, manifest version and filename suffix so old and new entries never collide.

## Edge Cases & Constraints

- Authorization naming the previous configuration revision is refused as an identity mismatch.
- A cache miss whose module bytes do not match the claimed digest fails before compilation and publishes nothing.
- A corrupt new-format entry is a host failure (as today); it is never silently recompiled around.
- Evidence produced before this change does not replay on the new helper; that is the existing changed-helper refusal, not a regression.
- Gomad formats may change and byte-compatibility with previous helpers is not required.
- fn-153 also edits the helper's compile cache: it moves the raw libc file operations to `rustix` and routes publication through shared helpers. The envelope change here builds on that cache code rather than diverging from it.
- Helper-identity and cache-format rotations should land in the fn-153/fn-154/fn-166 rotation batch, or immediately after it, so they share one WASM requalification instead of forcing another.

## Acceptance Criteria

- **R1:** The helper's engine runs with native unwind info disabled, and the configuration revision is rotated consistently across the helper, the cache identity and the Go engine identity. Errors: authorization or engine identity naming the old revision → identity mismatch, run refused before execution. [paraphrase]
- **R2:** After the terminal frame, the helper process exits within 1 s for the Temporal `./tests` module on linux/arm64 without relying on leaking the store/module; the success-path `mem::forget` workaround is removed if R1 alone meets this bound, kept with a comment otherwise. The bound also holds on at least one post-load error path (for example authorization mismatch). No error surface beyond R1.
- **R3:** On a compile-cache hit, the helper neither reads nor hashes the module file. On a miss, it verifies the module digest before compiling and publishing. Errors: digest mismatch on a miss → `invalid` outcome, nothing published; missing module file on a hit → no error (file unused); missing file on a miss → infrastructure outcome. [paraphrase]
- **R4:** The Go adapter performs no full-module hash per run; module identity comes from preparation-time records. Errors: a helper `invalid` module-digest mismatch on a miss is mapped by Go to the existing boundary error, as today's Go-side check reports it, never recorded as a target outcome; helper executable verification stays. [paraphrase]
- **R5:** The compiled-artifact envelope uses a new format version whose integrity check runs at ≥ 2 GB/s on linux/arm64; previous-format entries are ignored and preserved. Errors: truncated, oversized, mis-identified or checksum-failing new-format entries → host failure before `started`, as today. [paraphrase]
- **R6:** For the Temporal `./tests` workflow selection on linux/arm64 with a warm cache, helper `prepare_nanos` is ≤ 1 s (from ≈ 3.5 s), and same-seed runs before and after the change produce identical choice traces and transcript digests. The measurement is the median of at least 5 warm-page-cache runs, recorded with the artifact size. If cache read plus deserialize alone exceed the bound, the shortfall is recorded and the parked mapping decision goes to the owner instead of being implemented here. No error surface beyond R1–R5.

## Boundaries

- Execution-time (per host call) work is out of scope; it belongs to the sibling host-call and filesystem specs. [paraphrase]
- Embedding Wasmtime in Go (`wasmtime-go`) is rejected and out of scope. [paraphrase]
- No change to guest builds or the Go toolchain overlay.
- No cache eviction or pruning of earlier-format entries.
- No file-backed mapping of artifacts for deserialization (parked below).

## Decision Context

The user asked why per-run preparation is done at all and what would happen without the module hash. Analysis: on a cache hit the module bytes never reach execution, so verifying them protects nothing. On a miss, verification is required: compiling wrong bytes would publish code under the wrong identity and poison every later hit. The envelope checksum guards an `unsafe` deserialize against corruption only, so a fast non-cryptographic or fast cryptographic hash suffices. [paraphrase]

`native_unwind_info(false)` is preferred over leaking the module at exit: it also removes the registration cost at compile and load time. Wasmtime's own trap backtraces do not depend on it; native debuggers and profilers lose unwinding through guest frames (confirmed against the Wasmtime 47 `native_unwind_info` documentation).

### Planning decisions (2026-10-13, autonomous planning pass)

- **Trust model check.** On a hit, what runs was published only by a helper with the same executable digest after verifying the bytes on a miss. Retained replay materializes and verifies module bytes itself and uses a fresh private cache, so it is always a verified miss. Plan-time preparation verification still hashes once per plan. Skipping the hash on a hit therefore removes no evidence guarantee.
- **CRC-32 via `crc32fast`** as a direct dependency adds a lockfile edge but no new package. Vendoring the `sha2` aarch64 compression function was rejected: ≈ 2.2 GB/s is marginal on Linux arm64 for the throughput bound.
- **Remove `mem::forget`** once native unwind info is off and the success path and one post-load error path both exit within 1 s.
- **Go keeps verifying the helper executable** (≈ 7 ms with Go's hardware SHA-256); only the per-run module hash goes. The helper's own per-run self-hash for the cache key (≈ 40 ms) stays out of scope.
- **Rotation timing.** No spec-level dependency. The revision and cache-format rotations land inside the fn-153/fn-154/fn-166 rotation candidate or immediately after it (MILESTONES "joins or follows"); tasks carry that gate. fn-172 task 1 also centralizes the revision constant, so whichever lands second rebases.
- **Spec correction.** The `mem::forget` workaround and `wasi/zz_scratch_profile_test.go` are committed, not uncommitted as the original evidence said.

Sibling specs from the same session: WASM helper local host-call fast path; Filesystem namespace in the Rust helper; Durable volume crash model in the Rust helper; Network models in the Rust helper; Multi-instance clusters in one helper process; Guest-side cost experiments and baselines.

## Parked unknowns

- Whether to also map the artifact file for deserialization (≈ 10× faster load than reading into a private buffer) reverses a documented decision against mutable file-backed executable mappings; resolve by a design review of the mapping's mutation risk under the cache's ownership/permission rules.
- Current Linux/arm64 gain is unmeasured: Linux registers unwind info differently. Resolve through the required R6 measurement on the acceptance host.

## Early proof point

Task fn-173-gomad-wasm-helper-per-run-preparation.1 validates the core approach (disabled native unwind info removes the macOS teardown and cuts deserialize without changing evidence). If it fails, re-evaluate keeping the `mem::forget` workaround on every exit path before continuing with fn-173-gomad-wasm-helper-per-run-preparation.2+.

## Quick commands

```bash
cargo +1.94.1 test --manifest-path tools/gomad3/backends/wasm/wasmhost/Cargo.toml --release --locked
make -C tools/gomad3/backends/wasm helper
(cd tools/gomad3 && env -u GOROOT GOENV=off GOTOOLCHAIN=local GOWORK=off go test -tags test_dep -count=1 -run 'TestGuestTransport|TestTransportRejectsChangedFiles' ./backends/wasm/wasi)
```


## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | The helper's engine runs with native unwind info disabled, and the configuration revision is rotated consistently across the helper, the cache identity and the Go engine identity. Errors: authorization or engine identity naming the old revision → identity mismatch, run refused before execution. | fn-173-gomad-wasm-helper-per-run-preparation.1 | — |
| R2 | After the terminal frame, the helper process exits within 1 s for the Temporal `./tests` module on linux/arm64 without relying on leaking the store/module; the success-path `mem::forget` workaround is removed if R1 alone meets this bound, kept with a comment otherwise. The bound also holds on at least one post-load error path (for example authorization mismatch). No error surface beyond R1. | fn-173-gomad-wasm-helper-per-run-preparation.1, fn-173-gomad-wasm-helper-per-run-preparation.5 | — |
| R3 | On a compile-cache hit, the helper neither reads nor hashes the module file. On a miss, it verifies the module digest before compiling and publishing. Errors: digest mismatch on a miss → `invalid` outcome, nothing published; missing module file on a hit → no error (file unused); missing file on a miss → infrastructure outcome. | fn-173-gomad-wasm-helper-per-run-preparation.2 | — |
| R4 | The Go adapter performs no full-module hash per run; module identity comes from preparation-time records. Errors: a helper `invalid` module-digest mismatch on a miss is mapped by Go to the existing boundary error, as today's Go-side check reports it, never recorded as a target outcome; helper executable verification stays. | fn-173-gomad-wasm-helper-per-run-preparation.3 | — |
| R5 | The compiled-artifact envelope uses a new format version whose integrity check runs at ≥ 2 GB/s on linux/arm64; previous-format entries are ignored and preserved. Errors: truncated, oversized, mis-identified or checksum-failing new-format entries → host failure before `started`, as today. | fn-173-gomad-wasm-helper-per-run-preparation.4 | — |
| R6 | For the Temporal `./tests` workflow selection on linux/arm64 with a warm cache, helper `prepare_nanos` is ≤ 1 s (from ≈ 3.5 s), and same-seed runs before and after the change produce identical choice traces and transcript digests. The measurement is the median of at least 5 warm-page-cache runs, recorded with the artifact size. If cache read plus deserialize alone exceed the bound, the shortfall is recorded and the parked mapping decision goes to the owner instead of being implemented here. No error surface beyond R1–R5. | fn-173-gomad-wasm-helper-per-run-preparation.5 | — |
