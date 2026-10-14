---
satisfies: [R5]
---
# fn-180-gomad-replay-time-instrumentation-for.4 Invariant bisect over exact replays

## Description
Given a registered invariant check, `gomad replay` binary-searches the stop point for the first choice index at which the check fails, using repeated exact instrumented replays; report already-failing-at-first-choice and never-fails outcomes as such.

## Acceptance
- [ ] Bisect finds the first failing choice on a synthetic fixture with known answer
- [ ] Already-failing and never-failing outcomes reported (tests)

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
