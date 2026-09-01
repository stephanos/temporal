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
- Run the same checked Space, policy, PlannerPolicy, environment, candidate Limit, admitted Result/outcome stream, and terminal stop reason twice and compare the serial candidate sequence and canonical terminal encoding.
- Prove runner timing cannot change semantic identities, order, or canonical encoding for the same admitted stream, while wall-clock scheduling may change the terminal prefix and stop reason.
- Exercise a pinned Regression through its existing ordinary path and prove it consumes no exploration Limit.
- Cover one successful uncovered-coordinate campaign and one truthful Limit Reached campaign.

### Investigation targets
**Required** (read before coding):
- Completed tasks `.1` through `.4`.
- Existing ordinary Regression execution path.
- `CallerClosureFault` baseline/duplicate-delivery proof fixtures and their existing runner and Run Evaluation bindings.

## Acceptance
- [ ] Identical checked inputs, candidate Limit, admitted Result/outcome stream, and terminal stop reason produce the same candidate identities and canonical terminal encoding; wall-clock scheduling is excluded from the equality claim.
- [ ] The end-to-end fixture executes both candidates in the exact runnable two-choice Space and rejects the basic-lifecycle VariationSpace before runner preflight.
- [ ] Pinned Regressions remain outside the campaign and its exploration Limit.
- [ ] Focused Lean and Go end-to-end fixtures pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
