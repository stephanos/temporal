---
satisfies: [R5]
---
# fn-33-run-serial-bounded-semantic-exploration.5 Prove deterministic serial exploration and pinned independence

## Description
Prove the complete bounded campaign preserves fn-17 selection and ordinary Regression ownership.

**Size:** M
**Files:** `tools/umpire/campaign/integration_test.go`, `model/Temporal/Tool/ExplorationBridgeTests.lean`
**Touches:** [tools/umpire/campaign/integration_test.go, model/Temporal/Tool/ExplorationBridgeTests.lean]

### Approach
- Run the same checked Space, policy, PlannerPolicy, environment, and Limits twice and compare the serial candidate sequence and terminal output.
- Prove runner timing and progress rendering cannot change semantic identities or order.
- Exercise a pinned Regression through its existing ordinary path and prove it consumes no exploration Limit.
- Cover one successful uncovered-coordinate campaign and one truthful Limit Reached campaign.

### Investigation targets
**Required** (read before coding):
- Completed tasks `.1` through `.4`.
- Existing ordinary Regression execution path.
- Fn-17 Nexus proof fixtures.

## Acceptance
- [ ] Identical checked inputs and Limits produce the same candidate identities and terminal output.
- [ ] Pinned Regressions remain outside the campaign and its exploration Limit.
- [ ] Focused Lean and Go end-to-end fixtures pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
