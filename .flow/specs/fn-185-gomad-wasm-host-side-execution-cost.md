# Gomad: WASM host-side execution cost profile

**Owner acceptance amendment, 2026-10-24 (legacy compatibility retired).** Apply the [current compatibility policy](../artifacts/legacy-compatibility-policy-2026-10-24.md). Legacy Go/Python byte, schema, CLI/message and compatibility-only frozen-host requirements below are retired. Current behavior, integrity and same-candidate deterministic replay remain required; compatibility-only proof rows may retire explicitly without replacement proof.

## Conversation Evidence

> user: "File a host-import profiling spec for performance before committing to any optimization."

## Goal & Context

Attribute authorized execution time on the current WASM candidate to host-side cost centers precisely enough to decide whether another execution-speed lever exists, and record that decision before any optimization spec is opened. [paraphrase]

fn-178 measured guest share `G/T` at 0.148 to 0.192 on the Temporal two-cluster partition/heal/crash scenario: more than four fifths of authorized execution is spent in timed host-import spans, now served in-process by the Rust helper. fn-174 removed the encoding and transcript-hash cost (93 percent reduction), and fn-178 ruled out parallel instances (best-case bound 1.106x) and fibers (0.05 to 0.09 percent handoff share). What remains unmeasured is how the host-import time divides between the model owners that replaced the Go broker: namespace, volume, network, clock, World, storage transport, cache and scheduler. The current diagnostic profile cannot answer this: `AbiSpan` folds every in-process import into six coarse kinds (clock, scheduler, select, observation, idle, other-local), and the retained fn-172.4 breakdown predates the in-process migration, where `nested.remote_read_wait` was 90 percent of execute.

## Architecture & Data Models

- Extend the diagnostic-profile helper (`--features diagnostic-profile`, never qualification evidence) so each host import records its exact `module.op` and the model owner that served it, with count, total wall, exclusive wall (minus nested spans) and a bounded per-call distribution (min, median, p99, max). Add a reconciliation row: attributed host wall plus guest residual must equal execute wall within a stated tolerance, otherwise the profile is incomplete.
- Owner attribution happens where the helper dispatches an import to a model (`abi.rs` dispatch, `namespace_call.rs`, `storage_transport.rs`, `network`, `clock.rs`, `world`, `cache.rs`, cluster handoff). No change to release-helper code paths beyond the already compiled-out diagnostic feature.
- Rust-side CPU sampling of the helper with an external profiler (`perf record` on a Linux host, or the platform equivalent) on the same runs, folded to symbols and mapped to the same owners; retained as flamegraph data next to the profile JSON. No new Cargo dependency.
- Workloads: the fn-178 cluster scenario (`TestMatchedReleaseClusterCandidate` path, seeds 7, 19, 42) and the single-guest workflow and activity selections used by fn-172/fn-174 (seeds 0, 1000003, 2000006), each one warmup plus three warm repetitions, on a quiet host, following the owner WASM measurement-host decision.
- Output: one retained artifact under `.flow/artifacts/` with per-owner and per-op tables, reconciliation, identities, raw JSON and exact rerun commands, plus a ranked decision table.

### Rust measurement owner and comparison rules

Task .1 must expose diagnostic attribution on the actual Rust linked-execution path, including execute-phase session/span lifecycle and request-local bounded sinks; transport-helper instrumentation alone is insufficient. Collector/report code and sampling target for .2/.3 belong to `tools/gomad/rust/gomad` and the linked helper library. Existing Go `wasmmeasure` drivers are semantic references and retained historical evidence, not the final Rust measurement producer. R6 drivers must be adapted to current Rust owners and any-host policy without rerunning retired native execution or manufacturing historical baselines. Missing comparable inputs remain an explicit unavailable R6 result.

Use the same raw elapsed interval for a workload's wall denominator. Do not sum overlapping per-instance wall spans into an elapsed fraction: retain those as aggregate thread-time, with overlap explained. Rank reducible CPU/authorized-work costs separately from handoff waiting and instrumentation. Only a non-overlapping contribution to that denominator supports an Amdahl bound; otherwise report the bound as unavailable rather than deriving speedup from accumulated waits.

The Rust producer accepts explicit fn-221 sweep selections/configurations, including timeout and capacity representatives. Finalize bounded diagnostic profiles on interruption, retain termination, actual prefix and dropped/truncated counters, and never force a timeout cell through a successful-exit-only collector. Completed deterministic runs require full release witness equality. Host-wall timeout pairs can stop at different ordinals: authenticate and compare their common execution prefix and replay each retained prefix under its own identities, retain both cutoff observations, and make no full-run equivalence claim. Missing/truncated proof remains pending. Preserve all original limits; instrumentation slowdown is reported, not repaired by increasing the budget. A deterministic capacity boundary still requires matching evidence to that boundary.

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
- **R3:** Measurements cover the fn-178 cluster scenario on seeds 7, 19, 42 and the single-guest workflow and activity selections on seeds 0, 1000003, 2000006, one warmup plus three warm repetitions each, on a quiet host, with host, source, helper, module, engine and model identities, choice witness equality against release runs, raw JSON and exact rerun commands retained. Errors: an unavailable host or a witness mismatch leaves the measurement open.
- **R4:** The release helper is unchanged in behavior: the current runtime-cost producer/equivalent release-behavior check (formerly `TestHelperRuntimeCostProfile`) passes on the candidate release implementation and the retained qualified workloads show unchanged evidence; no Cargo dependency or lockfile change. Errors: any release-path change blocks completion.
- **R5:** A retained decision report ranks cost centers by share of authorized execution with the Amdahl bound, candidate mechanism and determinism/identity risk per center, and records one verdict: a named lever with its target, or no optimization spec opened. Errors: a report without the verdict, or an optimization started before it, is a finding.
- **R6:** Transferred from fn-182 R4 and fn-184 R5 by the owner decision of 2026-10-17. Retain matched baseline/candidate measurements on the same host in the same window: (a) fn-182 full versus slim guests, for cold compilation, warm preparation/load, fresh-instance startup, execution wall time and helper peak RSS on the one-box and multi-instance cells, plus the candidate compiled-cache bytes that were not retained at close; (b) fn-184 wall times for build, vet, lint and the default test targets (`test-harness`, `world-test`, `test-wasm`) on the pre-isolation baseline versus the isolated default set, and for the selected set. Report repetitions, median/range, host, source, helper, module and toolchain identities and exact rerun commands. Label every difference within the repetition range as within spread; claim no gain from bytes or package counts alone. Reuse the prepared drivers: fn-182 `.flow/artifacts/fn185-performance-inputs/measure.py` (SHA-256 `cae15958505e00796e3b9ee77ff90dbc151829ff33b4498afe0fd241d9037aa1`) with its frozen helper, `qualification.test` and inputs, and fn-184 `/home/agent/wt/evidence/fn185/r6/isolation/measure.sh` with base `3dae95b99f` and after `f22f8ef227`. Their frozen inputs and worktrees are local and may be gone; then fn-185 re-measures against its own baseline and candidate under the same contract. Errors: an unavailable measurement stays labeled unavailable and blocks R6; a failed cell is retained and rerun into a new root, never dropped.

## Boundaries

- No optimization, scheduler, engine, fuel or model-semantics change; this spec measures and decides only.
- No native measurement; native execution stays deferred under fn-128/fn-149.
- WASM performance measurements may run on any host under the 2026-10-17 owner decision. Baseline and candidate must use the same host and measurement window with all repetitions, spread, identities and raw evidence retained; historical fn-172/fn-174 numbers are context, not baselines.
- No CI run, PR or push.

## Decision Context

Owner decision (2026-10-16): measure before optimizing. fn-178 found guest execution under a fifth of authorized time and closed the parallelism and fibers questions, so the next speedup can only come from the in-process host owners, and their split is unknown.

Owner ordering decision (2026-10-16): fn-185 is last in the active WASM queue, immediately after fn-187. Profiling extends the post-restructure helper and moved measurement command. Workloads, seeds, repetitions, reconciliation, witness equality, CPU sampling and release-preservation acceptance remain unchanged.

Owner deferral decision (2026-10-17): "can we defer the perf checks to fn-185 and then close out fn-182 and fn-184?" The wall-clock measurements of fn-182 R4 and fn-184 R5 move here as R6, owned by task .2. fn-182 and fn-184 closed with their byte and count measurements; their closure records remain in Flow and Git history. R6 does not change the profiling scope of R1–R5.

Closed-owner cleanup (2026-10-20): the active full/slim driver and tests now belong to `.flow/artifacts/fn185-performance-inputs/`; isolation inputs belong to `/home/agent/wt/evidence/fn185/r6/isolation/`. Closed-spec evidence is deleted, not archived. `fn185-input-release-baseline` preserves source commit `51604889c9a9062f981cd02aa3daf04ca55fd940` for the transferred matched-release comparison; rebuild and authentically admit its helper under fn-185 paths before measurement. Old absolute-path admissions are not relocated or relabeled. All original R6 repetitions, comparisons, identity and spread requirements remain unchanged.

## Early proof point

Task .1 shows a single-guest activity run whose per-op and per-owner tables reconcile to execute wall within tolerance with the same choice witness as the release run. If reconciliation cannot close, stop and report the gap before the cluster measurements.

## Quick commands

```sh
cargo build --manifest-path tools/gomad/rust/Cargo.toml -p gomad --release --locked --bin gomad-measure --features helper-diagnostic-profile
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad --features helper-diagnostic-profile --test profile_corpus
gomad-measure host-owner-profile --help      # R1/R2/R3 producer
gomad-measure runtime-cost-profile --help    # R4 release-preservation check
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Exact module.op and owner attribution with count, total, exclusive wall and per-call distribution; reconciliation within tolerance with zero overlap errors. | fn-185-gomad-wasm-host-side-execution-cost.1 | — |
| R2 | Helper CPU samples folded by owner, retained, agreeing with wall attribution on the top three owners or reported. | fn-185-gomad-wasm-host-side-execution-cost.2 | — |
| R3 | Cluster seeds 7/19/42 and single-guest seeds 0/1000003/2000006, warmup plus three warm repetitions, quiet host, identities, witness equality, raw data and rerun commands retained. | fn-185-gomad-wasm-host-side-execution-cost.2 | — |
| R4 | Release helper unchanged in behavior; moved runtime-cost producer passes; no Cargo dependency or lockfile change. | fn-185-gomad-wasm-host-side-execution-cost.1 | — |
| R5 | Ranked decision report with Amdahl bound, mechanism and risk per center, ending in one verdict; roadmap and milestones updated. | fn-185-gomad-wasm-host-side-execution-cost.3 | — |
| R6 | Transferred fn-182 R4 and fn-184 R5 timing: matched same-host baseline/candidate, repetitions, median/range, gain-versus-spread labels, identities and rerun commands, reusing or re-creating the prepared drivers. | fn-185-gomad-wasm-host-side-execution-cost.2 | — |

## Amendments (2026-10-24, conductor)

1. **Owners are Rust.** The Go collector (`internal/measure`, `cmd/wasmmeasure`) named in API Contracts was retired with the Go host. The producer is the `gomad-measure` binary in `tools/gomad/rust/gomad` (`host-owner-profile`, `runtime-cost-profile`), reading the linked helper's `helper-diagnostic-profile` feature; the release build is the empty-feature `gomad`. Task .1's 2026-10-23 handover shows R1 attribution, reconciliation and witness equality proven on `gomad`; its R4 release-preservation check stays open until the fn-232.13 cycle supplies it.
2. **Tasks .2/.3 wait for the quiet-host window.** Owner decision 2026-10-23: timing results are non-blocking until the owner declares a quiet host. The profiles, the perf sampling and the verdict run then, on the commit current at that time, together with the stress timing and cycle timing reruns. The revival trigger is recorded in task .2.
3. **R6(b) recommendation (owner decision needed).** R6(b) compares build, vet, lint and default-test wall times of the Go host before and after its fn-184 isolation (base `3dae95b99f`, after `f22f8ef227`). Both are Go-host trees that no longer exist in the default build; the measurement would time a deleted codebase and cannot inform any current decision. The conductor recommends closing R6(b) as won't-do with that reason and keeping R6(a) (full versus slim guest cost on the Rust host, which still answers a live question). Until the owner rules, R6(b) stays as written and is reported unavailable.
