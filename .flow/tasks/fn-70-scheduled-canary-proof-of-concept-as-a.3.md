---
satisfies: [R5, R8]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.3 Honor CHASM canceled and terminated failure policy for dedicated local canaries

## Description
Honor CHASM canceled and terminated failure policy for dedicated local canaries.

**Size:** M
**Files:** chasm/lib/scheduler/scheduler.go; chasm/lib/scheduler/pause_on_cancel_test.go; chasm/lib/scheduler/scheduler_nexus_completion_test.go; tests/canary_schedule_policy_test.go; tests/schedule_test.go
**Touches:** [chasm/lib/scheduler/scheduler.go, chasm/lib/scheduler/pause_on_cancel_test.go, chasm/lib/scheduler/scheduler_nexus_completion_test.go, tests/canary_schedule_policy_test.go, tests/schedule_test.go]

### Approach
- Make CHASM pause classification consult its existing namespace-filtered Tweakables.CanceledTerminatedCountAsFailures through tweakablesFromContext. Preserve false default and existing failed/timed-out semantics; do not alter V1 static policy or general scheduling behavior.
- Unit-test absent/default/explicit false and true for canceled and terminated completion, PauseOnFailure false, normal failure/timeout/success, conflict token update and no new buffered dispatch after pause.
- Add TestCanaryScheduleCanceledTerminatedPause with isolated local testcore environment: EnableChasm=true, EnableCHASMSchedulerSentinels=true, EnableCHASMSchedulerCreation=true, CHASMSchedulerCreationRolloutPercent=100, EnableCHASMSchedulerRouting=true, migration disabled, and scheduler.CurrentTweakables copied from defaults with CanceledTerminatedCountAsFailures=true. Scope configuration to test/dedicated namespace, never production files.
- Live-test both direct cancellation and termination while worker unavailable, automatic pause, no subsequent interval action and explicit resume; false-policy control remains unpaused. Capture effective behavior; SDK options alone do not enforce this. Reuse scheduleCommonOpts/newScheduleEnv and qualified CHASM setup rather than invent fixtures.

### Investigation targets
**Required:**
- chasm/lib/scheduler/config.go:18,43,60 — existing flag and namespace accessor.
- chasm/lib/scheduler/scheduler.go:606,669 — current predicate ignores flag.
- chasm/lib/scheduler/pause_on_cancel_test.go:14 — default behavior.
- tests/schedule_test.go:65,305,3142 — fixture and cancellation/creation patterns.
- common/dynamicconfig/constants.go:3164 — exact creation/routing settings.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./chasm/lib/scheduler`
`mise exec -- go test -json -count=1 -tags test_dep,integration ./tests -run '^TestCanaryScheduleCanceledTerminatedPause$' -timeout 10m > /tmp/fn70-task3-live.jsonl`
Then verify both terminal exit 0 and JSON records: each exact named top-level test has Action=run and Action=pass, no matching test/subtest Action=skip or fail, and package pass. Parse with a bounded script; a missing record is failure. This command names a required new test, not a currently existing one.

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] Default/false behavior remains unchanged; true makes canceled and terminated executions count as pause failures.
- [ ] Live dedicated CHASM configuration proves both automatic pauses and no subsequent tick until explicit resume.
- [ ] PauseOnFailure false and existing failure/success behavior remain correct; conflict-token and unit regressions pass.
- [ ] Configuration is local/test-only and uses existing toggle; no manual-notice workaround, recovery service or production mutation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
