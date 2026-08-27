---
satisfies: [R1, R2]
---
# fn-28-authorized-remote-staging-black-box.1 Freeze the fixed staging binding and byte-identical v2 subject

## Description

Define the smallest compiled binding between the exact v2 caller-closure Artifact and the existing owner-supplied staging harness.

**Size:** M
**Files:** `model/Temporal/System/Execution/StagingBinding.lean`, `model/Temporal/System/Execution/StagingBindingTests.lean`
**Touches:** [`model/Temporal/System/Execution/StagingBinding.lean`, `model/Temporal/System/Execution/StagingBindingTests.lean`]

### Approach
- Pin one fixed nonproduction binding and prove its format version, Artifact Checksum, Behavior Fingerprints, and required harness capabilities without adding an Evaluation Profile or Receipt.
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
- [ ] Pin one fixed nonproduction binding and prove its format version, Artifact Checksum, Behavior Fingerprints, and required harness capabilities without adding an Evaluation Profile or Receipt.
- [ ] Exact bindings and Limits fail closed under representative one-field and N/N+1 mutations.
- [ ] Focused tests pass, existing comments are preserved, and no deferred API or persisted format is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
