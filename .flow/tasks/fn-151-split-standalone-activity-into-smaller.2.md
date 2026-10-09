---
satisfies: [R2, R3, R4]
---
# fn-151-split-standalone-activity-into-smaller.2 Extract focused verification from ActivitySystem into smaller subject models

## Description
Extract focused verification from ActivitySystem into complete subject models after task .1. Keep the three tasks serial. Re-anchor the source and generated artifacts against HEAD 4755faca73 and the completed task-.1 move; the pre-fn-145 IR is not the baseline.

**Size:** M

**Files:** Lifecycle core and focused Activity subjects, execution/export attachments, Scala regressions and the named owner-dependent Go tests below. Generated artifacts remain read-only until task .3.

The new `ActivitySubjectRegression` suite declares package `framework`, matching the existing Activity regression suites. Focused test acceptance requires the captured output to name `framework.ActivityRetryRegression`, `framework.ActivityResetRegression` and `framework.ActivitySubjectRegression`, with executed test counts and their actual results. A zero-suite or zero-test selector is a failure, not passing evidence.

**Scope and retained lifecycle.**

ActivitySystem keeps Phase, State, Fact, init/end, evidence, state helpers used by transitions, all effects and rules, its Product refinement including map/visibility/unobservable timers, shared capabilities and their controls, repeated Retries/Deadline bindings, reset-aware replacement factories and instance-specific overrides, and existing capability bounds. Preserve worker task-token and service by-ID rules independently. Keep cancelIsNotUndone, resetSettles, resetResumes, resetKeepsPaused and controlPrecedence with their existing free verify queries. Keep terminated/terminate and generated activitySystem.terminateSettles with the core Terminable capability. Existing reset, heartbeat and by-ID subject models retain task .1's declaration identities and semantics.

Use complete derived subjects in package temporal.features.activity.standalone.system. Each owns its properties, scenarios and queries. Do not introduce category files, new transitions, a new state projection, framework/schema/runtime changes, or fn-155 work. Keep filenames subject-first and integrate with task .1's Dispatch*, Retry*, Heartbeat*, Reset* and Response* files.

**Derivation proof before extraction.**

Use the existing explicit zero-change derivation ActivitySystem.rebind() for the new subjects. Bare Derived(ActivitySystem) is refused by the current lifter; unmonitored also removes refinement and must not substitute for this derivation. Before moving claims, lift an ordinary minimal derived subject through the existing gate and compare its complete transition relation with ActivitySystem: ordered state/action/outcome/fact catalogs, init/end, evidence, every enabled/disabled/hole row and ordered result including facts/outcomes/choice metadata, phase projection, unobservable timers, refinement target/map/visibility, assumptions and monitor attachments/evaluation points. Compare the retained core with the post-.1 baseline too. Explain target/owner identity differences separately. If the existing zero rebind fails this proof, stop dependent extraction and report the concrete incompatibility; do not change the DSL or switch to unmonitored to bypass it. This proof is an early task-.2 gate, not a second evaluator.

**Concrete owner and query map.**

All rows below keep family temporal.features.activity.standalone.system and the existing query name. The old owner is activitySystem. New objects derive with ActivitySystem.rebind(); each moved scenario uses its new owner. Existing query form, full action sequence, limit name/steps/actions/search, total, expectations including reasons and monitor expectations, and exploration metadata remain exact.

| File / owner | Property | Scenario | Query | IR file | Limits / total at 4755faca73 | Case standing |
| --- | --- | --- | --- | --- | --- | --- |
| Completion.scala / completion | completes | completed | completion | activity-standalone.json | three / 16848 | lowered |
| RetryFailures.scala / retryFailures | nonRetryableFails | nonRetryable | nonRetryableFailure | activity-standalone.json | three / 16848 | lowered |
| RetryFailures.scala / retryFailures | retryCompletes | retriedThenCompleted | retry | activity-standalone.json | six / 33696 | lowered |
| RetryFailures.scala / retryFailures | retryExhausts | exhausted | retryExhaustionByFailures | activity-standalone.json | six / 33696 | nothing-to-realize |
| Cancellation.scala / cancellation | canceledByWorker | cancelRequestedThenCanceled | cancel | activity-standalone.json | four / 22464 | unsupported |
| Cancellation.scala / cancellation | cancelRequestedWhileStarted | cancelRequestedThenCanceled | cancelRequest | activity-standalone.json | four / 22464 | unsupported |
| Pausing.scala / pausing | completes | pausedThenCompleted | pauseResume | activity-standalone.json | six / 28080 | lowered |
| DispatchEligibility.scala / dispatchEligibility | completes | delayedThenCompleted | startDelayedCompletion | activity-standalone.json | four / 22464 | lowered |
| DispatchEligibility.scala / dispatchEligibility | dispatchRequiresReady | any | delayedAttemptsAreNotDispatched | activity-standalone.json | eight / 5346432 | nothing-to-realize |
| DispatchEligibility.scala / dispatchEligibility | scheduleToStartRequiresDispatch | any | scheduleToStartWaitsForDispatch | activity-standalone.json | eight / 5346432 | nothing-to-realize |
| Timeouts.scala / timeouts | scheduleToStartFires | scheduleToStartExpires | scheduleToStartTimeout | activity-standalone.json | three / 16848 | lowered |
| Timeouts.scala / timeouts | startToCloseFires | startToCloseExpires | startToCloseTimeout | activity-standalone.json | three / 16848 | unsupported |
| Timeouts.scala / competingTimeouts | scheduleToStartFires | bothDeadlinesStartFirst | competingTimers.scheduleToStartFirst | activity-standalone-record.json | three / 11232 | no-realization |
| Timeouts.scala / competingTimeouts | scheduleToCloseFires | bothDeadlinesCloseFirst | competingTimers.scheduleToCloseFirst | activity-standalone-record.json | three / 11232 | no-realization |

DispatchEligibility avoids colliding with the existing Dispatch enum. Its focused file belongs beside task .1's cohesive Dispatch.scala protocol models, DispatchRaces.scala, DispatchWithTaskQueue.scala and DispatchWithWorker.scala. Timeouts and CompetingTimeouts are separate complete subjects in Timeouts.scala. Move the two System deadline scenarios and their competingTimers queries out of ActivityRecord.queries in task .1's Dispatch.scala together. Root CompetingTimeouts.queries only in activity-standalone-record; it has no realization and no expectation. Root Timeouts.queries and its execution attachment only in activity-standalone. Keep both timer orders and the no-realization standings; adding an execution attachment to CompetingTimeouts would change coverage and admission.

The former ActivitySystem/completes claim maps one-to-three to completion/completes, pausing/completes and dispatchEligibility/completes. Each new owner declares the same exact worker.respondCompleted predicate independently; no foreign-owned property alias serves another owner's scenario. The former scheduleToStartFires claim maps one-to-two to timeouts and competingTimeouts; scheduleToCloseFires maps to competingTimeouts. Preserve the exact predicates in each instance and account explicitly for these additional claim instances. Move completedOnRetry into RetryFailures.states as a shared state constant and update TimeoutRetry's two reads in task .1's Retry* file plus ActivityRetryRegression. Preserve the full state equality, attempt saturation and policy semantics.

**Realization and identity attachments.**

Keep Standalone as the core Realizes(ActivitySystem) binding. In system/Realization.scala add zero-change DerivesFrom(Standalone, Subject) objects CompletionExecution, RetryFailuresExecution, CancellationExecution, PausingExecution, DispatchExecution and TimeoutsExecution for the corresponding new owners. They reuse the complete base header, controller, workers, evidence, server-step derivation and controls. Root those typed objects in exports.activityStandalone in the root standalone/Standalone.scala. Do not copy or redesign scripts. CompetingTimeouts receives no realization in either IR.

Produce an explicit complete baseline-to-scratch inventory keyed by IR file + family + owner + query name, and a property/scenario inventory keyed by family + owner + kind + name. Include identity mappings for new subject targets, property predicate functions, scenario owners, realization qualified IDs/names, Definition IDs and behavior/projection fingerprints. Carry the baseline generated Case identities/local references/checksums and manifest standings into the map and give task .3 the exact expected final attachments/identity mapping to verify after regeneration; do not claim regenerated Case equality in task .2. Unchanged query names keep the existing family/query identity; moved owner and realization identities must be derived and checked, not assumed byte-identical. For every lowered row the expected managed filename remains activity-standalone-<query>-case.json; task .3 must map any generator-required change explicitly. Do not rewrite historical Runs or compatibility fixtures.

Every original query outside the table remains under its prior owner and IR export with its prior realization/standing. Account for retained core lifecycle queries, all 23 core capability queries, existing TimeoutRetry/heartbeat/by-ID/reset subjects, worker composition, dispatch protocol/race/queue models and unrelated IR files. Record old/new file/owner/query/realization/Case standing and identity for every inventory entry, including unsupported/nothing-to-realize/no-realization entries; keep unsupported construct/evidence details apart from allowed source-position changes.

**Consumer changes.**

Make TestActivityEveryClaimDeclarationIsLifted in tools/umpire/check/activity_parity_test.go compare exact family/owner/kind/name keys. Its current kind + simple-name set collapses the three completes owners. Preserve explicit constructor-name overrides, unnamed val names, existing repeated capability expansion and claim-origin checks. Include the new subject files and correct IR memberships, and add focused regressions proving missing one completes owner, crossing an owner and losing an explicit-name override each fail. Do not replace exact coverage with name-only membership or permissive discovery.

Update source consumers ActivityRetryRegression.test.scala and ActivityResetRegression.test.scala, retaining their exact predicates, selected action/path, limits, expectations and wrong-landing checks. Keep core controls/reset regressions under their retained owners. Update reader row/path/mutation expectations and conformance lookups from actual mapped query owners instead of hard-coded activitySystem. Retain all row counts, failing rows, witnesses and full catalog coverage through the exact mapping. Preserve fatal witness satisfied and its lack of reset coverage.

Inspect and update the named consumer surfaces below against the 4755faca73 generated baseline and new scratch owner map. Changes are limited to actual owner/function/file/attachment mappings and the exact-key source-coverage fix; keep negative cases and assertion strength. Task .2 owns these consumer source edits, not Go execution against stale production IR. Task .3 verifies them after the batch's single production regeneration. Update stale Record.scala/WithTaskQueue.scala filename comments retained verbatim inside task .1's declaration bodies; list those comment changes as explicit nonsemantic deltas. Task .3 is generated-artifact-only and must not inherit this source documentation work.

**Touches:** [model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/activity/standalone/system/Completion.scala, model/temporal/features/activity/standalone/system/RetryFailures.scala, model/temporal/features/activity/standalone/system/Cancellation.scala, model/temporal/features/activity/standalone/system/Pausing.scala, model/temporal/features/activity/standalone/system/DispatchEligibility.scala, model/temporal/features/activity/standalone/system/Timeouts.scala, model/temporal/features/activity/standalone/system/RetryTimeouts.scala, model/temporal/features/activity/standalone/system/Dispatch.scala, model/temporal/features/activity/standalone/system/DispatchRaces.scala, model/temporal/features/activity/standalone/system/DispatchWithTaskQueue.scala, model/temporal/features/activity/standalone/system/DispatchWithWorker.scala, model/temporal/features/activity/standalone/Standalone.scala, model/temporal/features/activity/standalone/system/Realization.scala, model/temporal/features/activity/standalone/ActivityRetryRegression.test.scala, model/temporal/features/activity/standalone/ActivityResetRegression.test.scala, model/temporal/features/activity/standalone/ActivitySubjectRegression.test.scala, tools/umpire/check/activity_parity_test.go, tools/umpire/check/activity_properties_test.go, tools/umpire/check/activity_pins_test.go, tools/umpire/check/activity_system_test.go, tools/umpire/lower/activity_cases_test.go, tools/umpire/lower/withholding_test.go, tools/umpire/lower/origin_identity_test.go, tools/umpire/lower/waits_test.go, tools/umpire/lower/inventory_test.go, tools/umpire/lower/published_test.go, tools/umpire/conformance/activity_test.go, tools/umpire/conformance/played_test.go, tools/umpire/conformance/outcome_test.go, tools/umpire/export/quint_test.go, tools/umpire/lint/api_test.go, tools/umpire/lint/kinds_test.go, tools/umpire/lint/bindings_test.go, tools/umpire/lint/holes_test.go]

This is a closed source-edit list. Generated model/ir and model/cases changes belong only to task .3. The four existing Dispatch* files allow mapped references and stale filename comments, not protocol redesign. Add ActivitySubjectRegression.test.scala for ordinary derivation and independent claim-owner assertions. Leave untouched listed consumer files that need no mapped change.

**Investigation targets:**

- `model/framework/Machine.scala:424` and `model/irgen/Declarations.scala:960` — existing zero-change rebind and emitted refinement/monitor semantics.
- `model/temporal/features/activity/standalone/system/System.scala:540` — exact focused/core/capability inventory and independently owned predicates.
- `model/temporal/features/activity/standalone/Standalone.scala` and `system/Realization.scala` — artifact roots and complete typed execution attachments; the latter path is relative to that standalone directory.
- `model/temporal/features/activity/standalone/ActivityRetryRegression.test.scala` and `ActivityResetRegression.test.scala` — shared state, fatal witness, paths and expectation pins.
- `tools/umpire/check/activity_parity_test.go:58` — exact owner-qualified declaration coverage, explicit names and capability origins.
- `tools/umpire/lower/origin_identity_test.go` and `withholding_test.go` — mapped origin and retry identities without changing negative controls.
- `model/check/Gate.scala:464` and `model/irgen/Lift.scala:183` — existing scratch packaging/classpath/lift stages and per-artifact emission.

**Quick:**

Run from the temporal repository root. Use the same existing limits/assertions; record resource failures as failures. Before source edits, seal the post-.1 source in scratch stage `before`. Repeat the exact packaging/classpath/lifting sequence at stages `early` (minimal ordinary derived subject, before claim moves) and `after` (complete extraction). The package, classpath and lift arguments below are the existing gate's stages from model/check/Gate.scala:464-520 and model/irgen/Lift.scala:183-215, with their output paths moved to ignored scratch. Do not add a gate flag or production update. Preexisting model/build/ir-scalapb.jar and api-scalapb.jar must be current from fn-145's close; report a stale prerequisite instead of mutating framework/schema/API outputs.

```sh
fn151_stage=before
mkdir -p ".flow/tmp/fn151/task2-$fn151_stage/ir"
mise exec -- scala-cli --power package --server=false --suppress-outdated-dependency-warning --library model/project.scala model/framework model/temporal -f -o ".flow/tmp/fn151/task2-$fn151_stage/model-scala.jar"
mise exec -- scala-cli compile --server=false --print-class-path model/project.scala model/framework model/temporal --suppress-outdated-dependency-warning > ".flow/tmp/fn151/task2-$fn151_stage/model-scala.classpath"
mise exec -- scala-cli run model/irgen --suppress-outdated-dependency-warning --main-class umpire.irgen.lift -- --ir ".flow/tmp/fn151/task2-$fn151_stage/model-scala.jar=model/" ".flow/tmp/fn151/task2-$fn151_stage/model-scala.classpath" ".flow/tmp/fn151/task2-$fn151_stage/ir"
mise exec -- scala-cli test --server=false model/project.scala model/framework model/temporal --suppress-outdated-dependency-warning --test-only 'framework.ActivityRetryRegression'
mise exec -- scala-cli test --server=false model/project.scala model/framework model/temporal --suppress-outdated-dependency-warning --test-only 'framework.ActivityResetRegression'
mise exec -- scala-cli test --server=false model/project.scala model/framework model/temporal --suppress-outdated-dependency-warning --test-only 'framework.ActivitySubjectRegression'
mise exec -- scala-cli fmt --scalafmt-conf model/.scalafmt.conf --check model/project.scala model/framework model/temporal
```

