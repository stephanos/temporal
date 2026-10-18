---
satisfies: [R1, R2, R3, R7]
---
# fn-165-inputs-as-records-named-classes.3 Lifter: classes from record construction, in.field reads, record refusals and fixtures

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

The lifter side: lower classes from record construction, lift `in.field` in function bodies, and refuse what records allow but the DSL does not (R1, R2, R3, R7).

**Size:** M
**Files:** `model/irgen/Declarations.scala` (action and input declarations, class lowering), `model/irgen/Expressions.scala` (function-body reads), `model/irgen/Syntax.scala` (the `Assigned` matcher), `model/irgen/Types.scala` (`firstValue`), `model/irgen/testdata/lifts/*.scala` and `expected/*.json`, reject fixtures
**Touches:** [model/irgen/*.scala, model/irgen/testdata/**]

### Approach
- Class lowering: a call `a(field = v, ...)` is an `Apply` with `NamedArg` children; each omitted field's value is the case class's parameter default (`$default$n` methods in TASTy, which must be enum cases or literals, R2). `a()` is all defaults; `a(In(...))` is the record's constructor call read the same way. After fn-141 Part B, classes export by running and this lowering reduces to reading the framework's `Class` value; write it so that the function-body part survives Part B unchanged.
- `Param` names and types from the record (field labels and types), replacing the token symbols; delete `inputTokens`, `tokens.indexOf` and the `Assigned` matcher.
- Function bodies: `in.field` lifts as a parameter read, as `s.field` does. Refuse `in.copy`, a pattern match on `In`, and a method call on `In`, with the state's wording.
- Refuse a positional call with two or more inputs (an argument that is not a `NamedArg`), at the call, naming the action; refuse a computed default at the record, naming the field.
- Fixtures: a record action, a single-input action and an action with no inputs in one lift fixture; a record whose default is not its domain's first value; a structural-bound consumer; reject fixtures for positional, computed default and `copy` in a body.
## Acceptance
- [ ] Empty, named and value classes lower to the same `Class` IR the token form produced; the lift fixture's expected IR is unchanged apart from the new fixtures.
- [ ] `Param` names and types come from the record; the token-identity sites in the IR generator are deleted.
- [ ] `in.field` lifts as a parameter read; `copy`, pattern matching and method calls on an input record are refused in function bodies with the stated text at the position.
- [ ] A positional call with two or more inputs and a computed default are refused at the call and at the record.
- [ ] The eight fixtures of R7 exist and pass.


## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
