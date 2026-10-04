---
satisfies: [R3]
---
# fn-124-shrink-and-simplify-the-umpire-go.3 Declare activity attempts, start carriers, lost admissions, causal parents, timeouts and history event names in the realization

## Description
Implements R3. Cross-spec entry gate: start after fn-118 lands (it edits waits, timeouts and realization surfaces). Each Temporal fact listed in R3 moves to one declaration in the realization (model/temporal/realize or the feature's Realization.scala), is lowered into the Case, and the Go runtime, lowering and conformance read the declared value instead of their own rule; remove the Go copies. Case bytes may change only by the new declared fields; record each change. Run the live generated Cases once.
## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
