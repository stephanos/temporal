---
satisfies: [R1, R5]
---
# fn-100-gomad-f6-a-package-level-functional.3 Add the slice to temporal.json with expectations and required probes

## Description
Tier 3 entries, per-suite expectations naming blockers, required_probes where modeled operations are needed; update CI assertion.

## Acceptance
- manifest check passes

## Done summary
The ten F6 slice suites are in `temporal.json` as tier 3 `functional-*` workloads. Every non-qualified expectation in the manifest now names a validated `finding` identity (R1): `qualify-set` rejects a failure-class expectation without one, rejects a `finding` on `qualified` or `unsupported_target`, and rejects unknown `required_probes` names at load time. CI assertions expect 28 workloads on both platforms.

| Suite (id) | Test | darwin/arm64 expectation and outcome (seeds 11, 17) |
|---|---|---|
| functional-activity | TestActivityTestSuite | qualified; qualified with exact replay |
| functional-cancel | TestCancelWorkflowSuite | qualified; qualified with exact replay |
| functional-child-workflow | TestChildWorkflowSuite | qualified; qualified with exact replay |
| functional-continue-as-new | TestContinueAsNewTestSuite | qualified; qualified with exact replay |
| functional-cron | TestCronTestSuite | qualified; qualified with exact replay |
| functional-query | TestQueryWorkflowSuite | qualified; qualified with exact replay |
| functional-signal-chasm | TestSignalWorkflowTestSuiteChasm | qualified; qualified with exact replay |
| functional-timer | TestWorkflowTimerTestSuite | qualified; qualified with exact replay |
| functional-update | TestWorkflowUpdateSuite | qualified; qualified with exact replay |
| functional-workflow | TestWorkflowTestSuite | qualified; qualified with exact replay |

- **required_probes (R5):** every slice suite requires `stdlib.net.interfaces`, `stdlib.os.getwd`, `stdlib.os.newfile`, and `stdlib.os.openfile`, which every measured run observed. `qualify` always runs semantic coverage, so a seed that misses a required probe fails instead of qualifying.
- **linux/amd64 expectation:** `intermittent`, with `finding: GOMAD_MILESTONES.md#f6-a-package-level-functional-slice`. Linux last measured the slice before the environment-filter and mark-start greying fixes, when 6 of 20 seed runs qualified. Those fixes are platform-neutral, but until a linux run re-measures the slice the manifest does not claim `qualified` there. The existing intermittent/unrepeatable entries now name their milestone sections the same way: F5 for user-timers and batch cancel, F3 for the frontend probe.

Evidence:

- **At 2097ed05c2 (manifest sha256:e664c61f):** the full `make gomad3-qualification` ran 28/28 supported, 0 failed, 0 infrastructure errors, with expectations met. The darwin CI jq assertion evaluates true on that report.
- **At 9b87c3c6d1 (manifest sha256:e10842d4, the `finding` change):** the full rerun was killed by its disk guard at 22/28, so it is inconclusive as a single run. All 22 completed workloads, including all ten slice suites, were `qualified` with exact replay, with 0 failed and 0 infrastructure errors. A subset manifest with the six unrun workloads (5 tier 2 suites and user-timers) then ran 6/6 qualified with exact replay on the same toolchain build key.
- **Retained artifacts:** deleted after both runs.
- **Reports:** `scratchpad/f6t3/temporal-qualification-set.json`, `scratchpad/f6t3r2/temporal-qualification-set.partial.json`, and `scratchpad/f6t3r2/subset-report.json`.

Follow-up for task .4: a single uninterrupted 28-workload run at the current manifest needs about 11 GiB of free disk for retained artifacts.

Tests: `TestLoadManifestRejectsUnknownRequiredProbes` and `TestLoadManifestRequiresFindingOnFailureExpectations` (tools/gomad3/qualification/set/set_test.go). Both were confirmed red without their fix.

stage: impl-review - ran [2026-09-28T08:14..2026-09-28T08:25] (codex fan-out: round 1 NEEDS_WORK on R1 finding identity, refunded after the fix commit moved HEAD; re-dispatched fan-out SHIP x3)
## Evidence
- Commits: 2097ed05c289bb76bc9cb8c491f6901084472680, 9b87c3c6d1f120e1e3915bcb29c52dc70891d6b6
- Tests: cd tools/gomad3 && go test ./qualification/... ./upgrade/, make -C tools/gomad3 validate, make gomad3-integration-test, tools/gomad3/.bin/gomad qualify-set --check --manifest=tools/gomad3integration/qualification/temporal.json (28 workloads), make gomad3-qualification @2097ed05c2 manifest sha256:e664c61f: 28/28 supported, failed=0, infrastructure_errors=0, darwin CI jq assertion true, make gomad3-qualification @9b87c3c6d1 manifest sha256:e10842d4: INCONCLUSIVE full run, stopped by disk guard at 22/28 completed (all 22 qualified, failed=0, infrastructure_errors=0), qualify-set subset manifest (the 6 unrun workloads) @9b87c3c6d1: 6/6 supported, failed=0, infrastructure_errors=0
- PRs: