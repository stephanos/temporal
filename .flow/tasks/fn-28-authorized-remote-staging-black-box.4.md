---
satisfies: [R4, R5]
---
# fn-28-authorized-remote-staging-black-box.4 Implement the fenced public-gRPC participant and cleanup lifecycle

## Description
Implement R4/R5's server-enforced lease, idempotent remote participant, bounded delivery evidence, and exact cleanup protocol.

**Size:** M
**Files:** `tools/umpire/temporal/remote/lease.go`, `tools/umpire/temporal/remote/participant.go`, `tools/umpire/temporal/remote/cleanup.go`, `tools/umpire/temporal/remote/lifecycle_test.go`, `tools/umpire/temporal/remote/testdata/**`
**Touches:** [tools/umpire/temporal/remote/lease.go, tools/umpire/temporal/remote/participant.go, tools/umpire/temporal/remote/cleanup.go, tools/umpire/temporal/remote/lifecycle_test.go, tools/umpire/temporal/remote/testdata/**]

### Approach
- Acquire the fixed workflow lease before starting the sole worker, using conflict-fail semantics, invocation binding, server execution timeout, zero client redispatch, and one read-only Describe for ambiguous acquisition; bind all later mutations to the returned run fence.
- Register the exact lease/caller workflows and Nexus handler on the preconfigured task queue, derive run-owned IDs from the bounded run-id, send one operation command, and collect runner receipts plus public history.
- Treat target-owned Nexus redelivery as observable operational behavior: correlate every delivery to the operation identity and use an idempotency guard that permits exactly one semantic force-close mutation; do not claim that the public SDK disables server retries.
- Apply fixed one-worker/two-workflow/one-operation-command/one-semantic-mutation/64-call/16-MiB-RawEvidence/eight-minute ceilings; evidence overflow or unresolved delivery ambiguity is non-qualified, never truncated into success.
- Run fresh-context cleanup after every post-lease exit, affect only exact fenced IDs, release/terminate the caller and lease, stop handles, verify terminal states, and repeat the target fingerprint before evidence is finalized.
- Use server timeouts as the runner-loss containment backstop; prove the adapter never mutates namespace, endpoint, task queue, deployment, search attributes, or server configuration.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.6.md` — exact participant/control binding and operational-only outputs
- `temporaltest/server.go:42-126` — worker/client lifecycle behavior for the public-boundary harness
- `tests/nexus_workflow_update_test.go:182-206` — public history and Nexus caller patterns
- `common/testing/umpire/canary/canary.go:146-240` — fresh cleanup context and bounded execution pattern
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.3.md` — opaque authority/target handles and no-side-effect boundary

### Key context
A worker must not start before lease ownership is proven. Public Nexus retry policy is target-owned, so the enforceable invariant is one client command and one idempotent semantic mutation with every observed delivery retained.

### Acceptance
- [ ] Exactly one lease winner can start the worker and send the exact operation command once.
- [ ] Target redelivery is identity-correlated, visible in evidence, and cannot repeat the semantic mutation.
- [ ] Ambiguous starts use one read-only resolution and never duplicate a client mutation.
- [ ] Every partial-start, cancel, timeout, crash, authority-loss, fence, scope, limit, cleanup, and postflight row yields the specified status with no unrelated mutation.
- [ ] Server timeouts bound runner-loss residue and focused race/public-boundary tests pass within RawEvidence v1's 16-MiB cap.

## Acceptance
- [ ] R4/R5 lease, participant, delivery evidence, cleanup, and postflight are complete.
- [ ] Bounded lifecycle, idempotency, and no-mutation matrices pass under the race detector.
- [ ] Existing participant/lifecycle comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
