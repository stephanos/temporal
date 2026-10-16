---
satisfies: [R4, R5]
---
# fn-188-framework-only-source-iteration.2 Establish the framework-only build and verification loop

## Description
Current admission: this broad exporter/verification task remains separately held. Its fn-188.3 dependency is complete with native-batch review SHIP; the admitted fn-188.4 prerequisite owns sample-suite discovery and explicit Go holds. Neither closure authorizes resumption here automatically. Preserve the full successful supported-generic gate obligation and failed/interrupted evidence; focused prerequisite or artifact compilation success is not this task's completion. Demonstrated unmet framework DSL requirements still take priority. Before later admission, re-anchor actual supported suites and their remaining semantic prerequisites; never count held/skipped checks as passing or run the loop as a DSL-only gate.

**Touches:** [Makefile, model/framework/**, model/README.md]. Reuse the existing check-framework entrypoint, framework.check Registry/Module/Tools, Boundary and export ModelBuild artifact seams rather than creating a parallel harness. The implementation and interrupted fn-174.2 repairs are preserved at checkpoint 437016b7d148b87cccee78c926af92119dec7fb4, not uncommitted work. Reconcile only current diagnosed gaps; do not rebuild the already implemented runner. Package fresh generic framework and compiler plugin artifacts without model/project.scala or Temporal source/API/Model jars, classify discovered suites honestly, and run every supported generic suite to terminal success. Account for SampleExpectTest, NativeMapsTest, NativeVotingTest and NativeHistoryTest; preserve Go-dependent SamplesSuite cases as held, with no pass credit. No model/temporal reads, schemas, Go execution/changes, tools/umpire, backend or regeneration work. Keep one build lane. Native expect and record-input implementation are already complete and do not wait for this loop.

## Acceptance
Historical gate evidence remains failed/incomplete: the first full run exited 2 with 280 export cases and six failures; the later rerun was interrupted with exit 143 and JUnit error markers, without a terminal export summary. The six focused repairs passed 43 cases, not the full gate. Keep .flow/tmp/fn1882-final-summary.md and its evidence intact; do not reinterpret an artifact-only checkpoint as full success.

A documented command builds/tests the framework with Temporal entirely excluded; no stale Temporal artifacts or API dependencies enter generic fixtures; every framework suite, including native samples and Go-dependent cases, is accounted for with explicit held integration reasons; boundary/command regression tests and the actual supported framework gate pass to terminal completion; checkpoint 437016b7d148b87cccee78c926af92119dec7fb4 repairs and failed/interrupted evidence are preserved; no Temporal source is restored or consumed and no Go check is run in this lane. Focused artifact compilation alone does not complete R4/R5 or this task.

## Done summary
The framework-only gate (`make check-framework`) terminated green at source 191f5b7333: boundary PASS, fresh core and plugin builds exit 0, core 466 tests, check 100 tests, export 286 tests, 117 generic suites and 21 held suites (record .flow/tmp/checks/framework.7448953529908255698, log .flow/tmp/fn1882-green-gate.log). Temporal sources stayed excluded; no Go, schema or regeneration ran.

Reaching it re-anchored the framework exporter, lifter, reify plugin capture/schema phases and their fixtures to the native DSL surface (expectation blocks, native bindings and evidence clauses, witness finishing view, applyDynamic classes, scalar and record inputs, rule cases and guard cases, member selections, native state data, retired totals and tokens), across commits 97265776e8..191f5b7333 including the three integrated constructed-suite lanes (f904974025, 17f33ff864, f633f7f67d). Frozen originals were re-pinned only where fixtures legitimately changed, each documented next to its test.

Held, with explicit reasons and no pass credit: the system-backed suites (Temporal sources, API, realization kit), the two Go-running SamplesSuite cases, and the sample lifts (ReifySamplesSuite and one FiniteSchemaSuite case), which need fn-173.4's held native-collection IR schema. The earlier failed/interrupted evidence (.flow/tmp/fn1882-final-summary.md, fn1882-goal-gate-summary.md) is preserved. fn-174.24 semantics, Temporal/Go/backend and full-spec obligations remain with their owners.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 97265776e8, 6bba1e6fc7, 06c12a13b1, 64896c0b59, 9d18a2c837, 8ca9723359, eb4d00a7d8, 88d0e220d8, 191f5b7333
- Tests: make check-framework: boundary PASS; core 466; check 100; export 286; 117 generic suites, 21 held (record .flow/tmp/checks/framework.7448953529908255698)
- PRs: