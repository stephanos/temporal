---
satisfies: [R7, R13]
---
# fn-160-one-ir-file-per-feature-collected-from.3 One machine per file; flat level folders grouped by name prefix: mini feature and shared code (real features at the sweep)

## Description
**Scope (2026-10-12, Scala DSL batch rules; supersedes the text below where they conflict):** re-scoped to `features/activity_mini` and the shared code it uses (foundations, actors, capabilities, realize, `Bounds`). The lifter enforces one machine per file, named after it, flat level folders grouped by family prefix (facet names, a declared `enum FamilyName`, the level's name or primary machine's), and the kind/form layout; the mini and shared sources are moved and renamed to satisfy it, with a rename map, an IR identity/position classification and the mini step-table equivalence. The enforcement holds every lifted Temporal Model, so a real feature fails it as *behind* until the owner's sweep; the mini/shared lift selects no real feature and is unaffected. The real features' splits, moves and renames (the close policy, `DispatchWithTaskQueue` and the rest listed below), regenerated checked-in IR/Cases, Go consumers and the `make umpire-check-*` gates go to the sweep and Batch 5b. A pure file move needs no scalafix rule; the reference renames of shared objects ship as `Fn160Task3FamilyNames`, and the sweep recipe is in `model/migrate/README.md`.

Require each Model file to declare at most one machine, composition or derivation (`Machine`, `Composition` or `Derived` object), in a file named after it. Level folders stay flat, with no nested packages. Files are grouped by a shared name prefix: the facet name from the feature's vocabulary (fn-161), or the design family's name. That way the sorted file list shows every machine and how they group. Today 53 machine objects sit in 24 files, and only 12 of those files hold exactly one. The largest, nexus workflow's close policy, has 9 designs in about 1,200 lines.

**Size:** L
**Files:** `model/irgen/Structure.scala` / `Order.scala` (the layout and order checks), `model/irgen/testdata/layout*` (new refusal fixture), `model/README.md` (layout rules, feature-file reading order), every multi-machine file under `model/temporal` (listed below) and the feature-file headers that list files, regenerated `model/ir/*`
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/testdata/layout/**, model/irgen/testdata/layoutRefusals/**, model/README.md, model/temporal/features/**/*.scala, model/temporal/foundations/**/*.scala, model/ir/*]

### Approach
- **The rule:** a file holds at most one top-level `Machine`, `Composition` or `Derived` object, and the file is named after it (`ResetSettlement.scala`). Types, signature declarations and realizations follow the existing layout rules. Decide with the owner whether realization objects fall under the rule too.
- **Flat folders, prefix grouping (owner's decision):** no subfolders or nested packages inside `product/`, `system/` or `realization/`. A family's files share a prefix and sort together, e.g. `system/Heartbeat.scala` (the facet), `HeartbeatCompletion.scala`, `HeartbeatRetry.scala`, `HeartbeatExhaustion.scala`; `ResetSettlement.scala`, `ResetKeepingPause.scala`, `ResetDeferred.scala`; `ClosePolicyRejectAfterClose.scala` and so on. Where an object's name doesn't start with its family prefix, rename the object so it does (e.g. `DeferredReset` → `ResetDeferred`; the close-policy designs → `ClosePolicy<Design>`, or another prefix the owner confirms). This renames IR ids, so list every rename in the done summary.
- **Prefix rule in the lint:** every file in a level folder is the level file, a shared kit file, or starts with a facet name from the feature's vocabulary or a declared design-family prefix. Refuse others, with a fixture.
- **Files to split** (as surveyed on 2026-10-10):
  - nexus workflow `system/ClosePolicy.scala` (9)
  - activity `system/DispatchWithTaskQueue.scala` (6)
  - task queue `system/System.scala` (4)
  - activity `system/Heartbeat.scala`, `system/Reset.scala`, `system/ResponseByID.scala` and nexus workflow `system/System.scala` (3 each)
  - activity `system/DispatchWithWorker.scala`, `system/Timeouts.scala`, `system/Dispatch.scala`, `system/DispatchRaces.scala` and task queue `product/Product.scala` (2 each)
  
  Re-survey before starting; some names may have changed.
- **Shared helpers:** where machines in one file share private helpers, values or comments, move them to the family's folder file, or to the level file. Keep comments with the machine they describe.
- **Enforce in the lifter's layout check** with a refusal naming the file and its second machine, plus a `layoutRefusals` fixture. Update the README layout rules and the reading order.
- **Drop the hand-maintained file lists** at the top of each feature file (e.g. the standalone activity's list of ~17 files and their contents) once the tree shows them. Keep a pointer to the template layout.
- **Regenerate:** object renames for prefixes change IR ids; and the IR records source file paths (`ResponseByID.scala` appears 52 times in `activity-standalone.json`), so moving machines changes the IR. Check that the diff contains only the listed renames, source paths and positions.

### Prerequisite
- Batch 5b (see MILESTONES.md): the lint rules land with Batch 5b's tooling; the file moves come after fn-162.3 removes the claim-container subjects (so they aren't split only to be deleted), and before fn-160.2's collection and the batch's single regeneration. No separate regeneration here. A working tree that builds the lifter (the in-progress IR `Duration` migration, plus `make protoc`).

### Batching
- Batch 1d (Scala DSL, immediate; see MILESTONES.md): land the Scala changes now. Checked-in IR, Cases and Go consumers may go stale. Verify with the Scala build, framework and model tests, and fn-160.5's step-table comparison under this task's rename map. Declare the IR change this task will cause; the IR is regenerated once, in Batch 5b, whose shared close supplies the gate evidence for the `make umpire-check-*` criteria. File moves come after fn-162.3 removes the claim-container subjects.

### Quick commands
```sh
make umpire-gen-model
make umpire-check-model
make umpire-check-lint
```
### Explicit R13 source coverage

The owner-confirmed kind/form layout belongs to this task as well as the
one-machine file rule. A kind holds its kind file and its forms; components
shared by two forms live under the kind's `shared/`, including the one Nexus
Product both forms refine. Each form's root feature file is `<Form><Kind>.scala`
(`StandaloneActivity.scala`, `StandaloneNexus.scala`, `WorkflowNexus.scala`).
Re-anchor to the integrated source and prove these existing migrations rather
than replaying moves or creating one Product per form. Preserve the exact machine
and Query population, source/identity maps and native table equivalence. The
structure lint must accept the intended shared Product and refuse both a
misnamed form root and shared components placed beside the forms. Use authored
source-coordinate fixtures, and retain the independent flat-level/prefix rules.
This records original R13 coverage, not new semantics or permission for held
managed regeneration, Go, independent review or live work.

## Acceptance

- [ ] No Model file declares more than one `Machine`, `Composition` or `Derived` object, and each such file is named after its object; level folders are flat, and families share a name prefix (facet or design family) enforced by the lint.
- [ ] The lifter refuses a second machine in a file, naming the file and the machine, with a refusal fixture; the README layout rules state the rule.
- [ ] Feature files no longer carry hand-maintained file lists.
- [ ] Regenerated IR differs only in the listed prefix renames, source paths and positions; `make umpire-check-model` and `make umpire-check-lint` pass.
- [ ] Kind/form/shared-package layout and `<Form><Kind>.scala` root names satisfy R13, with source-positioned acceptance/refusal fixtures and the existing single shared Nexus Product; identity/population and step-table proofs classify only intended source/name moves (R13).

## Done summary
Re-scoped to `features/activity_mini` and the shared code (chore commit 8fc711f8fc); implementation in adec9a39c6. Batch 1d (fa32b76586, d1536baa09) had already split the files one machine per file and added the second-machine/misnamed-file refusals, scoped to `model/temporal/`, but there was no family-prefix rule, no refusal fixture and no proof. This task adds those.

baseline: HEAD d7c9c57497 tree, lifted by the HEAD lifter from a frozen copy (`.flow/tmp/fn160/task3/before`, `head-irgen`).
stage: impl-review - skipped (REVIEW_MODE=none; independent review runs once at Scala batch end)

Enforcement scope (decision): the flat-layout rules hold every lifted Temporal Model (`model/temporal/`), plus any fixture with a `protocol/` or `realization/` folder. The older `system/` fixtures keep the earlier rules. Real features fail the rules as *behind* in their own lifts. The mini selection (`FEATURES=activity_mini`, the selected/copied lifts) contains no real feature, so the mini/shared lift is unaffected. The shared provider renames below also leave the real standalone activity uncompilable until the sweep runs `Fn160Task3FamilyNames` on it. I accepted that as "behind".

Requirement evidence:
- R7, one machine per file (existing rule, now pinned): the `layoutFiles` fixture (`model/irgen/testdata/layoutFiles/urn`, `LayoutFilesFixtures`) refuses `BoilSpare` as a second machine in `protocol/Protocol.scala:22`, naming the file and the machine, and refuses `BoilingOver` in `BoilOver.scala:6` (a misnamed file).
- R7, family prefixes (new, `Structure.families`): each file of a level folder must start, at a word boundary, with the level's name or the name of its primary machine, a facet (`enum FacetName`), or a family declared in the feature file (`enum FamilyName`). `Kit.scala` is exempt. Fixture: `Overflow.scala:6` is refused. `BoilDry` (family), `HeatingSlow` (facet) and `ProtocolMember` (level) are accepted. The HEAD mini/shared tree, lifted by the new lifter, is refused for exactly its 9 unprefixed files (`before-own.log`).
- Mini and shared moves/renames: mini declares `enum FamilyName: case answer, queue`. `TrustingAnswers`->`AnswerTrusting`, `ScheduledQueue`->`QueueScheduled`, `DispatchProgress`->`DispatchingProgress`, and `protocol/Faults.scala`->`AnswerFaults.scala` (file only). The task queue declares `enum FamilyName: case queue`, and `ForgetfulQueue`/`LossyMatchingQueue`/`VolatileQueue` become `QueueForgetful`/`QueueLossyMatching`/`QueueVolatile`. The hand-maintained file lists in `MiniActivity.scala` and `TaskQueue.scala` are dropped; the hard-instance inventory and a template pointer remain.
- R13 (kind/form layout, `<Form><Kind>.scala`): the mini root is `MiniActivity.scala`. A new fixture test materializes `layout/kinds` and refuses a misnamed form root (`relay/standalone/Standalone.scala`: "has no root feature file") and a shared Product beside the forms (`relay/product/`: "a kind keeps its sources in its general file, shared/product/ and named forms"). The existing single shared Product layout (`shared/product`) is unchanged.
- Rename map, IR identity: `.flow/tmp/fn160/task3/mini.map` and `steps.map` (6 machine ids, plus the composition member short name `scheduledQueue`). `irdiff.py`: the complete mini lift, before vs after, is equal modulo the rename map and source positions. Waivers are byte-identical. `posdiff.py`: every position is a before position under the file renames and the line maps of the edited files. Positions moved: 3 in MiniActivity.scala (header), 7 in TaskQueue.scala (header/FamilyName), plus whole-file moves for Faults (8), DispatchProgress (21), ScheduledQueue (12) and TrustingAnswers (29). Nothing else changed.
- Step tables: `FEATURES=activity_mini ... --check-equivalence <mini baseline> steps.map --steps-only`, exit 0, 16 tables, equivalent (equivalence-final.log). Baseline integrity.mjs PASS.
- Migration: `Fn160Task3FamilyNames` (scalafix) renames references, including imports. The mini names are scoped to the mini packages, comments and literals are kept, and a collision is refused. `FamilyNamesTest` 4/4. `prove-mini.sh umpire.migrate.RewriteFamilyNamesMini`: legacy names at legacy paths -> rule + moves restores all 16 accepted sources byte for byte (rule rewrote 7, 7 moves), and the complete mini lift and waivers are byte-identical (PASS). The file moves and family choices are manual. The recipe is in `model/migrate/README.md`, "Flat layout".
- Docs: model/README.md (layers paragraph: prefix rule, FamilyName, realization naming, tree as inventory; structure-lint paragraph; mini table; task queue tree), SEMANTICS citations, migrate README. Pins updated for the renames: Bounds.test paths, Fixtures.test lifts member, FixturePositions TaskQueue.scala lines, lifts identity-contracts (`forgetfulQueue`->`queueForgetful`), binding-order.tsv, migrate proofs/tests file names.

Verification: LayoutFilesFixtures 2/2; MiniRealizationFixtures 8/8; FacetAdmissionTest 1/1; FixtureIdentityContractSuite 2/2; migrate GuideSourcesTest, QueueAliasTest and FamilyNamesTest pass; prove-mini RewriteFamilyNamesMini, RewriteMini and RewriteEntersMini all PASS; selected mini lift exit 0 (final == after); equivalence exit 0; scalafmt --check clean on the new and changed lifter/migrate files (`MiniActivity.scala` import order is a pre-existing finding, untouched); git diff --check.

Failures retained (inherited, not fixed): FacetsTest 5 failures and BoundsTest 2. Both fail first on the pre-existing fn-149 grouping refusal ("is a Property, declared directly in ...properties"). BoundsTest lifts a frozen fn159/task5 mini jar. On top of that, FacetsTest's "unknown vocabulary" variants and BoundsTest's stale jar also now report my family-prefix refusal for their unprefixed `Breaking.scala`/old mini files. Not rerun on base.

Environment note: `model/build/model-scala.jar`/`.classpath` were rebuilt from the mini-selected sources (the full set no longer compiles). The previous ones are backed up in `.flow/tmp/fn160/build-backup/`.

Open obligations / deferred: real-feature splits, moves and renames (close policy designs, DispatchWithTaskQueue successors, `TrustingRecord…`, realization names) and running the rule on them, in the sweep. Checked-in `activity_mini/generated/ir.json` is stale in source paths and the 6 ids until Batch 5b regeneration. Go tests naming `forgetfulQueue`/`lossyMatchingQueue`/`volatileQueue`/`dispatchProgress` (tools/umpire/check, export) and table-pins (sealed, not edited) go to Batch 5b/sweep. The full irgen Fixtures suite was not run (heavy, inherited failures).

Deferred gates: make umpire-gen-model/-check-model/-check-lint, Go/proto/regeneration, full Fixtures/ConventionsFixtures suites, broad validation and independent review at batch end.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 8fc711f8fc, adec9a39c6
- Tests: timeout 2400 mise exec -- scala-cli test --server=false model/irgen --test-only umpire.irgen.LayoutFilesFixtures (2/2), scala-cli test model/irgen --test-only umpire.irgen.MiniRealizationFixtures (8/8), FacetAdmissionTest, FixtureIdentityContractSuite (pass); FacetsTest/BoundsTest inherited fn-149 grouping failures, scala-cli test model/migrate --test-only umpire.migrate.FamilyNamesTest (4/4), GuideSourcesTest, QueueAliasTest (pass), model/migrate/prove-mini.sh umpire.migrate.RewriteFamilyNamesMini / RewriteMini / RewriteEntersMini (PASS), .flow/tmp/fn160/task3/selected.sh final (mini lift exit 0, identical to after); irdiff.py and posdiff.py before(HEAD lifter) vs after: equal modulo mini.map and positions, FEATURES=activity_mini scala-cli run model/check -- --check-equivalence <mini-selected-baseline-00d22597d2> .flow/tmp/fn160/task3/steps.map --steps-only (equivalent, 16 tables); integrity.mjs PASS
- PRs: