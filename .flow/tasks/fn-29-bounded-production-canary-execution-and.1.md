---
satisfies: [R1]
---
# fn-29-bounded-production-canary-execution-and.1 Define QualificationProfile v4 and the exact production-canary policy

## Description
Implement R1's domain-neutral v4 qualification vocabulary and the single Temporal-owned production-canary policy without broadening v1-v3.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Umpire/Qualification.lean`, `model/Temporal/System/Qualification/ProductionCanary.lean`, `model/Temporal/System/Qualification/ProductionCanaryTests.lean`, `model/Temporal/Tool/QualificationProfile.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Qualification/**, model/Umpire/Qualification.lean, model/Temporal/System/Qualification/ProductionCanary.lean, model/Temporal/System/Qualification/ProductionCanaryTests.lean, model/Temporal/Tool/QualificationProfile.lean, model/TemporalModelTests.lean]

### Approach
- Extend reusable checked vocabulary only with generic canary environment, protected authority, isolation/scope, public-evidence, cleanup, trust, omission, claim-strength, and non-release-eligibility values; keep concrete Temporal meanings in the Temporal instance.
- Preserve exact local v1, CI v2, and remote-staging v3 constructors, bytes, digests, exports, and rejection boundaries; add one explicit v4 branch.
- Compile the exact zero-fault/zero-traffic/zero-deployment-mutation policy, hard limits, required and forbidden capabilities, evidence closures, isolation trust, formal absence, omissions, and `releaseEligibility:false`.
- Prove reusable imports and fixtures contain no Temporal, Nexus, target, production coordinate, credential, provider, repository, workflow actor, or checker vocabulary.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v1 qualification ownership and policy invariants
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.1.md` — v2 evolution/purity pattern
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.1.md` — v3 remote vocabulary and prior-version closure
- `.plans/UMPIRE4_DSL.md` — profile-qualified Result and semantic ownership
- `model/Umpire/ARCHITECTURE.md` — reusable package purity boundary

### Key context
A canary environment class is reusable vocabulary; the named Temporal profile and its production target meanings are not. Non-release eligibility must be checked data, not documentation.

### Acceptance
- [ ] V4 admits only the exact generic canary policy shape and compiled Temporal instance.
- [ ] Every unknown, duplicate, contradictory, broadened, secret-bearing, or N+1 mutation rejects.
- [ ] V1-v3 fixtures and exports remain byte-identical and reject v4.
- [ ] Focused Lean purity, canonicalization, identity, and mutation tests pass with comments preserved.

## Acceptance
- [ ] R1 profile vocabulary, exact policy, versioning, limits, and purity boundary are complete.
- [ ] Focused reusable and Temporal Lean suites pass.
- [ ] Existing comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
