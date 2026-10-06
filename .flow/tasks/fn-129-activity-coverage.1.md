---
satisfies: [R1, R5]
---
# fn-129-activity-coverage.1 Heartbeat: worker.heartbeat and a retryable heartbeat timeout

## Description
**Batch:** deferred Model batch (see MILESTONES.md, Deferred, fn-128). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at that batch's single regeneration against its baseline (the tree at the DSL batch's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
R1, R5 (comparison P2-6). Current source after fn-132.2: `model/temporal/features/activity/standalone/Standalone.scala` owns the form; `product/Product.scala` and `system/System.scala` each own their local `Phase`, `State` and `Fact`. Actions are `client.start` and `client.control` on the `activity` entity. All three System realizations remain in `system/Realization.scala`. The kind header `features/activity/Activity.scala` is package-only; fn-132.6 owns the later shared-declaration extraction. IR stems are `activity-standalone`, `activity-standalone-record` and `activity-standalone-race`. The retry policy is fn-128's.

A `worker.heartbeat` action, enabled while the worker holds the attempt and disabled elsewhere (a heartbeat on a closed activity follows fn-128's rejection rows). A `deadline.heartbeat` armed by a `heartbeat: Timeout` start input; its timeout is retryable under fn-128's retry policy (`retriesRemaining`). An observation of heartbeat details if the realization can read them (Describe); otherwise record why not.

Realize the heartbeat (RecordActivityTaskHeartbeat from the Case's worker) and the start input, with at least one live Query (e.g. a heartbeat timeout that retries and completes).
## Acceptance
- [ ] `worker.heartbeat` and `deadline.heartbeat` exist; the timeout retries while attempts remain.
- [ ] At least one live Query is realized and its Case ran once.
- [ ] Heartbeat details are observed, or the reason they are not is recorded.
- [ ] New Cases listed (for R5); the spec's Verification gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
