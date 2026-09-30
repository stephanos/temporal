---
satisfies: [R5]
---
# fn-106-gomad-close-the-remaining-tests-gaps.5 Resolve the traceback address leak and the worker-commands seed-11 hang

## Description
Find why the SDK panic traceback prints host thread-stack addresses and why seed 11 never delivers the worker cancel command; fix or classify each.

## Acceptance
- each is fixed (skip removed) or classified with its channel


## Done summary
Traceback leak fixed: while Gomad is enabled the runtime prints a possibly-dead traceback argument slot as a bare '?' instead of its stale value (patch hunk in runtime/traceback.go, added to the patch allowlist; regenerated with patch-regenerate from the pristine source). TestWFTFailureReportedProblemsTestSuite qualifies on seeds 11 and 17 with its skip removed. Worker-commands seed 11 classified: the cancel command does not reach the control queue before the test's 90s default context runs below the 2s long-poll minimum, after which every poll is refused with 'Context timeout is too short'; the test's 120s Awaitf exceeds its own context. The skip stays with that reason; seed 17 qualifies.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: gomad qualify-set TestWFTFailureReportedProblemsTestSuite: qualified seeds 11 and 17, gomad qualify-set TestWorkerCommandsTaskSuite (unskipped): seed 11 target_failure, seed 17 qualified
- PRs: