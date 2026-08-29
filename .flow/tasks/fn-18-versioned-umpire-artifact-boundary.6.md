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
- Encode only the parent's explicit projections of Model values/coordinates/traces, Observation
  plan/vocabulary/dispositions/diagnostics, Evidence Links, Implementation Link records/diagnostics,
  clause/property verdicts, Query summary, staged Limits, Known Gaps, and cleanup; do not serialize
  Lean constructors or add an open diagnostic payload.
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
- [ ] Field-order and canonical-order mutations cover every nested projection; Go admits the exact
  Lean bytes without knowing how Observation Evaluation, Implementation Link, or Property results
  were computed.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestResult`

## Done summary
Implemented the canonical v2 Evidence and Result transport boundary in Go and Lean with exact pretty-JSON checksum parity, closed bindings/provenance/status matrices, exhaustive mutation coverage, and no semantic evaluator logic. Final verification passed the focused/full Go suites, Lean aggregate, model lint, regression, vet, pinned lint, race, and fuzz; inherited missing `umpire-check-artifact*` targets remain deferred to task .11, and the green gate receipt was non-warrantable only because of the protected inherited `config/development.yaml` dirty state.

stage: impl-review - ran [2026-08-29T07:27:30Z..2026-08-29T07:55:31Z]
## Evidence
- Commits: 0e2088c6d205124d69777d6ffae6afeb9806bbfc, edbb3ad9c8112215f4b5e18d8ccdfd9ca5636371, 467a28f46e4ffc1c5d40f1a47e72e05620aae893
- Tests: baseline: green, mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestResult, mise exec -- go test -count=1 ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/common/artifactio/..., cd model && mise exec -- lake build Umpire.Artifact.Tests.Result Umpire.Artifact.Tests.Codecs UmpireTests, make lint-model, make umpire-check-regression, mise exec -- go vet ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/common/artifactio/..., mise exec -- ./.bin/golangci-lint-v2.13.1 run --timeout 10m --new-from-rev=edbb3ad9c --config=.github/.golangci.yml ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/common/artifactio/..., mise exec -- go test -race -count=1 ./tools/umpire/artifact/... ./tools/umpire/internal/artifactv2/... ./tools/common/artifactio/..., mise exec -- go test -count=1 ./tools/umpire/artifact -run '^$' -fuzz '^FuzzStrictJSONNoPanicOrPermissiveSuccess$' -fuzztime=5s, git diff --check, GATE_RECEIPT_NOT_WRITTEN:unittest:inherited protected config/development.yaml dirty state made receipt non-warrantable
- PRs: