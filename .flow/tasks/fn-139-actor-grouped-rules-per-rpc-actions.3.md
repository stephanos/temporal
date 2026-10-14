---
satisfies: [R1, R2, R3, R4]
---
# fn-139-actor-grouped-rules-per-rpc-actions.3 Lifter reads from/when/multi-action on; block-form rule in the model gate (not yet in lint-model)

## Description
Lifter half of the new rule forms from task .2: the lifter reads `from` blocks (their leading import), `when` cases and multi-action `on`, and drops its own one-block refusal. Also adds the model-gate rule that refuses the parenthesized single-case `on(x)(case)`. The rule gets its flag and tests here, but it is not added to `make lint-model` until task .7 has converted the 73 single-case lines.

**Size:** M
**Files:** model/irgen/Declarations.scala (`ruleSteps`, `cases`, `heading`), model/irgen/Order.scala (`appliesAtOnce` and the `on` check), model/irgen/testdata/lifts/Rules.scala + expected (new `from`/`when`/multi-`on` cases), model/irgen/testdata/lifts/Rejects.scala + expected/rejects.txt, model/irgen/test/Fixtures.test.scala, model/check/BlockFormRule.scala (new), model/check/Gate.scala (`--check-block-form`), model/check/test/BlockFormRule.test.scala (new)
**Touches:** [model/irgen/**, model/.scalafmt.conf, model/check/BlockFormRule.scala, model/check/Gate.scala, model/check/test/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `ruleSteps` (Declarations.scala:425-488) matches `from(d)(block)` and walks its statements. It accepts the wildcard import of `d` only as the first statement, then `on` blocks. Anything else is refused with the existing "not a rule" style, naming the statement. An `on` whose action symbol is not a member of `d` is refused naming the action and `d`. A `from` in a `from` is refused.
- Multi-action `on(a, b[, c[, d]])` yields one block per target with the same cases. Remove the "written twice ... one block of cases" refusal (:446-451). The framework's overlap check (task .2) now guards repeats, and the lifter has no overlap check of its own (Order.scala:174-211 reads the framework's result).
- `heading` (:533-552) maps `when(…)`, `when(set)` and `when[R]` (when present) to the same `Heading.In` / `Heading.InSet` as `in`, and `.where` on a `when` case to `Heading.And`. Keep `in` working until task .7.
- Order.scala:205-212 and :590 hard-code `on`, `in` and `where`. Add `when` and `from`, or rule initialization stops reading as immediate.
- Block-form rule: follow `CommentRule` (`findings(root)` returning `file:line: reason`, scanning with `ProtoLiterals.literals` so strings and comments are skipped). It flags an `on(...)` whose second argument list is parenthesized rather than a brace block, under `model/temporal`. Wire `--check-block-form` in Gate.scala like `--check-syntax` (:536-557). Do not add it to the Makefile here.
- `when` cases map to `Heading.In`/`Heading.InSet`. The existing `Heading.When` means `where(g)`, so do not route `when` through it. Renaming `Heading.When` to `Heading.Where` is optional.
- Formatter: model/.scalafmt.conf:9-10 enables `RedundantBraces`, and its `parensForOneLineApply` default rewrites a one-line `on(x) { case }` into `on(x)(case)`. Check that `make fmt-model` leaves a multi-line single-case block (spec, Architecture: rule layout) alone. If it does not, set `rewrite.redundantBraces.parensForOneLineApply = false` here, and confirm the formatter then changes no other file under model/.
- Keep new fixtures from copying Model text (Overlap.test.scala refuses more than 6 shared substantive lines).

### Investigation targets
**Required:**
- model/irgen/Declarations.scala:425-560
- model/irgen/Order.scala:170-215, :585-595
- model/check/CommentRule.scala and model/check/Gate.scala:470-560
- model/irgen/test/Fixtures.test.scala:1-60

**Optional:**
- model/irgen/testdata/lifts/Rejects.scala:1084-1090 and expected/rejects.txt:26: the repeated-block refusal that goes
## Acceptance
- [ ] Fixture: the same machine written with `from` + import, `when` and `on(a, b)` lifts to JSON identical to its `on` + `in` form, positions aside.
- [ ] Refusal fixtures, each refused at its line: a statement in a `from` other than the leading import and `on` blocks, an action not declared by the `from`'s declarer, a `from` in a `from`. The old repeated-block specimen is removed or replaced by a cross-block overlap specimen.
- [ ] BlockFormRule.test.scala: the parenthesized single-case `on` is reported with file and line, while brace blocks, `on` inside strings and comments, and `rebind(on(...) { ... })` are not.
- [ ] `make fmt-model` keeps a multi-line single-case block as written (shown by a fixture or test), and changes nothing else under model/.
- [ ] `scala-cli test model/irgen`, `scala-cli test model/check` and `make lint-model` pass. `model/ir` is untouched.
## Done summary
The lifter now reads the task-.2 rule forms. It reads `from(d) { import d.*; on(...) ... }`, the `when(...)` and `when(set)` cases (lifted exactly as `in`), and `on(a, b[, c[, d]])`, which gives each target the same cases. It drops its "written twice" one-block refusal. The new model-gate rule `gate --check-block-form` reports a parenthesized `on(x)(case)` under model/temporal, as task .3 asked; it is not in `make lint-model` yet.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: claude-opus-5-5 at high (lane D)

### What changed
- model/irgen/Declarations.scala `ruleSteps`: the statement loop is now a local `rule(stat, within)`, which recurses into a `from` with its declarer. New private helpers: `notRule`, `declarerName`, `fromBlock` (the first statement must be `Import(d, SimpleSelector("_"))`, and only the first) and `declaredBy` (the target's action val is owned by the declarer's module class). A multi-target `on` refuses a target named twice and a disabled target. `cases` refuses a `from` inside an `on`. `heading` maps `when` to `Heading.In`/`Heading.InSet`. `Heading.When` (meaning `where`) is not renamed.
- model/irgen/Order.scala `appliesAtOnce`: `when` and `from` are added beside `on` and `in`.
- model/check/BlockFormRule.scala (new) and the `--check-block-form` flag in Gate.scala (usage and doc lines included); model/check/test/BlockFormRule.test.scala (new). Not added to the Makefile.
- Fixtures: lifts/Grouped.scala (new) holds `Grouped` (from, when, when(set), when.where, on of two classes) and `Plain` (on/in). A new test requires one machine of the two, except for names and positions. It uses the `declarations` helper, so there is no expected file. lifts/RuleRejects.scala (new): a from in a from, an undeclared action, an import after the first statement. In Rejects.scala the old `BlockRepeated` specimen is replaced in place, with the same line count, by `FromStatement`, a from with no leading import. expected/rejects.txt is updated.
- Formatter: `make fmt-model` keeps a multi-line single-case brace block as written. It rewrites only a one-line `on(x) { case }` to `on(x)(case)`, and the block-form rule then flags that. So `parensForOneLineApply` needed no change: model/.scalafmt.conf is untouched, and `scala-cli fmt --check` passes across model/.

### Lifter messages (match the framework's from .2)
- `from(<d>) sits in from(<outer>): a from holds on blocks alone`
- `on(<a>) sits in from(<d>), and <d> declares no <a>: a from holds the blocks of the actions its declarer declares`
- `<a> is named twice in one on: name each action, or class of it, once`
- Lifter-only messages: `not a rule of <m>'s from(<d>): <stmt>; a from begins with its declarer's import, \`import <d>.*\`` and `...; a from holds its declarer's import, first, then \`on(action) { case ~> effect }\` blocks`, plus `from sits in a block of <m>'s rules: a block holds its cases alone`.

### Expected IR delta (batch regeneration)
None in model/ir: no Model uses the new forms, and the lowering of the existing forms is unchanged.

### For later tasks
- fn-139.7: `gate --check-block-form` reports 74 lines today, not the spec's 73. The extra line is one derivation, `ActivityRecord.rebind(on(worker.poll)(always ~> ActivityRecord.effects.admit))`, which R4 also covers. Wire it into `lint-model-syntax` (Makefile) once the Models are converted.
- The lifter checks declarer membership by symbol owner. A `from(worker)` whose vals forward another object's action (`val poll = kind.worker.poll.on(activity)`) declares that val itself, so `on(poll)` in `from(worker)` passes, and the framework's reflection agrees.
- `disabled(...)` inside a `from` is refused, as "not a rule of ...'s from(...)". It stays outside any from (spec decision).
- fn-136.4: `when[R]` needs its own `heading` case too; `call` yields `("when", List(List(evidence)))`-shaped args, which the current `when(set)` case would take for a set. Distinguish it by its type args or evidence type.

### Follow-ups (not built)
- None.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 64b4b8140c
- Tests: UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen (rc=0; own fixture expectation rejects.txt only), mise exec -- scala-cli test --suppress-outdated-dependency-warning model/check (rc=0, all suites incl. BlockFormRuleSuite and GateSuite), scala-cli run model/check -- --check-block-form on the repository: 74 findings (expected; not in lint-model until fn-139.7), make lint-model-irgen lint-model-irgen-lifts lint-model-check lint-model-syntax (each rc=0, one by one); scala-cli fmt --check model/... (rc=0), GATE_SKIPPED:umpire-check-model:batch - DSL batch rule
- PRs: