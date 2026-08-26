---
satisfies: [R1]
---
# fn-28-authorized-remote-staging-black-box.1 Define QualificationProfile v3 and the exact remote staging policy

## Description
Implement R1's domain-neutral v3 qualification vocabulary and the single Temporal-owned remote-staging policy without broadening v1/v2.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Umpire/Qualification.lean`, `model/Temporal/System/Qualification/RemoteStaging.lean`, `model/Temporal/System/Qualification/RemoteStagingTests.lean`, `model/Temporal/Tool/QualificationProfile.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Qualification/**, model/Umpire/Qualification.lean, model/Temporal/System/Qualification/RemoteStaging.lean, model/Temporal/System/Qualification/RemoteStagingTests.lean, model/Temporal/Tool/QualificationProfile.lean, model/TemporalModelTests.lean]

### Approach
- Extend the reusable checked profile vocabulary only with generic remote environment, authority, target/lease, public-evidence, cleanup, trust, omission, and claim values; keep every concrete Temporal meaning in the Temporal instance.
- Preserve exact local v1 and CI v2 constructors, bytes, digests, exports, and rejection boundaries; add one explicit v3 export branch for `remote-staging-public-grpc`.
- Compile the exact limits, required/forbidden authority capabilities, evidence closures, cleanup requirements, formal absence, omissions, and environment-qualified-remote claim.
- Prove reusable imports and string fixtures contain no Temporal, Nexus, staging coordinate, credential, workflow-provider, repository, or checker vocabulary.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v1 reusable/local ownership and policy invariants
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.1.md` — v2 evolution and purity pattern
- `.plans/UMPIRE4_DSL.md` — profile-qualified Result and semantic ownership rules
- `common/testing/umpire/environment_profile.go:10-157` — existing portable environment vocabulary to supersede or adapt, not duplicate
- `model/Umpire/ARCHITECTURE.md` — reusable package purity boundary

### Key context
The compiled profile may identify one Temporal environment; the reusable constructors and wire values may not. Do not add compatibility aliases or a permissive shared reader.

### Acceptance
- [ ] V3 admits only the exact generic remote policy shape and compiled Temporal instance.
- [ ] Every unknown, duplicate, contradictory, broadened, secret-bearing, or N+1 mutation rejects.
- [ ] V1/v2 fixtures and sibling exports remain byte-identical and reject v3.
- [ ] Focused Lean purity, canonicalization, digest, and mutation tests pass with comments preserved.

## Acceptance
- [ ] R1 profile vocabulary, concrete policy, export, limits, and purity boundary are complete.
- [ ] Focused reusable and Temporal Lean suites pass.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
