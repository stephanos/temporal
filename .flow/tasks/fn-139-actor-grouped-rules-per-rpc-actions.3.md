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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
