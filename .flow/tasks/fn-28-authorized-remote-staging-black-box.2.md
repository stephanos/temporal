---
satisfies: [R1]
---
# fn-28-authorized-remote-staging-black-box.2 Admit owner-supplied authority and target preflight

## Description

Validate the authority, target identity, isolation, concurrency-one, and remaining-Limit results supplied by the existing operational harness.

**Size:** M
**Files:** `tools/umpire/staging/preflight.go`, `tools/umpire/staging/preflight_test.go`
**Touches:** [`tools/umpire/staging/preflight.go`, `tools/umpire/staging/preflight_test.go`]

### Approach
- Reject missing, stale, broadened, or secret-bearing harness results before remote mutation; do not implement a target selector, lease, workflow, or authority loader.
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
- [ ] Reject missing, stale, broadened, or secret-bearing harness results before remote mutation; do not implement a target selector, lease, workflow, or authority loader.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
