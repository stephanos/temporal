---
satisfies: [R3]
---
# fn-216-mini-activity-vertical-slice-the-model.3 Go test: in-process server, SDK worker answering from the Model, observations judged

## Description
Exercise R3 with a real repository server and SDK worker against the committed Temporal guest.
Keep the production Testpilot dependency boundary unchanged.

**Size:** M
**Files:** server integration test, local module wiring and existing live-gate prerequisite.
**Touches:** [tests/testpilot_model_slice_test.go, go.mod, go.sum, Makefile]

## Approach
- Place TestTestpilotModelSlice in the existing tests package with test_dep and integration tags. Reuse newTestpilotTestEnvironment and the established standalone-activity dynamic configuration. External test packages under Testpilot are still scanned and cannot import tests/testcore; do not bypass that gate.
- Add root require/replace entries resolving `umpire.local/model` to the local generic host and `umpire.local/temporal` to the local extension module. No push, publication or new third-party SDK dependency is needed.
- Load the guest artifact from an explicit build prerequisite and use actual SDK Client.ExecuteActivity with maximumAttempts=1. Register a real activity before starting its SDK worker. The SDK body sends a partial callback to the owner and returns only the decoded Model reply; failure becomes a non-retryable ApplicationError.
- Follow the parent contract's phased single-owner loop. Judge initial Scheduled responses with the worker stopped, start the worker, handle its delivery callback and Answer, then synchronously poll Describe and Judge. Positively check repeats and the correct final phase; do not infer unseen Poll or discard stale responses.
- Run Complete and Fail inputs with reset/isolation. Assert a single attempt and callback. Add the impossible terminal mutation after the valid final assertion or in a separate reset run; require Unexplained and frozen-state behavior rather than continuing the positive scenario.
- Make callback-channel waits and test cleanup cancellation-aware. A host refusal/error must reach the owner, stop the scenario and release worker shutdown without a leaked reply waiter. Use require.Eventually or explicit bounded channel coordination, never sleeps.
- Add `umpire-check-model-slice` with the Temporal guest prerequisite and exact TestTestpilotModelSlice selector, failing if that test is absent or unselected. Make the existing umpire-check-live-tests target depend on this passing slice target without removing or filtering its legacy tests. A fake activity server or SDK test suite cannot substitute.
- Before implementation, record the existing TestTestpilotGeneratedCases baseline using the same tags and CGO setting. Its current root in tests/testpilot_generated_test.go points at model/temporal/features while the committed manifests are archived; tests/testcore/testpilot/model_fixture.go refuses a root without manifests. Record the broad umbrella's outcome separately before/after. New slice failures and introduced legacy regressions block; an unchanged pre-existing legacy failure remains visible and is not represented as a pass. Do not relocate frozen fixtures, rebuild Scala or redesign Cases to satisfy this task.
- Record per-call Answer latency and total live-test time with the environment/toolchain for .4. Do not fold live-server timing into the model feedback target.

## Investigation targets
**Required:**
- `tests/testpilot_testenv_test.go:14` - dedicated SQLite test environment.
- `tests/testpilot_generated_test.go:46` - standalone-activity feature configuration.
- `tests/activity_standalone_test.go:107` - real standalone RPC precedent.
- `common/testing/testpilot/temporal/worker/sdk.go:43` - existing SDK construction.
- `common/testing/testpilot/dependency_boundary_test.go:214` - all-source dependency scan.
- `Makefile:554` - actual TestTestpilot gate.

## Quick commands
Run CGO_ENABLED=0 mise exec -- go test -tags 'test_dep integration' ./tests -run '^TestTestpilotModelSlice$' -count=1 with the built guest, then make umpire-check-model-slice and the relevant Testpilot dependency controls. Attempt umpire-check-live-tests and compare its result to the recorded pre-change legacy baseline, reporting unchanged failures separately. Run make model-check to retain guest/host parity.

