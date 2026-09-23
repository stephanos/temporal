---
satisfies: [R3, R5, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.9 Recover lost iterations, reconcile without dispatch, and add the protected workflow

## Description
The recovery record is .4's; it only lets the same job's `reconcile` step name what was in flight, and the guard across jobs is the lease. Add the closed mode `umpire-canary reconcile --recovery <file> --output <dir>`: it acts only on the `(lease ID, run ID)` its job's recovery file names: a lease the job `took` at once, and a lease the job `found` only once that lease run is older than the invocation limit plus the cleanup reserve (younger, it exits 2 with status `lease-in-use`); it reads the close reason through the `leaseState` predicate .4 defines, verifies or terminates exactly the workflow IDs that run's `run-opened` signals name (not found on two reads an RPC timeout apart counts as never started, so closed), and then records the scope reconciled -- terminating the lease run with reason `umpire-canary: reconciled`, or, when it had closed any non-canary way (timed out, or terminated by hand with another reason), starting and at once terminating a fresh lease run with that reason -- or leaves it and reports it uncertain; a lease ID the server does not find is a clean scope; the report lists the fenced workflow IDs; it writes only its bounded reconciliation report (lost and unpublished iterations, what it closed, what it could not verify) and exits 0 closed, or `nothing-to-reconcile` when its job wrote no recovery file (preflight refused before any lease), 2 uncertain or `lease-in-use`, 3 tooling failure; it never prepares, dispatches, assesses, or writes a receipt or provenance. Add `.github/workflows/umpire-production-canary.yml`: `workflow_dispatch` only, one `concurrency` group (`umpire-production-canary`, `cancel-in-progress: false`), `if: github.ref == 'refs/heads/main'`, environment `production-canary`, `permissions: contents: read`, a GitHub-hosted runner, a 30-minute job timeout, pinned action SHAs, `make canary-build`, `umpire-canary run`, `umpire-canary reconcile` under `if: always()`, and the output directory (receipts, provenance, summaries, progress) uploaded under `if: always()`; pinned by `tools/canary/workflow_test.go`. The environment's protection (deployment branches `main` only, required reviewers) is a precondition the runbook states; the file's checks are defense in depth.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/regression/`

**Files:** `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/canary/workflow_test.go`
**Touches:** `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/canary/workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Reconcile terminates or verifies only the workflows its own job's lease run names, never touches a lease younger than an invocation could be, and cannot prepare, dispatch, assess, publish a receipt or synthesize a Verdict.
- [ ] A held lease, a missing, tampered, stale or crossed record, or an uncertain scope never allows a redispatch or an accepted receipt; only reconcile releases a held lease.
- [ ] The workflow is manual, protected, fixed to the trusted ref, least-privilege, bounded, and always reconciles and retains its output.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
