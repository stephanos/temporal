---
satisfies: [R1, R16]
---
# fn-141-shrink-the-ir-generator-one-description.1 Move the order lint, structure lint and marker checks out of the exporter

## Description
Part C: give the declaration-order lint, structure lint and marker checks a home and gate step of their own, preserving what they refuse. This task runs now under the owner's pull-forward and faster-tasks decisions in MILESTONES.md, from the integrated fn-169.6 source checkpoint. The original model/irgen paths are superseded by model/framework/export.

**Size:** M
**Touches:** [model/framework/export/**, model/framework/check/**, model/framework/lint/**, model/framework/inspect/**, model/temporal/export/**, model/temporal/check/**, model/README.md, model/SEMANTICS.md]
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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
