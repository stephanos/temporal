---
satisfies: [R5, R6, R7]
---
# fn-135-effect-and-is-blocks-for-effects-and.3 Lift effect { } blocks with their statement refusals

## Description
Lift `effect { }` section vals to the step function the method form gives, and add the statement refusals the spec lists. Split from .2 because rule-effect lowering and the statement walker are a separate path from predicates.

**Size:** M
**Files:** `model/irgen/Declarations.scala`, `model/irgen/Syntax.scala`, `model/irgen/Expressions.scala`, `model/irgen/testdata/lifts/Blocks.scala` (extend) and a new `BlockRejects.scala` with `expected/` output, `model/irgen/test/Fixtures.test.scala`
**Touches:** [model/irgen/*.scala, model/irgen/testdata/lifts/**, model/irgen/test/Fixtures.test.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Rule-effect lowering (`Declarations.scala` ~620-660) today requires `lambda(r.effect)` whose body is a call of a `def` ("the effect of a rule is a call of a def of effects"). Add the `val` member registered by .2's mechanism; leave the `def` branch as is. `givesNoEmpty` (`:693`) does not apply to a block, which never yields no step.
- Walk the block's statements (unwrapped with `contextBody`) once: setter calls, recognized by the fixed setter shape from .1 (any other shape is refused, naming the accessor), become updates of one copy of `s` (`E.Copy` over `E.Var("s")`), emitted in field-declaration order with each field's type as the expected type, as `copyOf` does (`Expressions.scala:689-700`); `record` calls append facts in call order; a lone `reject(o)` becomes `Step(o, Var s, [])`. Otherwise build `Step(ok, copy, facts)` with the `Ok` outcome resolved the way `enter` resolves it (`outcomeOf(ok, "enter")`, `irgen/Syntax.scala:19-60`). An assign-only or empty block lifts exactly as `enter(s.copy(…))` / `enter(s)` does.
- Refuse, with `fail(tree, msg)` naming the statement and position: any statement other than an assignment, `record` or `reject` (a local `val`, a bare expression such as `phase == started`, any other call); an assignment, `record` or `reject` inside `if`/`match`/a loop (`If`/`Match` lift generically today at `Expressions.scala:358-371`, so the walker must look before delegating); a rejecting block with any other statement; a field assigned twice; a field read after its assignment; a nested block.
- Extend the two-field equivalence fixture with effect vals beside their `def` twins (assign+record, partial assignment, assignments written out of field order, two `record`s, assign-only, empty, reject) bound in rules; assert identical IR but for names and positions.
- Placement lint: classify `effect { }` vals as step functions (`Order.scala:276-305`), so the rule that step functions belong in `effects` applies to them.

### Investigation targets
**Required** (read before coding):
- `model/irgen/Declarations.scala:615-700` — rule-effect lowering, `givesNoEmpty`
- `model/irgen/Syntax.scala:19-60` — `sugarCall`, `sugar`, how `enter`/`reject` lift
- `model/irgen/Expressions.scala:340-375` — generic `If`/`Match`/`While`/`Block` lifting
- the .2 fixture `model/irgen/testdata/lifts/Blocks.scala`

**Optional** (reference as needed):
- `model/irgen/testdata/lifts/Rejects.scala`, `expected/rejects.txt`

### Acceptance
- [ ] Effect vals lift to IR identical to their `def` twins but for names and positions, for every case listed in Approach
- [ ] One refusal fixture per refusal in the spec's Edge Cases (including any-other-statement and a malformed setter), each message naming the statement and position
- [ ] The placement lint classifies `effect { }` vals
- [ ] Every existing lifter fixture passes; no change to any checked-in IR (R7) is checked at the batch's regeneration
- [ ] `make lint-model` passes

## Acceptance
- [ ] TBD

## Done summary
The lifter now lifts `effect { }` blocks declared as section vals to the step function their method form gives. A rule binds such a val directly (`~> effects.pause`), and the statement refusals the spec lists are in place. The block walker, `effectSteps` in `model/irgen/Syntax.scala`, makes one pass over the statements:
- Setter calls become one `Copy` of `Var s`, with fields in declaration order and each value lifted against its field type.
- `record` facts append in call order.
- The outcome is resolved from the block's `Ok` given, as `enter`'s is.
- A lone `reject(o)` lifts as `reject(o, s)`.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

Tier: IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

Baseline: green (`mise exec -- scala-cli test model/irgen`: 94 passed, 1 skipped, before the first edit).

### What changed
- `Context.scala`: `blockCall` now recognizes `effect[S, O, F](using owner, ok)(body)` as well as `is`. It returns a `BlockCall(form, types, usings, body)`, and `SectionBlock` holds the val and that call. `misplacedBlock` gives the `effect` example (`val pause = effect { ... }` in `object effects`) when the block is an effect.
- `Expressions.scala`: `blockFunction` dispatches an effect to `effectSteps`, and an `is` still goes to `functionOf`. The getter shape check is factored out as `getterField`, and a new `setterField` recognizes the fixed setter shape `def phase_=(p)(using d: Draft[S, ?, ?]) = d.set(_.copy(phase = p))`.
- `Declarations.scala`, rule-effect lowering: a new branch takes a `Ref` to an effect block val, checks that it is in `effects`, and emits `Call(callee, [s])`. A block written inline in a rule is refused as misplaced. The lambda/`def` branch is unchanged apart from sharing the `outsideEffects` refusal. `givesNoEmpty` is not applied to blocks.
- `Order.scala`: `kindOf` classifies a val whose right-hand side is `umpire.Syntax$package$.effect` as `Kind.Step`.

### Acceptance
- Equivalence: in `lifts/Blocks.scala`, `Blocked.effects` holds effect vals and `Defined.effects` holds their def twins, each bound by a rule. The cases are:
  - assign + record (partial assignment)
  - assignments written out of field order, with two `record`s
  - an assignment that reads its own field before assigning it
  - assign-only
  - empty
  - reject

  The existing test (renamed "is and effect blocks lift as the defs they stand for, ... and a rule's effect") requires identical IR but for names and positions. It now also asserts that each of the 6 effects is a function of its own. Red-first: with the rule val branch disabled, the blocks lift fails with "the effect of a rule of blocked is a function, not ...effects.start".
- Refusals: 10 new roots in `lifts/BlockRejects.scala`, each with a line in `expected/rejects.txt` that names the statement and its position:
  - `BranchedEffect`: an assignment inside `if`
  - `RejectingEffect`: reject plus an assignment
  - `TwiceAssigned`
  - `ReadAfterAssigned`
  - `LocalVal`
  - `BareExpression`
  - `OtherCall`: `require(...)`
  - `NestedEffect`
  - `ShapedSetter`: a malformed setter, named with where it is declared
  - `RuleEffect`: a block written in a rule
- Placement lint: `sectionOrder/SectionOrder.scala` gains `Misfiled.strayBlock = effect(...)` at line 118, refused as "a step function ... belongs in the `effects` object". The test's later line numbers shift by 1. Red-first: without the `Order` change it is refused as vocabulary instead.
- Every existing lifter fixture passes. The update run changed only `expected/blocks.json` and `expected/rejects.txt`, with 10 added lines. No other expected file moved.
- `make lint-model`: each part ran on its own, as did the fmt check (with `--scalafmt-conf model/.scalafmt.conf`). All rc=0.

### Tests run
- `UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`, check mode: 94 passed, 1 skipped.
- `make lint-model-irgen`, `lint-model-irgen-lifts`, `lint-model-check`, `lint-model-models` and `lint-model-syntax`: each rc=0.
- `scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check` on model/project.scala, model/umpire, model/temporal, model/irgen and model/check: rc=0.
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration

### Declared IR delta
None for `model/ir`, `model/cases` or the Model lift expectations. No Model uses `effect { }`. The new paths fire only on a section val whose right-hand side is `effect`, a rule effect that is a `Ref` to such a val, or an inline block. The `is` path lifts as before: `blocks.json`'s `is` functions still equal their twins. The fixture changes are deliberate: `expected/blocks.json` (the new effects, actions and `Outcome.refused`) and 10 new lines in `expected/rejects.txt`.

### Touches
All edits are inside the declared Touches. `model/irgen/testdata/sectionOrder/SectionOrder.scala` falls under `model/irgen/*` testdata, not `testdata/lifts/**`. I edited it because the task's placement-lint AC needs a fixture, and the existing placement fixture lives there. This is reported as a small deviation for the conductor to accept.

### Note for the conductor
- Two owner commits (`0f25754d67`, `b14aaf293b`: MILESTONES.md and fn-143 flow files) landed between my base and my commit, so the evidence lists my commit `dfb6ada9fc` rather than a range.
- Mistake made and repaired: a plain `scala-cli fmt model/irgen` without `--scalafmt-conf` briefly reformatted the whole irgen tree with default settings. I restored every file I had not touched with `git checkout -- model/irgen` (none carried owner edits) and re-applied my edits. Only the 11 files in the commit differ from the base. Later workers should always pass `--scalafmt-conf model/.scalafmt.conf`.

### For later tasks
- fn-135.4 (ActivityProduct conversion):
  - Setters must have exactly the fixed shape `def phase_=(p: Phase)(using d: Draft[State, ?, ?]): Unit = d.set(_.copy(phase = p))`: two parameter lists, the second a given `Draft`, and a body of `d.set(_.copy(<field> = p))` with the parameter itself as the only replaced field. The field name need not match the setter's name.
  - A field can be read in its own assignment (`retried = !retried`), but not after it is assigned.
  - Rules bind effect vals as `~> effects.x`. A lambda wrapper such as `s => effects.x(s)` is not recognized for a val.
- fn-135.5 (derived status facts): the place to append derived facts is the `case _` branch at the end of `effectSteps` in `model/irgen/Syntax.scala`. The facts list is `lifted.collect { case r: Record => r.facts }`, and the assigned fields are `assigns`, which already sit beside each other there. The reject branch is separate, so a rejecting block naturally records no derived fact.
- Messages: an effect written outside a section val gets "`effect { ... }` declares a member of a machine object's section, `val pause = effect { ... }` in `object effects`, and is written nowhere else".
- Placement: `Order.kindOf` treats any val whose right-hand side is `effect(...)` as a step function, so a converted effect outside `effects` is refused in a feature file.

### Follow-ups (not built)
- `View`/`Draft` are still not in `sugarNames` (carried over from fn-135.1/.2).
- The task's own Acceptance mentions no MILESTONES.md edit, and none is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: dfb6ada9fc
- Tests: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (baseline: 94 passed, 1 skipped), UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen, mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (check mode: 94 passed, 1 skipped), make lint-model-irgen, make lint-model-irgen-lifts, make lint-model-check, make lint-model-models, make lint-model-syntax, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model and the byte-identical model/ir check run at the batch's single regeneration
- PRs: