---
satisfies: [R4, R5, R10]
---

# fn-29-bounded-production-canary-execution-and.4 Hold one lease, fence every Run and clean up exactly

## Description
Add `tools/canary/recovery`: the in-job mode-0600 record (invocation ID, the lease ID and run ID with whether the job `took` or `found` it, the current Run ID and phase), created before the lease is read, updated at each phase, and decoded strictly (unknown, repeated or case-folded keys, another version, a mode other than 0600 reject). Add `tools/canary/controller/lease.go`: before taking the lease, read the latest run under the lease ID and, when it is closed, its close event from `GetWorkflowExecutionHistory` (a describe gives `TERMINATED` without the reason) through one predicate, `leaseState`, that reconcile shares: open, or closed any way other than a termination with reason `umpire-canary: released` or `umpire-canary: reconciled` (a timeout included), is an unreconciled scope, recorded in the recovery file as `found` and refused. Otherwise the lease is a workflow with the policy's fixed ID and type on the policy's lease task queue, which no worker polls, started with `WORKFLOW_ID_CONFLICT_POLICY_FAIL` and the policy's lease run timeout, recorded in the recovery file as `took`; its run ID is the fence, and a start that collides refuses the same way. Add `fenced.go`: `FencedDriver` wraps a `testpilot.Driver`; `Open` captures the Run ID Testpilot chose, signals it to the lease (`run-opened`) and only then delegates, so the lease's history is the server-side list of every workflow ID the fence may touch; a signal that fails fails the Open before anything runs. Add `run.go`: the `PreparedCase` preflight returned, and an injected `decide(run, verdict) (accepted bool, err error)` that .8 wires to fn-26's admission and assessment; per iteration a fresh fenced Driver (`testpilotdriver.New` with the `Transport` value the controller is given, wrapped) and SDK client, one `PreparedCase.Run` bounded by the Temporal Profile's ceilings (`DefaultCeilings`: 30 seconds, 20 of cleanup), and the Driver released before the next; at most the policy's iterations, serially, stopping after the first iteration `decide` does not accept. The Case's own Nexus handler entrypoint is performed by the Driver's worker authority; the canary starts no handler of its own. Cleanup runs on every exit after the lease is held under a fresh context bounded by the reserve: close the iteration's Driver, terminate each workflow the lease's `run-opened` signals name that is still open, verify each is closed, then terminate the lease; anything unverified leaves the lease held and is recorded as uncertain, never hidden. The spec's early proof is `tests/testpilot_canary_lifecycle_test.go` (`TestTestpilotCanaryLifecycle`): resources from fn-83's `provision.Create`, the canary Case prepared under the test cluster's names, and the loop run in-process twice through the real Driver, fenced, the test passing a plaintext `Transport` directly (the Case's worker and handler entrypoints need reservations no scripted Driver grants), `decide` accepting a satisfied Verdict; both Runs close satisfied with distinct Run IDs signalled to one lease, and cleanup releases it. Lease and fence edge cases are unit tests against a fake server.

### Quick commands
`go test -count=1 -tags test_dep ./tools/canary/controller/`

**Files:** `tools/canary/recovery/**`, `tests/testpilot_canary_lifecycle_test.go`, `tools/canary/controller/fenced.go`, `tools/canary/controller/fenced_test.go`, `tools/canary/controller/lease.go`, `tools/canary/controller/lease_test.go`, `tools/canary/controller/run.go`, `tools/canary/controller/run_test.go`
**Touches:** `tools/canary/recovery/**`, `tests/testpilot_canary_lifecycle_test.go`, `tools/canary/controller/fenced.go`, `tools/canary/controller/fenced_test.go`, `tools/canary/controller/lease.go`, `tools/canary/controller/lease_test.go`, `tools/canary/controller/run.go`, `tools/canary/controller/run_test.go`

### Re-plan note (2026-09-23)
Rewritten on fn-85 (the canary set), fn-83 (provisioning), fn-22 (the recorded Run) and fn-26 (Claim Assessment); the spec's **Re-plan** section states the contracts.

## Acceptance
- [ ] A lease collision, a lease whose latest run timed out or closed without a canary termination, a stale fence, a second concurrent Run, a duplicate dispatch, a workflow ID the lease's signals do not name and an iteration past the limit each fail closed.
- [ ] A tenfold request is capped by the policy's iteration, invocation and progress limits, the Temporal Profile's `DefaultCeilings` (RPC, worker, duration, events) and fn-26's caps, never by added concurrency or unbounded retained state.
- [ ] Cleanup touches only the workflow IDs the lease's signals name, releases the lease only when each is verified closed, and records a failure or uncertainty apart from the Verdict.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
