---
satisfies: [R2, R5, R16]
---
# fn-112-make-the-standalone-activity-scala.3 Add machine derivation and canonical step helpers

Touches: [model/umpire/Machine.scala, model/umpire/Domain.scala, model/lifter/Declarations.scala, model/lifter/Expressions.scala, model/lifter/test/**, model/lifter/testdata/**]

## Description
Implement the framework/lifter surface for behavior-preserving machine derivation and compact step expressions.

**Size:** M
**Files:** model/umpire machine/step DSL; model/lifter Declarations.scala and expression lifting; dedicated positive/refusal fixtures.

### Approach
- Follow existing restrict-copy lifting for rebind, extend and unmonitored. Preserve starts, ends, results, assumptions, monitors and refinement metadata; replace only bound actions, add only unbound actions and retain binding order.
- Add `refining(product)(map)` to replace only the source product/map while preserving visible facts/outcomes, and `assuming(as*)` to append unique assumptions in declaration order.
- Add accept, disabled, stay, finite membership and implication declarations that lower directly to existing IR.
- Refuse rebind of an unbound action, extend of a bound action, duplicate assumptions, incompatible refinement state/map types, duplicate bindings and cyclic aliases with located diagnostics.
- Prove derived Definition IDs and complete tables against the task-1 archive.

## Acceptance
- [ ] rebind, extend, refining, assuming and unmonitored lift without a new IR field and preserve all untouched machine metadata and ordering.
- [ ] accept/because, disabled, stay, in and implies lower to the existing step/expression IR.
- [ ] Positive and refusal fixtures cover every new form, refinement replacement, assumption append and invalid binding case.
- [ ] Full tables, IDs, fingerprints and Query answers equal the original baseline under the narrow R1 projection.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
