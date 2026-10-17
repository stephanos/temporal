---
satisfies: [R2, R3, R4, R5]
---
# fn-193-compositions-are-plain-scala-members-as.2 Plain compositions in the DSL; fixtures, tests and docs rewritten

## Description
**DSL completion phase (owner, 2026-10-17).** Implement the form the prototype chose: `Composition` without a state type parameter, members as vals, syncs/own/synced by action reference, fn-190 claim sections and scenarios, the chosen derivation form; native runtime over the members' var snapshots. Rewrite the framework composition fixtures and DSL tests, SEMANTICS §17 and the README composition section. Leave nothing in the composition implementation that needs `RecordMachine`, `Owner`, `Declaring` or a retired `DerivedComposition`. The lifter/IR/Go half follows with fn-190.12.

**Size:** L

### Verification
- `mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests` passes; lint RuleSuite passes.
- Located refusals for a non-Machine member and a sync of an undeclared action (fixtures).

## Acceptance
- [ ] Compositions declare members as vals; no state case class, selectors or `->` pairing
- [ ] Composition fixtures/tests rewritten and passing natively; docs updated
- [ ] No composition code needs RecordMachine/Owner/Declaring

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
