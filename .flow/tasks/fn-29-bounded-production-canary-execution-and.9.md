---
satisfies: [R3, R5, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.9 Recover lost iterations, reconcile without dispatch, and add the protected workflow

## Description
The recovery record is .4's; it only lets the same job's `reconcile` step name what was in flight, and the guard across jobs is the lease. Add the closed mode `umpire-canary reconcile --recovery <file> --output <dir>`: it acts only on the `(lease ID, run ID)` its job's recovery file names: a lease the job `took` at once, and a lease the job `found` at once when that lease run is closed, and when it is open only once it is older than the invocation limit plus the cleanup reserve (younger, it exits 2 with status `lease-in-use`); it reads the close reason through the `leaseState` predicate .4 defines, verifies or terminates exactly the workflow IDs that run's `run-opened` signals name through .4's `workflowClosed`, and then records the scope reconciled -- terminating the lease run with reason `umpire-canary: reconciled`, or, when it had closed any non-canary way (timed out, or terminated by hand with another reason), starting and at once terminating a fresh lease run with that reason -- or leaves it and reports it uncertain; a lease ID the server does not find is a clean scope; the report lists the fenced workflow IDs; it writes only its bounded reconciliation report (on the `took` path, the iterations its job's record shows unpublished, as lost; on the `found` path, the fenced IDs as publication unknown, pointing at the earlier invocation's artifact; what it closed; what it could not verify) and exits 0 closed, or `nothing-to-reconcile` when its job wrote no recovery file (preflight refused before any lease) or wrote one that records no lease (reported as a lease that may be held, which the next dispatch's `found` path recovers), 2 uncertain or `lease-in-use`, 3 tooling failure; it never prepares, dispatches, assesses, or writes a receipt or provenance. Add `.github/workflows/umpire-production-canary.yml`: `workflow_dispatch` only, one `concurrency` group (`umpire-production-canary`, `cancel-in-progress: false`), `if: github.ref == 'refs/heads/main'`, environment `production-canary`, `permissions: contents: read`, a GitHub-hosted runner, a 30-minute job timeout, pinned action SHAs, `make canary-build`, `umpire-canary run`, `umpire-canary reconcile` under `if: always()`, each mode's stdout and stderr redirected into named files in the output directory, and that directory (receipts, provenance, summaries, progress) uploaded under `if: always()`, all checked by `workflow_test.go`; pinned by `tools/canary/workflow_test.go`. The environment's protection (deployment branches `main` only, required reviewers) is a precondition the runbook states; the file's checks are defense in depth.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/regression/`

**Files:** `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/canary/workflow_test.go`
**Touches:** `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/canary/workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [x] Reconcile terminates or verifies only the workflows its own job's lease run names, never touches a lease younger than an invocation could be, and cannot prepare, dispatch, assess, publish a receipt or synthesize a Verdict.
- [x] A held lease, a missing, tampered, stale or crossed record, or an uncertain scope never allows a redispatch or an accepted receipt; only reconcile releases a held lease.
- [x] The workflow is manual, protected, fixed to the trusted ref, least-privilege, bounded, and always reconciles and retains its output.

## Done summary
`controller.Reconcile` and `umpire-canary reconcile --output --recovery`: nothing-to-reconcile for no record or a record with no lease (no credential needed); a record not this job's, tampered or loosely permissioned is refused; otherwise, after scope checks, it acts only on the recorded lease run (took at once; found open only past invocation limit + reserve, else `lease-in-use`), verifies or terminates exactly its fenced workflows via `closeFenced` (shared with cleanup), and records the scope reconciled (terminate the open run, or start-and-terminate a fresh run when a closed-otherwise run is still the latest) or reports it uncertain with what an operator must close. The stdout report lists fenced/closed/terminated/unverified, lost iterations (not after a finished run), and on the found path the publication-unknown Runs with the earlier invocation and artifact name. `.github/workflows/umpire-production-canary.yml` is the manual, protected, pinned workflow; `tools/canary/workflow_test.go` pins it. Implementation review: NEEDS_WORK, then SHIP; P3 notes applied.
## Evidence
- Commits: d3f1813f7ee36f44d12e50512dd22f6b126b3bd8, e5d57337dc392d69d1f471e2c90fedaafe796ef1, 12f4bf38ab4b2c157ef2908c6436f1d0df977414
- Tests: go test -count=1 -race -tags test_dep ./tools/canary/..., go test -count=1 -tags 'test_dep integration' ./tests/ -run '^TestTestpilotCanaryLifecycle$', GOLANGCI_LINT_BASE_REV=HEAD make lint-code-fast
- PRs: