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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
