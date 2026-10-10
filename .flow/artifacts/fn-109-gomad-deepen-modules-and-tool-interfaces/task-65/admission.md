# Explicit private preparation for scripted Runner coverage

The conductor selects the two-operation design from the parallel [seam research](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/seam-research.md) and [fixture research](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/fixture-research.md), both bound to `f0ed4a36d47c12d72383ea5de80370cec8581c9f`. Their SHA-256 values are respectively `d374e64e54a242c4c111e6b99a5a2cda8275f82404e72eabd911bcdff76c06c3` and `f81d159f769083e21c73cadfef61760b92e7ad30ad155a44cdd868669a7596ed`.

Under the user's instruction to follow recommendations rather than ask when in doubt, this dated decision admits private per-call preparation and bootstrap functions with lazy real fallbacks. It extends existing private injection under R5, not the public API. Alternatives that replace whole campaign phases would bypass the behavior being tested; exposing preparation service internals would broaden a separate module boundary. Adapter verification and toolchain-identity hooks have later concrete consumers but are not admitted here. No unused four-operation scaffold, generic dependency framework or configurable production host guard.

Task 63's reviewed decomposition landed in primary at `f0ed4a36d4`. Its ordinary source acceptance remains open. This task is a corrective owner for that acceptance gap, not admission of subsequent simulation/storage tasks, and introduces no dependency cycle requiring task 63's failed tests to pass before they can be repaired. fn-109.15/.26, fn-152 and other downstream owners keep their existing dependencies. Nothing here completes fn-109.63 or fn-112.10.

## Production boundary

Use the existing private `executionDependencies` carrier, with only complete preparation and bootstrap function fields. Nil fields call the original operations at the original points with the original arguments and errors. Embed/copy that carrier in `campaignRuntime`; preserve it during resumed request reconstruction. Do not eagerly populate defaults. Keep the isolated rejection's current position and error text, and reject either function even with nil executor before invoking it or launching a coordinator.

Route only `preparation.Prepare` in `localCampaign.prepareTarget` and `profile.BootstrapFrame` in `runSeed`. Keep actual journal transitions, target-preparation root, progress, scheduling, counters, output opening/closing, target verification, World assessment, publication, resource lifetime, comments, error precedence and transactions unchanged. Do not change plan, replay, guidance, minimization, shard, adapters, toolchain discovery, runtime or storage owners. The resumed request must retain the dependency value, but portable resume coverage is not claimed from this slice.

## Explicit fixture scope

Attach a test-only adapter at exactly the existing calls in these six tests in `runner_test.go`:

- `TestRunPreparesOnceBoundsParallelismAndGroupsMatchingFailures`
- `TestRunPublishesConnectedWorldBundle`
- `TestRunClassifiesConnectedWorldDeadlock`
- `TestRunCountsConnectedWorldReplayDivergence`
- `TestRunRejectsInvalidConnectedWorldBeforePublication`
- `TestRunRejectsPreparedTargetMutationBeforeFailurePublication`

Keep every existing assertion and helper definition unchanged. The new helper requires an explicit preparer and scripted executor; call the real fixture preparer once with its actual journal-owned root, retain its actual copied file/hash/size/permissions and set the empty nonnil adapter slice. Use visibly synthetic bootstrap bytes only for executors that do not decode them. Verify the actual forwarded profile/target/Runner/seed and bytes in a focused contract case with concurrency-safe observations. Do not forge host/toolchain qualification or replace target verification.

Do not change `testConfig`, infer portability from executor dynamic type, install global hooks, migrate all `...With` calls, replace tests wholesale, weaken assertions or alter native fixtures. Real error/waiting preparers retain the real preparation owner and its StageTarget classification. Actual target/compiler calls, mixed public/private cases and the ordinary crash subprocess retain separate source/native dispositions from the full inventory.

## Verification and ownership

Before implementation, run the unchanged target-mutation regression on the current source and retain its expected preparation-stage failure. A compiler error, missing tool, timeout before that assertion or skipped test is not a meaningful RED. After implementation the same assertion must observe the actual mutating executor and `prepared_target_integrity`. Run all six unchanged behavioral assertions, dependency argument/error/default controls, preparation-only bootstrap refusal, isolated refusal of each new field with nil executor, public profile guards, real preparation error/cancellation controls, existing local phase controls and package architecture boundaries with `test_dep`.

At the frozen batch, run one ordinary Runner observation and compare all original named outcomes with the retained task 63 candidate: only these six admitted tests may change their original pass/fail disposition. New controls are listed separately. Retain any newly reached failure as evidence, not a reason to relax an assertion. Run configured affected lint, affected vet/errortype, formatting, generated validation and repository fast lint. Root serializes Go/build/lint/generator gates with task 64 and compares the current original-base lint against the retained 53 findings once for the combined batch. No diff-filtered lint or focused pass substitutes for aggregate acceptance.

The implementation worker's product surface is four production paths and four test paths, exactly as declared in the task description. Root owns lifecycle, independent review, commits and integration. Aggregate failures stay source-owned and must be resolved before formal completion; native fn-128/fn-149 remain deferred and unverified. No push, PR, CI, native execution, lint exception, policy widening or format migration follows from this admission.
