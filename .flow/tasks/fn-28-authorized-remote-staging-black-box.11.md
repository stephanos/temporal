---
satisfies: [R6]
---
# fn-28-authorized-remote-staging-black-box.11 Document black-box staging and deferred control-plane boundaries

## Description

Synchronize the exact fixed command, owner prerequisites, Evidence boundary, Limit behavior, canary dry-run, and blocked conditions.

**Size:** M
**Files:** `docs/admin/umpire-fixed-staging.md`, `docs/README.md`, `.plans/UMPIRE4_COMPONENTS.md`
**Touches:** [`docs/admin/umpire-fixed-staging.md`, `docs/README.md`, `.plans/UMPIRE4_COMPONENTS.md`]

### Approach
- State explicitly that protected workflows, leases, recovery, Evaluation Receipts/Profiles, Claim Assessment, canary Execution, and release eligibility are deferred.
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
- [ ] State explicitly that protected workflows, leases, recovery, Evaluation Receipts/Profiles, Claim Assessment, canary Execution, and release eligibility are deferred.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
