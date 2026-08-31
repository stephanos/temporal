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
Blocked:
The pre-implementation owner-control gate failed closed. No named owner-supplied fixed staging
profile or staging harness is present in the Umpire 4 tree, repository automation/configuration,
available environment capability names, or local Git refs.

The repository's only remote execution implementation is the legacy Umpire 3 adapter/profile. It
does not qualify: it consumes the Umpire 3 experiment family, permits caller-selected endpoint,
namespace, and task queue values, and does not provide fn-28's fixed owner authority/target
preflight or postflight target-identity contract. Its canary controller is production-oriented
control-plane machinery explicitly deferred by fn-28.

Missing external prerequisites:

- a named fixed nonproduction profile and harness;
- fail-closed owner authority and target-identity preflight;
- owner-enforced concurrency one and fixed Execution/Evidence Limits;
- owner-attested isolated namespace or Run-owned resources;
- cleanup verification and postflight target identity;
- a corresponding canary dry-run binding with no production Execution authority.

Per `.plans/UMPIRE4_ORDER.md`, Umpire must not implement substitutes for these controls. No remote
mutation or implementation was attempted.
## Evidence
- Commits:
- Tests:
- PRs:
