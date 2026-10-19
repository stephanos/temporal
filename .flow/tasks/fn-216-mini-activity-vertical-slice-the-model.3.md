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
TBD

## Evidence
- Commits:
- Tests:
- PRs:

