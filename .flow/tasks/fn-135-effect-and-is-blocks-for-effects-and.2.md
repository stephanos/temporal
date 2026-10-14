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
The lifter now lifts `is { }` blocks declared as section vals. One recognizer, `Context.blockVal`, treats a val in a machine object's section whose right-hand side is `is { }` as a function named by the val's `fullName`. `isFunction` consults it, so every existing `isFunction` / `forwardedDef` caller resolves such a val without a change of its own. The block lifts to the function its `def` twin lifts to: the lifter supplies the parameter `s` of the block's state type, and a call of a fixed-shape getter lifts as a read of that field of `s`. Every `DefDef` branch keeps its behavior. `function` now delegates to a new `functionOf(name, ps, body, at)`, which the block path shares.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

Tier: IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

Baseline: green (`mise exec -- scala-cli test model/irgen`: 92 passed, 1 skipped, before the first edit).

### Acceptance
- Equivalence: `lifts/Blocks.scala` puts `Blocked` (`is { }` vals `paused`, `busy`, `over`) beside `Defined` (def twins). Both use the members in:
  - `where(states.paused)` and `in(...).where(states.busy)`
  - a capability field, `Pausable(paused = states.paused)`, with its Property expanded
  - `end`, as `def end(s) = states.over(s)`
  - a claim, `property.stays(states.paused)`

  The test "is blocks lift as the defs they stand for, ..." takes each machine's machines, functions, Properties, Scenarios and Queries, swaps in the other machine's name, strips positions, and requires them equal. It was red first: with the recognizer disabled, the lift fails at `where(states.paused)`.
- Refusals, in `lifts/BlockRejects.scala`, each with a line in `expected/rejects.txt` that names its position:
  - `NestedIs`: an `is` nested in another block.
  - `HeaderIs`: a val of the machine object, not of a section.
  - `DefIs`: a section `def` whose body is `is { }`.
  - `ShapedAccessor`: a `View` accessor that is not a single field read. The message names the accessor and where it is declared.
- Every existing lifter fixture passes: `scala-cli test model/irgen` in check mode gives 94 passed, 1 skipped. The update run rewrote only `expected/rejects.txt` (4 added lines) and created `expected/blocks.json`.
- `make lint-model`: ran its parts one at a time (make 3.81), plus the fmt check. All rc=0.

### Tests run
- `make model/build/model-scala.jar` (packaging only: the jar predated fn-135.1's View/is).
- `UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`. It writes only this task's own fixture expectations.
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`, check mode: 94 passed, 1 skipped.
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check`: rc=0.
- `make lint-model-irgen`, `lint-model-irgen-lifts`, `lint-model-check`, `lint-model-models` and `lint-model-syntax`: each rc=0.
- `scala-cli fmt --check` on model/project.scala, model/umpire, model/temporal, model/irgen and model/check: rc=0.
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration

### Declared IR delta
None for `model/ir`, `model/cases` or the Model lift expectations. No Model uses `is { }`. The only existing paths that changed are `isFunction`, now true for a section `is` val, and a new lift case for a block or accessor call. No checked-in source has a section val built by `is`, a block call, or a def taking a `View`. Fixture changes are deliberate: the new `expected/blocks.json` and 4 new lines in `expected/rejects.txt`.

### Touches
All edits are inside the declared Touches. `Declarations.scala`, `Capabilities.scala`, `Claims.scala`, `Syntax.scala` and `Realizations.scala` needed no edit. Each reaches a val member through `isFunction` / `forwardedDef` / `callee`, which now consult the recognizer.

### For later tasks
- fn-135.3 (effect):
  - Add `"effect"` to `blockForms` in `model/irgen/Context.scala`. `blockCall` matches `effect[S, O, F](using owner, ok)(body)` only if its argument shape is adjusted: `is` has one using arg, `effect` has two (Owner, Ok). The current pattern is `Apply(Apply(TypeApply(fn, state :: _), List(_)), List(body))`.
  - `blockFunction` builds the `is` function: it scans for nested blocks, renames the view to `s`, then calls `functionOf`. Effects need their own body walker.
  - `accessor(fn)` already flags any def taking a `View` or `Draft`. `accessorRead` accepts only the getter shape, so .3 must accept the setter shape `def phase_=(p)(using d: Draft[S, ?, ?]) = d.set(_.copy(phase = p))` too, or setters will be refused as malformed accessors.
  - The rule-effect check in `Declarations.lowered` (`effect(r)`) still requires a lambda calling a def of `effects`. A val `~> effects.pause` is a bare Ref with no lambda, so .3 needs the val branch there. The plan review already noted this.
- Messages: a misplaced block is refused with "`is { ... }` declares a member of a machine object's section ...". `forwardedDef` and `lift` both raise it, and `blockVal` raises it for a non-section val.
- A block val owned by a section object counts as a section member. The test is: its owner is a module class, and that class's owner is a machine or composition object (`objectForm`).
- A capabilities section with no waivers still writes `<fixture>.waivers.json`, listing the section machines. `blocks` is not in the test's `waived` list, so that file is not pinned.

### Follow-ups (not built)
- `View`/`Draft` are still not in `sugarNames` (carried over from fn-135.1).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7f2783558c8abce97b62affcd44075a32534c8fc
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (94 passed, 1 skipped), UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (own fixture expectations only), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check, make lint-model-irgen lint-model-irgen-lifts lint-model-check lint-model-models lint-model-syntax (each rc=0, one by one), mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration
- PRs: