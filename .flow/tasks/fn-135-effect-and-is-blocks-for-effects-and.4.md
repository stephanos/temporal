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
`ActivityProduct` is now written in the block forms. Every effect is an `effect { }` val, and every yes/no `states` member is an `is { }` val. Each `Phase` case declares its status fact, so no effect records a status by hand (`startAttempt` is `effect { phase = started }`). The status projection is renamed `states.status`. The README, `.plans/DSL_OPERATORS.md` item 6 and the sugar doc comments describe the block forms and the status declaration.

stage: impl-review - skipped(config: REVIEW_MODE=none - DSL batch: reviews run once at the batch's end)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tier: lane C, IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

Integrated on `umpire` as `afa032162c`. After integration, `make model/build/model-scala.jar`,
the combined Models test suite, `make lint-model-syntax`, `make lint-model-models`, and
`git diff --check` all passed. The combined documentation conflict was resolved by retaining both
the already-integrated phase-role/protobuf material and this task's status/block material.

Baseline: green at 6f3aa06ce (`scala-cli test model/project.scala model/umpire model/temporal` rc=0, 43 passed; `scala-cli test model/irgen` rc=0, 105 passed, 1 skipped).

### What changed
- `product/Product.scala`:
  - `enum Phase(val status: Fact) extends Recorded[Fact]`, with each case written `case started extends Phase(Fact.statusStarted)`. Case and `Fact` order are unchanged.
  - The top-level accessors sit after `enum Fact` and before the machine object: `def phase(using v: View[State])` and the status-shape setter `def phase_=(p: Phase)(using d: Draft[State, ?, Fact]) = d.set(p)(_.copy(phase = p))`.
  - `states`: `status` (renamed from `phase`), `terminal` and `notFoundCode` keep their forms. `over`, `paused`, `running`, `held` and `pausable` are `is` vals. `Phase.paused` stays qualified.
  - `effects`: ten `effect { phase = X }` vals and `notFound = effect(reject(Outcome.notFound))`. The `import Fact.*` is dropped because nothing reads it any more.
  - `Closable(status = states.status, ...)`. The rules, `end` and the other capability fields are unchanged.
  - scalafmt writes the single-expression blocks with parentheses, `is(phase == started)` and `effect(reject(...))`. This is the same call, and the 135.2/.3 fixtures show the same thing.
- `model/README.md`:
  - Types bullet: the status declaration.
  - New **Blocks** bullet beside the Sugar bullet: both block forms, their statement rules, derived status facts and the fixed accessor shapes.
  - Feature-file order item 3: accessors in the signature.
  - Machine-object items 2 (`is` and the projection naming) and 4 (`effect`).
  - Layout listing.
  - Worked example: `states.status`.
- Header comment of `model/umpire/Syntax.scala`; header and `sugar` hook comment of `model/irgen/Syntax.scala`; class doc of `model/umpire/Machine.scala`. Comments only.

### Equivalence harness (R4, R10)
- Script: `/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/dsl-batch/fn-135.4/ir_equiv.py`
- Command at the batch regeneration: `python3 .flow/tmp/dsl-batch/fn-135.4/ir_equiv.py <baseline>/model/ir model/ir` and the same with `model/cases`. With no file names it compares every `activity-standalone*.json` (IR, laws, lint and case files).
- What it does: drops every `position` key, renames `ActivityProduct$.states$.phase` to `...status` in every string and key, keys every `functions` list by name, and diffs. Exit 1 on any other difference.
- Run here against a scratch lift of the Models, before (6f3aa06ce) and after. I did not regenerate model/ir; the lifts went to `/tmp/laneC3-ir-base` and `/tmp/laneC3-ir-1`. All five lifted files (`activity-standalone{,-race,-record}.json`, `{,-record}.laws.json`) report "equal but for positions and the renamed status projection". Every other IR file is byte-identical (`diff -rq`).
- Negative control: changing one `statusStarted` literal makes it report 10 differences.

### Declared IR delta (for the batch regeneration)
- `model/ir/activity-standalone.json`, `activity-standalone-record.json`, `activity-standalone-race.json`, `activity-standalone.laws.json` and `activity-standalone-record.laws.json` have two kinds of change:
  - source positions of `ActivityProduct`'s declarations;
  - the function `temporal.features.activity.standalone.product.ActivityProduct$.states$.phase`, renamed `...states$.status` wherever it appears (its own name and every `function` reference, 5 in activity-standalone.json). That function moves in the name-sorted `functions` list.
- `*.lint.json` of the same files: the same rename, if a lint key names that function. I did not run the Go lint.
- `model/cases`: none expected. No case file names the function, and the facts are the same.
- Lifter fixture expectations that come from the Models (not regenerated here, per worker-notes): `model/irgen/testdata/lifts/expected/hints.json`, `rejections.json` and `hintsRefused.json` are stale. Each lifts `ActivitySystem`, which refines `ActivityProduct`. I ran the harness on each stale file against its fresh lift, and each was "equal but for positions and the renamed status projection". The batch regeneration rewrites them, and they are the only 3 failures of `scala-cli test model/irgen` at this commit (102 passed).

### Tests run
- `make model/build/model-scala.jar` rc=0 (the Model compiles)
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal` rc=0, 43 passed. This includes IrFilesTest, which constructs every IR file's roots and the rules' overlap checks.
- `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen` rc=1: 102 passed. The 3 failures are the Model-derived stale fixtures above, the expected delta.
- `scala-cli run model/irgen --main-class umpire.irgen.lift -- --ir model/build/model-scala.jar=model/ model/build/model-scala.classpath /tmp/laneC3-ir-{base,1}`, then `ir_equiv.py`: equal.
- `make lint-model-syntax` rc=0, `make lint-model-models` rc=0, `make lint-model-irgen` rc=0
- `scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check` rc=0
- GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: make umpire-check-model, umpire-check-cases and the Go tests over model/ir run at the batch's single regeneration

### Touches
`.plans/DSL_OPERATORS.md` (one paragraph appended to item 6) is outside the declared Touches. The task's Approach names it explicitly, so I made the edit and report it here.

### For later tasks
- fn-136.2 (this lane, next): the product's predicates are now `is` vals. A role test inside one is written `phase.in[Closed]`.
- `Phase` cases now take a constructor argument. A role mixin goes after it: `case started extends Phase(Fact.statusStarted), Held`.
- The rename means a states member must not be named `phase` in any machine that adopts the accessors, because it would shadow the field getter.

### Follow-ups (not built)
- The README's Sugar bullet still does not list fn-136's `when[R]`/`p.in[R]`. That belongs to fn-136's docs.
- No MILESTONES.md edit is needed.
## Evidence
- Commits: afa032162c4ec41ba122f33bf4c9929300ccd026
- Tests: lane verification: baseline green; Model tests rc=0 (43 passed); lifter tests 102 passed with only 3 declared stale generated fixtures; IR equivalence harness equal except positions and states.phase->states.status rename; negative control detected, integration: make model/build/model-scala.jar rc=0, integration: mise exec -- scala-cli test --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal rc=0 (combined suite green), integration: make lint-model-syntax lint-model-models rc=0; git diff --check rc=0, GATE_SKIPPED:umpire-check-model:batch - DSL batch rule: model gate, cases check and Go tests over model/ir run at the batch regeneration
- PRs: