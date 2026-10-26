---
satisfies: [R2, R7]
---
# fn-255-thorough-framework-testing-generated.3 Nightly pin and cargo-fuzz targets for untrusted-input decoders

Touches: [model/umpire/fuzz/**, model/umpire/mise.toml, model/umpire/protocol/runtime/src/**, model/umpire/backends/tlc/src/**]

## Description
Fuzz every decoder that reads bytes the framework does not control (spec R2). This task also pins the nightly toolchain that .5 (Miri) reuses.

**Size:** M
**Files:** `model/umpire/fuzz/` (cargo-fuzz package, targets, seeded corpora), `model/umpire/mise.toml` (dated nightly, cargo-fuzz), decoder fixes if findings need them.
**Touches:** [model/umpire/fuzz/**, model/umpire/mise.toml, model/umpire/protocol/runtime/src/**, model/umpire/backends/tlc/src/**]

### Approach
- Pin a dated nightly and cargo-fuzz through mise. The nightly is used by fuzz, Miri and branch-coverage jobs (bounded smoke and scheduled), isolated from stable gate builds.
- Targets, run natively:
  - guest dispatch: arbitrary method id and bytes through `Dispatch::prepare` and `abi.rs::Guest::dispatch_to` with actual arbitrary request bytes, plus `runtime.rs::decode`;
  - `annotations.rs::AnnotationRuntime::prepare`;
  - TLC: `runner.rs::{messages, parse, decode_trace}` (with a fixed schema), `decode.rs::{decode_dump, decode_state}`.
  Expose the minimum needed for fuzzing under a `fuzzing` feature, not as public API.
- Oracle: `Ok` or a typed refusal; no panic, no hang beyond a per-input timeout, no allocation beyond a cap.
- Bounds (spec R7): timed libFuzzer runs set `-timeout`, `-rss_limit_mb`, `-max_len` and `-max_total_time`. Default corpus replay enumerates a sorted manifest and calls the shared decoder harness exactly once per file, without mutation or minimization; `-runs=N` is not an exactly-once replay contract. Enforce corpus count/byte and per-input work/time limits, and verify visited IDs/counts with a control. Keep minimization in the scheduled tier. See [libFuzzer options](https://llvm.org/docs/LibFuzzer.html#options).
- Seed corpora from existing goldens and recorded TLC output (`expected/tlc.txt.snap` files, guest test requests).
- Planted-panic control: a feature-gated panic in one decoder is found within the target's time budget.
- Planted-slow-input control: a bounded outer supervisor terminates a deliberately non-returning decoder input and fails with stage, target and corpus input ID. Exercise both timed fuzzing and stable replay supervision.
- Use the shared production bytes/dispatch path for native fuzzing; assert a valid seed reaches dispatch and successful response publication. Native `u32` pointer allocation refusal cannot count as dispatch-path coverage. Reuse `Guest::dispatch_to` for bytes/dispatch coverage; .5 adds the native-sized allocation/ownership seam and tests it. This task does not depend on .5.

### Investigation targets
**Required:**
- `model/umpire/protocol/runtime/src/abi.rs::Guest::dispatch_to`, `lib.rs::Dispatch::prepare`, `runtime.rs::decode`, `annotations.rs::AnnotationRuntime::prepare`
- `model/umpire/backends/tlc/src/runner.rs::{messages, parse, decode_trace}`, `decode.rs::{decode_dump, decode_state}`
- `model/umpire/mise.toml` — tool pins

## Acceptance
- [ ] One target per listed decoder, with a seeded corpus.
- [ ] A timed run per target (budget recorded) finds no panic, hang or unbounded allocation, or each finding is fixed and kept as a corpus regression.
- [ ] The planted-panic control is found; planted slow inputs fail both timed fuzzing and stable replay within the outer timeout, naming stage, target and input.
- [ ] A valid native dispatch seed reaches the production decoder/dispatcher and publishes a response; allocation refusal cannot satisfy that control.
- [ ] Per-input timeout, memory limit, input length and corpus-size cap set for every target; corpus replay runs each input exactly once.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
