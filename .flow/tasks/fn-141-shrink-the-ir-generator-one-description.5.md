---
satisfies: [R7, R8, R9, R15]
---
# fn-141-shrink-the-ir-generator-one-description.5 Capture points and function lookup by span (proof for Part B)

## Description
Part B's early proof. The framework learns to record where a declaration is written and where a function is written. The lifter learns to lift the function found at a recorded span. Nothing is migrated yet: one fixture shows that a Property declared this way gives the IR the tree path gives.

**Size:** M
**Files:** new capture files in `model/umpire`, `model/umpire/*.test.scala`, `model/irgen/{Context,Expressions}.scala`, a new fixture under `model/irgen/testdata/lifts/`, `.plans/DSL_OPERATORS.md`
**Touches:** [model/umpire/**, model/irgen/**, .plans/DSL_OPERATORS.md]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Naming context: a context parameter the framework supplies from the enclosing `val` or object. It carries the name, the qualified path the Definition ID is made of, and file and line. Hold it to the lifter's `capturedName`, `qualifiedName` and `familyOf`, which define today's values.
- Recorded function: the function, its source span, and the values it closes over, by name. A def reference and a lambda both convert at the call site.
- These are the only `inline` or macro uses in the framework. Weigh `sourcecode` against a small macro of our own and record both line counts (fn-113 R25).
- Lifter: find the function whose tree has the recorded span, lift it under the liftable subset, and bind each captured value. A finite Model value becomes the IR value it is; a recorded function becomes a call. The spike of 2026-10-06 proved the capture and did not prove this lookup.
- Refuse a captured value that is neither, naming it and its type, and a span with no function.
- Revise rule 5 of `.plans/DSL_OPERATORS.md` to say where `inline` and macros are allowed (R15).

### Investigation targets
**Required:**
- `model/irgen/Context.scala` (`capturedName`, `qualifiedName`, `familyOf`, `definitionId`, `pos`)
- `model/irgen/Expressions.scala` (`lambda`, `function`, `parameters`, `forwardedDef`, `boundValues` use)
- `model/irgen/Constants.scala` and `Declarations.scala` (`literalValue`): a Model value as an IR value
- `model/umpire/Syntax.scala` (`Rules.on`, `codeOf`, `writtenAction`): the existing inline exception
- `.plans/DSL_OPERATORS.md` rule 5; fn-113 R15 and R25; `.plans/SCALA.md`, How Scala produces the IR

## Acceptance
- [ ] munit: name, ID and line are recorded from a `val`, an object, a helper def with the context, and a comprehension; a declaration with nothing to name it does not compile or is refused at its position.
- [ ] munit: a recorded function carries its span and captured values; a def reference and a lambda both convert.
- [ ] A fixture Property declared through the capture points exports the same IR as its tree-lifted twin.
- [ ] Reject fixtures: a captured value of no finite Model type; a function body outside the liftable subset is refused with today's text at today's position.
- [ ] The `sourcecode` weighing and both line counts are in the done summary.
- [ ] If the lookup or the binding cannot be made to work, stop: Part B does not continue (see the spec's Early proof point).

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
