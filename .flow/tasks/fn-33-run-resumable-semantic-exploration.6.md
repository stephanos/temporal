---
satisfies: [R6]
---
# fn-33-run-resumable-semantic-exploration.6 Enforce deferred concurrency and resume boundaries

## Description

Add import/API/CLI guards and synchronize architecture and operator documentation.

**Size:** M
**Files:** `tools/umpire/campaign/boundary_test.go`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `docs/README.md`
**Touches:** [`tools/umpire/campaign/boundary_test.go`, `model/ARCHITECTURE.md`, `model/Umpire/ARCHITECTURE.md`, `docs/README.md`]

### Approach
- Fail if worker pools, leases, lock files, generation graphs, checkpoints, crash recovery, resume, adaptive corpora, or generalized multi-environment orchestration appear.
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
- [ ] Fail if worker pools, leases, lock files, generation graphs, checkpoints, crash recovery, resume, adaptive corpora, or generalized multi-environment orchestration appear.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
