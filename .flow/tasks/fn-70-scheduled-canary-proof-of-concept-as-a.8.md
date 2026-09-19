---
satisfies: [R1, R5, R6, R8]
---
# fn-70-scheduled-canary-proof-of-concept-as-a.8 Reconcile owned Schedules and expose explicit operator control and status

## Description
Reconcile owned Schedules and expose explicit operator control and status.

**Size:** M
**Files:** tools/canary/schedule.go; tools/canary/schedule_test.go; tools/canary/status.go; tools/canary/status_test.go; tools/canary/cli.go; tools/canary/cli_test.go; tools/canary/cmd/canary/main.go; tests/canary_schedule_test.go
**Touches:** [tools/canary/schedule.go, tools/canary/schedule_test.go, tools/canary/status.go, tools/canary/status_test.go, tools/canary/cli.go, tools/canary/cli_test.go, tools/canary/cmd/canary/main.go, tests/canary_schedule_test.go]

### Approach
- Expose list/apply/worker/pause/resume/status with explicit installation/config/selection and bounded operations. Use stable installation/check IDs and versioned ownership manifest; Describe and update callback verify owner and expected action/artifact/binding/policies. Full selection/conflict validation precedes mutations; stage new schedules paused, report partial network outcomes accurately.
- Apply same selection is idempotent and preserves operator/failure pauses; empty/removal pauses only owned schedules. Explicit changed artifact/binding requires update intent; resume revalidates exact resources/config and explicitly acknowledges reconciliation. Neither list/status/worker startup nor reporting creates or resumes schedules.
- Use SDK ScheduleOptions Spec.Intervals Every60s, Overlap SCHEDULE_OVERLAP_POLICY_SKIP, CatchupWindow10s, PauseOnFailure=true and bounded Workflow action from task7. Dedicated local CHASM configuration from task3 is required; verify effective routing/pause behavior, not an unchecked config assertion. No SDK default reliance or manual workaround for direct termination.
- Status combines schedule/action/orchestration observations and retained references; report missing/stale/unreachable separately from Verdict, freshness3minutes by default. Bounded shutdown pauses owned scheduling before cancellation/drain; unavailable server yields uncertain failure, no claimed pause. Never touch unrelated resources.
- Add TestCanaryOwnedScheduleLifecycle: create/reapply/conflict/remove/resume, canceled/terminated automatic pause, explicit10s Describe, slow>60s no overlapping/buffered tick, catch-up boundary and partial failures. Tests may use controlled short intervals for isolated policy cases but must retain explicit60s configuration assertions and task9 real cadence.

### Investigation targets
**Required:**
- tools/tdbg/app.go:45 — CLI pattern.
- tests/schedule_test.go:454,3149 — Describe/catch-up and SDK Create.
- common/dynamicconfig/constants.go:3164 — CHASM settings.
- chasm/lib/scheduler/config.go:60 — dedicated namespace policy.

### Quick commands
`mise exec -- go test -count=1 -tags test_dep ./tools/canary/...`
`mise exec -- go test -race -count=1 -tags test_dep ./tools/canary/...`
`mise exec -- go test -json -count=1 -tags test_dep,integration ./tests -run '^TestCanaryOwnedScheduleLifecycle$' -timeout 10m > /tmp/fn70-task8-live.jsonl`
Then verify both terminal exit 0 and JSON records: each exact named top-level test has Action=run and Action=pass, no matching test/subtest Action=skip or fail, and package pass. Parse with a bounded script; a missing record is failure. This command names a required new test, not a currently existing one.

### Execution constraints
Read native fn70 R1–R10 and repository guides. Preserve comments and unrelated dirty source. Do not stage, commit, or push; the user owns commits. Fn77 Producer edits finish first for source serialization only: re-anchor exact delivered source/artifact before fn70 edits, without introducing a semantic prerequisite. No fn79 operation cancellation, replacement Driver/evaluator, runtime Lean invocation, general recovery/lease framework, production deployment/config mutation, or fn29 machinery. Proposed new file owners may reuse an established equivalent; record actual paths. Run Lean jobs serially. Baseline existing focused tests before edits; new named tests apply after creation and must be wired into the actual package/module roots. No silent skips, unmatched test regexes, fixture expectation weakening, or inferred passing gates.

## Acceptance
- [ ] Invalid complete selections/conflicts cause no activation; apply is idempotent and removal affects owned schedules only.
- [ ] Actual60s/Skip/10s policy and disabled retries are verified; slow/catch-up cases do not buffer uncontrolled work.
- [ ] Canceled/terminated/uncertain failure pauses automatically under qualified local policy; resume is explicit and reapply cannot resume.
- [ ] CLI status/shutdown preserve uncertainty, freshness, ownership and separate semantic/operational outcomes under partial failure.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