## Acceptance
- [ ] Complete and Fail scenarios run against a real in-process server and real SDK worker; every valid observation is judged and the final phase agrees with the Model reply.
- [ ] The test proves owner ordering, a single attempt/callback, positive repeats, reset/isolation and an impossible terminal mutation without changing the independent Model.
- [ ] Cancellation, host errors and cleanup release callback waiters and terminate the worker; no sleep-based synchronization or fake-server substitution is used.
- [ ] Root local module wiring works without publication; the dedicated slice gate runs the test and passes, the unchanged-scope umbrella invokes it, boundary controls stay intact, and Answer/total timing evidence is recorded. Existing legacy Case failures are classified against a pre-change baseline; introduced regressions block.

## Done summary
TestTestpilotModelSlice now exercises Complete and Fail against a real in-process Temporal server and SDK worker using the independent WASM Model. This is a bespoke Temporal SDK/server/Model-host proof; it does not execute generic Testpilot Prepare/PreparedCase.Run, Case, Program or Contract integration.

The owner judges Scheduled responses with the worker stopped, answers the SDK delivery, then synchronously judges every Describe response. Complete/Fail check one attempt and callback, repeats, reset, instance isolation and opposite-terminal freeze. HostRefusal and Cancellation check reply-waiter release and shutdown. The activity returns only the decoded Model reply, using the SDK failure converter for non-retryable failure. The independent Model hash remains e9df73c0a44e0b4cd1ab5fcfdca7228b3653b9f9a27e531b76886c4beac87ad6.

Local root require/replace entries connect the two nested modules without changing pinned versions. make umpire-check-model-slice builds the real guest and requires the parent plus all four scenarios to pass. make umpire-check-live-tests depends on it and supplies the guest to its unchanged broad selector. Missing test, unselected parent, skipped parent, skipped Fail, missing guest and producer-failure controls all failed as intended. Temporary mutations were restored before commit. Logs are fn216-3-control-*.log under .flow/tmp.

baseline: red. The pre-edit GeneratedCases command failed for absent manifests, and the broad umbrella had seven failures. Model and dependency checks were green. The fixed pre-review and final post-review umbrellas retain all seven identities and assertion causes. The final Go suite took 360.505s and Make exited 2. Both slice invocations passed with relative CARGO_TARGET_DIR. The umbrella is not green and has no green receipt.

| TestTestpilot suffix | Unchanged cause |
| --- | --- |
| AssessRecordedRuns | Expected 1, received 3 with rejected-subject at testpilot_assess_test.go:101. |
| ExplorationDiscoversUnpinnedExecution | Missing model/ir/nexus-workflow.json. |
| GeneratedCases | No manifest.json under model/temporal/features. |
| NexusControlForgedCompletionIsViolated | Invalid or duplicate correlated transition. |
| NexusControlReplaysThroughTheCommand | Missing model/ir/nexus-workflow-control.json. |
| NexusPairCase | Expected 1, received 3 after optional.none() dereference in the relation capture predicate. |
| UmpireRunRunsACheckedInCaseAgainstAnyEndpoint | Exit 3 from the missing generated asyncCompletion Case. |

The complete comparison is .flow/tmp/fn216-3-legacy-comparison.md. Raw before/after logs are fn216-3-baseline-umbrella.log and fn216-3-final-umbrella-relative.log. The initial introduced umbrella guest-environment failure is retained in fn216-3-pre-review-umbrella.log; the explicit environment wiring fixed it. No legacy checks or fixtures changed.

Post-review exact slice, dedicated slice, dependency controls, lint and full make model-check passed. Logs are fn216-3-final-{exact,slice-relative,boundary,lint,model-check}.log. Receipts are .flow/tmp/green-receipts/b65f9fac-{smoke,unittest}.json. Concurrent preliminary compilation killed one compiler and timed out two checks; those observations remain inconclusive in fn216-3-concurrent-compilation.md and their original logs. Serial replacement checks reached the intended assertions and passed their controls.

