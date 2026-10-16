# Gomad: WASM host-side execution cost profile

## Conversation Evidence

> user: "File a host-import profiling spec for performance before committing to any optimization."

## Goal & Context

Attribute authorized execution time on the current WASM candidate to host-side cost centers precisely enough to decide whether another execution-speed lever exists, and record that decision before any optimization spec is opened. [paraphrase]

fn-178 measured guest share `G/T` at 0.148 to 0.192 on the Temporal two-cluster partition/heal/crash scenario: more than four fifths of authorized execution is spent in timed host-import spans, now served in-process by the Rust helper. fn-174 removed the encoding and transcript-hash cost (93 percent reduction), and fn-178 ruled out parallel instances (best-case bound 1.106x) and fibers (0.05 to 0.09 percent handoff share). What remains unmeasured is how the host-import time divides between the model owners that replaced the Go broker: namespace, volume, network, clock, World, storage transport, cache and scheduler. The current diagnostic profile cannot answer this: `AbiSpan` folds every in-process import into six coarse kinds (clock, scheduler, select, observation, idle, other-local), and the retained fn-172.4 breakdown predates the in-process migration, where `nested.remote_read_wait` was 90 percent of execute.

## Architecture & Data Models

- Extend the diagnostic-profile helper (`--features diagnostic-profile`, never qualification evidence) so each host import records its exact `module.op` and the model owner that served it, with count, total wall, exclusive wall (minus nested spans) and a bounded per-call distribution (min, median, p99, max). Add a reconciliation row: attributed host wall plus guest residual must equal execute wall within a stated tolerance, otherwise the profile is incomplete.
- Owner attribution happens where the helper dispatches an import to a model (`abi.rs` dispatch, `namespace_call.rs`, `storage_transport.rs`, `network`, `clock.rs`, `world`, `cache.rs`, cluster handoff). No change to release-helper code paths beyond the already compiled-out diagnostic feature.
- Rust-side CPU sampling of the helper with an external profiler (`perf record` on Linux/arm64) on the same runs, folded to symbols and mapped to the same owners; retained as flamegraph data next to the profile JSON. No new Cargo dependency.
- Workloads: the fn-178 cluster scenario (`TestMatchedReleaseClusterCandidate` path, seeds 7, 19, 42) and the single-guest workflow and activity selections used by fn-172/fn-174 (seeds 0, 1000003, 2000006), each one warmup plus three warm repetitions, on a quiet real Linux/arm64 host, following the owner performance-acceptance-host decision.
- Output: one retained artifact under `.flow/artifacts/` with per-owner and per-op tables, reconciliation, identities, raw JSON and exact rerun commands, plus a ranked decision table.

## API Contracts

- Profile JSON gains `abi_ops` (keyed by `module.op`) and `owners` (keyed by owner name) with `count`, `wall_nanos`, `exclusive_wall_nanos`, `execute_wall_nanos` and the per-call distribution; existing buckets (`phases.*`, `abi.*`, `nested.*`) keep their names and meaning. The Go collector moved by fn-187 into `internal/measure` and `cmd/wasmmeasure` reads the new fields and fails if reconciliation misses tolerance.
- Release helper (`make -C internal/backends/wasm helper`, empty features) keeps byte-identical behavior: the `wasmmeasure` runtime-cost producer that replaces the moved runtime-cost producer (formerly `TestHelperRuntimeCostProfile`) passes on the candidate release helper and retained qualified workloads show unchanged choice, WASI, network and output evidence.
- The decision report states, per cost center: share of authorized execution, the Amdahl upper bound if eliminated, the candidate mechanism, and its risk to determinism, evidence identity and replay. It ends with one explicit verdict: either name the lever that can plausibly reach the stated target or record that no optimization spec is opened.

## Edge Cases & Constraints

- Diagnostic-profile and guest-profile helpers refuse qualification roles; nothing in this spec records a performance pass or changes a diagnostic default.
- Spans must not overlap or double count: nested spans (encoding, hashing, remote write) are subtracted to produce exclusive wall; an overlap error counter must be zero in every retained run.
- Fuel: profiling must not change fuel accounting or choice consumption; a profiled run must reproduce the same choice witness hash as the release run at the same seed.
- Gain versus spread: differences within the repetition range are reported as such; a cost center under 5 percent of authorized execution is not a lever.
- Coordination: run after fn-187 completes its corpus, measurement-command migration and module ownership work, with fn-180 already integrated. Extend the moved profile collector and current dispatch owners. Preserve fn-187's corpus contracts; keep production diagnostic behavior compiled out of the empty-feature release helper.

## Acceptance Criteria

- **R1:** The diagnostic-profile helper attributes every host import to an exact `module.op` and a model owner with count, total, exclusive wall and per-call distribution, and the profile reconciles attributed host wall plus guest residual to execute wall within a stated tolerance with zero overlap errors. Errors: an unattributed import, a missing owner or a reconciliation miss fails the collector rather than producing a partial table.
- **R2:** Helper CPU samples on the same runs are folded by symbol and mapped to the same owners, retained with the profile, and agree with the wall attribution on the top three owners. Errors: disagreement is reported, not reconciled by hand.
- **R3:** Measurements cover the fn-178 cluster scenario on seeds 7, 19, 42 and the single-guest workflow and activity selections on seeds 0, 1000003, 2000006, one warmup plus three warm repetitions each, on a quiet real Linux/arm64 host, with host, source, helper, module, engine and model identities, choice witness equality against release runs, raw JSON and exact rerun commands retained. Errors: an unavailable host or a witness mismatch leaves the measurement open.
- **R4:** The release helper is unchanged in behavior: `TestHelperRuntimeCostProfile` passes on the candidate release helper and the retained qualified workloads show unchanged evidence; no Cargo dependency or lockfile change. Errors: any release-path change blocks completion.
- **R5:** A retained decision report ranks cost centers by share of authorized execution with the Amdahl bound, candidate mechanism and determinism/identity risk per center, and records one verdict: a named lever with its target, or no optimization spec opened. Errors: a report without the verdict, or an optimization started before it, is a finding.

## Boundaries

- No optimization, scheduler, engine, fuel or model-semantics change; this spec measures and decides only.
- No native measurement; native execution stays deferred under fn-128/fn-149.
- No Darwin or amd64 figures as acceptance; historical fn-172/fn-174 numbers are context, not baselines.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-16): measure before optimizing. fn-178 found guest execution under a fifth of authorized time and closed the parallelism and fibers questions, so the next speedup can only come from the in-process host owners, and their split is unknown.

Owner ordering decision (2026-10-16): fn-185 is last in the active WASM queue, immediately after fn-187. Profiling extends the post-restructure helper and moved measurement command. Workloads, seeds, repetitions, reconciliation, witness equality, CPU sampling and release-preservation acceptance remain unchanged.

## Early proof point

Task .1 shows a single-guest activity run whose per-op and per-owner tables reconcile to execute wall within tolerance with the same choice witness as the release run. If reconciliation cannot close, stop and report the gap before the cluster measurements.

## Quick commands

```sh
cargo +1.94.1 build --manifest-path tools/gomad/internal/backends/wasm/wasmhost/Cargo.toml --release --locked --features diagnostic-profile
cd tools/gomad && go run ./internal/backends/wasm/cmd/wasmmeasure --help
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Exact module.op and owner attribution with count, total, exclusive wall and per-call distribution; reconciliation within tolerance with zero overlap errors. | fn-185-gomad-wasm-host-side-execution-cost.1 | — |
| R2 | Helper CPU samples folded by owner, retained, agreeing with wall attribution on the top three owners or reported. | fn-185-gomad-wasm-host-side-execution-cost.2 | — |
| R3 | Cluster seeds 7/19/42 and single-guest seeds 0/1000003/2000006, warmup plus three warm repetitions, quiet real Linux/arm64, identities, witness equality, raw data and rerun commands retained. | fn-185-gomad-wasm-host-side-execution-cost.2 | — |
| R4 | Release helper unchanged in behavior; moved runtime-cost producer passes; no Cargo dependency or lockfile change. | fn-185-gomad-wasm-host-side-execution-cost.1 | — |
| R5 | Ranked decision report with Amdahl bound, mechanism and risk per center, ending in one verdict; roadmap and milestones updated. | fn-185-gomad-wasm-host-side-execution-cost.3 | — |
