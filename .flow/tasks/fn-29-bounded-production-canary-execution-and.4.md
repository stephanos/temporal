---
satisfies: [R4, R5, R10]
---

# fn-29-bounded-production-canary-execution-and.4 Hold one lease, fence every Run and clean up exactly

## Description
Add `tools/canary/controller/lease.go`: the lease is a workflow with the policy's fixed ID and type on the policy's lease task queue, which no worker polls, started with `WORKFLOW_ID_CONFLICT_POLICY_FAIL` and a 24-hour run timeout; its run ID is the fence, and a start that finds the ID running is a collision that refuses as an unreconciled scope. Add `fenced.go`: `FencedDriver` wraps a `testpilot.Driver`; `Open` captures the Run ID Testpilot chose, signals it to the lease (`run-opened`) and only then delegates, so the lease's history is the server-side list of every workflow ID the fence may touch; a signal that fails fails the Open before anything runs. Add `run.go`: the `PreparedCase` preflight returned; per iteration a fresh fenced Driver (`testpilotdriver.New` with the authority's credentials, wrapped) and SDK client, one `PreparedCase.Run` bounded by the Run limit, and the Driver released before the next; at most the policy's iterations, serially, stopping after the first iteration that is not accepted; a canary-owned handler worker on the handler queue completing the operation synchronously with the Case's fixed result. Cleanup runs on every exit after the lease is held under a fresh context bounded by the reserve: stop the handler worker, terminate each workflow the lease's `run-opened` signals name that is still open, verify each is closed, then terminate the lease; anything unverified leaves the lease held and is recorded as uncertain, never hidden. A unit test runs the prepared canary Case twice through a fenced in-process Driver: the spec's early proof.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/controller/`

**Files:** `tools/canary/controller/fenced.go`, `tools/canary/controller/fenced_test.go`, `tools/canary/controller/lease.go`, `tools/canary/controller/lease_test.go`, `tools/canary/controller/run.go`, `tools/canary/controller/run_test.go`, `tools/canary/controller/handler.go`
**Touches:** `tools/canary/controller/fenced.go`, `tools/canary/controller/fenced_test.go`, `tools/canary/controller/lease.go`, `tools/canary/controller/lease_test.go`, `tools/canary/controller/run.go`, `tools/canary/controller/run_test.go`, `tools/canary/controller/handler.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] A lease collision, a stale fence, a second concurrent Run, a duplicate dispatch, a workflow ID the lease's signals do not name and an iteration past the limit each fail closed.
- [ ] A tenfold request is capped by the iteration, Run, invocation and progress limits, never by added concurrency or unbounded retained state.
- [ ] Cleanup touches only the workflow IDs the lease's signals name, releases the lease only when each is verified closed, and records a failure or uncertainty apart from the Verdict.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
