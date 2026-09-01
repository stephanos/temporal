---
satisfies: [R4, R5]
---

# fn-28-portable-evaluation-contract-and.6 Deepen the resident executor and Evidence-closure lifecycle
## Description

Compose contract admission, the existing runner, explicit Evidence closure, portable evaluation, cleanup, and local decision mapping behind one small resident executor interface.

**Size:** L
**Files:** `tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/temporal/local/**`
**Touches:** [`tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/temporal/local/**`]

### Approach
- Expose one request/result seam; keep phases, adapters, resource accounting, and status mapping internal to the module.
- Reuse an attached authority across bounded requests while assigning fresh run correlations and owning only per-run workers/endpoints/workflows; never close the enclosing cluster/client.
- Wait for contract-declared terminal receipts and source closure within explicit Limits. Mark the executor poisoned after uncertain cleanup and reject further work.
- Guard `idle`/`active`/`poisoned` atomically: reject overlap as typed pre-I/O `busy`/`inconclusive`, return to idle only after complete cleanup, and never queue requests internally.

### Investigation targets

**Required** (read before coding):
- Existing `runner.Run`, runtime engine phases, `nexus.Binding`, and local authority/resource ownership.
- Parent contract/evaluator tasks and current cleanup/source-closure validation.
- Existing cancellation and failure classification tests.

## Acceptance
- [ ] A caller can execute a complete contract through one small interface without orchestrating admission, execution, evaluation, or cleanup phases.
- [ ] Multiple closed runs reuse the resident process/authority safely; run identity or resource leakage and post-uncertain-cleanup reuse fail closed.
- [ ] Eventual closure, deadline, cancellation, cleanup, race, and N/N+1 tests preserve independent statuses and never infer absence from quiet time.
- [ ] Overlap loses atomically before runtime I/O, active cancellation cannot expose idle early, and poisoned state permanently rejects reuse.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
