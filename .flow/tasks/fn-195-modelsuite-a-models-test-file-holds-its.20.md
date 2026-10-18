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
# fn-195.20 native question checkpoint

ModelSuite exploration tests now register deferred find and sample requests through the existing receiver, options, slug and lifecycle admission. One immutable native suite execution delegates those requests to the existing find and seeded-sampling engines, retaining ordinary results, positioned target conditions, exact owners and starts, sample descriptors and selected paths.

Final source checkpoint: `ad087b4e70eab24c6e9606c00bd26912e722b3a2`; original audited checkpoint: `708d21473df0d2ebba034a434ba814c158720914`. Baseline: green, 15 SuiteExecutionTest, 8 QuestionsTest and 7 SamplingTest controls before implementation at `4c540b75e7aa4ce1658a15b07107049c53c2771e`.

Tier: session (jev-unavailable(no_key)); AGENTS pinned worker6.1medium

stage: impl-review - skipped(config: REVIEW_MODE=none; policy: owner phase-batch formal review deferral)
stage: plan-sync - skipped(config: planSync=false)
Tracker sync: n/a (bridge inactive).

### Acceptance evidence

- `review-native-v2` executes 20 SuiteExecutionTest controls: 5 new question controls and all 15 existing replay/claims controls. Assertions inspect actual native Results, budgets, witness/counterexample paths, sample metadata and Model state. They cover found, completed-horizon notFound, unknown cuts/holes, decisive found-before-cut and throwing-effect refutation; every target condition and its evaluation order; distinct failing lines, including nested Boolean failures; actual before/outcome/records context; nested successful and unsuccessful questions; exact-instance and foreign-start refusals; missing multi-start from; invalid horizons/counts and missing targets. Samples retain claims, monitors, progress, seed/count/horizon/failing index, repeated paths and finite-experiment limitations.
- `review-public-v2` and `review-client-v2` each execute 5 outside-package controls against the real public ModelSuite: 4 execution controls plus a compiletime executable-prefix refusal control. Each harness checks 17 nested test results, including 8 intended failures: notFound (1), unknown find (1), bad sampled claim plus automatic claims (2), and the exact `outsidequestions.ExplorationFixture$BudgetSuite=1` override across replay/find/sample/claims (4). These intended native failures remain in the logs; their successful assertion by the outer harness supplies no passing native-question credit. The test-level from callback stays deferred at registration and runs once for each question. `review-direct-v2` and `review-client-direct-v2` each directly select the real PublicExplorationSuite and execute its 4 tests.
- The separate-library client compiles only materialized authored clients against a freshly packaged native jar and the verified retained IR dependency. Its public control checks the Explore class's actual jar origin. Every recipe uses a fresh `--workspace`, Scala 3.9.0/JVM 27/MUnit 1.2.0 and the unchanged strict compiler options, including assertion-tree retention.
- `review-declarations-source-v2` and `review-declarations-client-v2` prove declaration loading/registration without checker classes and zero authored callbacks. Execution of the registered public suite produces 4 explicitly asserted checker-unavailable failures per route. Deferred multi-start registration and the prefix compiletime refusal are also checked. These harnesses print 5 outer passes, but their 2 execution-specific branches are inactive in declaration-only mode and earn no semantic credit. `review-declarations-library-v2` packages DSL/realize without checker sources.
- `review-controls-v2` executes 28 ingredient controls: the existing RunBody/ModelSuite controls (21) plus all SpacesTest controls (7), including the ordinary fixed-input run and receiver/context regressions from .19. `review-quick-v2` executes the 4 CheckerTest controls through the admitted scratch recipe.
- Source and library each independently produce 6 located grammar refusals: an exploration body without a question, find/sample in examples, empty/non-Boolean targets, and the separate actual executable-prefix reproduction. The prefix reproduction is refused before any execution output. Source and library each independently refuse 2 unrelated warning controls under unchanged flags. Count these 12 direct compiler refusals and 4 warning controls separately from executed tests. Each compiler exits 1 as intended, and `validate-v2.py` verifies diagnostics, positive test counts and statuses. The encompassing `audit-verify-v2.sh` exits 0; no original failed or successful recipe was overwritten.

### Red and failed preparation evidence

`questions-red-v1` fails compilation because NativeSuiteExecution has no find/sample methods, with consequent typing errors. `registration-red-v1` fails on the explicit uninstalled-exploration diagnostic, plus a collateral warning. These are compile-red observations, not executed failing tests.

`foreign-target-red-v1` executes the new early-false/later-foreign-owner control and fails because no ownership exception occurs. `foreign-target-green-v1` passes all 19 controls after owner validation moves ahead of Boolean evaluation; predicates retain their order and short-circuit behavior.

The fresh two-axis audit found an executable-prefix admission defect and a nested first-failure position defect. `audit-mutation-red-v1` executes the actual public ModelSuite: its two native checks pass with initial State(count=99), and the probe fails because Model.count remains 99 instead of 0. `audit-line-red-v1` executes 20 controls: the new nested-line control fails because the outer expression's line overwrites the inner false line, and the other 19 pass. Joined wrapper 44856 observes both exit 1 results before fixes. Checkpoint ad087 rejects authored prefix blocks while preserving direct inlined question calls, and mirrors replay's first-failure guard. The complete fresh v2 matrix above verifies both fixes in source/library arrangements. Original reports and conductor dispositions remain verbatim in `quality-reports.md`; this supplies no formal phase-batch verdict.

All other failed attempts remain immutable. `baseline-v1` selected zero tests and is inconclusive; corrected exact selectors establish the green baseline. `adapters-v1` has two fixture typing failures. `adapters-v2` reveals over-specific callback expectations for unspecified space positions and a mismatched owner-error expectation; the corrected control pins both positions explicitly and preserves native search semantics. `registration-v2` exposes the long inherited whole-action warning/nowarn mismatch in an ordinary-run neighbor, which uses the equivalent explicit fieldless call in the final fixture. `registration-v3` additionally selects ineligible nested suites through an overbroad selector. `registration-v3/v4` expose an unretained bare claim initializer, and `registration-v6` reaches the named claim but refuses its same-line initializer. The final fixture follows the established named, separate-line claim form. `registration-v5` exits 2 during preparation before compiler startup and earns no test credit.

### Immutable bindings and scope

`source-review-v2.sha256` binds the five final task-owned repository files. `artifacts-review-v2.sha256` binds the versioned recipes, copied sources, dependency, commands, tool versions, logs, terminal statuses and produced libraries under `.flow/tmp/fn195-native-questions/`. Their SHA256 values are `dd44664fa5c803c480432a0cc3f7b0f5b7f2afd9b9ccd3b6a051e250f0d40c23` and `2ea44161ca5324da133f7b62e521e90ffc3ebfbc0ff2ed093e1c5ede12073165`. Repeated formatting leaves the final source binding unchanged. Task-scoped `git diff --check` passes. The original source binding remains a historical708 checkpoint; the original artifact binding still independently verifies and has not been refreshed to bless changed source. Mutable quality supplements live outside these execution artifact manifests.

- Final native library SHA256: `7453eaf722be19e66a7b4bc7d32f0602092fcb0b71236111c6fc0576dacbd82b`.
- Final declaration library SHA256: `f2360fd267a997ea8d0e5a81d9cae7c1f20bfeaa66c349c2f84927c9bc979959`.
- Retained IR dependency SHA256: `b9e0538afffad2b33b9fc8fc9eb2a3f4e63b10928fa38506b585747535e85c92`, copied only from `.flow/tmp/fn195-native-spaces/native-library-v1/native/model/build/ir-scalapb.jar` into new task scratch roots.

The user-removed repository `model/build/ir-scalapb.jar` remains absent. The canonical Quick receipt check refuses reuse because the worktree contains user changes; the canonical recipe's dependency is unavailable. `review-quick-v2` is an honest scratch observation with a different command, not a canonical receipt. `classify.log` reports exit 1, FULL, naming user-owned MILESTONES.md. No docs-only skip or full changed-scope gate credit is claimed. The owner-held Make/export/plugin/lifting/IR/schema/backend/generated/Go/Temporal/server/Testpilot/recordedRun/Case operations remain unrun; export compile-only approval remains ungranted.

Refutation and the final refusal/reference matrix remain .21's work. Actual Registry-selected integration remains original .9's work; .8/.14/.11 and the stronger fn-196.2 ordering obligation remain open. Descriptors and selected native sample paths supply reusable inputs only, with no Case or realization execution and no conformance eligibility claim.

The source commits contain only the five declared task paths. Unrelated roadmap/spec/task changes and the user's index are preserved through explicitly authorized task-scoped staging. Intervening commit `a651ad74ea7cc91b4fa4c288dae6f50e34ff4f93` belongs to the conductor's fn209 planning and contains only its 12 Flow files; it is not .20 implementation ownership. A broad range whitespace observation in that unrelated plan does not make a task-scoped whitespace pass a globally clean claim.

### Processes and quality

Every started task compiler/JVM command is joined. Original matrix 81156, expected-RED audit wrapper 44856 and final v2 matrix 46151 all have terminal exit 0. Their individual expected compiler/test failure statuses remain in the logs and in `processes.md`. No task compiler lane remains; the conductor transferred that lane to the separate fn209 worker. Historical Bloop PID 93084 was untouched.

The conductor accepts final source ad087 after independent source/artifact checks and the two supplemental audits. Both original findings are empirically resolved; no outstanding quality finding remains. All four reports and the final bounded acceptance remain in `quality-reports.md`. Formal phase-batch review remains deferred, with no formal review verdict claimed.
## Evidence
- Commits: 708d21473df0d2ebba034a434ba814c158720914, a651ad74ea7cc91b4fa4c288dae6f50e34ff4f93, ad087b4e70eab24c6e9606c00bd26912e722b3a2
- Tests: env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-questions/run.sh baseline-suite-v2 baseline, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-questions/run.sh baseline-questions-v2 baseline checkercontrol.QuestionsTest, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-questions/run.sh baseline-sampling-v2 baseline framework.checker.SamplingTest, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-questions/audit-run-v1.sh audit-mutation-red-v1 mutation outsidequestions.MutationProbeTest, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-questions/audit-run-v1.sh audit-line-red-v1 source, env -u BASH_ENV bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn195-native-questions/audit-verify-v2.sh, sha256sum -c .flow/tmp/fn19520/source-review-v2.sha256, sha256sum -c .flow/tmp/fn19520/artifacts-review-v2.sha256, git diff --check -- model/framework/dsl/NativeRuntime.scala model/framework/dsl/ModelSuite.scala model/framework/checker/SuiteExecution.scala model/framework/checker/SuiteExecution.test.scala model/framework/dsl/testdata/modelsuite/ExplorationClient.scala.txt
- PRs: