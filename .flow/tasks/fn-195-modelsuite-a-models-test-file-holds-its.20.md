---
satisfies: [R3, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.20 Connect native find and seeded sample questions to ModelSuite

## Description
Implement R3's bounded find and seeded sample execution over the preceding native space declarations, including local R7 refusals. Consume the same committed .2 and partial .8 lifecycle/runtime checkpoint as the space leaf. Original .9 still owns complete Registry-selected integration.

**Size:** M, five expected files and four acceptance groups adapting the shipped checker engines.
**Files:** `model/framework/dsl/NativeRuntime.scala`, `model/framework/dsl/ModelSuite.scala`, `model/framework/checker/SuiteExecution.scala`, `model/framework/checker/SuiteExecution.test.scala`, new `model/framework/dsl/testdata/modelsuite/ExplorationClient.scala.txt`.
**Touches:** [model/framework/dsl/NativeRuntime.scala, model/framework/dsl/ModelSuite.scala, model/framework/checker/SuiteExecution.scala, model/framework/checker/SuiteExecution.test.scala, model/framework/dsl/testdata/modelsuite/ExplorationClient.scala.txt, .flow/tmp/fn195-native-questions/**]

### Approach
- Extend the existing DSL-owned `NativeSuiteExecution` request interface and the suite exploration registration branch. Keep all public signatures free of checker-owned Space/Expectation/Result types. Reuse the space carrier; if a necessary public find/sample declaration method belongs in `Spaces.scala`, explicitly widen Touches and serialize with its owner rather than introducing a parallel wrapper API.
- Capture `space.find(steps = horizon): expect: ...` using existing positioned Boolean capture and exact owner ingredients. Preserve every assertion and its order, source span and deferred state/preceding-step context. Reuse `checker/Spaces.scala:17` target restoration and `:37` find semantics; combine all assertions into the actual target without dropping earlier conditions. Include early-false/last-true and separately failing-line controls, then prove restoration after both successful and failing questions.
- Resolve starts through the existing test-level `from` or single-start rule and pass the exact retained starts into both find and sample. Missing `from` for a multi-start Model and a foreign start refuse; add no second start API.
- Pass find/sample requests through the one immutable suite execution selected in `ModelSuite.scala:33` and `SuiteExecution.scala:9`. Reuse `Execution.scala:21` and `Sampling.scala:17`. Horizon, sample count and seed remain independent of the execution budget. Preserve native witness, notFound, unknown and counterexample precedence; MUnit find requires found and sampling requires its successful finite result.
- Retain an immutable seeded sample request and actual selected paths as inputs the later C task can consume. Preserve seed, count, walk horizon, failing sample index, claims/monitors/progress and finite cuts. Do not bind a realization, select a singular Property, invent Case identity/defaults or make conformance eligible.
- Use directly targeted real public ModelSuites to prove registration and result disposition. The new testdata directory is a task-owned source fixture, not an existing runner. Execute its materialized clients with legal source/test and fresh separate-library arrangements, preserving strict warning and assertion-retention controls from .2.

### Investigation targets
**Required:**
- `model/framework/dsl/ModelSuite.scala:213` - exploration registration extension point.
- `model/framework/dsl/RunBody.scala` - positioned expectation capture and deferred context.
- `model/framework/dsl/NativeRuntime.scala:10` - DSL-owned native execution interface.
- `model/framework/checker/SuiteExecution.scala:9` - one exact-suite resolved execution.
- `model/framework/checker/Spaces.scala:17` - prior-step target context and ordinary find results.
- `model/framework/checker/Sampling.scala:17` - seeded paths and sampled claims.
**Optional:**
- `model/framework/checker/Questions.test.scala:78` - existing native question evidence controls.

### Verification and handoff
Baseline focused existing QuestionsTest, SamplingTest and SuiteExecutionTest through the current native README recipe, then run the new direct public suites and fresh outside-package library clients. Preserve exact selected tests, nonzero counts, terminal exits, strict compiler options, source/artifact hashes, intended refusal diagnostics and ordinary result evidence under this task's scratch root. Check zero declaration callbacks, restored Model state and actual previous-step outcome/before/records context, repeated mixed question use, unrelated-warning refusal and every assertion retained. Prove DSL/realize declaration compilation/loading without checker classes and explicit unavailable-checker refusal when execution is requested.

This leaf's direct selectors earn no complete global discovery, held-suite inventory or actual gate credit. Keep original .9's Registry-selected public suite check, .8/.14/.11 and fn-196.2 ordering unchanged. No new engine, policy parser or backend. All existing Temporal/Make/Go/export/IR/generated/server/Case holds persist; pending compile-only export approval remains ungranted. Formal review stays phase-batch deferred.

## Acceptance
- [ ] Find uses the retained exact-instance space, requested horizon and every positioned Boolean target. Controls distinguish found witness, completed-horizon notFound, unknown budget cut and decisive counterexample, with actual prior-step context and restored Model state. Missing/empty/non-Boolean targets and horizons outside `1..space.steps` refuse at the authored operation.
- [ ] Sample delegates to existing seeded walks and claims/monitors/progress, repeats identical paths for identical descriptor/seed and retains count/horizon/seed/failing index. A failing claim fails MUnit, cuts remain unknown and finite samples never imply exhaustive verification. Count below one and find/sample in examples refuse locally.
- [ ] Claims, replay, find and sample receive the exact same immutable resolved suite budget. Horizon/count/seed do not replace it. Foreign instances and repeated/interleaved questions cannot share state, mutate the reusable descriptor or reuse stale target context. The retained sample descriptor and selected paths need no Case or realization execution to be read by later C.
- [ ] Focused controls plus directly targeted public source and fresh separate-library ModelSuites execute nonzero counts, preserve strict warning/assertion-retention checks and exact diagnostics, and demonstrate deferred declaration-only loading plus explicit checker-absent execution refusal. Original .9 still owns actual Registry-selected integration.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
