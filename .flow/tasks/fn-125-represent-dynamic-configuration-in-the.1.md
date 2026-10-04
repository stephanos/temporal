---
satisfies: [R1, R2]
---
# fn-125-represent-dynamic-configuration-in-the.1 Fix the Nexus switch, limit it to workflow-scheduled Nexus Cases, and stop deriving schedule-to-close from the instruction timeout

## Description
Implements R1 and R2 (spec "Fix-now items"). Go harness and Driver only: no Scala, IR or Case-byte change. This task produces the HSM/CHASM evidence that the owner's Q2 and task 8 wait for.

**Cross-spec entry gate:** none; may start now. fn-121.3 also edits `tests/testpilot_generated_test.go` (build tags); whichever lands second rebases. Paths are before fn-114.9's move.

**Size:** M
**Files:** `tests/testcore/testpilot/switch.go` (+ test); `tests/testpilot_generated_test.go`; `tests/testpilot_nexus_caller_case_test.go`, `tests/testpilot_run_case_test.go` (switch users); `common/testing/testpilot/temporal/worker/typed.go` (+ test).
**Touches:** [tests/testcore/testpilot/**, tests/testpilot_*_test.go, common/testing/testpilot/temporal/worker/**]

### Approach
- Switch values: `chasm` sets every key `tests/nexus_workflow_test.go:82-94` sets for CHASM, including `nexusoperation.chasmWorkflowOperationsRolloutPercent=100`; `hsm` sets the HSM side of the same keys (rollout 0). `SwitchSetting.Value` is a bool today, so generalize it to a typed value (the rollout percent is an int) and let `Configuration()` spell each value in the registry codec's text form.
- Unit test: build a dynamic-config collection from exactly the settings the harness passes under each value and assert `chasmnexus` `UseChasmForWorkflow` (`chasm/lib/nexusoperation/config.go:69-75`) is true under `chasm` and false under `hsm`.
- The harness refuses one key given two values (name the key, both values and both sources) instead of "last wins". This ends the `EnableChasm=true` append after `hsm`'s `false` (`testpilot_generated_test.go:182`). If a workflow-Nexus Case truly needs `history.enableChasm=true` for something other than the implementation, the refusal names it; record it for the owner, never pick a value silently.
- The switch applies only to Cases whose Program schedules a workflow Nexus operation (a `ScheduleNexusOperation` command, cf. `scheduleNexusOperation` in `typed.go`), not to every Case that binds an endpoint (`bindsNexusEndpoint`). Standalone Nexus operation Cases run once with their required settings (plus today's blanket settings, which task 6 removes).
- Driver: `scheduleNexus` passes `ScheduleToCloseTimeout`, `ScheduleToStartTimeout` and `StartToCloseTimeout` exactly as the command carries them and derives none from `instruction.TimeoutMilliseconds()` (`typed.go:76`). Test that changing the Profile's default instruction timeout or `BoundScale` changes no request field the Driver sends. The caller's timeout paths set their deadline explicitly (`model/temporal/nexuscaller/Realization.scala:382-394`); a Case whose operation then never closes fails by its declared wait bound, naming it. If a pinned Run records the request, re-record (`make umpire-rerecord-pinned-runs`) and say why.
- Run every workflow-Nexus generated Case live once under each value (and the `testpilot_nexus_caller_case_test.go` Queries). List every divergence, with both Verdicts, in the done summary for the owner (Q2). Do not hide one by a Model, Case or expectation change.

### Investigation targets
**Required:**
- `tests/testcore/testpilot/switch.go`
- `tests/testpilot_generated_test.go:150-260`
- `tests/nexus_workflow_test.go:78-96`; `chasm/lib/nexusoperation/config.go:34-75`; `chasm/lib/workflow/nexus_commands.go:35-45`
- `common/testing/testpilot/temporal/worker/typed.go:55-95`
- `.plans/DYNAMIC_CONFIG.md` sections 1-2
**Optional:**
- `tests/testpilot_nexus_caller_case_test.go:25-140`; `.flow/tasks/fn-121-shard-generated-cases-per-case-in-ci.1.md`

### Quick commands
```bash
go test -count=1 -tags test_dep ./tests/testcore/testpilot/... ./common/testing/testpilot/...
go test -count=1 -tags 'test_dep integration' ./tests -run 'TestTestpilotGeneratedCases|TestTestpilotNexusCaller'
make umpire-check-cases
```

### Execution constraints
- Case bytes, the lowered manifest and `tests/testcore/testpilot/testdata/generated-case-names.txt` are unchanged (depth-2 names; only the switch's depth-3 subtests move).
- No Model, realization, expectation or Contract change. A divergence is reported, not fixed.

## Acceptance
- [ ] `chasm` sets every CHASM-selecting key, rollout percent 100 included; `hsm` sets the HSM values of the same keys; a unit test shows `UseChasmForWorkflow` true under `chasm` and false under `hsm` for exactly the settings the harness passes.
- [ ] The harness refuses one key given two values, naming the key and both values; the `EnableChasm` override at `testpilot_generated_test.go:182` is gone.
- [ ] The switch applies only to Cases whose Program schedules a workflow Nexus operation; standalone Nexus Cases run once.
- [ ] The Driver sends a schedule command's timeouts exactly as carried and derives none from the instruction timeout; a test shows the Profile's default instruction timeout and scale factor change no request field.
- [ ] The workflow-Nexus Cases ran live once under each value; every divergence (or none) is listed with both Verdicts in the done summary for the owner's Q2.
- [ ] Case bytes and the Case-name golden are unchanged; Testpilot and tooling tests and `make lint-code-fast` pass.


## Done summary
# fn-125.1 done summary

### What changed
- `tests/testcore/testpilot/switch.go`: `SwitchSetting.Value` is typed (`any`, validated against the setting by
  `ResolveSettings`) and carries a `Source`; `Configuration()` spells values in the registry text form (`true`, `100`).
  `hsm`/`chasm` now set the six keys `tests/nexus_workflow_test.go:82-94` sets, including
  `nexusoperation.chasmWorkflowOperationsRolloutPercent` 0/100. `ResolveSettings` refuses one key given two values,
  naming the key, both values and both sources; equal values are kept once. `SchedulesWorkflowNexusOperation`
  scopes the switch to Programs with a `ScheduleNexusOperation` command. `CaseSettings` builds a Case's cluster settings
  (switch value or `StandaloneSettings` + required settings); `requiredSettingKinds` moved here from the harness so the
  unit test uses exactly the harness's settings.
- `tests/testpilot_generated_test.go`: the switch runs only for workflow-Nexus Cases; standalone Nexus operation Cases run
  once; the `EnableChasm=true` append after `hsm`'s `false` is gone (a key given two values now fails the Case).
- `common/testing/testpilot/temporal/worker/typed.go`: `scheduleNexus` sends schedule-to-close, schedule-to-start and
  start-to-close exactly as carried; none derives from the instruction timeout (QLF-01).
- Tests: `TestNexusImplementationSwitchSelectsTheImplementation` (for every generated workflow-Nexus Case, a dynamic
  config collection holding exactly `CaseSettings` gives `UseChasmForWorkflow` true under `chasm`, false under `hsm`;
  without the rollout key `chasm` gives false), `TestResolveSettingsRefusesOneKeyGivenTwoValues`,
  `TestSchedulesWorkflowNexusOperationSelectsTheSwitchedCases`, `TestSDKScheduleCommandTimeoutsDoNotDependOnTheProfile`
  (three Profiles differing in default instruction timeout and BoundScale send identical options);
  `TestSDKAwaitUsesItsOwnTimeout` updated (its "start expires first" case asserted the forbidden derivation).
- Pinned Runs re-recorded (control and canary): each recorded the schedule request's Profile-derived `10s`
  schedule-to-close; they now record `0s`. Receipt goldens re-rendered. Case bytes, manifest and Case-name golden unchanged.
- `.plans/API_BEHAVIOR_HINTS.md` unexplained item 2 marked resolved.

### HSM/CHASM evidence for the owner's Q2 (live, local cluster, commit c5f5e4eac8)
Every switched Case ran 3 times under each value (generated harness and the hand caller Queries). Server logs
confirm the path: under `hsm` the operation's tasks are `StateMachineOutbound`/`StateMachineTimer`, under `chasm`
`Chasm`/`ChasmPure`, in every subtest (`live-analysis.txt`).

| Case | hsm | chasm | Divergence |
| --- | --- | --- | --- |
| nexus-caller syncCompletion, asyncCompletion, asyncFailure, handlerError, startToCloseTimeout (generated + hand) | 3/3 each | 3/3 each | none |
| nexus-caller-retry (generated + hand) | 3/3 | 3/3 | none |
| nexus-control-forgedCompletion (Violated as expected) | 3/3 | 3/3 | none |
| nexus-caller-scheduleToStartTimeout, generated | 3/3 | 2/3 (1 Incomplete) | none: known race |
| nexus-caller-scheduleToStartTimeout, hand | 2/3 (1 Inconclusive) | 1/3 (1 Inconclusive, 1 Incomplete "nexus handler entrypoint completed without a reply") | none: known race |
| same two, extra diagnostic, defaults | generated 6/6 | generated 6/6 | |
| same two, diagnostic with `frontend.enableMatchingFanOutForPollCancellation=false` (uncommitted patch) | 10/10 | 10/10 | |
| nexus-operation cancelIsRequested, terminateSettles (standalone, now run once) | n/a | 3/3 | n/a |

The scheduleToStart failures hit both values, stop a worker, and vanish with the fan-out setting off: they are the
ShutdownWorker race MILESTONES records (#9424, Q1, task 6), not an implementation divergence.

**No Verdict diverged.** Why the retry Case agrees although `attempt` is counted differently: HSM increments when an
attempt completes (`service/history/hsm/nexusoperations/statemachine.go:91-95`), CHASM at schedule and reschedule
(`chasm/lib/nexusoperation/operation_statemachine.go:22,92`). During the first backoff, where the Case reads it, both
are 1. They differ before the first attempt completes (HSM 0, CHASM 1) and after the second attempt is scheduled
(HSM 1, CHASM 2); no current Query reads `attempt` there. So Q2 has no failing evidence today; the difference is real
in source and would surface only for a Query that reads `attempt` at those points.

Activity Cases (not switched, settings unchanged): 9/10 once; `activity-activityProtocol.terminateSettles` was
Incomplete after its 10 s worker stop, the same ShutdownWorker race signature.

### Decisions
- Equal values from two sources are allowed (kept once); only differing values are refused.
- `hsm` sets `nexusoperation.enableStandalone=false` because the upstream suite does; a workflow-Nexus Case requiring it
  true would be refused naming both sources (none does).
- The hand caller test keeps applying the switch value directly (no required settings in its fixtures).
- `scheduleActivity` still defaults its schedule-to-close from the instruction timeout: out of R2's scope (R11, task 10).
- Re-recorded the pinned Runs although their probes still passed, because each records the Profile-derived request field.

### Review
Round 1: SHIP (`flowctl claude impl-review --spec claude:claude-opus-5-5:high`, base eaaa8c58c9; log
`.flow/tmp/fn125-1/review-r1.log`). Reviewer claude-opus-5-5 at high is the same family as the writer (Opus 5.5).
Deferred P3: `formatSettingValue` handles float64/Duration/string beyond today's bool/int producers (kept: task 9's
duration assumptions will use them). FYI items: `scheduleActivity` defaulting (R11), `activityEnvironment` could reuse
`StandaloneSettings()`, the standalone-Case check keys on the `nexus-operation-` name prefix.

### Gates (head 084eac92f6)
Go tooling + Testpilot suite (`-p 2`, 397 s), `make umpire-check-cases` (199 s), `make lint-code-fast` (86 s): exit 0.
Logs: `gate-*.log`, `gates-summary.txt`; live: `live-*.jsonl`, `live-summary*.txt`, `live-analysis.txt`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c5f5e4eac8, 084eac92f6
- Tests: go test -tags test_dep -count=1 ./tests/testcore/testpilot/... ./common/testing/testpilot/temporal/... (exit 0), go test -json -count=1 -tags test_dep -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... ./tests/testcore/testpilot/... (exit 0), make umpire-check-cases (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), go test -json -count=3 -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases$/^nexus-' (exit 1: scheduleToStartTimeout/chasm 1 of 3 Incomplete, ShutdownWorker race; all else 3/3 both values), go test -json -count=3 -tags 'test_dep integration' ./tests -run '^TestTestpilotNexusCaller' (exit 1: ScheduleToStartTimeout hsm 1/3 and chasm 2/3 failed, ShutdownWorker race; all else 3/3 both values), go test -json -count=6 -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases$/^nexus-caller-scheduleToStartTimeout$' (exit 0), scheduleToStartTimeout generated+hand, count=5, enableMatchingFanOutForPollCancellation=false diagnostic patch, uncommitted (exit 0, 20/20), go test -json -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases$/^activity-' (exit 1: activityProtocol.terminateSettles Incomplete, ShutdownWorker race; 9/10 pass), pinned Runs re-recorded via UMPIRE_CONTROL_RECORD / UMPIRE_CANARY_RECORD live tests, receipt goldens and probe packages (exit 0), flowctl claude impl-review --spec claude:claude-opus-5-5:high (SHIP, round 1)
- PRs: