---
satisfies: [R1, R2, R3, R4, R7]
---
# fn-134-capabilities-own-their-properties.3 Migrate the Temporal kit and every Model to capabilities sections; regenerate

## Description
Move every law into its capability's companion and every declaration site into a `capabilities` section, with bounds in `queries`, then run the one regeneration. This is the task R7's equivalence proof belongs to. The tests that read law data move here too, so the gate is green at the end.

**Size:** M
**Files:** `model/temporal/capabilities/{Capabilities,Close,Pause,Terminate,Cancel}.scala` (replaced by one file per capability: `{Closable,Terminable,Pausable,Cancelable,Pollable,Describable}.scala`), `model/temporal/capabilities/Catalog.test.scala` (the two-machines test, rewritten), `model/temporal/features/activity/standalone/{Standalone,product/Product,system/System,system/Record,system/WithTaskQueue}.scala`, `model/temporal/features/nexus/standalone/{Standalone,system/System}.scala`, `tools/umpire/check/{capabilities_test,activity_parity_test}.go`, `model/ir/**`, `model/cases/**`
**Touches:** [model/temporal/**, model/ir/**, model/cases/**, tools/umpire/check/capabilities_test.go, tools/umpire/check/activity_parity_test.go]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- Kit, one file per capability (owner decision 2026-10-06): each capability's case class and its companion, with the companion's Property defs, live in a file named after the capability: `Closable.scala`, `Terminable.scala`, `Cancelable.scala`, `Pausable.scala`, `Pollable.scala`, `Describable.scala`. Each `object x extends Law(...)` (Close.scala:19,40; Pause.scala:12; Terminate.scala:13; Cancel.scala:12) becomes a def in its kind's companion, with `promises`/`doesNotPromise` and the cites moved to Scaladoc. `pausedIsNotDispatched` goes into `Pausable`'s companion and reads Pollable's `running`. Pausable and Pollable stay separate. `Capabilities.scala`, `Close.scala`, `Pause.scala`, `Terminate.scala` and `Cancel.scala` are deleted; the folder header comment moves to the file that best introduces the folder or is dropped if each file's header says it. `Catalog.scala` goes in fn-134.4 as planned.
- Sites: Product.scala:120, System.scala:312 and nexus System.scala:155 become `capabilities` sections, keeping nexus's `overriding`. Record.scala:276-290 and WithTaskQueue.scala:101-117 and :189-205 become shared sets (task 2's form), keeping each `except`. Rewrite the `.claim` uses (Record.scala:256-264, 346; WithTaskQueue.scala:87-94, 181-183) and the `irFile` roots (activity Standalone.scala:144-145, nexus Standalone.scala:53).
- Bounds move unchanged: three (product, activity system, nexus), five (record, over-queue), twelve (over-matching).
- Two-machines test: rewrite Catalog.test.scala over the capability companions and the declared machines (its hand-written list at :18-43 updates).
- Go tests that read `activity-standalone.laws.json` (`capabilities_test.go:54-97`, `activity_parity_test.go:90-104`, which also has a regex on `object implements`) now read the IR `origin` and the new section form.
- Equivalence harness: before editing, capture a projection on the current tree with the gate's own outputs: every Query name and answer, every Check receipt, every Definition ID, the exploration identity map, `model/cases` bytes, and `model/ir/*.lint.json`. After `make umpire-gen-model`, diff it. The only allowed IR differences are `origin` fields, generated-Property positions and the three deleted `*.laws.json`. `.lint.json` keys and reasons must be unchanged.

### Investigation targets
**Required:**
- `model/temporal/capabilities/*.scala`
- `model/temporal/features/activity/standalone/system/{Record,WithTaskQueue}.scala`
- `model/temporal/features/nexus/standalone/system/System.scala:155-175`
- `tools/umpire/check/capabilities_test.go`, `tools/umpire/check/activity_parity_test.go:90-110`
**Optional:**
- `model/temporal/IrFiles.test.scala:29`

### Key context
This runs after fn-132 and fn-133 close, rebased on their regenerated `model/ir`, so the baseline is clean. No other regeneration may run alongside it.

## Acceptance
- [ ] No `Law` object and no `implements` section remain under `model/temporal`; Pausable and Pollable are separate capabilities.
- [ ] Every machine's generated Queries are bounded in `queries` with the bounds they had.
- [ ] The two-machines test passes over the companions.
- [ ] Equivalence diff: Query names, answers, receipts, Definition IDs, exploration identity, Case bytes and `.lint.json` unchanged; the IR diff shows only `origin`, generated-Property positions and the removed `*.laws.json`.
- [ ] `make umpire-gen-model`, `make umpire-check-cases`, `make umpire-check-lint` and the Go suite pass.

## Done summary
Temporal Models now declare typed capabilities in sections and bound generated Queries in their queries sections. Five Properties live in the six capability companions, and the two-machines and Go reader tests read companion definitions and IR origins.

Task: fn-134-capabilities-own-their-properties.3
Status: in_progress. Ready for conductor review; no review verdict claimed.
Workspace: /tmp/umpire-fn134-3.dQ8Jx2
Branch: codex/fn134-3
Commit: 6a10012d13c4924a1aa223c7a959a4923d43aa97
Base: 3e1fd115f0c56bef7ced655e401c5a7b36a82de2
stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
stage: plan-sync - skipped(config: planSync.enabled=false)

Integrated on `umpire` as `a1c39e5b4e`. Conductor review found no blocking issue. On the
integration branch, `make model/build/model-scala.jar`, the combined 47-test Model suite,
`make lint-model-models lint-model-syntax`, `git diff --check`, and the no-generated-tree-delta
check all passed.

Implementation and interpretation:

- Shared admission/over-queue/over-matching capability sets preserve the original waivers and bounds. Pausable and Pollable remain separate. No Law objects or implements sections remain under model/temporal.
- Catalog.scala remains as an empty transitional Catalog for fn-134.4. It supplies no Temporal Law entries.
- Promises, nonpromises and code citations use adjacent // documentation. The executable CommentRule and current README forbid Scaladoc, so they take precedence over the task's stale wording. No gate or flag was weakened.
- Scala 3.9 SemanticDB rejects eta-expanding the closed-rejection def when its final parameter uses dependent m.Outcome. An explicit type application reproduced the failure and was removed. Naming O and refining the model argument to Declares[S] { type Outcome = O } fixes the compiler warning while retaining the outcome constraint and Property body. Full reproduction and failed hypothesis logs remain in .flow/tmp/fn134-3-lint.log and fn134-3-lint-typed.log.
- Describable.status became statusTable. Closable also binds status, and the intentional name-only ambiguity refusal rejects Nexus's combination. The table-field rename brings no Property and no semantic IR change. The red lift is fn134-3-final-lift.log; the successful lift is fn134-3-accepted-lift.log.

Verification:

- Test-first companion inventory reproduction failed with the expected missing definitions in fn134-3-red.log. Final model/framework suite passed all 47 tests in fn134-3-status-table-tests.log.
- RolesSuite passed 3 tests. Go vet passed. Four selected Go reader tests passed against scratch IR, including exact Nexus receipt kinds and origins, activity claim inventory and capability-section fixture receipts.
- make fmt-model lint-model-models lint-model-syntax exited 0. Scalafix prints nonfatal NoSuchFieldException: path diagnostics under the pinned JDK; no lint rule errors or SemanticDB failure remained.
- git diff --check passed. Tracked IR, Cases and expected lifter fixtures remain byte-unchanged by this commit.
- The pre-edit model suite printed all passing tests, but its command exit status was not retained. Baseline evidence is inconclusive on exit status. The post-edit foreground suite captured exit 0.

Per-task scratch equivalence only:

- Scratch-lifted the exact task parent and current Model sources using the same lifter. Seven Model JSONs have identical ordered semantic contents after classifying source positions, 26 added generated Property origins and the exact two root provenance substitutions below. Query names, bodies, bounds, static totals, machines, realizations and other semantic declarations are unchanged. Three old laws sidecars disappear; three scratch waivers sidecars preserve declaring-machine sets and every waiver subject/reason.
- activity-standalone.json source changes only temporal.features.activity.standalone.product.ActivityProduct$.implements to $.capabilities and temporal.features.activity.standalone.system.ActivitySystem$.implements to $.capabilities.
- nexus-standalone.json source changes only temporal.features.nexus.standalone.system.NexusSystem$.implements to $.capabilities.
- Both IRs containing Exploration declarations, nexus-workflow.json and nexus-workflow-control.json, are byte-identical between parent/current, so their exploration identities cannot change from this task.
- Initial strict comparison caught the source substitutions. The classified comparison asserts these exact substitutions before comparing all remaining semantic data. See fn134-3-equivalence.log, fn134-3-equivalence-classified.log and fn134-3-equivalence.py.
- This is not the required union proof against batch baseline 96de1fd92d. Actual complete Check receipts, Case bytes, downstream Definition IDs and accepted .lint generation must be checked at the batch's single regeneration.

Expected stale artifacts and integration risks:

- No regeneration, full model/Go gate or flowctl done ran, as required by the DSL batch override. Checked-in Models retain old origins/positions/source spellings and the three .laws.json files; Cases and .lint files have not changed.
- Legacy lifter fixture sources still reference removed Temporal Law symbols and Describable(status = ...). The following cleanup tasks must migrate those before the fixture/full gate. This task ran RolesSuite and existing capability-section reader fixtures without regenerating any fixture.
- fn-135.4 and fn-136.2 changes remain in the task parent and implementation. No conflicts occurred and no conductor checkout files were touched.
- The conductor should add the exact two root source changes to the batch's expected-delta contract, integrate this commit, perform review and run the union proof at the single regeneration.
## Evidence
- Commits: a1c39e5b4e5b9334ec0f3f67cfcad6c5b0e4225b
- Tests: baseline: inconclusive exit status; pre-edit model suite log shows passing tests, command exit not retained, RED: companion inventory test; suite_rc=1 in .flow/tmp/fn134-3-red.log, timeout 600 mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/project.scala model/umpire model/temporal (exit 0; 47 tests; .flow/tmp/fn134-3-status-table-tests.log), timeout 600 make fmt-model lint-model-models lint-model-syntax (exit 0; .flow/tmp/fn134-3-final-style.log), mise exec -- go vet -tags test_dep ./tools/umpire/check (exit 0; .flow/tmp/fn134-3-go-vet.log), mise exec -- go test -tags test_dep ./tools/umpire/check -run '^TestCapabilitiesGenerated' -count=1 (exit 0; .flow/tmp/fn134-3-go-fixture.log), mise exec -- scala-cli test --server=false --suppress-outdated-dependency-warning model/irgen --test-only umpire.irgen.RolesSuite (exit 0; 3 tests; .flow/tmp/fn134-3-lifter-tests.log), timeout 600 mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/umpire model/temporal -f -o model/build/model-scala.jar (exit 0; .flow/tmp/fn134-3-status-table-package.log), timeout 600 mise exec -- scala-cli compile --server=false --suppress-outdated-dependency-warning --print-class-path model/project.scala model/umpire model/temporal (exit 0), timeout 600 mise exec -- scala-cli run --server=false --suppress-outdated-dependency-warning model/irgen --main-class umpire.irgen.lift -- --ir model/build/model-scala.jar=model/ model/build/model-scala.classpath .flow/tmp/fn134-3-accepted-ir (exit 0; scratch only), python3 .flow/tmp/fn134-3-equivalence.py .flow/tmp/fn134-3-parent-ir .flow/tmp/fn134-3-accepted-ir (exit 0; per-task semantic comparison; exact root source substitutions classified), timeout 600 mise exec -- go test -tags test_dep -c ./tools/umpire/check -o .flow/tmp/fn134-3-check.test (exit 0), scratch cwd .flow/tmp/fn134-3-reader-proof/tools/umpire/check: timeout 600 /tmp/umpire-fn134-3.dQ8Jx2/.flow/tmp/fn134-3-check.test -test.v -test.run '^Test(NexusOperationReceivesTheLaws|ActivityEveryClaimDeclarationIsLifted|CapabilitiesGenerated.*)$' (exit 0; 4 tests; .flow/tmp/fn134-3-scratch-readers.log), cmp parent/current nexus-workflow.json and nexus-workflow-control.json (both exit 0), git diff --check (exit 0), git diff --quiet 3e1fd115f0c56bef7ced655e401c5a7b36a82de2..HEAD -- model/ir model/cases model/irgen/testdata/lifts/expected (exit 0), integration: make model/build/model-scala.jar; combined Model suite 47/47; make lint-model-models lint-model-syntax; git diff --check; no tracked generated-tree delta (all exit 0), flowctl gate classify --base 3e1fd115f0 (FULL; full gates deferred by task DSL Batch override)
- PRs: