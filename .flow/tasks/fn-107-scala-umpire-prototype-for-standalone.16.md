---
satisfies: [R2, R11]
---
# fn-107-scala-umpire-prototype-for-standalone.16 Make Scala v2 independent and restore the legacy Scala baseline

## Description
**Touches:** [model/scalav2/**, Makefile, model/scala/umpire/Action.scala, model/scala/umpire/Compose.scala, model/scala/umpire/Domain.scala, model/scala/umpire/Machine.scala, model/scala/umpire/Refine.scala, model/scala/umpire/Search.scala, model/scala/umpire/Sets.scala, model/scala/umpire/Assume.scala, model/scala/umpire/Channel.scala, model/scala/umpire/Monitor.scala, model/scala/umpire/test/Declarations.test.scala]

Make the prototype's Scala sources and tooling independent of model/scala. This is a bounded adoption of the reviewed framework and selected activity/Nexus models into model/scalav2/scala, followed by guarded removal of task 2's legacy delta. Preserve package names and semantics. No feature policy, Go API or IR schema change is required.

**Size:** L
**Files:** v2 authoring project/framework/models, compiler wrapper and proto-jar/gate wiring, provenance fixtures, README, Scala Makefile prerequisites, exact eleven legacy paths above.

### Approach
- Copy the necessary framework, case producer and selected native models/tests into model/scalav2/scala. Keep comments and user baseline bytes. Own compiler helper, toolchain directives and both generated Java jars inside v2; reuse shared protobuf inputs and generic Go models.
- Rewire v2 generation, fixture compilation/lifting, native tests and lint. Native declarations compile independently. The Makefile discovers the new scala/project.scala. No legacy source/helper/jar may be a build or runtime input.
- Actual copied-source lint probe found 51 DisableSyntax findings (19 casts,5 nulls,18 vars,6 loops,2 runtime type checks,1 return): .flow/tmp/fn-107/isolation-lint-probe/lint.log, rc96. Resolve lint with behavior-preserving changes or justified existing line-scoped conventions; preserve comments and lint coverage/config. Record normalization; do not weaken semantic or identity pins.
- Regenerate IR from source. Compare strictly after accounting only for source-location/provenance relocation and recorded formatting line moves. Tables, IDs, refinement, evidence, stuck states and behavior fingerprints stay equal.
- Before restoring each legacy path, compare its current hash to task2-review-r2/manifest.json. For seven existing paths restore exact task2-baseline/bytes; remove four new task-owned files only after adoption is verified. Preserve any later user drift and reconcile it rather than overwrite it. No Git reset/revert/checkout/stash.
- Run meaningful positive and negative isolation controls in a temporary snapshot lacking model/scala and v2 generated jars. Regenerate, compile, native-test, lift, check, lint and compare deterministic artifacts there. The saved pre-isolation gate must fail in the same snapshot. Never move/delete the user's live legacy directory.

### Investigation targets
**Required:** model/scalav2/{run.sh,gen.sh,lifter/Lift.scala,README.md}; model/scala/{project.scala,scala.sh,gen-proto.sh}; selected umpire/temporal sources; Makefile Scala section; task2 baseline bytes/review manifest; current task15 admission/interpretation API; isolation-spec drafts and lint probe in .flow/tmp/fn-107.
**Optional:** legacy proof/view code for dependency inventory only; existing shared Go/Lean/Testpilot fixture inputs.

### Quick commands
GOFLAGS=-tags=test_dep make umpire-check-scala; make lint-scala; mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/...; scoped Go IR lint if its diagnostic paths change. Final isolation snapshot runs the same gates and deterministic regeneration. Record inherited global lint failures separately.

### Wrap-up handoff
The implementation is in commit `e1d0753f417ab0791d94a018f0148b0390f1b350`. V2 owns the framework, selected activity/Nexus Models, compiler helper and generated proto jars. The eleven task-2 legacy changes were guarded and restored; unrelated legacy files and user edits were preserved. Independent native review has no verdict, so this task is unfinished.

Fresh wrap-up verification: `mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` passes. `mise exec -- scala-cli test model/scalav2/scala/project.scala model/scalav2/scala/umpire model/scalav2/scala/temporal --server=false --suppress-outdated-dependency-warning` passes all 66 tests. The scoped Go lint command also fails during package loading with `no go files to analyze`; its cause is not established. The normal Scala gate cannot start Bloop because its cache output is outside the writable workspace; Scala lint cannot acquire Coursier's `.structure.lock` for the same reason. Earlier full live and isolated-snapshot gates passed before these execution restrictions; those earlier results do not establish a fresh full gate.

The prior isolated snapshot omitted the legacy tree and all v2 jars, regenerated all three jars, passed build/test/lint, and reproduced checked-in IR twice byte for byte. Restoring the old gate or introducing legacy references/source positions failed for the expected reasons. A separate snapshot ran the legacy gate with v2 absent. Conductor reconciliation found no semantic IR differences after documented provenance/line relocation, no lost comment text across 40 adopted files, no drift in 53 other legacy files, and no change to `specs/KNOWN_BUG.md`.

Resume by repairing native reviewer startup, then run task 16's review and final gates before marking it done. Do not reset the transport ledger or widen the reviewer sandbox. Before task 3, reconcile the additive generic key-level claim/query/composition prerequisite; the current typed constructors cannot bind arbitrary IR-built tables. Its proposals and task-3 binding draft are preserved under [continuation](../artifacts/fn-107-scala-umpire-prototype-for-standalone/continuation/orchestration-recommendation.md). They remain drafts; no new task was admitted or implemented.

The user's wrap-up request explicitly authorizes committing and pushing this checkpoint, superseding the earlier no-commit constraint. It does not declare the prototype or this task complete.
## Acceptance
- [ ] V2 owns its authoring framework, selected models, compiler helper and generated jars; build/lift/test/runtime inputs do not read model/scala. Existing package names and source-linked IDs are preserved.
- [ ] In an isolated snapshot without the legacy Scala tree or prebuilt v2 jars, generation, native compile/tests, IR checks, focused Go tests and configured Scala lint pass. The pre-move gate fails in that same snapshot, and repeat generation produces byte-identical artifacts.
- [ ] Strict pre/post semantic comparison preserves tables, Definition IDs, refinement rows, evidence, stuck states and fingerprints. Only documented source provenance/positions and normalization line moves differ; no pin is weakened and feature IR is not handwritten.
- [ ] Existing comments and preexisting user edits are retained. Only the eleven guarded task-2 legacy paths are restored/removed; all other legacy source bytes remain unchanged, and the restored legacy native baseline still passes in a separate snapshot.
- [ ] Legacy-dependency checks inspect matched content and IR source-location fields accurately, with an explicit negative control; they cannot discard every hit by filtering the containing v2 filename. Every v2 IR source position resolves within v2.
- [ ] Configured lint covers the adopted Scala project without relaxing rules or dropping source coverage. Exact normalization and exception reasons are recorded; all required live/snapshot gates and command closures have actual exit-code receipts. No staging, commits, pushes or worktrees.


## Done summary
Scala v2 owns its framework, selected activity and Nexus Models, compiler helper and generated jars under `model/scalav2`; the eleven task-2 legacy paths were restored. The implementation is commit `e1d0753f41`.

The Codex implementation review found one defect: composition admission in `goir/load.go` keyed and counted a member action that a sync consumes as a class of its own, unlike `model/go/umpire` composition. It is fixed (`syncedActions`), with tests for a sync named like the class it takes, a Scenario that schedules a synced member action, and the corrected key count (100, not 120). The re-review in the same session returned SHIP with no findings.

Isolation was re-run on 2026-09-30 in a snapshot with `model/scala` absent and no v2 jars: `make umpire-check-scala`, `make lint-scala` and `make umpire-gen-scala` exit 0, regeneration leaves every checked-in IR file byte-identical, and no IR source position names `model/scala/`. Not re-run: the saved pre-isolation gate failing in that snapshot, and the legacy gate with v2 absent; their earlier logs were lost with the previous checkout. The Lean-dump parity comparisons skip here because the dumps and the Lean toolchain are gone.

The review fix is uncommitted; the owner makes the commits.

stage: implement - ran (earlier session; commit e1d0753f41)
stage: impl-review - ran (codex; round 1 NEEDS_WORK by gpt-6-astra at high, the flowctl default, not the routed gpt-5.6-sol; round 2 SHIP in the same session 01a0f4fe-a647-7e30-9565-2359a77932c7 over the uncommitted fix diff)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e1d0753f417ab0791d94a018f0148b0390f1b350
- Tests: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc 0; Lean-dump comparisons skipped), GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), make lint-scala (rc 0), GOFLAGS=-tags=test_dep make lint-code LINT_CODE_TARGETS=./model/scalav2/goir (rc 0, 0 issues), isolated snapshot without model/scala and v2 jars: umpire-check-scala, lint-scala, umpire-gen-scala rc 0; IR byte-identical (.flow/tmp/fn-107/task16/iso.log), codex impl-review: .flow/tmp/fn-107/task16/review-16-r1.json (NEEDS_WORK), review-16-r2.md (SHIP), review-16-fix1.diff
- PRs: