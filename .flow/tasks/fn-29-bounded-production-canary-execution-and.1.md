---
satisfies: [R1]
---
# fn-29-bounded-production-canary-execution-and.1 Define EvaluationProfile v5 and the exact production-canary policy

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement R1's domain-neutral v5 Claim Assessment vocabulary and the single Temporal-owned production-canary policy without broadening v2-v3.

**Size:** M
**Files:** `model/Umpire/Evaluation/**`, `model/Umpire/Evaluation.lean`, `model/Temporal/System/Evaluation/ProductionCanary.lean`, `model/Temporal/System/Evaluation/ProductionCanaryTests.lean`, `model/Temporal/Tool/EvaluationProfile.lean`, `model/TemporalModelTests.lean`
**Touches:** [model/Umpire/Evaluation/**, model/Umpire/Evaluation.lean, model/Temporal/System/Evaluation/ProductionCanary.lean, model/Temporal/System/Evaluation/ProductionCanaryTests.lean, model/Temporal/Tool/EvaluationProfile.lean, model/TemporalModelTests.lean]

### Approach
- Extend reusable checked vocabulary only with generic canary environment, protected authority, isolation/scope, public-evidence, cleanup, trust, Known Gap, claim-strength, and non-release-eligibility values; keep concrete Temporal meanings in the Temporal instance.
- Preserve exact local v2, CI v3, and remote-staging v4 constructors, bytes, digests, exports, and rejection boundaries; add one explicit v5 branch.
- Compile the exact zero-fault/zero-traffic/zero-deployment-mutation policy, hard limits, required and forbidden capabilities, evidence closures, isolation trust, formal absence, Known Gaps, and `releaseEligibility:false`.
- Prove reusable imports and fixtures contain no Temporal, Nexus, target, production coordinate, credential, provider, repository, workflow actor, or checker vocabulary.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — v2 Claim Assessment ownership and policy invariants
- `.flow/tasks/fn-27-hermetic-ci-execution-and-qualification.1.md` — v2 byte-identical CI parity pattern
- `.flow/tasks/fn-28-authorized-remote-staging-black-box.1.md` — v4 remote vocabulary and prior-version closure
- `.plans/UMPIRE4_DSL.md` — profile-evaluated Result and semantic ownership
- `model/Umpire/ARCHITECTURE.md` — reusable package purity boundary

### Key context
A canary environment class is reusable vocabulary; the named Temporal profile and its production target meanings are not. Non-release eligibility must be checked data, not documentation.

### Acceptance
- [ ] V5 admits only the exact generic canary policy shape and compiled Temporal instance.
- [ ] Every unknown, duplicate, contradictory, broadened, secret-bearing, or N+1 mutation rejects.
- [ ] V2-v4 fixtures and exports remain byte-identical and reject v4.
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
