---
satisfies: [R2, R4, R11]
---
# fn-139-actor-grouped-rules-per-rpc-actions.7 in to when and block form across every Model and fixture; rule-case in retired; block-form lint on; docs

## Description
Makes `when` the only rule-case form and the brace block the only `on` form across every Model and lifter fixture. Then it removes rule-case `in` (including the role form `in[R]`, if fn-136.4 landed it under that name) from the framework and the lifter, with a refusal that names `when`. It turns on the block-form rule in `make lint-model` and updates the docs for everything fn-139 adds (R2, R4, R11).

**Size:** M (mechanical across about 11 Model files and the fixtures, plus docs)
**Files:** every Model with rules under model/temporal (activity product, System, Record; Nexus product, standalone and workflow Systems, TrustingCaller, ClosePolicy; shared taskqueue product and System; shared worker), model/irgen/testdata/** (`in(` cases), model/umpire/Rules.test.scala, model/umpire/Syntax.scala (remove rule-case `in` overloads, keep membership `in`), model/irgen/Declarations.scala and Order.scala (drop `in`, add the refusal), model/check/SyntaxRule.scala, Makefile (`lint-model-syntax`), model/README.md, model/SEMANTICS.md, .plans/DSL_OPERATORS.md
**Touches:** [model/temporal/**, model/irgen/**, model/umpire/**, model/check/**, Makefile, model/README.md, model/SEMANTICS.md, .plans/DSL_OPERATORS.md]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Rename first, then remove. Rewrite every rule case `in(…)` → `when(…)`, `in(set)` → `when(set)`, `in(…).where(g)` → `when(…).where(g)` and `in[R]` → `when[R]`. Then rewrite every parenthesized single-case `on(x)(case)` as a block: brace, newline, the case, closing brace on its own line (73 lines today; spec, Architecture: rule layout). Leave membership `p.in(a, b)` alone everywhere. `scala-cli run model/check -- --check-block-form` finds the single-case lines.
- Only the activity's product and System use `from` (task .5). Other machines keep `on` outside `from` (spec, Boundaries).
- Remove `Rules.in[Q](first, rest*)` and `Rules.in(set)` (Syntax.scala:322-330), and `in[R]` if present. Keep both membership extensions (:34, :345). Overload resolution with a membership `in` still in scope means a leftover rule case may resolve to the Boolean extension and fail with a confusing type error. Make it a clear refusal instead: add an `@implicitNotFound`/`compiletime.error` overload, or a lifter refusal, whose message names `when`, and test it.
- Lifter: delete the `in` branches in `heading` and in Order.scala's name lists, and add the refusal fixture `in(...)` as a rule case → message naming `when`. Reword the projection refusal (Declarations.scala:472-473, "in names phases") to name `when`.
- Lint: add `$(MODEL_GATE) --check-block-form` to `lint-model-syntax` in the Makefile, beside `--check-syntax` and `--check-comments` (Makefile:748-751).
- Docs: see the spec's Resolved via Research, docs-gap-scout. In model/README.md, update the rules section (`from`, `when`, multi-action `on`, `rejects`, per-(class, phase) overlap, the new refusals), the sugar list, the Store fixture's single-case `on`, the `client.control(Control.*)` examples and the shared-outcome paragraph. In model/SEMANTICS.md, update the Rules section and its `rejects` lowering. In .plans/DSL_OPERATORS.md, update the rule-block row and keep rule 5 with `on` as the one inline exception. Keep doc text free of names the vocabulary gate retires.
- Equivalence pin: rule headings are not written to `model/ir`, so this task changes no IR line (spec, Edge Cases). The batch regeneration's diff should show nothing attributable to this commit. Positions shift only where block rewrites move lines.

### Investigation targets
**Required:**
- model/umpire/Syntax.scala:300-350
- model/irgen/Declarations.scala:533-552 and model/irgen/Order.scala:205-212
- Makefile:723-751
- model/README.md rules section (around :654-700) and model/SEMANTICS.md:165-210
## Acceptance
- [ ] No rule case under model/ or in a fixture reads `in(…)` or `in[R]`. Every `on` in model/temporal is a brace block with one case per line.
- [ ] A rule-case `in(...)` is refused with a message naming `when`, in the framework (munit) and in the lifter (refusal fixture). Membership `p.in(a, b)` still compiles and lifts.
- [ ] `make lint-model` runs `--check-block-form` and passes. A planted single-case `on` fails it with file and line.
- [ ] model/README.md, model/SEMANTICS.md and .plans/DSL_OPERATORS.md describe `from`, `when`, multi-action `on`, `rejects`/`because`, the overlap rule across blocks, and the block form.
- [ ] `scala-cli test model/umpire`, `scala-cli test model/irgen` and `scala-cli test model/check` pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
