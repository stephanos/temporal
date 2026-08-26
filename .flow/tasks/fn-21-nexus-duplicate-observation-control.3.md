---
satisfies: [R3, R7]
---
# fn-21-nexus-duplicate-observation-control.3 Realize one participant-owned duplicate observation

## Description
Implement the exact Nexus participant negative-control realization for R3/R7 behind Task `.2`'s closed binding. It performs one real cancellation lifecycle and contributes one explicitly labeled duplicate observation, never a second request chain or synthetic history event.

**Size:** M
**Files:** `tools/umpire/temporal/nexus/participant.go`, `tools/umpire/temporal/nexus/workflow.go`, `tools/umpire/temporal/nexus/binding.go`, `tools/umpire/temporal/nexus/participant_test.go`, `tools/umpire/temporal/nexus/binding_test.go`
**Touches:** [tools/umpire/temporal/nexus/participant.go, tools/umpire/temporal/nexus/workflow.go, tools/umpire/temporal/nexus/binding.go, tools/umpire/temporal/nexus/participant_test.go, tools/umpire/temporal/nexus/binding_test.go]

### Approach
- Reuse fn-19's four-command participant and exact force-close binding; select the closed negative-control branch only from the already checked program/request, never from a CLI flag, environment value, or arbitrary argument.
- Pin the workflow Nexus cancellation mode to `WaitRequested`; retain exactly one normal lifecycle chain containing `NEXUS_OPERATION_CANCEL_REQUESTED` followed by `NEXUS_OPERATION_CANCEL_REQUEST_COMPLETED` and intercept the corresponding handler receipt/correlation.
- After the completed receipt, contribute exactly one synthetic observation before participant-output closure while preserving one force-close, one cancellation request chain, mechanical callback count one, and single-attempt semantics.
- Make activation idempotence structural: one run-scoped closed state admits one transition from completed real receipt to one injected contribution; retries/timing cannot activate it.
- Test normal/faulted branches and every participant/runtime row in the spec mutation table, including no readiness, rejection/failure/timeout/cancel, duplicate activation, second request chain, synthetic history mutation, wrong correlation/program/fault, and cleanup after every acquired resource.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-19-bounded-local-temporal-execution-and.6.md:13-31` — current Nexus participant/binding contract
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md:55-59` — four commands and one force-close lifecycle
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md:65-86` — operational precedence, history, and source closure
- pinned Go SDK v1.44.0 workflow Nexus cancellation-mode and history-event APIs
- pinned Nexus Go SDK v0.6.0 idempotent asynchronous `Operation.Cancel` contract

### Key context
`WaitRequested` completes through the ordinary requested/completed cancellation event pair. Keep the injected contribution at the participant observation edge and never mutate/count that history chain as two requests.

### Acceptance
- [ ] The faulted branch proves one completed real cancellation lifecycle before one and only one injected observation.
- [ ] Normal mode emits no marker/contribution and preserves existing behavior.
- [ ] No path issues a second force-close/request chain or synthetic history event; the normal requested/completed pair remains intact.
- [ ] Missing/failed/unbound real cancellation follows the exact operational/tooling status table and cannot emit a successful synthetic claim.
- [ ] Focused participant/binding tests prove deterministic activation and cleanup without a general injection surface.
## Acceptance
- [ ] R3 exact real-cancellation/injected-observation lifecycle is implemented and bounded.
- [ ] R7 no-framework/no-new-authority boundaries hold.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
