---
satisfies: [R1]
---
# fn-33-run-serial-bounded-semantic-exploration.2 Bind Lean-owned candidate selection and coverage

## Description
Connect the bridge to fn-17 so Lean alone chooses each candidate and owns semantic coverage.

**Size:** M
**Files:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`
**Touches:** [model/Temporal/Tool/ExplorationBridge.lean, model/Temporal/Tool/ExplorationBridgeTests.lean]

### Approach
- Initialize fn-17 from one checked Space, one retained policy, fn-40 PlannerPolicy, and explicit Limits.
- Return at most one checked v2 ExperimentSpec from `next` and keep coordinate ordering and exhaustion opaque to Go.
- Admit only the exact checked observation for the outstanding candidate before advancing.
- Test exhaustive and uncovered-coordinate policies at the bridge seam.

### Investigation targets
**Required** (read before coding):
- Fn-17 public facade and task `.8` session API.
- Fn-40 canonical PlannerPolicy constructors and default seed.
- Parent spec `API Contracts`.

## Acceptance
- [ ] The bridge delegates selection and semantic coverage exclusively to fn-17.
- [ ] At most one candidate is returned and no Go-visible coordinate-scoring API exists.
- [ ] Focused policy, binding, and Limit tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
