---
satisfies: [R5]
---
# fn-28-authorized-remote-staging-black-box.7 Prove canary dry-run binding without production authority

## Description

Add a no-RPC fixture that validates the same v2 Artifact against a compiled canary binding.

**Size:** M
**Files:** `model/Temporal/System/Execution/CanaryDryRun.lean`, `model/Temporal/System/Execution/CanaryDryRunTests.lean`
**Touches:** [`model/Temporal/System/Execution/CanaryDryRun.lean`, `model/Temporal/System/Execution/CanaryDryRunTests.lean`]

### Approach
- Prove byte/fingerprint/capability compatibility and the explicit absence of credentials and production Execution authority.
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
- [ ] Prove byte/fingerprint/capability compatibility and the explicit absence of credentials and production Execution authority.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
