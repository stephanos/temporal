---
satisfies: [R5]
---
# fn-128-close-the-activitys-precision-gaps.5 Adopted from the Go model: precedence Property, attempt counts, time-window note

## Description
**Batch:** ACTIVE approved source batch fn-128.1–.5 → fn-138.1–.3 → fn-129.1–.4 (MILESTONES.md). Production regeneration, exact artifact comparison, full Model/Go/lint/fixture/canary gates, fresh independent review and live run occur once at shared fn-128.6/fn-129.5. They are deferred here, not waived. Framework/lifter fixtures, focused munit and actual-current-source scratch lifting/admission/lowering/recording proofs run in this task. Commit this task on its own.

**Touches:** `model/temporal/features/activity/standalone/system/{System,Realization}.scala`, focused activity regression tests in `model/temporal/features/activity/standalone/`, direct observation/conformance/lowering tests in `tools/umpire/{conformance,lower}/`, necessary activity documentation. No DSL/schema/runtime expansion or production generated artifacts. The conductor owns MILESTONES and shared Flow lifecycle.

R5 (comparison section 5 items 5–7). (1) A checked Property `cancelIsNotUndone`: cancellation remains requested or legally closes; no pause undoes it (reset joins in fn-129). (2) A typed raw `attemptCount` observation read finally in every activity Case through all four realizations, distinct from correlated retry evidence and from the independent Model's delivered-attempt counter. Preserve the all-or-none confirming groups. (3) The Go model's nominal vs real time window note (`model.go:300–310`) as `because` text on all seven timer/deadline branches. Record R6 source changes and deferred Program/identity/artifact changes.
## Acceptance
- [ ] `cancelIsNotUndone` is checked and holds.
- [ ] Every activity Case reads the attempt count.
- [ ] The deadline rules carry the time-window `because`.
- [ ] Changes listed for R6; the spec's Verification gates pass.

## Done summary
# fn128.5 integrated source verification

The cancellation-precedence Property, final raw attempt-count reads in all four
activity Realizes declarations, and seven nominal-time-window explanations are
integrated at original commit6ba93a5c02906d7eca20864f523c8d0594dc95e3.
Base13cfbc0e1b0296ac28bf536ad0f51f04e59d4ab7; original fast-forward, no rewrite.
Worker summary and complete provenance remain in fn1285-summary.md and
fn1285-evidence.json. Both main and isolated source inputs are identical.

Conductor reran the current-source precision proof and 13-Case lower/Prepare
inventory on the integrated target. Both exited0. Logs:
fn1285-integrated-precision.log and fn1285-integrated-inventory.log.
Cancellation verify is exercised within8, expanded883/explored918/holes0; the
seeded undo produces a replayed three-action counterexample. Behavioral tables,
named choices and unknown counts remain equal; only seven timer explanations
change. Product9/59/97; System and TimeoutRetry2376/63/25488; unknown0.

All13 supported Cases have exactly one final typed ActivityExecutionInfo read:
activitySystem9/timeoutRetry2/heldDispatch1/lostStartAnswer1. Prior controller
prefixes, Contracts and expected Runs remain equal. Failure1/poll2 and
timeout1/poll2 confirming groups remain strict. Public scheduling count is raw
API data, not equality with the independent Model's delivery count.

Scope-valid worker evidence is reused: focused Activity25, lifter4, format,
current-source scratch lift, selected recording/replay4 (public1/delivered0 and
public2), and checked-IR regression6. The checked-IR regression is NOT proof of
the new source. No production IR/Cases or external planning files were changed.

Two full six-Case recording probes FAILED, identically before and after this
task. Neither is passed or waived: nonRetryableFailure has never_evaluated
ambiguity, and pauseResume hits correlated per-event work ceiling4m at sequence21.
Their exact diagnostics, cause inventory and failed logs are retained in the
worker handoff. They are required shared-batch close fixes. Race fake Describe
data also needs its final raw observation populated at regeneration.

This completion records R5 SOURCE work only under the approved batch contract.
Production regeneration, exact artifact/identity/R6 comparisons, free Queries,
full Model/Go/lint/fixture/canary gates, fresh independent implementation and
completion reviews, and live proof remain mandatory at fn128.6/fn129.5.
No parent spec or goal closes; real conformance disagreements require a human,
never fitting the independent Model to the server.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6ba93a5c02906d7eca20864f523c8d0594dc95e3
- Tests: Integrated: /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- go run -tags test_dep .flow/tmp/activity-batch/fn1285-precision-proof.go .flow/tmp/activity-batch/fn1285-ir/activity-standalone.json .flow/tmp/activity-batch/fn1285-before-ir/activity-standalone.json; exit0; fn1285-integrated-precision.log, Integrated: /usr/bin/flock /tmp/umpire-heavy-gates.lock timeout 600s mise exec -- go run -tags test_dep .flow/tmp/activity-batch/fn1285-observation-inventory.go .flow/tmp/activity-batch/fn1285-ir .flow/tmp/activity-batch/fn1285-before-ir; exit0; fn1285-integrated-inventory.log, Reused exact-input worker evidence: Activity25, lifter4, package/lift, scoped format, selected current-source recording/replay4 and checked-IR regression6; full commands/provenance in fn1285-evidence.json, FAILED before and after source: full six-Case current-source recording/replay; nonRetryableFailure never_evaluated and pauseResume work-ceiling at sequence21. Required batch-close follow-ups; not passed or waived., DEFERRED approved batch: production regeneration, R6 artifacts/identities, free Queries, full Model/Go/lint/fixture/canary gates, independent reviews and live proof at fn128.6/fn129.5
- PRs: