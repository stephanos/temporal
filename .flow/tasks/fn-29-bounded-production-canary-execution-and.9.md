---
satisfies: [R3, R5, R8, R9, R10]
---

# fn-29-bounded-production-canary-execution-and.9 Recover lost iterations, reconcile without dispatch, and add the protected workflow

## Description
Add `tools/canary/recovery`: the mode-0600 record (invocation ID, lease workflow ID and fence, the active Run's ID prefix, the dispatch phase, the cleanup reserve, the expiry), written before the lease is taken and updated at each phase, decoded strictly; a record left open by a lost process makes the next `run` refuse. Add the closed mode `umpire-canary reconcile --recovery <file> --output <dir>`: it verifies or terminates only workflows carrying the recorded fence, releases the lease, writes the lost iteration's provenance (outcome `lost`, no receipt) and marks the record closed, or uncertain when anything could not be verified; it never prepares, dispatches, assesses or publishes a receipt. Add `.github/workflows/umpire-production-canary.yml`: `workflow_dispatch` only, `if: github.ref == 'refs/heads/main'`, environment `production-canary`, `permissions: contents: read`, a job timeout, pinned action SHAs, `make canary-build`, `umpire-canary run`, `umpire-canary reconcile` under `if: always()`, and the output directory uploaded as an artifact under `if: always()`; pinned by a regression test beside the CI workflow's.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/... ./tools/umpire/regression/`

**Files:** `tools/canary/recovery/**`, `tools/canary/cmd/umpire-canary/**`, `.github/workflows/umpire-production-canary.yml`, `tools/umpire/regression/canary_workflow_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] Reconcile terminates or verifies only fenced workflows and cannot prepare, dispatch, assess, publish a receipt or synthesize a Verdict.
- [ ] A missing, tampered, stale, crossed or uncertain record never allows a redispatch or an accepted receipt.
- [ ] The workflow is manual, protected, fixed to the trusted ref, least-privilege, bounded, and always reconciles and retains its output.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
