---
satisfies: [R4]
---

# fn-22-deterministic-replay-semantic.3 Compile ordered semantic reductions into complete Cases
## Description
Add a Lean-owned finite reduction protocol for the first negative Case. Enumerate applicable Producer-authored Program edits in fixed order and compile each accepted edit into a complete canonical Case while keeping the Contract unchanged and semantic coordinates opaque to Go.

**Size:** L
**Touches:** `model/Temporal/Tool/ReplayReduction.lean`, `model/Temporal/Tool/ReplayReductionTests.lean`, `tools/umpire/replay/bridge.go`

## Acceptance
- [ ] Lean owns applicability, ordering, candidate identity, and Case compilation.
- [ ] Invalid, duplicate, stale, crossed, and oversized candidates reject before preparation.
- [ ] Go exposes no Case mutation or semantic-coordinate editing API.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
