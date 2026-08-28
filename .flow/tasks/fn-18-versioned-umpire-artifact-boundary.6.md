---
satisfies: [R1, R4, R8]
---
# fn-18-versioned-umpire-artifact-boundary.6 Persist Evidence and Result without merging semantics

## Description
Persist the output of Observation Evaluation and the separate Run Evaluation Result without copying either evaluator.


**Size:** M
**Files:** `model/Umpire/Artifact/Result.lean`, tests, and `tools/umpire/artifact/result.go`
**Touches:** [model/Umpire/Artifact/Result.lean, model/Umpire/Artifact/Tests/Result.lean, tools/umpire/artifact/result.go, tools/umpire/artifact/result_test.go]

### Approach
- Implement exactly the parent Evidence/Result formats, top-level/nested field order,
  ArtifactBinding/provenance/checksum formulas, nullable fields, closed diagnostics, and independent
  status matrix.
- An accepted Observation Result carries one complete Evidence-backed Model Trace and an Evidence Link for every established Model Fact.
- Unknown, conflict, and unsupported results carry diagnostics and no partial Model Trace.
- Result binds exact Artifacts and keeps operational, Observation Evaluation, Implementation Link, Property, Known Gap, Limit, and cleanup outcomes independent.
- Retain observation and Implementation Link Definition IDs/Behavior Fingerprints, exact
  dispositions and coordinate Evidence Links, and recompute the domain-separated
  `evaluationOutcomeChecksum` only for complete resolved satisfied/violated semantics.
- Perform no Claim Assessment and retain no prohibited raw field value.

### Investigation targets
**Required:** `model/Umpire/Observation/**`, fn-4's accepted-result rules, and fn-20's Run Evaluation contract.

## Acceptance
- [ ] Accepted Evidence is complete and every Model Fact has an Evidence Link.
- [ ] Missing links, partial traces, raw-value leakage, invalid status combinations, stale references, and checksum drift reject.
- [ ] Transport never maps raw Evidence, applies an Implementation Link, evaluates a Property, or assesses a claim.
- [ ] Every invalid observation/link/property/semantic/evaluation-checksum/nullability combination
  rejects while valid operational-failure and semantic-non-success combinations remain admissible.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestResult`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
