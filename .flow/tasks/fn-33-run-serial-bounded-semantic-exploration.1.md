---
satisfies: [R1, R3]
---

# fn-33-run-serial-bounded-semantic-exploration.1 Define the one-candidate canonical Case bridge
## Description
Define canonical `initialize`, `next`, `observe`, and `finish` frames. Each `next` carries one whole Lean-produced Case plus opaque candidate/lineage identity; each `observe` carries only the exact closed Run/Verdict result for the outstanding candidate.

**Size:** M
**Touches:** `model/Temporal/Tool/ExplorationBridge.lean`, `model/Temporal/Tool/ExplorationBridgeTests.lean`, `tools/umpire/campaign/bridge.go`

## Acceptance
- [ ] Candidate, Case, policy, seed, Profile/catalog, and Limit bindings are canonical and exact.
- [ ] Duplicate, stale, crossed, incomplete, and N+1 frames fail before preparation or coverage.
- [ ] Go sees no semantic-coordinate scoring or Case-family parameterization API.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
