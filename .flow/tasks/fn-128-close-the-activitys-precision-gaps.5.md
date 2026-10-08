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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
