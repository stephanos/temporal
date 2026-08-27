---
satisfies: [R6]
---
# fn-28-authorized-remote-staging-black-box.8 Expose the closed staging command through the existing harness

## Description

Add one exact command that invokes the fixed staging binding and existing operational harness.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-run-fixed-staging/**`, `Makefile`
**Touches:** [`tools/umpire/cmd/umpire-run-fixed-staging/**`, `Makefile`]

### Approach
- Expose only admitted-set and output-root inputs; block before work when required owner controls are unavailable and add no protected workflow or control plane.
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
- [ ] Expose only admitted-set and output-root inputs; block before work when required owner controls are unavailable and add no protected workflow or control plane.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
