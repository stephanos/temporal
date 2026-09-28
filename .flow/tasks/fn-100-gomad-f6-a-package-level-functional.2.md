---
satisfies: [R3, R4]
---
# fn-100-gomad-f6-a-package-level-functional.2 Triage and fix every evidence divergence in the slice

## Description
For each diverging suite, diff runtime event logs / replay observed streams, find the channel, fix in Gomad (or upstream test bug).

## Acceptance
- no suite classified as evidence divergence
- at least eight of ten qualify

## Done summary
None of the six suites diverges on darwin/arm64 at 01e2ef92ec (the same toolchain and `gomad` binary as task .1). The five suites that diverged on linux, plus the cancel suite, each ran with `--repeat 4` on seeds 11 and 17. All 12 seed runs qualify, all four fresh repetitions of every run produce the same evidence digest, and every repetition replays with `match=true` and an exact choice replay. Every digest is the one task .1 recorded with `--repeat 2`. No suite is classified as evidence divergence, and this task changes no code.

Flags are identical to task .1 except `--repeat 4`. Script: `/private/tmp/claude-501/-Users-stephan-Workspace-temporal-gomad/ab9b6622-9568-4591-8873-30108517815b/scratchpad/f6t2/run.sh`. The runs went one at a time, and retained artifacts were deleted after each. Raw reports: `.../scratchpad/f6t2/<Suite>-<seed>/report.json`. Table: `.../scratchpad/f6t2/table.tsv`.

| Suite | Seed | Class | Evidence digest (4 reps) | Replay (4 reps) | Transcript rec | Decisions | Choice records | Virtual time | Exec wall s (reps) | Peak gor. | Outcome | Denied |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| SignalChasm | 11 | qualified | 1dc5c8164551 ×4 | match/exact ×4 | 17 | 57986 | 86512 | 57.016 s | 2.1/2.1/2.6/7.0 | 1944 | exit/success | 0 |
| SignalChasm | 17 | qualified | b41a14a15316 ×4 | match/exact ×4 | 17 | 59741 | 89312 | 57.016 s | 2.2/2.1/2.1/2.1 | 1658 | exit/success | 0 |
| Update | 11 | qualified | ac81a7f94f68 ×4 | match/exact ×4 | 17 | 132626 | 206774 | 22.465 s | 3.9/4.3/3.5/4.1 | 4586 | exit/success | 0 |
| Update | 17 | qualified | ed0aef39a718 ×4 | match/exact ×4 | 17 | 132540 | 206561 | 22.411 s | 2.8/3.2/2.7/2.7 | 4588 | exit/success | 0 |
| Child | 11 | qualified | 8ccfaa1f789f ×4 | match/exact ×4 | 17 | 26045 | 39014 | 13.014 s | 1.6/1.7/1.7/3.1 | 1246 | exit/success | 0 |
| Child | 17 | qualified | cb00e8036059 ×4 | match/exact ×4 | 17 | 25863 | 38763 | 13.014 s | 1.6/1.6/1.7/1.7 | 1248 | exit/success | 0 |
| Cron | 11 | qualified | 4125ff4caff5 ×4 | match/exact ×4 | 17 | 9901 | 14473 | 15.010 s | 1.5/1.5/1.5/1.6 | 712 | exit/success | 0 |
| Cron | 17 | qualified | 9ec30f61456b ×4 | match/exact ×4 | 17 | 10008 | 14651 | 15.010 s | 1.6/1.6/1.6/1.6 | 720 | exit/success | 0 |
| Timer | 11 | qualified | 0ce91d7b5f4e ×4 | match/exact ×4 | 17 | 8720 | 12796 | 8.000 s | 1.5/1.7/1.5/1.6 | 705 | exit/success | 0 |
| Timer | 17 | qualified | 5fba5b945780 ×4 | match/exact ×4 | 17 | 8596 | 12646 | 8.000 s | 1.5/1.5/1.5/1.5 | 707 | exit/success | 0 |
| Cancel | 11 | qualified | 88d0f5945a79 ×4 | match/exact ×4 | 17 | 21777 | 30927 | 0 s | 1.7/1.7/1.7/1.8 | 1355 | exit/success | 0 |
| Cancel | 17 | qualified | 75a95a2311a4 ×4 | match/exact ×4 | 17 | 21774 | 30973 | 0 s | 1.7/1.7/1.7/1.7 | 725 | exit/success | 0 |

The whole qualify command took 69 s to 150 s of host wall time per run. No run had a failure directory, a watchdog termination, or a `GOMAD_CAPABILITY_DENIED`. The one 7.0 s execution (signal 11, rep 4) is host wall only: its virtual time and digest are identical to the other repetitions.

Together with task .1, the slice qualifies 20 of 20 seed runs on darwin, and these six suites add 24 fresh repetitions and 24 exact replays with no divergence. The linux-era divergences are explained by channels fixed after that measurement, not by anything left in the slice:

- **Green Tea scan-work accounting** (97d9925acc): targets now build with `GOEXPERIMENT=nogreenteagc`. This is platform-neutral.
- **Environment filtering copied Gomad control variables into Go strings before dropping them** (97d9925acc): the replay's heap differed from the recording's. This explains the replay-only differences, including cancel's replay-evidence difference. The fix is in the runtime overlay and is platform-neutral.
- **Mark-start greying and the M's allp snapshot** (2d8f937d3e): which m structs the first assist greyed, and whether allp was greyed from an M or from the globals, depended on host timing. That timing moved the end-of-mark boundary and every later run-queue order. The fix is in the runtime overlay and is platform-neutral. It covers the fresh-repetition divergences (signal, update, child, cron 17, timer 17).
- **Server update-admission ordering** (97d9925acc, `compareAdmission` in `service/history/workflow/update`): this was the update suite's test failure. It was fixed upstream in server code and is platform-neutral.
- **ASLR re-execution** (cf1bc982c2, 85c71462b8): this fix is darwin-only (`gomad_aslr_darwin.go`), so it cannot account for the linux divergences.

Linux/amd64 is not re-measured here. That the platform-neutral fixes also close the linux divergences is an inference until the slice is requalified on linux.

stage: impl-review - skipped(empty: measurement-only task, no commits in BASE_COMMIT..HEAD to review; no divergence observed, so code changes were forbidden by the task prompt)
## Evidence
- Commits:
- Tests: baseline: none (spec defines no Quick commands), for S in TestSignalWorkflowTestSuiteChasm TestWorkflowUpdateSuite TestChildWorkflowSuite TestCronTestSuite TestWorkflowTimerTestSuite TestCancelWorkflowSuite; seed 11,17: tools/gomad3/.bin/gomad qualify --json --seed SEED --repeat 4 --choices --choice-bytes=64MiB --replay-successes --success-limit=1 --success-bytes=1GiB --capability-mode closure --build-tag disable_grpc_modules --build-tag gomad --build-tag test_dep --io-ro-mount ./schema=/go.temporal.io/server/schema --execution-timeout 6m --overall-timeout 9m go-test ./tests -- -test.run=^S$ (12/12 rc=0 qualified)
- PRs: