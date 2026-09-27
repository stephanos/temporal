---
satisfies: [R4, R9]
---
# fn-90-resolve-the-intermittent-live-testpilot.4 Resolve the umpire-run teardown: system worker and deletion asserted

## Description
Resolve (1) at its cause (R4): run the umpire-run live test on a cluster with the system worker, so
the `--create` namespace deletion finishes, and assert that both resources are gone. Unconditional:
every run pays the stuck delete today, whatever fn-90.3 measured.

**Size:** S
**Files:** `tests/testpilot_umpire_run_test.go`, `common/testing/testpilot/temporal/README.md` (Provisioning note)
**Touches:** [tests/testpilot_umpire_run_test.go, common/testing/testpilot/temporal/README.md]

### Approach
- Build the environment with `newTestpilotTestEnvironment(t, testcore.WithWorkerService("..."), <ack-interval options>)`, following `tests/namespace_test.go:39-45` (system worker plus 1 s transfer and visibility ack intervals). Probe once that the worker service combines with in-memory SQLite and does not make the test logger fail on its own error logs; if it does, record the log line in the receipt and solve it at its source, not by filtering.
- Replace the comment at `tests/testpilot_umpire_run_test.go:57-59` with the assertion: describing the namespace by its original name answers not-found within a bounded `require.Eventually` (the delete renames it to `<ns>-deleted-<id>` and reclaims it asynchronously). The failure message names the resource.
- Keep the stderr check: any `delete ...` leak line fails the test, now for the namespace as well as the endpoint.
- Record the test's run time before (from fn-90.3) and after; it should drop by the 30 s teardown wait.
- Leave `TestTestpilotUmpireRunRejectsAnUnreachableEndpoint` untouched (exit 3).
- Provisioning README (`common/testing/testpilot/temporal/README.md:68-72`): add that a `--create` namespace deletion finishes only on a cluster that runs the system worker service.
- Loop the test afterwards: 50 process-mode iterations with `make umpire-repeat-run`; any failure is triaged by signature, not retried.

### Investigation targets
**Required:**
- `tests/testpilot_umpire_run_test.go`
- `tests/testpilot_testenv_test.go`
- `tests/namespace_test.go:36-60`
- `tests/testcore/test_env.go:150-195` (`WithWorkerService`, dedicated cluster)
- `common/testing/testpilot/temporal/provision/provision.go:80-140` (release order: endpoint before namespace)

### Key context
- `frontend.allowDeleteNamespaceIfNexusEndpointTarget` defaults to false: the delete-namespace workflow refuses while an endpoint targets the namespace. Release order already deletes the endpoint first; keep it.
- No change to the umpire-run CLI, `binding` or `provision` behaviour (fn-83 contract).
## Acceptance
- [ ] The test asserts endpoint gone and namespace not-found; a forced leak (local, not committed) fails naming the resource.
- [ ] 50 process-mode iterations of `^TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint$` pass with zero failures; the receipt gives before/after rate and run time.
- [ ] `go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotUmpireRun'` passes; `make lint-code-fast` is clean.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
