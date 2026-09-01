---
satisfies: [R5, R6]
---

# fn-28-portable-evaluation-contract-and.7 Add the attached disposable-cluster authority adapter
## Description

Add the narrow adapter that lets the resident executor use a caller-owned Temporal SDK client and namespace while retaining strict ownership of Umpire-created workers and run resources.

**Size:** M
**Files:** `tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/attached_test.go`
**Touches:** [`tools/umpire/temporal/local/attached.go`, `tools/umpire/temporal/local/attached_test.go`]

### Approach
- Accept only the minimum attached-authority capabilities needed by the existing local environment; do not import `tests/testcore` into production packages.
- Treat the cluster and SDK client as borrowed, start fresh run-owned workers/task queues, and stop/delete only resources acquired by Umpire.
- Exercise the adapter with focused fakes before the tagged test supplies the concrete `testcore.NewEnv` implementation.

### Investigation targets

**Required** (read before coding):
- `tools/umpire/temporal/local/environment.go` authority and resource accounting.
- `tests/testcore.TestEnv` SDK client, namespace, worker, and cleanup ownership.
- Parent executor interface and existing runner adapter contract.

## Acceptance
- [ ] The exported attached-authority seam is minimal and has both existing loopback/fake and later testcore adapters.
- [ ] Borrowed cluster/client resources are never stopped or reported as Umpire-owned; every Umpire-created worker/resource closes exactly once.
- [ ] Drift, nil/incomplete authority, cancellation, cleanup failure, and reuse tests fail closed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
