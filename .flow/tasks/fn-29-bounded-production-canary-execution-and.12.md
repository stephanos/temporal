---
satisfies: [R1, R2, R6, R7, R9]
---
# fn-29-bounded-production-canary-execution-and.12 Prove schema compatibility and aggregate regression closure

## Description
### Umpire4 reconciliation (normative)

All canary-specific policy, profiles, claims, approvals, production authority, credentials, leasing, fencing, recovery, cleanup, rate/concurrency/blast-radius controls, audit, commands, workflows, and documentation belong to the independently owned `tools/canary` module. Umpire supplies stable generic artifact, runner, participant, Run Evaluation, and Claim Assessment interfaces only; it never imports `tools/canary` and gains no canary-specific types. The Lean model may define and verify the eligible trace subset, while the standalone canary owns operational policy and consumes the same complete `ExperimentSpec`. Replace legacy `tools/umpire` canary paths and Umpire-specific canary schema extensions accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Complete R1/R2/R6/R7/R9 with the cross-language schema/version matrix and repository aggregate gates after operational adversaries pass.

**Size:** M
**Files:** `model/Umpire/Evaluation/**`, `model/Umpire/Artifact/Tests/**`, `model/Temporal/System/Execution/ProductionCanaryTests.lean`, `model/Temporal/System/Evaluation/ProductionCanaryTests.lean`, `model/Temporal/Tool/RunEvaluationTests.lean`, `tools/umpire/artifact/**`, `tools/umpire/runevaluation/**`
**Touches:** [model/Umpire/Evaluation/**, model/Umpire/Artifact/Tests/**, model/Temporal/System/Execution/ProductionCanaryTests.lean, model/Temporal/System/Evaluation/ProductionCanaryTests.lean, model/Temporal/Tool/RunEvaluationTests.lean, tools/umpire/artifact/**, tools/umpire/runevaluation/**]

### Approach
- Run independent cross-language fixtures and mutate every profile/configuration/provenance/receipt/set version, identity, status, reason, order, nullability, relation, Known Gap, release-eligibility, cardinality/token/byte, and publication edge.
- Prove local, CI, and staging profile/receipt/set fixtures and readers remain byte-identical, reject descendant versions, and preserve the six ordinary source-member bytes.
- Prove isolation/authority/release fields never enter semantic Observation/Result identity, and public evidence corruption retains exact unknown/unsupported/conflict behavior through the canonical checker.
- Run focused Lean/Go/model suites followed by the aggregate regression gates, secret scans over serialized artifacts, and an unchanged-generated-Generated View check.
- Assert that codec-valid bytes are not authenticated production provenance and that no synthetic fixture is installed as a retained accepted production claim.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.1.md` — profile v5 version boundary
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.5.md` — canonical Run Evaluation admission
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.6.md` — receipt v5 contract
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.7.md` — ArtifactSet v6 closure
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.11.md` — preceding operational/security proof

### Key context
This task owns compatibility and aggregate closure, not live protected production execution or external environment-policy verification.

### Acceptance
- [ ] Exact cross-language v5/v6 fixtures pass every identity/version/status/relation/limit mutation.
- [ ] All prior bytes/readers/source members remain unchanged and reject descendants.
- [ ] Semantic purity and public evidence status matrices pass without authority/isolation leakage.
- [ ] Aggregate regression, secret-scan, and unchanged-generated-output gates pass.
- [ ] No fixture or codec claim is presented as authenticated production origin.
## Acceptance
- [ ] R1/R2/R6/R7/R9 schema compatibility and aggregate regression closure are complete.
- [ ] Focused and aggregate Lean/Go/model checks pass.
- [ ] Existing schema and test comments are preserved.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
