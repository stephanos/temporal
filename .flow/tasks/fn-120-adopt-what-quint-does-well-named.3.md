---
satisfies: [R5, R6, R7]
---
# fn-120-adopt-what-quint-does-well-named.3 Add model lint after the Scala root inventory stabilizes

Touches: [tools/umpire/model/**, tools/umpire/cmd/**, model/gate/**, model/ir/**, model/README.md]

## Description
Implement Part B against fn-114's final Scala-owned IR roots and fn-120.2's named alternatives. Inventory findings only after those roots are stable; do not accept a finding merely because an earlier inventory named it.

**Size:** M
**Files:** Go IR reader/checker, lint command and fixtures, gate integration, checked-in finding acceptances.

### Approach
- Reuse reader tables, Query and realization indexes rather than adding a second evaluator.
- Emit the R5 finding kinds with locations, fixtures for presence and absence, and reasoned checked-in acceptances.
- Fail the gate for new findings and stale acceptances.

## Acceptance
- [ ] R5 kinds report stable kind, machine, message and Scala location; malformed IR produces reader errors only.
- [ ] R6 each kind has positive and negative fixtures.
- [ ] R7 gate covers every checked-in IR root; first-run findings, fixes and accepted reasons are recorded.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
