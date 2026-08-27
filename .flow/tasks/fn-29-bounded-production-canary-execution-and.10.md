---
satisfies: [R3, R4, R5, R6, R8, R9]
---
# fn-29-bounded-production-canary-execution-and.10 Build the controlled public-boundary end-to-end harness

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Prove R3-R6/R8/R9's composed production protocol through one bounded synthetic mTLS/public-gRPC harness before running exhaustive adversarial matrices.

**Size:** M
**Files:** `tools/umpire/temporal/canary/testharness/**`, `tools/umpire/remoteassessment/**`, `tools/umpire/canaryassessment/integration_test.go`, `tools/umpire/cmd/umpire-assess-production-canary/integration_test.go`
**Touches:** [tools/umpire/temporal/canary/testharness/**, tools/umpire/remoteassessment/**, tools/umpire/canaryassessment/integration_test.go, tools/umpire/cmd/umpire-assess-production-canary/integration_test.go]

### Approach
- Build one controlled TLS/public-Temporal boundary that exercises the production authority, target/routing, lease, participant, cleanup/postflight, evidence, Run Evaluation, Claim Assessment, construction, and publication interfaces without server-internal telemetry.
- Provide independent literal expected identities for one successful synthetic run plus one pre-dispatch failure, one post-dispatch incomplete run, and one semantic rejection; assert exact stage order, status 0/1/2, call partitions, cleanup reserve, and publication counts.
- Exercise lease sequential-after-terminal, simultaneous-running-conflict, stale-completed-run, caller-duplicate rejection, request-owned ambiguity validation with newly discovered run-ID fencing, and target-owned duplicate delivery with one semantic mutation.
- Route accepted-path output only to a test-owned ephemeral sink that refuses retained production destinations, mark every diagnostic synthetic, and remove it during the test lifecycle.
- State in assertions and fixtures that a schema-valid receipt is not an authenticity proof; the harness boundary is a retention/policy rule, not cryptographic differentiation.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-29-bounded-production-canary-execution-and.md` — exact stage, trust, status, and budget contract
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.4.md` — lease/reuse/call-budget lifecycle
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.8.md` — controller and status composition
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.9.md` — protected workflow/recovery boundary
- `temporaltest/server.go` — controlled public lifecycle pattern

### Key context
This task proves one composed protocol path and a small terminal partition only. Task `.11` owns adversarial/security expansion; Task `.12` owns schema and aggregate compatibility closure.

### Acceptance
- [ ] One bounded public TLS/gRPC harness reaches every production controller stage using independent expected values.
- [ ] Success, pre-dispatch failure, post-dispatch incomplete, and semantic rejection preserve exact stage/status/publication facts.
- [ ] Lease policies, discovered-run-ID ambiguity resolution, redelivery idempotency, v3-persisted cleanup reserve, and exact non-panicking worker options work end to end with zero activity polling/responses.
- [ ] Synthetic accepted-path bytes are confined to the ephemeral refusing sink and no test claims self-authentication or retains a production destination.
## Acceptance
- [ ] R3-R6/R8/R9 bounded public-boundary integration proof is complete.
- [ ] Focused harness and command integration tests pass within one implementation iteration.
- [ ] Existing integration comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
