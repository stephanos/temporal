---
satisfies: [R5]
---

# fn-33-run-serial-bounded-semantic-exploration.5 Prove deterministic exploration and regression independence
## Description
Run identical checked inputs, seed, limits, decisive observations, and terminal reason twice and compare candidate order plus canonical semantic summary. Prove wall-clock timing can change only the completed prefix and that pinned regressions consume no campaign limit.

**Size:** M
**Touches:** `tools/umpire/campaign/integration_test.go`, `model/Temporal/Tool/ExplorationBridgeTests.lean`

## Acceptance
- [ ] Deterministic campaigns produce identical candidate identities, coverage, and semantic summaries.
- [ ] A 10x candidate input is bounded by admission/campaign limits and retained state stays constant-size per active candidate.
- [ ] Focused Lean and Go tests pass using `-tags test_dep` where applicable.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
