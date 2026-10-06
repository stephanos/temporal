---
satisfies: [R5, R7]
---
# fn-135-effect-and-is-blocks-for-effects-and.2 Lift is { } blocks and val section members

## Description
Teach the lifter to resolve section members declared as `val`s and to lift `is { }` blocks, leaving every `def` path untouched. This is the early proof point: the predicate paths (rule conditions, `in`, capability fields, `end`) are where `val` resolution is riskiest. Effects follow in .3.

**Size:** M
**Files:** `model/irgen/Context.scala`, `model/irgen/Expressions.scala`, `model/irgen/Declarations.scala`, `model/irgen/Capabilities.scala`, `model/irgen/Claims.scala`, `model/irgen/Syntax.scala`, new fixture `model/irgen/testdata/lifts/Blocks.scala` with `expected/` output, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/irgen/*.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- The single gate is `Context.isFunction` (`model/irgen/Context.scala:48`, true only for `DefDef`). Write ONE recognizer for a section `ValDef` whose right-hand side is an `is { }` (or, in .3, `effect { }`) call, register it as a function named by the val's `fullName` (`functionName` already uses `sym.fullName`, identical for val and def), and have every site below consult that recognizer rather than deciding per site. Every `DefDef` branch stays byte-for-byte.
- Route each caller of `isFunction` / `forwardedDef` (`Expressions.scala:112-129`) through the recognizer: rule conditions and `InSet` (`Declarations.scala` ~590-610), `Capabilities.checked` (`Capabilities.scala:457`, the "names a def" refusal) and `etaDef` (:580-582), claims (`Claims.scala:392`, :405, :519-522), realizations (`Realizations.scala:103-127`). `end` stays a `def` and calls the val; check that `endOf` (`Declarations.scala:214`) lifts `states.over(s)` with a val `over`.
- Lift the block body through `contextBody` (`Declarations.scala:733-742`), which already unwraps a context function with given parameters. The block has no `s: S` parameter, so synthesize the IR parameter (name `s`, the machine's state type) rather than going through `function(name, params, …)` (`Expressions.scala:277`), which takes `ValDef`s; the result must equal the method form's function.
- Accessors: a call to a getter (`Apply(fn, args) if isFunction(fn.symbol)` would otherwise route it to `defCallee`, `Expressions.scala:466`, `:249-277`, and lift the view-typed parameter, which has no IR form) is recognized before that path by checking the accessor's body against the fixed getter shape from .1, and lifts as a read of that field of `s`. An accessor of any other shape is refused, naming it.
- Refuse an `is` nested in another block, and `is { }` anywhere but a section `val`, with `fail(tree, msg)` naming the position.
- Equivalence fixture in the style of `testdata/lifts/Sugar.scala` and its test at `Fixtures.test.scala:1169-1180`: two machines over a two-field state, one with `is { }` vals, one with `def` predicates, used in `where`, a capability field, `end` and a claim (e.g. `stays(states.paused)`); assert one IR but for names and positions. Not `in(...)`: it takes a status predicate, which stays a plain function (spec, Architecture).

### Investigation targets
**Required** (read before coding):
- `model/irgen/Context.scala:40-60` — `isFunction`, `defs`
- `model/irgen/Expressions.scala:100-130`, `:225-290` — `forwardedDef`, `callee`/`defCallee`, `function`, `parameters`
- `model/irgen/Declarations.scala:150-160`, `:210-280`, `:580-660`, `:730-745` — `stepFunction`, `endOf`, header `function`, rule lowering, `contextBody`
- `model/irgen/Capabilities.scala:450-470`, `:575-590` — `checked`, `etaDef`
- `model/irgen/testdata/lifts/Sugar.scala` and `Fixtures.test.scala:1165-1185` — the equivalence precedent

**Optional** (reference as needed):
- `model/irgen/Claims.scala:385-410`, `:515-525`
- `model/irgen/Order.scala:166`, `:305` — declaration-order lint's handling of member kinds
- `model/irgen/testdata/lifts/Rejects.scala`, `expected/rejects.txt` — refusal fixture format

### Key context
- A behavior-neutral change must not add stricter validation to existing paths (memory: bug/integration/behavior-neutral-refactors-must-not-2026-09-04). The only new refusals are the block-specific ones.
- Inside `object states`, a member named like an enum case shadows the wildcard import; fixtures write `Phase.paused`-style qualified names.

### Acceptance
- [ ] The equivalence fixture lifts `is { }` vals and their `def` twins to identical IR but for names and positions, across `where`, a capability field, `end` and a claim
- [ ] Refusal fixtures for a nested `is`, an `is` outside a section `val` and a getter not of the fixed shape, each naming the position
- [ ] Every existing lifter fixture passes; no change to any checked-in IR (R7) is checked at the batch's regeneration
- [ ] `make lint-model` passes

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
