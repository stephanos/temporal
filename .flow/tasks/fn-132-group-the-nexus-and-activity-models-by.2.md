---
satisfies: [R1]
---
# fn-132-group-the-nexus-and-activity-models-by.2 Standalone activity moves under features/activity/standalone

## Description
**Size:** M
**Touches:** [model/temporal/features/standaloneactivity/**, model/temporal/features/activity/**, model/ir/**, model/cases/**, tools/umpire/**, common/testing/testpilot/**, tools/canary/**, tests/*.go, tests/testcore/testpilot/**, Makefile, model/temporal/IrFiles.test.scala, model/temporal/capabilities/Catalog.test.scala, model/irgen/test/Fixtures.test.scala, model/irgen/testdata/hintsInvalid/Invalid.scala, model/irgen/testdata/lifts/HintRejects.scala, model/irgen/testdata/lifts/Hints.scala, model/irgen/testdata/lifts/ScriptRejects.scala, model/irgen/testdata/lifts/Scripts.scala, model/irgen/testdata/lifts/Admission.scala, model/irgen/testdata/lifts/Members.scala, model/irgen/testdata/lifts/expected/hints.json, model/irgen/testdata/lifts/expected/hintsRefused.json, model/irgen/testdata/lifts/expected/rejects.txt, model/check/Metrics.scala, model/check/SourceMetrics.scala, model/check/test/SyntaxRule.test.scala, model/check/test/Gate.test.scala, model/irgen/Order.scala, model/irgen/Structure.scala, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, .plans/ACTIVITY_MODEL_COMPARISON.md, MILESTONES.md]

**Required investigation:** current activity feature/product/system trees; `tools/umpire/ir/layout_test.go:454`; current split-reader projection helper; fn-128/fn-129 via flowctl. Update downstream Flow prose only through flowctl. Preserve local product/system Phase/State/Fact and actual `client`/`activity` identities; no semantic renames beyond the package/file map.

Part A for the activity. Move `features/standaloneactivity` to `features/activity/standalone`, as folders and packages, with `product/` and `system/` (`System.scala`, `Record.scala`, `WithTaskQueue.scala`, `Realization.scala`). Keep the three System realizations in `standalone/system/Realization.scala`; do not create a root or Product realization file. `StandaloneActivity.scala` becomes `Standalone.scala`; the existing Scala tests move with it. Add `features/activity/Activity.scala` as the kind's general feature file (a header comment and package declaration; task 6 fills it).

The exact IR export stems are `activity` → `activity-standalone`, `activity-record` → `activity-standalone-record` and `activity-race` → `activity-standalone-race`. Corresponding export vals follow those stems; no machine, Query or intrinsic instruction is renamed beyond its declared package identity. Record the complete finite map before regeneration, including the source-file-derived compiler package owner where fixture diagnostics name it. Update every live reference: Go `tools/umpire`, Makefile targets, Case trees, canary, docs, and the paths in the fn-128 and fn-129 specs/tasks and `.plans/ACTIVITY_MODEL_COMPARISON.md`. Update only fn-133's current placement parenthetical to the new path via flowctl; preserve its explicitly historical review context and already-correct future placement. Force-add only exact CLI-mutated downstream authoring files if local exclusions hide them; do not change exclusions or sweep other ignored files.

**Observed live fixture/compiler dependencies:** Canonical regeneration compiled and tested the Models and lifted the production IR, then failed because the exact named lifter fixtures still import the retired activity package. Update only their imports, package/source-name assertions and explanatory references. The two expected hint IR files and refusal text receive the independently declared finite identity/source-path map, preserving every other field, source line, diagnostic and assertion; no golden recapture. Add this expected-fixture mapping to the proof ledger and retain all positive/negative fixtures. `Order.scala` and `Structure.scala` are authorized only for current example comments, not executable code or classifier/validator behavior. `Metrics.scala`/`SourceMetrics.scala` examples and SyntaxRule/Gate test sample paths/export names follow the same declared map without weakening checks. Task 3 still owns the final R4 specimens and classifier/docs completion.

The Model root-construction assertion in `IrFiles.test.scala` compares exports to the actual checked-in IR filenames. Stage the declared IR filename move before regeneration so this dynamic assertion can run; do not replace it with a fixed set or introduce an update-mode bypass. Preserve the first failed regeneration and the later fixture-import failure as actual evidence, not full-gate passes.

Independent of task 1; the two may run in either order but not at the same time (both regenerate `model/ir`).

**Finite downstream context maintenance (root authorization):** Update only fn-119.3's Required Nexus realization to `features/nexus/workflow/Realization.scala` and Activity realization to `features/activity/standalone/system/Realization.scala`, removing obsolete line ranges; fn-119.4's style target to `features/activity/standalone`; and fn-125.10's current Files/Required targets to `activity/standalone/{Standalone.scala,system/System.scala,system/Realization.scala}` as appropriate and `nexus/workflow/Realization.scala`. Preserve deferred statuses, dependencies, behavior and unrelated retired Go-reader targets. In fn-123's parent add one explicit historical-context sentence identifying its pre-fn-112 source paths/ranges; preserve the snapshot content and proposed semantics. These authoring edits use flowctl only; force-add exact CLI-mutated hidden authoring paths, never change exclusions. No whole-Flow-clear claim covers unrelated historical contexts.
## Acceptance
- [ ] `features/activity/standalone` exists with the spec's layout; `features/standaloneactivity` does not.
- [ ] A before/after projection with the path and IR-file-name map applied is identical.
- [ ] A path/package/export-aware search finds no retired standaloneactivity Model references outside history, archived plans and closed specs; fn-128/fn-129 use new paths and current local level types. All three realizations remain in the System file, with no root/Product replacement.
- [ ] Focused move proofs pass, then the shared required full model/Go/runtime/artifact/lint batch passes once after both Part A moves. The done evidence explicitly discharges the prerequisite/first move's deferred gates, retains JSON/exit/separate wall-time logs and states any not-required backend/live-run scope honestly. No golden recapture or weakened lint is accepted.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
