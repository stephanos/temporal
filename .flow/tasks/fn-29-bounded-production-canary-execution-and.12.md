---
satisfies: [R1, R2, R6, R7, R9]
---
# fn-29-bounded-production-canary-execution-and.12 Prove schema compatibility and aggregate regression closure

## Description
Complete R1/R2/R6/R7/R9 with the cross-language schema/version matrix and repository aggregate gates after operational adversaries pass.

**Size:** M
**Files:** `model/Umpire/Qualification/**`, `model/Umpire/Artifact/Tests/**`, `model/Temporal/System/Execution/ProductionCanaryTests.lean`, `model/Temporal/System/Qualification/ProductionCanaryTests.lean`, `model/Temporal/Tool/ConformanceTests.lean`, `tools/umpire/artifact/**`, `tools/umpire/conformance/**`
**Touches:** [model/Umpire/Qualification/**, model/Umpire/Artifact/Tests/**, model/Temporal/System/Execution/ProductionCanaryTests.lean, model/Temporal/System/Qualification/ProductionCanaryTests.lean, model/Temporal/Tool/ConformanceTests.lean, tools/umpire/artifact/**, tools/umpire/conformance/**]

### Approach
- Run independent cross-language fixtures and mutate every profile/configuration/provenance/receipt/set version, identity, status, reason, order, nullability, relation, omission, release-eligibility, cardinality/token/byte, and publication edge.
- Prove local, CI, and staging profile/receipt/set fixtures and readers remain byte-identical, reject descendant versions, and preserve the six ordinary source-member bytes.
- Prove isolation/authority/release fields never enter semantic Observation/Result identity, and public evidence corruption retains exact unknown/unsupported/conflict behavior through the canonical checker.
- Run focused Lean/Go/model suites followed by the aggregate regression gates, secret scans over serialized artifacts, and an unchanged-generated-projection check.
- Assert that codec-valid bytes are not authenticated production provenance and that no synthetic fixture is installed as a retained accepted production claim.

### Investigation targets
**Required** (read before coding):
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.1.md` — profile v4 version boundary
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.5.md` — canonical conformance admission
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.6.md` — receipt v4 contract
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.7.md` — ArtifactSet v5 closure
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.11.md` — preceding operational/security proof

### Key context
This task owns compatibility and aggregate closure, not live protected production execution or external environment-policy verification.

### Acceptance
- [ ] Exact cross-language v4/v5 fixtures pass every identity/version/status/relation/limit mutation.
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
