---
satisfies: [R6]
---
# fn-100-gomad-f6-a-package-level-functional.1 Requalify the ten-suite slice on darwin after the environment-filter fix

## Description
Run each suite through `gomad qualify --repeat 2` on seeds 11 and 17 with retention and replay; tabulate outcomes, counts, watchdogs, denials.

## Acceptance
- table of 20 seed runs recorded

## Done summary
All 20 seed runs of the ten-suite slice qualify on darwin/arm64 at d40c24b5cd (toolchain key bb4304eb…). Both fresh repetitions of every run produce the same evidence digest, and every retained success replays with `match=true` and an exact choice replay. No run diverges, none hits a watchdog, and none prints `GOMAD_CAPABILITY_DENIED`.

Each run used `gomad qualify --json --seed S --repeat 2 --choices --choice-bytes=64MiB --replay-successes --success-limit=1 --success-bytes=1GiB --capability-mode closure --build-tag disable_grpc_modules --build-tag gomad --build-tag test_dep --io-ro-mount ./schema=/go.temporal.io/server/schema --execution-timeout 6m --overall-timeout 9m go-test ./tests -- -test.run=^SUITE$`. The first run (activity 11) used `--overall-timeout 20m`. The other runs used 9m so that each one fit inside a single blocking call. No run came close to either limit. The raw reports are under `/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/ab9b6622-9568-4591-8873-30108517815b/scratchpad/f6t1/<Suite>-<seed>/report.json`, next to `rc.txt` and `events.jsonl`. Retained artifacts were deleted after each run, and 12 GiB stayed free throughout.

| Suite | Seed | Class | Evidence digest (both reps) | Replay | IO transcript rec/bytes | Decisions | Choice records | Virtual time | Wall/exec (max) | Host wall | Peak gor. | Watchdog | Denied | >2 min |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Activity | 11 | qualified | 2fadf5e22da2 | match/exact ×2 | 17 / 2176 | 27907 | 42309 | 13.618 s | 1.78 s | 57 s | 1452 | 0 | 0 | no |
| Activity | 17 | qualified | 7738fed5d14f | match/exact ×2 | 17 / 2176 | 27595 | 41842 | 13.118 s | 1.71 s | 40 s | 1455 | 0 | 0 | no |
| SignalChasm | 11 | qualified | 1dc5c8164551 | match/exact ×2 | 17 / 2176 | 57986 | 86512 | 57.016 s | 2.15 s | 41 s | 1944 | 0 | 0 | no |
| SignalChasm | 17 | qualified | b41a14a15316 | match/exact ×2 | 17 / 2176 | 59741 | 89312 | 57.016 s | 2.18 s | 41 s | 1658 | 0 | 0 | no |
| Query | 11 | qualified | 2160fb48480a | match/exact ×2 | 17 / 2176 | 25633 | 39046 | 11.262 s | 1.73 s | 39 s | 1408 | 0 | 0 | no |
| Query | 17 | qualified | f9982349d4c3 | match/exact ×2 | 17 / 2176 | 25581 | 38998 | 11.262 s | 1.72 s | 40 s | 1409 | 0 | 0 | no |
| Update | 11 | qualified | ac81a7f94f68 | match/exact ×2 | 17 / 2176 | 132626 | 206774 | 22.465 s | 3.06 s | 44 s | 4586 | 0 | 0 | no |
| Update | 17 | qualified | ed0aef39a718 | match/exact ×2 | 17 / 2176 | 132540 | 206561 | 22.411 s | 2.89 s | 44 s | 4588 | 0 | 0 | no |
| Child | 11 | qualified | 8ccfaa1f789f | match/exact ×2 | 17 / 2176 | 26045 | 39014 | 13.014 s | 1.74 s | 38 s | 1246 | 0 | 0 | no |
| Child | 17 | qualified | cb00e8036059 | match/exact ×2 | 17 / 2176 | 25863 | 38763 | 13.014 s | 1.74 s | 38 s | 1248 | 0 | 0 | no |
| ContinueAsNew | 11 | qualified | 3349341f9c15 | match/exact ×2 | 17 / 2176 | 51612 | 80173 | 36.432 s | 2.05 s | 40 s | 1170 | 0 | 0 | no |
| ContinueAsNew | 17 | qualified | 7e157a4c3f5c | match/exact ×2 | 17 / 2176 | 52239 | 80963 | 36.422 s | 2.04 s | 40 s | 1170 | 0 | 0 | no |
| Cron | 11 | qualified | 4125ff4caff5 | match/exact ×2 | 17 / 2176 | 9901 | 14473 | 15.010 s | 1.59 s | 37 s | 712 | 0 | 0 | no |
| Cron | 17 | qualified | 9ec30f61456b | match/exact ×2 | 17 / 2176 | 10008 | 14651 | 15.010 s | 1.60 s | 38 s | 720 | 0 | 0 | no |
| Workflow | 11 | qualified | 3786e45f5cf2 | match/exact ×2 | 17 / 2176 | 56053 | 86253 | 13.072 s | 2.08 s | 40 s | 2135 | 0 | 0 | no |
| Workflow | 17 | qualified | 048d2ca5ebd8 | match/exact ×2 | 17 / 2176 | 55687 | 85681 | 13.372 s | 2.03 s | 41 s | 2105 | 0 | 0 | no |
| Cancel | 11 | qualified | 88d0f5945a79 | match/exact ×2 | 17 / 2176 | 21777 | 30927 | 0 s | 1.88 s | 39 s | 1355 | 0 | 0 | no |
| Cancel | 17 | qualified | 75a95a2311a4 | match/exact ×2 | 17 / 2176 | 21774 | 30973 | 0 s | 1.73 s | 38 s | 725 | 0 | 0 | no |
| Timer | 11 | qualified | 0ce91d7b5f4e | match/exact ×2 | 17 / 2176 | 8720 | 12796 | 8.000 s | 1.55 s | 38 s | 705 | 0 | 0 | no |
| Timer | 17 | qualified | 5fba5b945780 | match/exact ×2 | 17 / 2176 | 8596 | 12646 | 8.000 s | 1.55 s | 38 s | 707 | 0 | 0 | no |

