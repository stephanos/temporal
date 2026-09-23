---
satisfies: [R3, R5, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.9 Recover lost iterations, reconcile without dispatch, and add the protected workflow

## Description
Add `tools/canary/recovery`: the in-job mode-0600 record (invocation ID, fence, the current Run ID and phase), written before the lease is taken and updated at each phase, decoded strictly; it only lets the same job's `reconcile` step name what was in flight. The guard across jobs is the lease: a lost process leaves it held, and the next `run` refuses on the collision. Add the closed mode `umpire-canary reconcile --recovery <file> --output <dir>`: it reads the lease by its fixed ID, verifies or terminates exactly the workflow IDs its `run-opened` signals name, and terminates the lease once each is verified closed, or leaves it held; it writes only its bounded reconciliation report (lost and unpublished iterations, what it closed, what it could not verify) and exits 0 closed, 2 uncertain, 3 tooling failure; it never prepares, dispatches, assesses, or writes a receipt or provenance. Add `.github/workflows/umpire-production-canary.yml`: `workflow_dispatch` only, `if: github.ref == 'refs/heads/main'`, environment `production-canary`, `permissions: contents: read`, a GitHub-hosted runner, a job timeout, pinned action SHAs, `make canary-build`, `umpire-canary run`, `umpire-canary reconcile` under `if: always()`, and only the output directory (receipts, provenance, summaries, progress) uploaded under `if: always()`, the runner-local records never; pinned by a regression test beside the CI workflow's. The environment's protection (deployment branches `main` only, required reviewers) is a precondition the runbook states; the file's checks are defense in depth.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/regression/`

**Files:** `tools/canary/recovery/**`, `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/umpire/regression/canary_workflow_test.go`
**Touches:** `tools/canary/recovery/**`, `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/umpire/regression/canary_workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Reconcile terminates or verifies only fenced workflows and cannot prepare, dispatch, assess, publish a receipt or synthesize a Verdict.
- [ ] A held lease, a missing, tampered, stale or crossed record, or an uncertain scope never allows a redispatch or an accepted receipt; only reconcile releases a held lease.
- [ ] The workflow is manual, protected, fixed to the trusted ref, least-privilege, bounded, and always reconciles and retains its output.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
