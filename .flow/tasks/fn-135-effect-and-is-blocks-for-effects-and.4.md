---
satisfies: [R1, R2, R3, R4, R8, R10]
---
# fn-135-effect-and-is-blocks-for-effects-and.4 Convert ActivityProduct, prove identical IR, update docs

## Description
Rewrite `ActivityProduct` in the new forms, rename the status projection to `status`, regenerate, and prove the IR and Cases unchanged apart from positions and that one name. It also adopts .5's status declarations on `Phase` (R8), so no effect records a status fact, and R4's comparison covers that too (R10). Docs for the new forms fold in here.

**Size:** M
**Files:** `model/temporal/features/activity/standalone/product/Product.scala`, `model/ir/activity-standalone*.json` (+ `.laws.json`, `.lint.json`, regenerated), `model/README.md`, `model/umpire/Syntax.scala` (header comment), `model/irgen/Syntax.scala` (hook list comment), `model/umpire/Machine.scala` (class doc)
**Touches:** [model/temporal/features/activity/standalone/product/Product.scala, model/ir/**, model/cases/**, model/README.md, model/umpire/Syntax.scala, model/irgen/Syntax.scala, model/umpire/Machine.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Baseline: the batch baseline (the tree at fn-132's close), per the Batch line; this task takes no snapshot of its own.
- Add the hand-written `phase` getter/setter in `Product.scala` in the fixed shape from .1, after `enum Fact` and before `object ActivityProduct` (the feature-file order lint ranks top-level types 1 and defs 2, `Order.scala:477-489`, so between `State` and `Fact` is refused), scoped so `object states` members cannot shadow it. Convert every effect to `effect { }` and every yes/no `states` member to `is { }`; keep `terminal` (a `Phase` predicate), `notFoundCode` and `end` as they are; rename the projection to `status` and update `Closable(status = …)`. Keep `Phase.paused` qualified inside `states`.
- Status declarations (R8): declare each `Phase` case's status fact on the case in .5's form, and drop every explicit `record(...)` of a status fact, so `startAttempt` is `effect { phase = started }`. `requestCancel` out of `cancelRequested` must still record `statusCancelRequested` (.5's assignment reading); the harness below catches it if not. Keep the `Phase`/`Fact` declaration order so the enums' IR is unchanged. Preserve every existing comment (owner rule).
- Do not regenerate here: `make umpire-gen-model` and `make umpire-gen-cases` (`Makefile:628`, :771) run once at the batch regeneration.
- **Equivalence harness (R4, R10):** write a small script under `.flow/tmp/` (run at the batch regeneration, not here) that, for baseline and new trees, parses each JSON file, strips every `position` value, renames `ActivityProduct$.states$.phase` to `…status` everywhere, and compares `functions` as a map keyed by name (the IR writes functions sorted by name, `Lift.scala:117-121`, so the renamed function moves between `running` and `terminal`), then diffs the normalized documents. Record the script's path and command in this task's evidence; its output is recorded at the batch regeneration, where any other difference stops the batch and is traced to this task.
- Docs: the status declaration beside the block forms (README sugar bullet ~:467-475 and enum bullet ~:441, `.plans/DSL_OPERATORS.md` item 6 at ~:152). `model/README.md` machine-object section (:636-670, items 2 and 4), the example at :1038-1050 (`states.phase`), :506-508, :572, :782 — show the block forms beside the method form, which stays valid. Add the forms to the sugar lists in the two `Syntax.scala` header comments and the `Machine.scala` class doc.

### Investigation targets
**Required** (read before coding):
- `model/temporal/features/activity/standalone/product/Product.scala` — the file being converted
- `model/ir/activity-standalone.json` around :546, :797, :824 — where the projection's name appears
- `model/README.md:500-520`, `:560-580`, `:630-680`, `:1030-1055`

**Optional** (reference as needed):
- `model/temporal/features/activity/standalone/system/System.scala:17`, `:90`; `Record.scala:27`, `:146`, `:335`, `:377`; `Standalone.scala:36`, `:140-141` — consumers of `ActivityProduct` (none reads `states`/`effects` members by name)
- `tools/umpire/lint/lawtable_test.go:135`

### Key context
- Product/System level validators require `Phase`, `State` and `Fact` to stay declared in `product/Product.scala`; keep the accessors in that file, not a sibling (memory: bug/integration/paired-level-validators-must-check-each-2026-10-06).
- Run the model gate with `MODEL_GATE_ARGS=--skip-go-checks` when the Go suite runs separately, and serialize heavy suites with the shared `flock` lock (MILESTONES verification instructions).

### Acceptance
- [ ] No effect in `ActivityProduct` takes a state parameter or calls `copy`; yes/no `states` members are `is { }`; the projection is `states.status` (R1-R3)
- [ ] Every `Phase` case declares its status fact and no effect records a status fact explicitly; `startAttempt` is `effect { phase = started }` (R8)
- [ ] The equivalence harness exists and its command is in the evidence; at the batch regeneration it reports no difference beyond positions and the renamed function (R4, R10)
- [ ] The Model compiles and `make lint-model-syntax lint-model` passes here; `make umpire-check-model`, `make umpire-check-cases` and the Go tests over `model/ir` (`tools/umpire/lint`, `tools/umpire/ir`, `tools/umpire/interp`) pass at the batch regeneration
- [ ] README and the sugar doc comments describe both block forms and the status declaration

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
