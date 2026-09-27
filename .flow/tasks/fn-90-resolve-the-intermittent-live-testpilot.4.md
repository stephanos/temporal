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
The umpire-run live test now runs on a cluster with the system worker service and 1 s transfer and visibility ack intervals. It asserts that the Nexus endpoint is gone and that the namespace's original name answers not-found within a bounded `await.Requiref`, and any `delete namespace` or `delete Nexus endpoint` leak line fails it. Each failure message names the resource. The provisioning README now says a `--create` namespace deletion finishes only on a cluster that runs the system worker.

Probing the worker service found a separate defect, fixed at its source in its own commit (08de7fb687, `tests/testcore/onebox.go`). The onebox gave frontend and history an `otellog.Logger` but not the worker, so every `WithWorkerService` cluster failed fx construction with "missing type: log.Logger". `TestNamespaceSuite` failed the same way. This file is outside the task's declared Touches, but the task asked for the probe's failure to be solved at its source.

Run time: before the fix (ed55fb0646) the test took about 36 s per iteration, 0/5 failed. After the fix it took about 6 s per iteration (5.9 s over 50 at 301e23526e, 6.7 s over 50 at bdf949c1e2), with 0/50 failures in both loops. The 30 s teardown wait is gone.

Forced leaks (local, reverted):
- A no-op namespace release fails with "the namespace umpire-run-nexus-caller umpire-run created was not deleted on exit".
- A namespace release that returns an error fails on the stderr leak line.

`TestTestpilotUmpireRunRejectsAnUnreachableEndpoint` is unchanged. The range from the base commit also contains fn-90.6 commits by another worker; they are not this task's.

stage: impl-review - ran [2026-09-27T05:05Z..2026-09-27T05:14Z] (claude:opus:high; SHIP with one P3, fixed in bdf949c1e2, re-review SHIP)
## Evidence
- Commits: 08de7fb6870718a90386566380e524a640d3ce3f, 301e23526ed0ffd7b53f32ec6ec5e2bc8edf11a0, bdf949c1e253d5f43699d7af597820d70d875473
- Tests: baseline: green (go vet -tags 'test_dep integration' ./tests; go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/...), go test -count=1 -tags 'test_dep integration' ./tests -run '^TestNamespaceSuite$/Test_NamespaceDelete_Empty$' (red before the onebox fix: fx missing otellog.Logger; green after), go test -v -count=1 -tags 'test_dep integration' ./tests -run '^TestTestpilotUmpireRun' (both pass at bdf949c1e2), forced leaks, local and reverted: namespace release no-op fails naming umpire-run-nexus-caller; namespace release error fails on the 'delete namespace' stderr leak line, make umpire-repeat-run SELECT='^TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint$' COUNT=5 MODE=process: before (ed55fb0646) 0/5 at ~36 s/iteration; after (301e23526e) 0/5 at ~6 s/iteration, make umpire-repeat-run SELECT='^TestTestpilotUmpireRunRunsACheckedInCaseAgainstAnyEndpoint$' COUNT=50 MODE=process: 0/50 at 301e23526e (5.9 s/iteration) and 0/50 at bdf949c1e2 (6.7 s/iteration), make lint-code-fast (0 issues); golangci-lint --build-tags test_dep,integration --new-from-rev=base ./tests/ (0 issues), go vet -tags 'test_dep integration' ./tests; go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/..., make umpire-check-live-tests: green, 45 passing identities, empty failure set (first attempt INCONCLUSIVE: link failed with no space left on device)
- PRs: