---
satisfies: [R2, R3, R6]
---
# fn-185-gomad-wasm-host-side-execution-cost.2 Capture host-side profiles on cluster and single-guest workloads

## Description
On a quiet host, run the fn-178 cluster scenario (the wasmmeasure producer replacing TestMatchedReleaseClusterCandidate, seeds 7, 19, 42) and the single-guest workflow and activity selections (seeds 0, 1000003, 2000006), one warmup plus three warm repetitions each, with the diagnostic-profile helper, and the same runs under `perf record` on the helper. Fold perf samples by symbol, map them to the same owners and compare the top three owners with the wall attribution. Verify the choice witness of every profiled run equals the release-helper run at the same seed. Retain under .flow/artifacts/fn185-*/ the raw profile JSON, folded stacks, host cpuinfo, boot id, source/helper/module/engine/model identities and exact rerun commands; summarize per-owner and per-op medians with ranges in a markdown artifact. Diagnostic and guest-profile helpers are never qualification evidence.

Also run the R6 measurements transferred from fn-182 R4 and fn-184 R5 (owner decision 2026-10-17), with the release helper, each baseline/candidate pair on the same host in the same window. fn-182: `python3 -I .flow/artifacts/fn185-performance-inputs/measure.py <checkout> <new root> --stock-go … --helper … --binary … --inputs … --run` (one checkout, profiles `full` versus `slim`, ABBAAB success cells plus canaries). The closed fn-182 frozen helper, `qualification.test` and inputs predate the cluster finish-window fix `6ead645c9e`; rebuild them from the measured candidate. fn-184: `measure.sh run` from `/home/agent/wt/evidence/fn185/r6/isolation/` with `BASE_SHA_EXPECTED=3dae95b99f AFTER_SHA_EXPECTED=f22f8ef227` (REPS=3, cold and warm, alternating side order). If a driver, its worktrees or its inputs are gone, re-create the same contract against fn-185's own baseline and candidate. Retain results under .flow/artifacts/fn185-*/.

**Size:** M
**Files:** .flow/artifacts/fn185-host-cost-profile-<date>.md and directory, measurement scripts kept with the artifact

**Touches:** .flow/artifacts/fn185-host-cost-profile-<date>.md and directory, measurement scripts kept with the artifact


**Re-scope (owner decision 2026-10-19):** this task waits for the fn-221 triage sweep and profiles sweep workloads, not only the fixtures above. Add the functional tests the sweep retains as passing, slowest-passing and timeout/capacity representatives (at least one of each, named by the sweep's support matrix) to the measured workload set with the same warmup, repetition, identity and retention contract. The fixture and cluster sets above remain for comparison with fn-178. Task .1's attribution tooling is not gated and may land before the sweep so the sweep harness can label timeouts by cost center.

**Delivery ordering (2026-10-20):** under the owner's instruction to follow recommendations autonomously, tasks .2/.3 run after the functional milestones in MILESTONES.md. Task .1's attribution tooling remains available alongside. Targeted profiling may come forward only for a demonstrated performance blocker. Unmeasured timeout/capacity cases remain unresolved, not unsupported; their cost-dependent follow-on classification waits for the measured verdict. This changes scheduling only: all workloads, repetitions, identities, witness comparisons and transferred R6 acceptance above remain required.
Closed-owner cleanup (2026-10-20): the active full/slim driver and tests now belong to `.flow/artifacts/fn185-performance-inputs/`; isolation inputs belong to `/home/agent/wt/evidence/fn185/r6/isolation/`. Closed-spec evidence is deleted, not archived. `fn185-input-release-baseline` preserves source commit `51604889c9a9062f981cd02aa3daf04ca55fd940` for the transferred matched-release comparison; rebuild and authentically admit its helper under fn-185 paths before measurement. Old absolute-path admissions are not relocated or relabeled. All original R6 repetitions, comparisons, identity and spread requirements remain unchanged.

## Acceptance
Use the parent Rust measurement producer, not the Go-only collector. Accept explicit sweep selections and finalize bounded profiles on timeout/capacity. Full completed witnesses match release; wall-time interrupted pairs authenticate/compare their common prefix and replay retained prefixes without claiming full-run equality or widening limits. Preserve unavailable/truncated measurements as pending. Adapt R6 drivers to the current Rust owner and any-host policy, with exact comparable inputs retained.

- [ ] All six workload/seed sets measured with warmup plus three warm repetitions; medians and ranges retained
- [ ] perf-derived owner ranking agrees with wall attribution on the top three owners, or the disagreement is reported
- [ ] Choice witnesses match release runs for every profiled run
- [ ] Identities, raw data and rerun commands retained; host is quiet and recorded
- [ ] R6: fn-182 full/slim timing and helper peak RSS (plus candidate compiled-cache bytes) and fn-184 build/vet/lint/default-test wall times (default and selected sets) measured as matched same-host baseline/candidate pairs with repetitions, median/range, gain-versus-spread labels, identities and rerun commands

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

### Deferred to the quiet-host window (owner decision 2026-10-23)

**Revival trigger:** the owner declares a quiet-host window. This task then runs on the commit current at that time, alongside the stress timing and cycle timing reruns. Producer: `gomad-measure host-owner-profile` and `runtime-cost-profile` (Rust); the Go `wasmmeasure` paths above are retired references. Sweep representatives come from the fn-221 Rust sweep matrix. Nothing in the acceptance is waived.