How to read the columns:

- "Host wall" covers the whole qualify command: the build, two executions, and two replays. The build dominates.
- "Wall/exec" is the longest execution wall time in the report.
- IO transcript bytes are the record count times the fixed 128-byte record size.
- Watchdog comes from the evidence outcome, which is `exit/success` for every run.
- Denied counts `GOMAD_CAPABILITY_DENIED` occurrences in the retained stdout and stderr.

First divergence: none. No run diverged, so no seed was rerun for host load.

Checks on suspicious greens:

- Query 11 was rerun with its artifacts kept. Query's stdout is only `PASS` because the suite logs to stderr. The stderr names each query test, and the rerun reproduced digest 2160fb48480a.
- Cancel 11 was rerun the same way to check its 0 s of virtual time. Every log timestamp is `2000-01-01T00:00:00Z`, and all three `TestRequestCancelWorkflowCommandExecution_*` subtests plus `TestExternalRequestCancelWorkflowExecution` appear. The rerun reproduced digest 88d0f5945a79.

Compared with the earlier linux/amd64 picture:

- These are 20 of 20 qualified runs. The linux run qualified 6 of 20 before the environment-filter, mark-start greying, and allp snapshot fixes.
- The six seed runs that had diverged between fresh repetitions (signal, update, child, cron 17, timer 17) and cancel's replay-evidence difference do not reproduce on darwin. That leaves task .2 with no darwin divergence to triage. Linux still needs a requalification.
- The transcript holds 17 records here, against 15 on linux.

stage: impl-review - skipped(empty: measurement-only task, no commits in BASE_COMMIT..HEAD to review; code changes forbidden by the task prompt)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: tools/gomad3/.bin/gomad qualify --json --seed {11,17} --repeat 2 --choices --choice-bytes=64MiB --replay-successes --success-limit=1 --success-bytes=1GiB --capability-mode closure --build-tag disable_grpc_modules --build-tag gomad --build-tag test_dep --io-ro-mount ./schema=/go.temporal.io/server/schema --execution-timeout 6m --overall-timeout 9m go-test ./tests -- -test.run=^<Suite>$ (x10 suites), baseline: none (spec defines no Quick commands)
- PRs: