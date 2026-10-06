---
satisfies: [R1]
---
# fn-128-close-the-activitys-precision-gaps.1 Dispatch is a field: replace the backingOff phase; start delay

## Description
R1 (comparison P1-1). Current source after fn-132.2: `model/temporal/features/activity/standalone/Standalone.scala` owns the form; `product/Product.scala` and `system/System.scala` each own their local `Phase`, `State` and `Fact`. Actions are `client.start` and `client.control` on the `activity` entity. All three System realizations remain in `system/Realization.scala`. The kind header `features/activity/Activity.scala` is package-only; fn-132.6 owns the later shared-declaration extraction. IR stems are `activity-standalone`, `activity-standalone-record` and `activity-standalone-race`. Machine names stay `ActivitySystem` and `ActivityProduct`; worker actions stay `worker.poll`/`respond`.

`system.State` gains `dispatch: Dispatch` (`now`, `startDelay`, `backoff`) and loses the `backingOff` phase. `client.start` gains a `startDelay: Timeout` input; `timers.startDelay` sets `dispatch = now`. A retryable failure sets `dispatch = backoff` and stays `scheduled`; `timers.backoff` sets `now`. Pause and unpause keep `dispatch` (`chasm/lib/activity/model/model.go:239-244`). `worker.poll` is enabled only while `dispatch = now`. Schedule-to-start is armed only while dispatchable and waiting (`model.go:312-322`). The refinement reads every waiting state as the Product's `scheduled`, whatever `dispatch` holds.

The realization sets the start request's start delay when the input is `expires`. Recompute the state count and any Limits/`total` the change moves.

Add a Query or fixture per divergence fixed: an unpause after a pause in backoff does not dispatch before the backoff timer; schedule-to-start does not fire while backing off.
## Acceptance
- [ ] No `backingOff` phase remains; `dispatch` is a `system.State` field and the refinement checks.
- [ ] Each of the two divergences is a Query or fixture that now refuses what the server refuses, citing `model.go`.
- [ ] `startDelay` is a start input with its timer, realized in the start request.
- [ ] Every table, Query answer and Case that changed is listed with why (for R6).
- [ ] The spec's Verification gates pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
