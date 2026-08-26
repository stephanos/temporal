---
satisfies: [R1, R4]
---
# fn-27-hermetic-ci-execution-and-qualification.1 Define QualificationProfile v2 and the exact CI policy

## Description
Implement the versioned reusable policy and concrete CI profile from R1/R4 without broadening local v1.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Umpire/Qualification.lean`, `model/Temporal/System/Qualification/CI.lean`, `model/Temporal/System/Qualification/CITests.lean`, `model/Temporal/Tool/QualificationProfile.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Qualification/**, model/Umpire/Qualification.lean, model/Temporal/System/Qualification/CI.lean, model/Temporal/System/Qualification/CITests.lean, model/Temporal/Tool/QualificationProfile.lean, model/TemporalModelTests.lean]

### Approach

- Keep the fn-26 v1 constructors, canonical bytes, digest, and local export branch unchanged; add separate v2 checked constructors for the closed CI class, execution boundary, provenance requirements, claim, and exact omissions.
- Define bounded `CIProvenance/v1` inert vocabulary with reusable strings/digests only; keep repository/workflow/runtime meanings in the Temporal instance.
- Compile `umpire.qualification-profile.ci-hermetic` with the exact required/forbidden capabilities and pin both its v2 profile export and workflow-definition digest.
- Extend the fixed profile sibling with one explicit `ci-hermetic` branch and v2 export handshake; retain the local v1 protocol byte-for-byte.
- Add equality, ordering, validation, digest, unknown/duplicate/contradiction, exact-limit, and v1 regression tests.

### Investigation targets

**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v1 profile and export invariants to preserve
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.1.md` — planned reusable/local ownership
- `.flow/tasks/fn-26-local-qualification-receipts-and-staged.4.md` — fixed sibling protocol
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — exact runtime/evidence/cleanup capability closure
- `.plans/UMPIRE4_DSL.md` — reusable profile-qualified Result boundary

### Acceptance

- [ ] V1 local profile/export fixtures remain byte-identical and reject v2.
- [ ] V2 accepts only the exact CI environment/claim/trust/boundary/provenance policy and rejects every malformed, contradictory, broadened, or N+1 variant.
- [ ] Reusable Umpire declarations contain no Temporal, scenario, endpoint, credential, repository, or workflow vocabulary.
- [ ] Compiled CI bytes/digests and the v2 sibling handshake are pinned by independent tests.

## Acceptance
- [ ] R1/R4 checked vocabulary and concrete profile are complete.
- [ ] Focused reusable and Temporal Lean suites pass while v1 outputs remain byte-identical.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
