---
satisfies: [R2, R3]
---
# fn-28-authorized-remote-staging-black-box.3 Bind the shared runner to the public-gRPC staging harness

## Description

Adapt the existing runner to the single fixed staging binding while preserving the admitted v2 Artifact and one-at-a-time lifecycle.

**Size:** M
**Files:** `tools/umpire/staging/runner.go`, `tools/umpire/staging/runner_test.go`
**Touches:** [`tools/umpire/staging/runner.go`, `tools/umpire/staging/runner_test.go`]

### Approach
- Execute through the owner-supplied harness with no endpoint/namespace/credential flags, retries, generalized remote adapter, or second runner lifecycle.
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
- [ ] Execute through the owner-supplied harness with no endpoint/namespace/credential flags, retries, generalized remote adapter, or second runner lifecycle.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
