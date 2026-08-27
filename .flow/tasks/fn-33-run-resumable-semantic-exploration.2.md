---
satisfies: [R1]
---
# fn-33-run-resumable-semantic-exploration.2 Drive Lean-owned one-at-a-time candidate selection

## Description

Connect the bridge to fn-17 so Lean chooses the next candidate and owns semantic coverage/exhaustion.

**Size:** M
**Files:** `model/Umpire/Exploration/Runtime.lean`, `model/Umpire/Exploration/RuntimeTests.lean`
**Touches:** [`model/Umpire/Exploration/Runtime.lean`, `model/Umpire/Exploration/RuntimeTests.lean`]

### Approach
- Return at most one checked v2 ExperimentSpec per next call and keep coverage coordinates, ordering, and exhaustion opaque to Go.
- Reuse the exact v2 Artifact, shared runner, and Run Evaluation boundaries named by the parent plan; do not add a parallel semantic or persistence authority.
- Add focused positive, N/N+1, stale/crossed-binding, cancellation, and mutation fixtures at the responsible boundary.

### Investigation targets

**Required** (read before coding):
- `.plans/UMPIRE4_ORDER.md` — retained prototype scope and deferred infrastructure.
- Parent Flow spec — exact contracts, Limits, failure ownership, and task boundary.
- Existing fn-18/fn-19/fn-20 implementation — Artifact, runner, cleanup, and Run Evaluation authority to reuse.

### Key context

This task implements only its retained serial/black-box slice. Deferred control-plane, concurrency, recovery, checkpoint, resume, receipt, and Claim Assessment machinery must not appear as placeholders.

## Acceptance
- [ ] Return at most one checked v2 ExperimentSpec per next call and keep coverage coordinates, ordering, and exhaustion opaque to Go.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
