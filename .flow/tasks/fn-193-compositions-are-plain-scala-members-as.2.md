---
satisfies: [R2, R3, R4, R5]
---
# fn-193-compositions-are-plain-scala-members-as.2 Plain compositions in the DSL; fixtures, tests and docs rewritten

## Description
**DSL completion phase (owner, 2026-10-17).** Implement the form the prototype chose: `Composition` without a state type parameter, members as vals, syncs/own/synced by action reference, fn-190 claim sections and scenarios, the chosen derivation form; native runtime over the members' var snapshots. Rewrite the framework composition fixtures and DSL tests, SEMANTICS §17 and the README composition section. Leave nothing in the composition implementation that needs `RecordMachine`, `Owner`, `Declaring` or a retired `DerivedComposition`. The lifter/IR/Go half follows with fn-190.12.

**Size:** L

### Execution notes from the accepted prototype

- Follow model/design/plain-composition-prototype.md's adopted spelling and its fixture migration map. Reuse the prototype's member/schema and sync work where appropriate, but generalize the existing on-demand NativeCheck.Search rather than copying the prototype's separate breadth-first explorer or introducing a synthetic RecordMachine adapter.
- R3 includes progress, monitors, scenarios, starts, fairness, witnesses and derivation-owned claims; the prototype's omitted forms are implementation work here, not a reason to narrow the acceptance gate. Preserve on-demand budget/cut behavior, vacuity and pinned-path ordering.
- R2 admits Machine and Composition members. A nested composition must therefore step and expose its member state and tagged facts correctly; accepting it at compile time and later refusing it as having no vars is the prototype's gap, not the final implementation.
- Preserve the existing fixture assertions through the plain-form rewrites, including reordered facts, derived end predicates, replaced-member claims and refused undeclared actions. Where the accepted prototype refuses a formerly written shape (repeated/shared vars or an unrelated replacement), retain the corresponding semantic control on a valid replacement fixture and add the refusal control.
- Widen the existing claim/target lookup for compositions rather than maintaining parallel copies of the claim words. Remove composition dependencies on RecordMachine, Owner and Declaring, leaving held export/IR adapters for fn-190.12.
- Keep framework/REFERENCE.md user-facing when updating the author surface: no Scala implementation/test filename citations or source/fixture columns, and preserve public-surface coverage.

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