Final dedicated live timing was 381.104ms including cleanup, with Answer calls of 55.417us Complete, 48.750us Fail and 31.166us HostRefusal. The final umbrella recorded 461.192ms and 321.701ms for its two live slice invocations; all per-call values remain in its log. Environment was linux/arm64, Go 1.27.0, SDK 1.48.0, CGO_ENABLED=0 and Rust 1.99.1. Verification ran serially in this task lane; no globally idle-machine claim is made. Compilation time is separate from these test timings. Final make model-time measured 9.746s full and 1.819s inner against 30s/10s soft targets, in fn216-3-final-model-time.log.

Three fresh Codex contexts returned SHIP with zero findings. Their rollout turn_context events confirm gpt-6.1-sol at high; .flow/tmp/fn216-3-review-execution.json records the evidence. Writer and reviewers share the Codex family. Fresh reviewer test commands were blocked by the read-only filesystem, and worker logs supply execution evidence. Receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-216-mini-activity-vertical-slice-the-model.3.json; fan-out evidence is .flow/review-fanout/27513ded9f2c43cc9c8ac7ddc3f3f185/.

stage: impl-review - ran [2026-10-19T18:50:56Z..2026-10-19T18:53:37Z] (model: gpt-6.1-sol at high)

Tier: session (jev-unavailable(no_key))

Tracker sync: n/a (bridge inactive).

Task .4 remains unstarted under the instruction to finish in-flight work and stop. Its final documentation and broader integration obligations remain open.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b65f9facc6ef7dccc247fc59814ce9a21bf64430, 618a313e93efe77eda4bb525c99ab01b15100c11
- Tests: UMPIRE_TEMPORAL_WASM=/home/agent/.cache/codex-build/cargo-target/wasm32-wasip1/release/umpire_temporal_guest.wasm CGO_ENABLED=0 mise exec -- go test -tags "test_dep integration" ./tests -run "^TestTestpilotModelSlice$" -count=1, CARGO_TARGET_DIR=../../../../../../../home/agent/.cache/codex-build/cargo-target make umpire-check-model-slice, CGO_ENABLED=0 mise exec -- go test -tags test_dep ./common/testing/testpilot -run "^TestTestpilot(DependencyBoundary|DependencyBoundaryRejectsForbiddenEdges|HelperDriverBoundary)$" -count=1, CGO_ENABLED=0 make lint-code-fast TEST_TAG=integration GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=37afaadf9e531c2c9846cf25d64f7b70e953f410, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-check, CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target make model-time, baseline: red (pre-edit GeneratedCases and seven broad umbrella failures; exact causes retained in fn216-3-legacy-comparison.md), CGO_ENABLED=0 mise exec -- go test -tags "test_dep integration" ./tests -run "^TestTestpilotGeneratedCases$" -count=1 [exit 1, unchanged baseline missing manifest], CARGO_TARGET_DIR=../../../../../../../home/agent/.cache/codex-build/cargo-target CGO_ENABLED=0 make umpire-check-live-tests [exit 2, exactly seven unchanged baseline identities and causes; slice passes twice], GOFLAGS="-skip=^TestTestpilotModelSlice$" make umpire-check-model-slice [expected rejection, no selected test], GOFLAGS="-skip=^TestTestpilotModelSlice$/^Fail$" make umpire-check-model-slice [expected rejection, Fail omitted while parent passes], UMPIRE_TEMPORAL_WASM=/missing/fn216-3-guest.wasm CGO_ENABLED=0 mise exec -- go test -tags "test_dep integration" ./tests -run "^TestTestpilotModelSlice$" -count=1 [expected missing-guest assertion, serial run], make umpire-check-model-slice [missing-function, skipped-parent and test-producer-failure mutations rejected; restored before commit; fn216-3-control-*.log], INCONCLUSIVE: initial nested Bash legacy attempt observed outer directory and no go.mod; corrected before edits, fn216-3-baseline.md, INCONCLUSIVE: concurrent preliminary lint timed out exit 124 during package loading, fn216-3-lint.log; superseded by final serial lint, INCONCLUSIVE: concurrent skipped-child control timed out exit 124 during compilation, fn216-3-control-child-skipped.log; superseded by serial intended-assertion control, INCONCLUSIVE: concurrent missing-guest control compiler signal:killed, fn216-3-control-missing-guest.log; superseded by serial file-not-found assertion
- PRs: