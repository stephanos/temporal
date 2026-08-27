---
satisfies: [R1]
---
# fn-28-authorized-remote-staging-black-box.1 Define EvaluationProfile v4 and the exact remote staging policy

## Description
Implement R1's domain-neutral v4 Claim Assessment vocabulary and the single Temporal-owned remote-staging policy without broadening v2/v2.

**Size:** M
**Files:** `model/Umpire/Evaluation/**`, `model/Umpire/Evaluation.lean`, `model/Temporal/System/Evaluation/RemoteStaging.lean`, `model/Temporal/System/Evaluation/RemoteStagingTests.lean`, `model/Temporal/Tool/EvaluationProfile.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Evaluation/**, model/Umpire/Evaluation.lean, model/Temporal/System/Evaluation/RemoteStaging.lean, model/Temporal/System/Evaluation/RemoteStagingTests.lean, model/Temporal/Tool/EvaluationProfile.lean, model/TemporalModelTests.lean]

### Approach
- Extend the reusable checked profile vocabulary only with generic remote environment, authority, target/lease, public-evidence, cleanup, trust, Known Gap, and claim values; keep every concrete Temporal meaning in the Temporal instance.
- Preserve exact local v2 and CI v3 constructors, bytes, digests, exports, and rejection boundaries; add one explicit v4 export branch for `remote-staging-public-grpc`.
- Compile the exact limits, required/forbidden authority capabilities, evidence closures, cleanup requirements, formal absence, Known Gaps, and environment-accepted-remote claim.
- Prove reusable imports and string fixtures contain no Temporal, Nexus, staging coordinate, credential, workflow-provider, repository, or checker vocabulary.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v2 reusable/local ownership and policy invariants
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.1.md` — v2 byte-identical CI parity pattern
- `.plans/UMPIRE4_DSL.md` — profile-evaluated Result and semantic ownership rules
- `common/testing/umpire/environment_profile.go:10-157` — existing portable environment vocabulary to supersede or adapt, not duplicate
- `model/Umpire/ARCHITECTURE.md` — reusable package purity boundary

### Key context
The compiled profile may identify one Temporal environment; the reusable constructors and wire values may not. Do not add compatibility aliases or a permissive shared reader.

### Acceptance
- [ ] V4 admits only the exact generic remote policy shape and compiled Temporal instance.
- [ ] Every unknown, duplicate, contradictory, broadened, secret-bearing, or N+1 mutation rejects.
- [ ] V2/v3 fixtures and sibling exports remain byte-identical and reject v3.
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
