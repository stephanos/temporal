---
satisfies: [R8, R9, R10, R13]
---
# fn-120-adopt-what-quint-does-well-named.4 Add IR explorer and semantic-level refusals

Touches: [tools/umpire/model/**, tools/umpire/cmd/**, model/lifter/**, model/SEMANTICS.md]

## Description
Implement Parts C and E after task 3's lint work settles the shared Go model/command surfaces, the choice schema and reader contract settle, and fn-112.10 closes its structural Case freeze. Fn-118's derived-wait work remains independently schedulable only where actual touched files are disjoint; shared lifter or generated-schema edits run serially.

**Size:** M
**Files:** Go model evaluator/command, lifter refusal fixtures, SEMANTICS.md.

### Approach
- Share one evaluator path between single commands and the interactive shell. Report bounded branch decisions with Scala positions.
- Name semantic levels and locate each reachable wrong-level expression; record type-impossible cases instead of inventing fixtures.

## Acceptance
- [ ] R8 lists starts, enabled classes and named outcomes; unknown names get useful refusals.
- [ ] R9 disabled reasons show bounded branch decisions at their Scala positions, including unbound classes.
- [ ] R10 single-command and interactive modes agree on a fixture.
- [ ] R13 level contract and reachable refusal fixtures are documented and checked.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