Keep all three complete scratch IR trees and comparison receipts. Compare the early derivation and final focused source/scratch IR through the explicit mappings, including all rows and same-property truth sets, totals/limits/expectations and realization declarations. Native Scala assertions preserve selected paths/query expectations; task .3 records reader-evaluated answers/exercised status/witnesses and generated Case semantic contents against the regenerated artifacts. Do not execute updated Go consumers against stale production owner IR or run a strict live-conformance suite solely for this task's Quick check. Task .3 runs the canonical Go verification with `-tags test_dep -p 2 -timeout 30m -json`, preserving exit status/logs and the current machine-wide heavy-suite lock, after the single production regeneration. Export tests requiring Quint/Apalache retain the fn-154 memory deferral and their assertions. Known completion/fatal/pause live failures remain Batch 5 with unchanged strict expectations. Do not diagnose/fix those failures, fit the Model to runtime, or reduce coverage/bounds/assertions to make this task cheaper.
## Acceptance
- [ ] Focused completion/failure-retry/cancellation/pausing/dispatch/deadline verification is extracted into the named complete subject models after task .1; ActivitySystem retains all original lifecycle transitions, refinement/evidence, capabilities/controls and cross-control/reset laws without framework/schema/runtime or fn-155 changes.
- [ ] An early ordinary lift proves ActivitySystem.rebind() preserves complete catalogs, starts/ends, every transition row/result, evidence, phase projection, refinement map/visibility, unobservable timers, assumptions and monitor semantics. Every new subject uses that existing derivation; no bare Derived or unmonitored substitution bypasses the proof.
- [ ] The exact IR/family/owner/query and family/owner/kind/name inventories account for every original claim/query, the one-to-three completes and one-to-two scheduleToStartFires mappings, shared completedOnRetry consumers and all retained capability/subject queries. Property truth sets, query forms/actions/answers/witnesses, totals/limits and expectations remain unchanged through the mapping.
- [ ] Executable subjects receive zero-change DerivesFrom(Standalone, Subject) attachments and consistent activity-standalone exports. CompetingTimeouts stays exclusively in activity-standalone-record with two no-realization queries, no realization/expectation and both independent deadline orders.
- [ ] TestActivityEveryClaimDeclarationIsLifted checks exact family/owner/kind/name keys, preserving explicit-name overrides, unnamed declarations, capability expansion and origins. Focused negative checks reject a missing completes owner, crossed owner and dropped explicit override; all other regression/mutation/row assertions retain their strength.
- [ ] The complete owner/query/realization/Definition-ID/fingerprint map retargets the named Scala, reader, lowerer, conformance, lint and export consumer sources and gives task .3 exact Case identity/standing checks. Baseline lowered/nothing-to-realize/unsupported/no-realization standings and unsupported meanings remain exact. Embedded stale Record.scala/WithTaskQueue.scala filename comments are updated as documented nonsemantic deltas; historical recordings and unrelated artifacts remain untouched.
- [ ] Focused Scala tests/formatting and complete before/early/after scratch-lift comparisons have recorded results. No production model/ir or model/cases regeneration and no Go checks against stale owner IR occur in task .2. Task .3 owns the single production regeneration, generated identity/standing/Case checks and canonical `-tags test_dep -p 2 -timeout 30m -json` Go/model/lint verification. Resource failures remain failures, inherited completion/fatal/pause strict failures remain Batch 5 and fn-154 Quint memory work remains deferred; no limits, coverage, assertions or expectations were weakened.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
