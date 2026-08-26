---
satisfies: [R4, R5]
---
# fn-29-bounded-production-canary-execution-and.4 Implement the fenced canary participant and cleanup lifecycle

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, conformance, and qualification interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R4/R5's exact lease, idempotent canary mutation, forbidden-capability closure, cleanup, and postflight behavior by reusing the remote lifecycle seams.

**Size:** M
**Files:** `tools/umpire/temporal/remote/**`, `tools/umpire/temporal/canary/lease.go`, `tools/umpire/temporal/canary/participant.go`, `tools/umpire/temporal/canary/cleanup.go`, `tools/umpire/temporal/canary/lifecycle_test.go`, `tools/umpire/temporal/canary/testdata/**`
**Touches:** [tools/umpire/temporal/remote/**, tools/umpire/temporal/canary/lease.go, tools/umpire/temporal/canary/participant.go, tools/umpire/temporal/canary/cleanup.go, tools/umpire/temporal/canary/lifecycle_test.go, tools/umpire/temporal/canary/testdata/**]

### Approach
- Acquire the fixed canary lease before worker startup with reuse ALLOW_DUPLICATE, running-conflict FAIL, invocation binding, server timeout, zero client redispatch, and one Describe that validates fixed workflow ID, invocation binding, workflow type, task queue, running state, and other request-owned start fields after ambiguity; adopt the run ID discovered by Describe as the fence rather than requiring a pre-known server run ID.
- Register only the exact lease/caller/handler set on the preconfigured dedicated task queue; derive the caller from workflow run plus attempt, set caller reuse REJECT_DUPLICATE/running-conflict FAIL, send one operation command, and allow one idempotent semantic force-close mutation.
- Retain every target-owned delivery attempt as operational evidence and prove duplicates cannot repeat the mutation or create semantic evidence.
- Enforce one-worker/two-workflow/one-command/one-mutation/zero-fault/zero-forbidden-action/16-MiB/eight-minute ceilings plus 10 preflight, 6 lease, 10 dispatch/control, 14 evidence, and 24 unborrowable cleanup/reconcile controller RPC attempts. Persist remaining cleanup reserve in RemoteRecoveryRecord v2 while preserving staging v1 unchanged.
- Configure pinned SDK v1.44 with `WorkflowTaskPollerBehavior` and `NexusTaskPollerBehavior` from `worker.NewPollerBehaviorSimpleMaximum(worker.PollerBehaviorSimpleMaximumOptions{MaximumNumberOfPollers: 1})`, `LocalActivityWorkerOnly: true`, no activity/local-activity registrations, and `worker.NewFixedSizeTuner` with one workflow, activity, local-activity, and Nexus slot. Leave legacy MaxConcurrent poller/execution fields zero and test startup, regular/sticky workflow tasks, and Nexus handling without panic or activity polling.
- Run fresh-context cleanup after every post-lease exit, touch only exact fenced resources, verify terminal state and postflight routing, and reject any unrelated-resource or scope encounter.
- Prove the production command has no namespace/endpoint/task-queue/deployment/configuration/traffic/fault mutation capability.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.4.md` — reusable remote lease/participant/cleanup contract
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.6.md` — participant/control binding and operational-only receipts
- `temporaltest/server.go` — controlled public client/worker harness lifecycle
- `tests/nexus_workflow_update_test.go` — public history and caller observation pattern
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.3.md` — opaque target and authority handles

### Key context
No customer-facing rollback exists because the canary performs no deployment or traffic mutation. Abort and cleanup are strictly run-resource containment operations.

### Acceptance
- [ ] Only one verified lease winner may start the worker or send the operation command.
- [ ] Sequential lease start after a terminal execution succeeds, simultaneous start conflicts, stale completed runs are never adopted, ambiguous success discovers and fences the request-matching run ID, and caller identity reuse always rejects.
- [ ] Redelivery is visible and deduplicated to exactly one semantic mutation.
- [ ] Every forbidden capability is structurally absent; foreground RPC N+1 enters the cleanup reserve, and reserve/worker-bound N+1 yields exact incomplete containment.
- [ ] Exact SDK worker options start without panic, process regular/sticky workflow plus Nexus tasks under one-slot bounds, and emit no activity poll or activity-task response.
- [ ] Partial start, cancel, crash, authority loss, fence/scope violation, cleanup failure/uncertainty, and drift yield exact non-success without unrelated mutation.
## Acceptance
- [ ] R4/R5 bounded lifecycle, containment, cleanup, and postflight are complete.
- [ ] Focused race/public-boundary/idempotency/no-capability matrices pass.
- [ ] Existing participant and lifecycle comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
