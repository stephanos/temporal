---
satisfies: [R5, R6, R7, R9]
---
# fn-64-umpire-case-runtime.18 Implement bounded Host reservation delivery and carrier encoding

## Description
Build the Host-local delivery ledger and Temporal start/header adapter that task6 and task7 compose with actual SDK workers.

**Size:** M
**Files:** `tools/umpire/temporal/internal/delivery/**`, focused tests and README, server README
**Touches:** [tools/umpire/temporal/internal/delivery/**, tools/umpire/temporal/server/README.md]

### Approach
- Implement one bounded, context-cooperative ledger for explicit reservation identities, trigger binding, first delivery, replay/redelivery, cancellation, terminal release and quarantine ownership. Use fake activation handles to prove lifecycle independently of SDK registration.
- Use immutable prepared carrier topology from the preceding task. Preserve source-node/ordinal routing under out-of-order workflow and Nexus arrivals; never use a cross-Run FIFO.
- Define the private versioned canonical route codec and reserved workflow/Nexus header keys. Validate bounds and exact identities without echoing route data in diagnostics.
- Add the concrete StartWorkflowExecution request carrier: clone the constructed request, reject header collisions, verify namespace/type/queue bindings, inject only delivery metadata and enforce transmitted size. Return the request for ordinary server InvokeRPC; do not wrap or alter application Value payloads.
- Pin actual Temporal workflow RunID on admission/start result and stable Nexus RequestID on first handler delivery. Reject crossed/stale retries. Expose admission data immutably to the future SDK adapter.
- Retain every accepted handle. Retire unconsumed routes after failed triggers, cancellation or parent workflow terminal state; distinguish unused reservation release from activation success and never cancel an unpinned foreign workflow ID.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/host.go` — public handle/session contracts
- `tools/umpire/internal/execution/contracts.go` — identity/ownership semantics
- `tools/umpire/temporal/server/session.go` — bounded effects and typed unary outcomes
- `tools/umpire/temporal/server/nexus.go` — context-aware serialization and recoverable claims
- `tools/umpire/temporal/server/README.md` — completion payload/bridge handoff
- `.flow/tmp/fn64-task6-investigation.md` — pinned SDK header feasibility

### Key context
This task supplies real routing/lifecycle primitives tested with fake transports/activations. Actual SDK interpreters and registrations remain task6; public composite/live integration remains task7.

## Acceptance
- [ ] Deterministic tests cover two identical concurrent Runs, reordered arrivals, absent/crossed/stale/duplicate routes, replay without second consumption, cancellation before/during admission, trigger rejection/uncertainty and unused reservation release.
- [ ] Carrier tests prove authored request immutability, reserved-key collision rejection, namespace/type/queue mismatch rejection, final byte bounds and exact round-trip route decoding; unrelated unary calls and application payloads remain unchanged.
- [ ] Race tests prove bounded lock waits, atomic Stop/admission, one terminal release, capacity retention until actual completion, and no cross-session cancellation or closed-result mutation.
- [ ] The ledger exposes a complete task6/task7 lifecycle handoff without SDK evidence or placeholder public Run APIs. Focused tagged tests/race, formatting and scoped lint pass.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: