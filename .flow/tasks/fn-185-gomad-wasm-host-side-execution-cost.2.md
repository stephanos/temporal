---
satisfies: [R2, R3]
---
# fn-185-gomad-wasm-host-side-execution-cost.2 Capture host-side profiles on cluster and single-guest workloads

## Description
On a quiet host, run the fn-178 cluster scenario (the wasmmeasure producer replacing TestMatchedReleaseClusterCandidate, seeds 7, 19, 42) and the single-guest workflow and activity selections (seeds 0, 1000003, 2000006), one warmup plus three warm repetitions each, with the diagnostic-profile helper, and the same runs under `perf record` on the helper. Fold perf samples by symbol, map them to the same owners and compare the top three owners with the wall attribution. Verify the choice witness of every profiled run equals the release-helper run at the same seed. Retain under .flow/artifacts/fn185-*/ the raw profile JSON, folded stacks, host cpuinfo, boot id, source/helper/module/engine/model identities and exact rerun commands; summarize per-owner and per-op medians with ranges in a markdown artifact. Diagnostic and guest-profile helpers are never qualification evidence.

**Size:** M
**Files:** .flow/artifacts/fn185-host-cost-profile-<date>.md and directory, measurement scripts kept with the artifact

## Acceptance
- [ ] All six workload/seed sets measured with warmup plus three warm repetitions; medians and ranges retained
- [ ] perf-derived owner ranking agrees with wall attribution on the top three owners, or the disagreement is reported
- [ ] Choice witnesses match release runs for every profiled run
- [ ] Identities, raw data and rerun commands retained; host is quiet and recorded

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
