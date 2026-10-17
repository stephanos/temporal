---
satisfies: [R1, R16]
---
# fn-141-shrink-the-ir-generator-one-description.1 Move the order lint, structure lint and marker checks out of the exporter

## Description
Part C: give the declaration-order lint, structure lint and marker checks a home and gate step of their own, preserving what they refuse. This task runs now under the owner's pull-forward and faster-tasks decisions in MILESTONES.md, from the integrated fn-169.6 source checkpoint. The original model/irgen paths are superseded by model/framework/export.

**Size:** M
**Touches:** [model/framework/export/**, model/framework/check/**, model/framework/lint/**, model/framework/inspect/**, model/temporal/export/**, model/temporal/check/**, model/README.md, model/framework/SEMANTICS.md]
New lint/shared-reader directories are options, not a prescribed layout. Keep modules deep, reuse the existing typed-tree index, and do not duplicate a TASTy read or create lint/export dependency cycles.

### Approach
- Re-read Order.scala, Structure.scala, Markers.scala, Lift.scala, Lifting.scala and Context.scala in model/framework/export before changing them. Preserve the single Index/TASTy read: both passes may use a small shared reader, or lint may own that reader and the exporter reuse it. Record the choice.
- Move the non-IR lints to their own module and expose their own gate step. Order and structure refusals stop export; marker checks remain after lifting and before any file is written, through an interface taking the index and lifted Models. Existing direct lift entry points must preserve refusal behavior.
- Move affected fixture tests with the code and split them out of Fixtures.test.scala beside their construct. Retain expected text/positions and unchanged expected IR. Historical fixture paths may remain as input data where a move would otherwise change author positions; record that choice.
- The mini feature, framework samples and shared code are the production scope. Real-feature sources/generated artifacts are untouched. Go, Makefile edits, regeneration, broad lint/full gates and independent review remain held.
- Before changes, record source lines by category for fn-141.14: declaration/DSL re-parsing, sugar matching, realization-kit matching, function-body/type lifting, lints and refusals. Record after counts and additions in the receipt; no standalone proof artifact is needed.
- Iterate only the affected suites. Baseline is .flow/tmp/checks/20261012-235203/record.md from fn-169.6. End once with make model-check-changed BASE=<task-base>, including mini step-table equivalence. The strict make model-mini gate remains red on held generated IR and is not a prerequisite. No bespoke captures, hash manifests or proof scripts.

## Acceptance
- R1: model/framework/export holds no Order.scala, Structure.scala or Markers.scala; these non-IR passes live in their own module and run as the gate's lint/check step with one shared TASTy read.
- Every moved order/structure/marker reject fixture retains its refusal text and source position. A source refused by a lint writes no IR, through the gate and direct lift routes.
- Fixture expected IR and mini/shared semantics are preserved. The task's make model-check-changed record passes, including mini step-table equivalence; generated artifacts are unchanged.
- R16: before/after source counts and category baseline, the shared-reader choice, affected tests, and deferred gates are recorded in the done summary.

## Done summary
Model authors retain the same refusals and lifted IR while declaration-order, structure, convention and marker checks now live in `model/framework/lint`. The gate reports source lint, IR lifting and marker checking from one inspector invocation, and the existing direct lift entry point runs the same checks before writing any files.

baseline: green via the preceding fn-169.6 source checkpoint, `.flow/tmp/checks/20261012-235203/record.md` (472 tests), reused under the activated task's owner-approved baseline policy.
Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(config: REVIEW_MODE=none)

The shared reader is `framework.inspect.Index`, extracted from Context with the positioned LiftError type. Export and lint depend on that leaf. Lint owns admission; Order runs the structure and convention checks. Export passes the existing Java service plug-in through lint's ModulePolicy; the service name and Temporal registration stay intact. Markers takes the shared index, a lifted Model, ordered declaration identities and enabled-rule action IDs. The last input preserves the disabled-fault behavior. Marker checks run after all files are lifted and before publishing IR or waivers; negative-control refutations still aggregate across the run. Model declaration names are unique within a lift under the existing distinctModelName check.

The order and structure tests were split from Fixtures.test.scala beside their checks. Admission, Conventions, EligibleWitness, LayoutFiles, PropertyGroups and QualifiedPhases suites moved beside the lint code and their coverage entries moved with them. MarkersFixtures now asserts every marker refusal against the existing pinned refusal contract through both direct and checked routes. The original 226-line rejection golden and all expected IR remain unchanged. Fixture source inputs remain under export/testdata so their authored file names and positions remain stable. SourcePositionsFixtures stays beside Context because it also tests export positions.

The new LintStagesSuite was run red before implementation. Its assertion failed because the checked-lift entry point did not exist. It now checks the three stages in order and proves that a source refusal stops before lifting and creates no output directory. MarkersFixtures also proves a post-lift marker refusal writes no IR through the gate entry point.

Source census for fn-141.14 follows exact whole-file buckets, excluding test sources and project.scala directives. These buckets identify the current homes of the work; refusal handling overlaps every bucket.

| Source category | Before | After | Files |
| --- | ---: | ---: | --- |
| Declaration and DSL re-parsing | 3,570 | 3,570 | Declarations, Claims, Compositions, Roles |
| Sugar matching | 2,837 | 2,837 | Syntax, Capabilities |
| Realization-kit matching | 2,711 | 2,711 | Realizations |
| Function-body and type lifting | 1,162 | 1,162 | Expressions, Types |
| Lints and their refusals | 2,824 | 2,885 | Order, Structure, Conventions, ConventionsLint, Markers; new ModulePolicy |
| Reader, orchestration and other plumbing | 2,121 | 2,174 | Admission, Constants, Context, Equality, Gaps, Lift, Lifting, Plugin, Switches; shared Index/LiftError and compatibility adapters |
| Total | 15,225 | 15,339 | Export, lint and inspect |

Exporter production sources alone fell from 15,225 to 12,249 lines, a reduction of 2,976. Lint owns 2,995 lines and inspect owns 95. The combined 114-line addition covers the shared-reader boundary, policy/stage/marker adapters, imports and formatting. The original Order/Structure/Markers subset was 2,227 lines. Its current lint home is 2,264 lines. Textual `fail(`, `refuse(` and `LiftError(` occurrences, including definitions, remain 706 across the production sources; this is a lexical cross-cutting count, not the future refusal ledger.

Focused verification passed 43 extraction tests, 27 remaining boundary/lint tests and 61 check/gate/coverage tests. The lint module compiled on its own with no exporter or system sources. Targeted scalafmt checking passed. The authoritative task check passed once in 458.9 seconds with 278 lifter tests and 154 check tests (432 total). Its command was `make model-check-changed BASE=1935e54127`, with its scope, test counts, exit codes and mini step-table equivalence recorded at `.flow/tmp/checks/20261013-023527/record.md`.

The strict model-mini comparison, regeneration, real-feature checks, production Go reader/lint/Case-lowering gates, broad Model/lifter scalafix lint, live runs and independent review remain deferred under the activated task contract. Makefile and Go changes are outside this task; their broader module-map/formatting reconciliation remains with the conductor at the batch boundary.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 048e23e2733d1581e806b7ab461e70f6ed44c710
- Tests: make model-check-changed BASE=1935e54127, mise exec -- scala-cli compile model/framework/lint, mise exec -- scala-cli run --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.LintStagesSuite framework.lint.OrderFixtures framework.lint.StructureFixtures framework.lint.MarkersFixtures framework.lint.ConventionsFixtures framework.lint.AdmissionSuite, mise exec -- scala-cli run --test model/framework/export model/temporal/export --main-class org.junit.runner.JUnitCore -- framework.export.LintStagesSuite framework.lint.MarkersFixtures framework.lint.LayoutFilesFixtures framework.lint.PropertyGroupsFixtures framework.lint.QualifiedPhasesTest framework.lint.EligibleWitnessFixtures framework.export.GapsSuite framework.export.SourcePositionsFixtures, mise exec -- scala-cli run --test model/framework/check model/temporal/check --main-class org.junit.runner.JUnitCore -- temporal.check.CoverageSuite temporal.check.GateSuite temporal.check.GateLintSuite, mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/framework/inspect model/framework/lint model/framework/export/Lift.scala model/framework/export/LiftError.scala model/framework/export/LintStages.test.scala model/framework/export/Context.scala model/framework/export/Lifting.scala model/framework/export/Gaps.scala model/framework/export/Gaps.test.scala model/framework/export/ConventionsLint.scala model/framework/export/Plugin.scala model/framework/export/ModelBuild.test.scala model/framework/export/SourcePositions.test.scala model/temporal/check/Coverage.scala model/temporal/export/TemporalPlugin.scala model/temporal/export/SuiteCoverage.test.scala
- PRs: