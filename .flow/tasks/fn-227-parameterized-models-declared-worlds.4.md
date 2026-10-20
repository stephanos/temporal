---
satisfies: [R5]
---
# fn-227-parameterized-models-declared-worlds.4 After .3: timeouts, Heartbeat, Tick and asynchronous cancel on the activity Model with progress and bound claims

## Description
Add the four Option<Ticks> timeouts and asynchronous_cancel to the activity World, per-deadline budgets and cancel_requested to the state, Worker::Heartbeat and Deadline::Tick (decrements live budgets, expires the first exhausted one with the expired deadline as a fact; disabled with no live budget). Schedule-to-close spans attempts; start-to-close and heartbeat budgets reset on Poll. Asynchronous cancel sets cancel_requested in Started and the next worker response or heartbeat closes as Canceled. Claims: a started activity under a fair worker eventually closes; schedule-to-close bounds total ticks. Verified natively on the default world and by TLC on the larger worlds under UMPIRE_BACKENDS=require; native default world stays inside R30.

## Acceptance
- [ ] Timeouts, Heartbeat, Tick and asynchronous cancel modeled; world with no timeouts has no Tick edge
- [ ] Progress and bound claims verified natively and by TLC
- [ ] Goldens, cover test and controls updated; R30 budgets hold

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
