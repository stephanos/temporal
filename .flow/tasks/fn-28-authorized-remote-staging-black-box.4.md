---
satisfies: [R3]
---
# fn-28-authorized-remote-staging-black-box.4 Capture public Evidence and participant Execution Receipts

## Description

Collect only public gRPC facts and participant-owned Execution Receipts into the existing bounded RawEvidence shape.

**Size:** M
**Files:** `tools/umpire/staging/evidence.go`, `tools/umpire/staging/evidence_test.go`
**Touches:** [`tools/umpire/staging/evidence.go`, `tools/umpire/staging/evidence_test.go`]

### Approach
- Preserve source closure and causal order while excluding server-internal facts, authority material, and any Go-side semantic interpretation.
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
- [ ] Preserve source closure and causal order while excluding server-internal facts, authority material, and any Go-side semantic interpretation.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
