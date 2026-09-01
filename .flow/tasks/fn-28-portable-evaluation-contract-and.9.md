---
satisfies: [R6, R7]
---

# fn-28-portable-evaluation-contract-and.9 Run the no-Lean end-to-end disposable-cluster mini-demo
## Description

Add the sole tagged Go integration test using `testcore.NewEnv`; keep one disposable cluster and resident HTTP executor alive while running the pre-generated normal and duplicate-delivery contracts.

**Size:** L
**Files:** `tests/umpire4_portable_executor_test.go`
**Touches:** [`tests/umpire4_portable_executor_test.go`]

### Approach
- Construct the concrete attached-authority adapter from `testcore.NewEnv` without exposing `testing.T` or testcore types through production interfaces.
- Send both contracts through the same live HTTP handler/executor/cluster with fresh run identities and isolated run-owned workers/resources.
- Require normal detailed statuses plus local `pass`, and negative-control detailed statuses plus trustworthy local `fail`; assert the process invokes no Lean/Make/shell checker.

### Investigation targets

**Required** (read before coding):
- `tests/testcore.NewEnv`, its SDK client/namespace/cleanup interfaces, and existing tagged functional test conventions.
- Existing generated caller-closure path, live negative control, and parent executor/HTTP adapter.
- Project test-tag and `require` conventions.

## Acceptance
- [ ] `go test -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$'` runs the exact two live contracts through one resident process and one disposable cluster.
- [ ] The normal run passes locally; duplicate delivery fails locally for the expected semantic clause; all detailed stages and cleanup remain inspectable.
- [ ] No Lean, Make, `mise`, `lake`, nested `go test`, or generated per-verification Go process is invoked at test runtime.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
