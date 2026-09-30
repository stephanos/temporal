---
satisfies: [R3]
---

# fn-103-gomad-seeded-virtual-clock-ticks.2 Measure forward on the tie-excluded suites and decide the default
# fn-103-gomad-configurable-virtual-clock-tick.2 Measure the tick on the tie-excluded suites and retire their exclusions

## Description
Run the tie-excluded suites and the smoke selection under forward across several seeds; if the smoke set stays green and the tie failures resolve, make forward the default, requalify the core, representative and smoke sets, and resolve tie exclusions (fixed upstream, pinned with a finding, or seed-named); otherwise keep strict and record the evidence. No full ./tests runs.
## Acceptance
- report lists exclusions removed; manifest updated; validate green

## Done summary
Measured on darwin/arm64 (2026-09-30): the seven ./tests suites carrying timestamp-tie skips (TestAdvancedVisibilitySuite and its legacy variant, TestDescribeTestSuite, TestNexusOTELSuite, TestNexusStandaloneTestSuite, TestStandaloneActivityTestSuite, TestTaskQueueSuite) ran under --clock-tick=forward with those 18 skips removed on seeds 11 and 17. Six qualified with exact replay on both seeds; TestStandaloneActivityTestSuite failed on seed 11 in TestStartDelay/UpdateWhilePaused_AfterWindow_ExtendsDispatch because, with time.Now ahead of the timer clock, a gRPC deadline lands microseconds later on the server than on the client and the 3s long poll hits the client deadline first. With that one subtest skipped (finding recorded), the suite qualifies on both seeds.

Decision (R3): the default stays strict. Switching globally changes every identity and would expose further deadline races of this kind across ./tests; instead the seven suites carry clock_tick forward per workload in the generated manifest, which resolves the tie exclusions as the spec allows ("pinned per workload with a finding").

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2f9d55f90
- Tests: gomad qualify-set /tmp/tie-forward.json: 6/7 qualified, StandaloneActivity seed 11 target_failure, gomad qualify-set StandaloneActivity forward with one skip: qualified seeds 11 and 17, make validate-qualification
- PRs: