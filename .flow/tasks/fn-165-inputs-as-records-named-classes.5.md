---
satisfies: [R1, R4]
---
# fn-165-inputs-as-records-named-classes.5 Migrate the Models to input records with byte-identical IR

## Description
**Note 2026-10-18 (one-language series):** Sweep proofs use the in-process IR (fn-201), fn-200's native lint and fn-203's Cases, not `model/ir` or `.lint.json`.

Migrate the Models to records, one action at a time, against the inventory (R1, R4).

**Size:** M
**Files:** every feature, foundation and actor file that declares an action with inputs or writes a class: `StandaloneActivity.scala`, the nexus feature files, `foundations/taskqueue/*.scala`, their product and protocol machines, Scenarios and witnesses, capability laws that name an input
**Touches:** [model/temporal/**/*.scala]

### Approach
- Per action in the inventory: declare its record with the recorded field order and defaults; replace `.input(token)` chains with `.inputs[In]`; rewrite `a(x := v)` as `a(x = v)`, `a()` stays; rewrite step functions from `(s, i1, i2, ...)` to `(s, in)` with `in.field` reads; delete the tokens the action no longer needs (a token shared with an unmigrated action stays until both are done).
- Single-input actions per the inventory keep `a(value)`.
- After each action: compile, framework and Model tests, lifter tests; after all: the step-table equivalence and a byte comparison of `model/ir/*.json` against the pre-migration tree.
## Acceptance
- [ ] Every action in the inventory is declared from a record or a single value; no Model file writes `:=` on a class or chains `.input(token)`.
- [ ] Every step function that reads inputs takes `(s, in)`.
- [ ] `model/ir/*.json` and `model/cases/*.json` are byte-identical to the pre-migration tree; the step-table equivalence passes.


## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
