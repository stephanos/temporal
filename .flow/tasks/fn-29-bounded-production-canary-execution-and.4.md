---
satisfies: [R4, R5, R10]
---

# fn-29-bounded-production-canary-execution-and.4 Hold one lease, fence every Run and clean up exactly

## Description
Add `tools/canary/controller/lease.go`: the lease is a workflow with the policy's fixed ID in the canary namespace, started with `WORKFLOW_ID_CONFLICT_POLICY_FAIL` and a run timeout of the invocation limit; its run ID is the fence, and a start that finds the ID running is a collision that refuses. Add `run.go`: `PreparedCase` from `testpilot.Prepare` once; per iteration a fresh Driver (`testpilotdriver.New` with the authority's credentials) and SDK client, the Profile's identity scope carrying the fence, one `PreparedCase.Run` bounded by the Run limit, and the Driver released before the next; at most the policy's iterations, serially; a canary-owned handler worker on the handler queue completing the operation synchronously with the Case's fixed result. Cleanup runs on every exit after the lease is held under a fresh context bounded by the reserve: stop the handler worker, terminate any workflow carrying the fence that is still open, verify each is closed, then terminate the lease; an uncertainty is recorded, never hidden.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/controller/`

**Files:** `tools/canary/controller/lease.go`, `tools/canary/controller/lease_test.go`, `tools/canary/controller/run.go`, `tools/canary/controller/run_test.go`, `tools/canary/controller/handler.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] A lease collision, a stale fence, a second concurrent Run, a duplicate dispatch, an identity outside the fence and an iteration past the limit each fail closed.
- [ ] A tenfold request is capped by the iteration, Run, invocation and progress limits, never by added concurrency or unbounded retained state.
- [ ] Cleanup touches only workflows carrying the fence, and records a failure or uncertainty apart from the Verdict.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
