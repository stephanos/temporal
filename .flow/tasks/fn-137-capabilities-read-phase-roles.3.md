---
satisfies: [R4, R5]
---
# fn-137-capabilities-read-phase-roles.3 Migrate every Model to Phased and argument-less Rules

## Description
A mechanical migration of the Temporal Models: each object that names phases mixes in `Phased` and drops its `Rules` argument. Compositions also mix in `Phased` now so later tasks can read them. Behaviour pin: `make umpire-gen-model` must leave model/ir and model/cases byte-identical.

**Size:** M
**Files:** model/temporal/features/activity/standalone/product/Product.scala, model/temporal/features/activity/standalone/system/{System,Record,WithTaskQueue}.scala, model/temporal/features/nexus/workflow/system/{System,TrustingCaller,ClosePolicy}.scala, model/temporal/features/nexus/product/Product.scala, model/temporal/features/nexus/standalone/system/System.scala, model/temporal/shared/worker/Worker.scala, model/temporal/shared/taskqueue/{product,system}/*.scala
**Touches:** [model/temporal/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `Rules(...)` sites: activity product Product.scala:87; activity system System.scala:179; Record.scala:209 (`ActivityRecord`) and :395 (`HeldDispatch`, a plain `Machine` declared at :369); nexus workflow System.scala:213, TrustingCaller.scala:53, ClosePolicy.scala:559 (`_.caller`, object `RejectAfterClose` at :266); nexus product Product.scala:70; nexus standalone System.scala:93; Worker.scala:68; task queue product Product.scala:75 (`_.outstanding`); task queue system System.scala:143 (`_.custody`). Record.scala:480 is already argument-less and needs no projection.
- Compositions: the activity composition (activity System.scala, around :412, `_.activity.phase`), the Nexus composition (nexus System.scala, around :431, `_.operation.phase`) and the WithTaskQueue compositions.
- Write `Phased[<State>, <PhaseType>](<projection>)` after `Machine[...]` in the parent list. Keep the existing comments.
- If fn-135 has landed, ActivityProduct's projection is named `status`. Mirror whatever the code says at that point.

### Investigation targets
**Required:**
- each site above, read in place
- model/temporal/IrFiles.test.scala: constructs every IR root

### Acceptance
- [ ] No `extends Rules(` remains under model/temporal (grep).
- [ ] Every object listed in spec R4 mixes in `Phased`.
- [ ] `make umpire-gen-model` leaves `git diff -- model/ir model/cases` empty.
- [ ] `make umpire-check-model` and `make lint-model` pass.

## Acceptance
- [ ] TBD

## Done summary
# fn-137-capabilities-read-phase-roles.3

The Temporal Models now declare all 16 phase projections through explicit `Phased[S, P](projection)` parents. All 12 projection-bearing rules objects use argument-less `Rules`, including activity/Nexus products and systems, `HeldDispatch`, worker/taskqueue providers, and the activity, Nexus and WithTaskQueue compositions. Derived models keep their inherited projection. `LostStartAnswer` remains unphased.

The inventory test covers all 16 declarations. It failed before migration because `ActivityProduct` was not `Phased`, then passed after migration. Existing fn-134.3, fn-135.4 and fn-136.2 changes remain intact. ActivityProduct keeps `_.phase`, matching its actual field despite its `states.status` observation name.

stage: impl-review - skipped(policy: parallel-wave - conductor owns review)
stage: plan-sync - skipped(config: planSync.enabled=false)

Integrated on `umpire` as `63e1b831f8`. Conductor review found no blocking issue. On the
integration branch, the Model package and combined 48-test suite passed, as did
`make lint-model-models lint-model-syntax`, the zero-legacy-`Rules(` grep, `git diff --check`, and
the check that no tracked IR, Cases, or lifter fixture changed.

### Verification

The pre-edit framework/Model baseline passed. The final framework/Model run passed all 48 tests, including the inventory, existing Phased/argument-less Rules tests, capability catalog and role refinement tests. Logs are under `.flow/tmp/fn-137.3/`.

- `model-tests.log` records the final Model/framework suite; `red-phased-models.log` records the intended pre-migration failure.
- `fmt-check.log` records a passing format check over model sources, irgen and check.
- `final-lints.log` records passing `make lint-model-models lint-model-syntax`. Scalafix prints the inherited JDK 27 `NoSuchFieldException: path` warnings and exits 0; there are no lint findings or compiler errors.
- Acceptance grep finds no `extends Rules(` under `model/temporal`; `git diff --check` passes.
- `focused-phased-fixtures.log` records isolated compilation/lifting of the existing `phasedRules` and `phasedMixin` fixtures. Their JSON is byte-identical to each other and checked expected JSON. Derived machine/composition lifts succeed, and `Unprojected` fails with the exact source-located refusal and no output file.
- `before-package.log`, `after-package.log`, `before-lift.log` and `after-lift.log` record successful worktree Model packages and scratch lifts of every declared IR root. Only source-independent ignored API/IR dependency jars came from the main checkout.

The full lifter suite exits 1. `baseline-lifter-tests.log` reproduces the same four failures with the pre-task Model jar; `lifter-tests.log` records the post-task run. They compile legacy fixture imports of the retired Law API and a retired `Describable(status=...)` named argument from fn-134.3. The failing entries are:

- `umpire.irgen.Fixtures.beforeAll(umpire.irgen.Fixtures)`
- `umpire.irgen.QualifiedNames.a declaration moved into another file of its package keeps its ID, type and family names`
- `umpire.irgen.QualifiedNames.a declaration moved under an object takes the object into its ID`
- `umpire.irgen.QualifiedNames.a declaration moved into another package takes the package into its ID, type and family`

The batch assigns legacy Law fixture retirement to fn-134.4, after fn-137.4. This task changes no fixture.

### Exact expected delta and staleness

The scratch comparison does not establish exact byte identity for all IR. `ir-delta.log` inventories every residual JSON path and old/new value without normalizing outputs. The 1,475 changed leaves across six files are exclusively `.position.line`; all names, ordering, graph structure, reads, conditions, effects, obligations, claims, domains, waivers and other values match. These four files are byte-identical: `activity-standalone-record.waivers.json`, `activity-standalone.waivers.json`, `nexus-standalone.json`, `nexus-standalone.waivers.json`.

The remaining line pairs below include multiplicities from derived copies. Each artifact's full paths are in `ir-delta.log`.

| Artifact | Changed leaves | Old to new line (count) |
| --- | ---: | --- |
| activity-standalone-race.json | 40 | 120->121 (16), 121->122 (3), 385->387 (3), 386->388 (11), 387->389 (1), 392->393 (3), 393->394 (3) |
| activity-standalone-record.json | 554 | 16->18 (28), 17->19 (10), 21->22 (1), 21->24 (12), 22->23 (7), 25->27 (216), 27->28 (5), 98->99 (8), 120->121 (220), 121->122 (15), 122->123 (11), 123->124 (1), 186->187 (20) |
| activity-standalone.json | 5 | 418->419 (5) |
| nexus-workflow-close.json | 821 | 268->271 through 274->277 (108 each), 276->279 (27), 277->280 (11), 278->281 (3), 279->282 (1), 284->286 (6), 288->289 (1), 289->290 (12), 290->291 (4) |
| nexus-workflow-control.json | 50 | 28->31 (4), 29->32 (6), 30->33 (3), 31->34 (1), 36->38 (6), 41->42 (3), 42->43 (18), 43->44 (9) |
| nexus-workflow.json | 5 | 438->439 (5) |

Normal scalafmt wraps the longer parent lists. Local blank-line compensation preserves every existing `object rules` declaration line; `before-rules-lines.log` and `after-rules-lines.log` compare byte-identically. Header-adjacent init/end/evidence/early-effect nodes retain the shifts above because available blank lines cannot compensate before those declarations. No comments were deleted or rewritten, and no aliases or format exemptions were introduced to hide the positions. The conductor accepted this traced position inventory for the batch union gate.

`model/ir`, `model/cases` and all lifting fixtures remain unchanged. Checked outputs remain intentionally unregenerated through this DSL batch. This scratch comparison is per-task diagnostic evidence; the authoritative union comparison and any Cases impact remain unobserved until the conductor's single batch regeneration against `96de1fd92d`.

GATE_SKIPPED:umpire-check-model:batch - full gates and authoritative IR/Cases checks deferred to the DSL batch close.
GATE_SKIPPED:umpire-gen-model:batch - regeneration deferred; no tracked output was written.

The implementation is committed and ready for conductor review. The task remains `in_progress`; the conductor owns review, integration and completion.
## Evidence
- Commits: 63e1b831f87b999875263ed07895b785ca5d4fe2
- Tests: baseline green: timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal; exit 0; .flow/tmp/fn-137.3/baseline-model-tests.log, test-first red: timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal --test-only temporal.IrFilesTest; exit 1 because ActivityProduct lacked Phased; .flow/tmp/fn-137.3/red-phased-models.log, timeout 600s mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal; exit 0, 48 tests; .flow/tmp/fn-137.3/model-tests.log, timeout 600s mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/umpire model/temporal model/irgen model/check; exit 0; .flow/tmp/fn-137.3/fmt-check.log, timeout 600s mise exec -- make lint-model-models lint-model-syntax; exit 0, inherited JDK 27 NoSuchFieldException warnings; .flow/tmp/fn-137.3/final-lints.log, timeout 600s bash /tmp/umpire-fn137-3.bbLBRX/.flow/tmp/fn-137.3/check-phased-fixtures.sh; exit 0, exact fixture identity, derived lifts and unprojected refusal; .flow/tmp/fn-137.3/focused-phased-fixtures.log, timeout 600s env UMPIRE_LIFTER_UPDATE= mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen; exit 1, four inherited legacy Law/Describable failures reproduced with pre-task jar; .flow/tmp/fn-137.3/lifter-tests.log and baseline-lifter-tests.log, mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/umpire model/temporal -f -o model/build/model-scala.jar; exit 0 before and after migration; .flow/tmp/fn-137.3/before-package.log and after-package.log, mise exec -- scala-cli run --suppress-outdated-dependency-warning model/irgen -- --ir .flow/tmp/fn-137.3/before.jar=model/ model/build/model-scala.classpath .flow/tmp/fn-137.3/before-ir; exit 0; .flow/tmp/fn-137.3/before-lift.log, mise exec -- scala-cli run --suppress-outdated-dependency-warning model/irgen -- --ir model/build/model-scala.jar=model/ model/build/model-scala.classpath .flow/tmp/fn-137.3/after-ir; exit 0; .flow/tmp/fn-137.3/after-lift.log, python3 .flow/tmp/fn-137.3/compare_ir.py; exit 0 classification only, byte identity fails for six IR artifacts with exactly 1475 position.line changes and no other differences; .flow/tmp/fn-137.3/ir-delta.log, cmp .flow/tmp/fn-137.3/before-rules-lines.log .flow/tmp/fn-137.3/after-rules-lines.log; exit 0, every object rules declaration line preserved, rg -n 'extends Rules\(' model/temporal; exit 1, zero matches as required, git diff --check; exit 0, git diff --name-only -- model/ir model/cases model/irgen/testdata; empty, tracked outputs and fixtures unchanged, integration: make model/build/model-scala.jar; combined Model suite 48/48; make lint-model-models lint-model-syntax; zero legacy Rules(projection) grep; git diff --check; no tracked generated/fixture delta (all exit 0), GATE_SKIPPED:umpire-check-model:batch - full gates and authoritative IR/Cases checks deferred to the DSL batch close., GATE_SKIPPED:umpire-gen-model:batch - regeneration deferred; no tracked output was written.
- PRs: