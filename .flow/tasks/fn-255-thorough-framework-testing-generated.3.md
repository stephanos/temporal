---
satisfies: [R2, R7]
---
# fn-255-thorough-framework-testing-generated.3 Nightly pin and cargo-fuzz targets for untrusted-input decoders

Touches: [model/umpire/Cargo.toml, model/umpire/Cargo.lock, model/umpire/dev/decoder-testing/**, model/umpire/fuzz/**, model/umpire/mise.toml, model/umpire/mise.lock, model/umpire/protocol/runtime/Cargo.toml, model/umpire/protocol/runtime/src/**, model/umpire/backends/tlc/Cargo.toml, model/umpire/backends/tlc/src/**, model/umpire/dev/xtask/src/main.rs, model/umpire/dev/xtask/src/lib.rs, model/umpire/dev/xtask/src/gate.rs, model/umpire/dev/xtask/src/gate/**, model/umpire/dev/xtask/src/quick/**, model/umpire/dev/xtask/src/test_run.rs, model/umpire/dev/xtask/tests/**, model/umpire/README.md, model/umpire/dev/xtask/README.md]

## Description
Fuzz every decoder that reads bytes the framework does not control (spec R2). Add a stable workspace-owned decoder harness and deterministic replay path, with thin isolated nightly cargo-fuzz wrappers. This task pins the dated nightly and components reused by .5 (Miri) and .6 (branch coverage).

**Size:** M
**Files:** `dev/decoder-testing/` (shared stable harness, replay binary and tests), root workspace manifest/lockfile, `fuzz/` (nightly-only wrappers, corpora, manifest and independent lockfile), local mise pins, minimal decoder feature seams, explicit command/ownership inventory and focused controls, framework documentation.
**Dependency:** canonical fn-254-fast-validation-gates-and-a-real-inner.12. Source preparation may precede its committed closure; real execution and task closure require its single committed R7 budget authority. The conductor records this cross-spec gate with Flow blocked status until the committed closure and admitted bounds are verified. This task does not depend on .5.

### Approach
- Make `model/umpire/dev/decoder-testing` a member of the root framework Cargo workspace with a library, stable deterministic replay binary and tests. Put shared decoder calls, target identities and corpus admission here. Both replay and cargo-fuzz call the same production-byte harness. No libFuzzer dependency, optional or otherwise, enters the stable workspace dependency graph.
- Keep `model/umpire/fuzz` an isolated cargo-fuzz package with its own `[workspace]` and committed `Cargo.lock`. Its nine targets are thin nightly-only wrappers depending on the stable harness. Commit corpus seeds and a sorted manifest with input IDs, hashes and source provenance. The two lockfiles must independently support locked builds without contaminating stable toolchain selection.
- Pin a dated nightly and cargo-fuzz through the framework-owned mise configuration. Install/select that nightly with `miri`, `rust-src` and `llvm-tools` components for .5/.6, isolated from the stable production/gate Rust pin. Resolve shared pin and decoder files serially through the integration owner; preserve the frozen .16 provider branch. No execution path implicitly installs tools or uses parent configuration.
- Provide one target per decoder (nine targets), run natively:
  - `dispatch_prepare`: arbitrary method ID and request bytes through `Dispatch::prepare`;
  - `guest_dispatch`: `abi.rs::Guest::dispatch_to` with actual arbitrary request bytes;
  - `runtime_decode`: `runtime.rs::decode` for production request types;
  - `annotations`: `annotations.rs::AnnotationRuntime::prepare`;
  - `tlc_messages`, `tlc_parse`, `tlc_trace`: `runner.rs::{messages, parse, decode_trace}` with a fixed valid schema for traces;
  - `tlc_dump`, `tlc_state`: `decode.rs::{decode_dump, decode_state}` with fixed valid schemas and independently bounded dump counts.
  Expose only the minimum production forwarding seams behind `fuzzing` features in runtime/TLC manifests. Keep ordinary production APIs unchanged. Do not duplicate decoder implementations or catch panics as typed refusals.
- Oracle: a valid value or a typed refusal; a panic, hang or allocation-cap breach fails. Assert that a valid native seed reaches real production decoding/dispatch and successful response publication. A native `u32` pointer allocation refusal cannot satisfy this control; reuse `Guest::dispatch_to`. The native-sized allocation/ownership seam belongs to .5.
- Seed all targets from existing guest request controls, decoder test literals, goldens and retained recorded TLC output, with exact provenance. Normalized `expected/tlc.txt.snap` summaries are only unframed/refusal inputs; they do not establish valid framed TLC-output coverage. Include positive framed/typed seeds as well as refusals.
- Consume the committed fn-254.12 table for corpus count/bytes, maximum input length and per-input work/time/memory limits, target run durations and aggregate stage caps. Do not create a second authoritative constants table. Timed libFuzzer invocations set `-timeout`, `-rss_limit_mb`, `-max_len` and `-max_total_time`; a bounded outer supervisor also owns cancellation, descendants and the whole stage. Reconcile worker spawning with the current supervision policy; use one process where possible. `Limits.heap_mb` alone is not a generic RSS limit.
- Stable replay validates sorted unique manifest IDs, exact corpus membership, regular-file admission, digests, count/byte/input-size limits and target selection before execution. Call the shared harness exactly once per admitted input, without mutation/minimization. Verify visited IDs/order/counts; `-runs=N` is not an exactly-once replay contract. Reject duplicate, missing, unlisted, digest-mismatched and over-cap inputs with stage/target/input identity where applicable. Keep minimization in the scheduled tier.
- Use only framework-root `target/` or `build/`, or an admitted external output directory, for scratch, build and fuzz artifacts. Never create nested `fuzz/target` outputs that enter the framework source inventory/standalone copy. Preserve source corpora during timed campaigns; mutations use bounded owned scratch.
- A feature-gated planted panic in a real decoder must be found within its target budget. A deliberately non-returning decoder input must fail both timed fuzzing and stable replay through outer supervision, naming stage, target and corpus input ID. Run these controls separately from positive campaigns and verify cleanup/cancellation.
- Register the stable replay and timed fuzz commands explicitly in the framework entry-point command inventory and test ownership. The stable harness has real root-workspace/nextest ownership; thin fuzz wrappers stay outside that inventory. Add deliberate quick-source ownership/exclusion controls so these new sources do not trigger an unmapped whole-workspace fallback or introduce fuzz/replay/Miri work into `model-quick`. Preserve existing package-owned tests. Final check/validate tier activation and measured gate integration remain .7, after admitted bounds.
- Keep all harnesses framework-only and validate standalone containment. Coordinate shared root manifest/lockfile, decoder manifests, pins and xtask files serially with other fn-255 lanes; do not transplant stale full files from retained candidates.

### Investigation targets
**Required:**
- `protocol/runtime/src/abi.rs::Guest::dispatch_to`, `lib.rs::Dispatch::prepare`, `runtime.rs::decode`, `annotations.rs::AnnotationRuntime::prepare`
- `backends/tlc/src/runner.rs::{messages, parse, decode_trace}`, `decode.rs::{decode_dump, decode_state}`
- `Cargo.toml`, `Cargo.lock`, `mise.toml`, `mise.lock`; current xtask command inventory, nextest inventory, quick ownership and standalone source inventory
- fn-254.12 committed budget table and current supervision/ownership contracts; frozen fn-254.16 interface without modifying that branch

## Acceptance
- [ ] The stable shared decoder library/replay binary/tests are owned by root workspace `dev/decoder-testing`; all nine thin cargo-fuzz targets exist with seeded corpora and an independent committed fuzz lockfile. The stable dependency graph contains no libFuzzer dependency.
- [ ] A dated nightly with `miri`, `rust-src`, `llvm-tools` and cargo-fuzz is pinned through framework-local mise, isolated from stable gate builds and usable by .5/.6.
- [ ] A timed run per target, with recorded authoritative budget and aggregate cap, finds no panic, hang or unbounded allocation, or each finding is fixed and retained as a corpus regression.
- [ ] The planted panic is found; planted slow inputs fail timed fuzzing and stable replay within outer timeouts, naming stage, target and input; owned processes/scratch are cleaned up.
- [ ] A valid native seed reaches the production decoder/dispatcher and publishes a successful response; allocation refusal cannot satisfy this control.
- [ ] Per-input timeout, memory/input limits and corpus count/byte bounds come from committed fn-254.12 authority. Stable replay visits each admitted input exactly once; inventory, digest, membership and over-cap rejection controls pass.
- [ ] Focused command/nextest ownership, quick-exclusion and standalone controls pass. Artifacts remain outside the source inventory; no fn-255 execution enters `model-quick`. .7 tier activation remains separate.
- [ ] Evidence records selected inputs/targets, tool versions, exact source, controls and limits; source preparation alone earns no execution or task-closure credit.

## Done summary
Blocked:
Execution and task closure wait for the committed fn-254-fast-validation-gates-and-a-real-inner.12 closure in umpire and admission of this task's work/time limits to its single authoritative budget table. Disjoint source preparation does not satisfy this gate. Flow task dependencies only accept same-spec IDs; after verifying that closure and admitted entries, the conductor may reset this block and admit the task normally.
## Evidence
- Commits:
- Tests:
- PRs:
