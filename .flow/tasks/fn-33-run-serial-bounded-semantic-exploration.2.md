---
satisfies: [R1]
---

# fn-33-run-serial-bounded-semantic-exploration.2 Bind Lean-owned Case selection and coverage
## Description
Connect the bridge to the retained fn-17/fn-40 exploration authority. Lean selects and compiles each complete Case, validates the returned decisive Run/Verdict observation, owns semantic coverage, and determines finite exhaustion.

**Size:** M
**Touches:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`

## Acceptance
- [ ] At most one checked Case is returned and selection order is deterministic for fixed inputs and seed.
- [ ] Only completed decisive Run/Verdict values with successful cleanup may update coverage.
- [ ] Inconclusive, lost, cleanup-uncertain, duplicate, or crossed work receives no coverage.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
