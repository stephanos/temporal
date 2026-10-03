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
- Add accept, disabled, stay, finite membership, implication and fact-recording declarations that lower directly to existing IR, following the operator policy in the spec's Decision Context (`.plans/DSL_OPERATORS.md`): words, no new symbol, no `inline`/macro, each a plain `def`/`extension` the lifter matches by name.
  - `a implies b`: `extension (a: Boolean) infix def implies(b: => Boolean)`, lowered to `or(not a, b)` so the by-name right side is read only when the left holds (a hole in `b` is not reached when `a` is false). Alphabetic infix has the lowest precedence, so `a == b && c implies d` reads as intended. No `iff`, no `and`/`or`/`not`.
  - `phase.in(a, b, c)`: `extension [A](a: A) def in(first: A, rest: A*)`, dotted (never infix), lowered to `OP_CONTAINS(a, list(first, rest...))`, the `a in b` SEMANTICS.md already defines. The signature forbids `x.in()`.
  - `step.records(fact)`: dotted like `contains`, lowered to `OP_CONTAINS(fact, field(step, facts))`, exactly what `s.facts.contains(f)` lifts to today. This is the definition fn-112.4's composition form `after.records(_.member, fact)` shares.
- Refuse rebind of an unbound action, extend of a bound action, duplicate assumptions, incompatible refinement state/map types, duplicate bindings and cyclic aliases with located diagnostics. `in()` with no member is a compiler refusal by signature.
- Prove derived Definition IDs and complete tables against the task-1 archive.
## Acceptance
- [ ] rebind, extend, refining, assuming and unmonitored lift without a new IR field and preserve all untouched machine metadata and ordering.
- [ ] accept/because, disabled, stay, in, implies and records lower to the existing step/expression IR: `implies` to `or(not a, b)` with a by-name right side, `in` to `OP_CONTAINS` over a list literal, `records` to `OP_CONTAINS` over the step's facts; no new IR node.
- [ ] `implies` is an `infix` extension of `Boolean`; `in` and `records` are dotted, `in` takes at least one member by its signature; none is `inline`/`transparent inline`; no `and`/`or`/`not`/`iff` word is added and no new symbolic operator appears.
- [ ] Positive and refusal fixtures cover every new form, refinement replacement, assumption append and invalid binding case, including a lifting fixture for each of `implies`, `in` and `records` and the short-circuit case where the right side of `implies` is not read.
- [ ] Full tables, IDs, fingerprints and Query answers equal the original baseline under the narrow R1 projection.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
